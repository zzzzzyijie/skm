package planner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/zzzzzyijie/skm/internal/adapter"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"github.com/zzzzzyijie/skm/internal/store"
)

type Engine struct {
	Store *store.Store
	Now   func() time.Time
}

func New(storage *store.Store) *Engine {
	return &Engine{Store: storage, Now: time.Now}
}

func (e *Engine) Build(skills []domain.Skill, state domain.State) (domain.Plan, error) {
	byID := make(map[string]domain.Skill, len(skills))
	for _, value := range skills {
		byID[value.ID] = value
	}
	deployments := make(map[string]domain.Deployment, len(state.Deployments))
	for _, deployment := range state.Deployments {
		deployments[deployment.Target] = deployment
	}
	var operations []domain.Operation
	for _, activation := range state.Activations {
		value, err := e.resolveActivation(activation, byID)
		if err != nil {
			return domain.Plan{}, err
		}
		for _, agentName := range activation.Agents {
			target, err := adapter.Target(agentName, activation.Placement, e.Store.Paths.UserHome, activation.ProjectRoot, value.Name)
			if err != nil {
				return domain.Plan{}, err
			}
			mode := activation.Mode.Effective()
			operation := domain.Operation{
				SkillID: value.ID, Name: value.Name, Agent: agentName,
				Placement: activation.Placement, ProjectRoot: activation.ProjectRoot,
				Target: target, SourcePath: value.Path, Mode: mode, Hash: value.Hash,
			}
			actual, err := os.Lstat(target)
			if os.IsNotExist(err) {
				if _, managed := deployments[target]; managed {
					operation.Status = domain.StatusBroken
					operation.Message = "managed target is missing"
				} else {
					operation.Status = domain.StatusCreate
				}
				operations = append(operations, operation)
				continue
			}
			if err != nil {
				return domain.Plan{}, err
			}
			deployment, managed := deployments[target]
			if !managed {
				operation.Status = domain.StatusConflictUnmanaged
				operation.Message = "target exists and is not managed by skm"
				operations = append(operations, operation)
				continue
			}
			stillManaged, err := targetMatches(target, actual, deployment.Mode, deployment.SourcePath, deployment.Hash)
			if err != nil || !stillManaged {
				operation.Status = domain.StatusConflictUnmanaged
				operation.Message = "managed target was modified outside skm"
				operations = append(operations, operation)
				continue
			}
			unchanged, err := targetMatches(target, actual, mode, value.Path, value.Hash)
			if err == nil && unchanged && deployment.SkillID == value.ID && deployment.Hash == value.Hash && deployment.Mode == mode {
				operation.Status = domain.StatusUnchanged
			} else {
				operation.Status = domain.StatusReplaceManaged
			}
			operations = append(operations, operation)
		}
	}
	collapsed := make(map[string]domain.Operation, len(operations))
	for _, operation := range operations {
		if existing, ok := collapsed[operation.Target]; ok && existing.SkillID != operation.SkillID {
			return domain.Plan{}, fmt.Errorf("multiple Skills target %s: %s and %s", operation.Target, existing.SkillID, operation.SkillID)
		}
		collapsed[operation.Target] = operation
	}
	operations = operations[:0]
	for _, operation := range collapsed {
		operations = append(operations, operation)
	}
	sort.Slice(operations, func(i, j int) bool { return operations[i].Target < operations[j].Target })
	data, _ := json.Marshal(operations)
	digest := sha256.Sum256(data)
	return domain.Plan{Digest: hex.EncodeToString(digest[:]), Operations: operations}, nil
}

func (e *Engine) resolveActivation(activation domain.Activation, byID map[string]domain.Skill) (domain.Skill, error) {
	if activation.PinnedHash == "" {
		value, ok := byID[activation.SkillID]
		if !ok {
			return domain.Skill{}, fmt.Errorf("enabled skill %q is missing from Library", activation.SkillID)
		}
		return value, nil
	}
	path := activation.PinnedPath
	if path == "" {
		path = e.Store.ObjectPath(activation.PinnedHash, activation.Name)
	}
	hash, err := fsx.HashDir(path)
	if err != nil {
		return domain.Skill{}, fmt.Errorf("read pinned Skill %s: %w", activation.SkillID, err)
	}
	if hash != activation.PinnedHash {
		return domain.Skill{}, fmt.Errorf("pinned Skill %s hash mismatch: got %s, want %s", activation.SkillID, hash, activation.PinnedHash)
	}
	return domain.Skill{ID: activation.SkillID, Name: activation.Name, Hash: hash, Path: path}, nil
}

func (e *Engine) Apply(plan domain.Plan, state *domain.State) error {
	for _, operation := range plan.Operations {
		if operation.Status == domain.StatusConflictUnmanaged {
			return fmt.Errorf("refusing to overwrite unmanaged target %s", operation.Target)
		}
	}
	for _, operation := range plan.Operations {
		switch operation.Status {
		case domain.StatusUnchanged:
			continue
		case domain.StatusCreate, domain.StatusBroken, domain.StatusReplaceManaged:
			if err := deploy(operation); err != nil {
				return fmt.Errorf("deploy %s: %w", operation.SkillID, err)
			}
			state.Deployments = upsertDeployment(state.Deployments, domain.Deployment{
				SkillID: operation.SkillID, Name: operation.Name, Agent: operation.Agent,
				Placement: operation.Placement, ProjectRoot: operation.ProjectRoot,
				Target: operation.Target, SourcePath: operation.SourcePath,
				Mode: operation.Mode, Hash: operation.Hash, UpdatedAt: e.Now().UTC(),
			})
		}
	}
	return e.Store.SaveState(*state)
}

func (e *Engine) AddActivations(state *domain.State, skills []domain.Skill, placement domain.Placement, projectRoot string, agents []domain.Agent, mode domain.LinkMode) {
	for _, value := range skills {
		activation := domain.Activation{
			SkillID: value.ID, Name: value.Name, Placement: placement,
			Agents: uniqueAgents(agents), Mode: mode, UpdatedAt: e.Now().UTC(),
		}
		if placement == domain.PlacementProject {
			activation.ProjectRoot = projectRoot
			activation.PinnedHash = value.Hash
			activation.PinnedPath = value.Path
		}
		state.Activations = upsertActivation(state.Activations, activation)
	}
}

func (e *Engine) SetProjectActivations(state *domain.State, projectRoot string, desired []domain.Activation, force bool) error {
	desiredTargets := make(map[string]struct{})
	for _, activation := range desired {
		for _, agentName := range activation.Agents {
			target, err := adapter.Target(agentName, domain.PlacementProject, e.Store.Paths.UserHome, projectRoot, activation.Name)
			if err != nil {
				return err
			}
			desiredTargets[target] = struct{}{}
		}
	}
	keptDeployments := state.Deployments[:0]
	for _, deployment := range state.Deployments {
		if deployment.Placement != domain.PlacementProject || deployment.ProjectRoot != projectRoot {
			keptDeployments = append(keptDeployments, deployment)
			continue
		}
		if _, keep := desiredTargets[deployment.Target]; keep {
			keptDeployments = append(keptDeployments, deployment)
			continue
		}
		if err := removeManaged(deployment, force); err != nil {
			return err
		}
	}
	state.Deployments = keptDeployments
	keptActivations := state.Activations[:0]
	for _, activation := range state.Activations {
		if activation.Placement != domain.PlacementProject || activation.ProjectRoot != projectRoot {
			keptActivations = append(keptActivations, activation)
		}
	}
	state.Activations = append(keptActivations, desired...)
	return e.Store.SaveState(*state)
}

func (e *Engine) Disable(state *domain.State, skillIDs map[string]struct{}, placement domain.Placement, projectRoot string, agents map[domain.Agent]struct{}, force bool) error {
	keptDeployments := state.Deployments[:0]
	for _, deployment := range state.Deployments {
		if !matchesSelection(deployment.SkillID, deployment.Placement, deployment.ProjectRoot, deployment.Agent, skillIDs, placement, projectRoot, agents) {
			keptDeployments = append(keptDeployments, deployment)
			continue
		}
		expected, err := adapter.Target(deployment.Agent, deployment.Placement, e.Store.Paths.UserHome, deployment.ProjectRoot, deployment.Name)
		if err != nil || filepath.Clean(expected) != filepath.Clean(deployment.Target) {
			return fmt.Errorf("deployment target failed safety check: %s", deployment.Target)
		}
		if err := removeManaged(deployment, force); err != nil {
			return err
		}
	}
	state.Deployments = keptDeployments
	keptActivations := state.Activations[:0]
	for _, activation := range state.Activations {
		if _, selected := skillIDs[activation.SkillID]; !selected || activation.Placement != placement || (placement == domain.PlacementProject && activation.ProjectRoot != projectRoot) {
			keptActivations = append(keptActivations, activation)
			continue
		}
		if len(agents) == 0 {
			continue
		}
		remaining := activation.Agents[:0]
		for _, agentName := range activation.Agents {
			if _, selected := agents[agentName]; !selected {
				remaining = append(remaining, agentName)
			}
		}
		if len(remaining) > 0 {
			activation.Agents = remaining
			keptActivations = append(keptActivations, activation)
		}
	}
	state.Activations = keptActivations
	return e.Store.SaveState(*state)
}

func targetMatches(path string, info os.FileInfo, mode domain.LinkMode, sourcePath, hash string) (bool, error) {
	switch mode {
	case domain.ModeSymlink:
		if info.Mode()&os.ModeSymlink == 0 {
			return false, nil
		}
		link, err := os.Readlink(path)
		if err != nil {
			return false, err
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(path), link)
		}
		return filepath.Clean(link) == filepath.Clean(sourcePath), nil
	case domain.ModeCopy:
		if !info.IsDir() {
			return false, nil
		}
		actualHash, err := fsx.HashDir(path)
		return actualHash == hash, err
	default:
		return false, fmt.Errorf("unsupported link mode %q", mode)
	}
}

func deploy(operation domain.Operation) error {
	if err := os.MkdirAll(filepath.Dir(operation.Target), 0o755); err != nil {
		return err
	}
	switch operation.Mode {
	case domain.ModeSymlink:
		temp, err := os.CreateTemp(filepath.Dir(operation.Target), ".skm-link-")
		if err != nil {
			return err
		}
		tempName := temp.Name()
		if err := temp.Close(); err != nil {
			return err
		}
		if err := os.Remove(tempName); err != nil {
			return err
		}
		defer os.RemoveAll(tempName)
		if err := os.Symlink(operation.SourcePath, tempName); err != nil {
			return err
		}
		return fsx.ReplacePath(tempName, operation.Target)
	case domain.ModeCopy:
		return fsx.CopyDirAtomic(operation.SourcePath, operation.Target)
	default:
		return fmt.Errorf("unsupported link mode %q", operation.Mode)
	}
}

func removeManaged(deployment domain.Deployment, force bool) error {
	info, err := os.Lstat(deployment.Target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !force {
		matches, matchErr := targetMatches(deployment.Target, info, deployment.Mode, deployment.SourcePath, deployment.Hash)
		if matchErr != nil || !matches {
			return fmt.Errorf("refusing to remove modified managed target %s; use --force", deployment.Target)
		}
	}
	return os.RemoveAll(deployment.Target)
}

func upsertActivation(values []domain.Activation, value domain.Activation) []domain.Activation {
	for i := range values {
		if values[i].SkillID == value.SkillID && values[i].Placement == value.Placement && values[i].ProjectRoot == value.ProjectRoot {
			value.Agents = uniqueAgents(append(values[i].Agents, value.Agents...))
			values[i] = value
			return values
		}
	}
	return append(values, value)
}

func upsertDeployment(values []domain.Deployment, value domain.Deployment) []domain.Deployment {
	for i := range values {
		if values[i].Target == value.Target {
			values[i] = value
			return values
		}
	}
	return append(values, value)
}

func uniqueAgents(values []domain.Agent) []domain.Agent {
	seen := make(map[domain.Agent]struct{}, len(values))
	result := make([]domain.Agent, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func matchesSelection(skillID string, deploymentPlacement domain.Placement, deploymentProject string, agentName domain.Agent, ids map[string]struct{}, placement domain.Placement, projectRoot string, agents map[domain.Agent]struct{}) bool {
	if _, ok := ids[skillID]; !ok || deploymentPlacement != placement {
		return false
	}
	if placement == domain.PlacementProject && deploymentProject != projectRoot {
		return false
	}
	if len(agents) == 0 {
		return true
	}
	_, ok := agents[agentName]
	return ok
}
