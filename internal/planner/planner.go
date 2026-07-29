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
	for _, installation := range state.Installations {
		value, ok := byID[installation.SkillID]
		if !ok {
			return domain.Plan{}, fmt.Errorf("installed skill %q is missing from catalog", installation.SkillID)
		}
		for _, agentName := range installation.Agents {
			target, err := adapter.Target(agentName, installation.Scope, e.Store.Paths.UserHome, installation.ProjectRoot, value.Name)
			if err != nil {
				return domain.Plan{}, err
			}
			mode := installation.Mode.Effective()
			operation := domain.Operation{
				SkillID:     value.ID,
				Name:        value.Name,
				Agent:       agentName,
				Scope:       installation.Scope,
				ProjectRoot: installation.ProjectRoot,
				Target:      target,
				SourcePath:  value.Path,
				Mode:        mode,
				Hash:        value.Hash,
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
		existing, ok := collapsed[operation.Target]
		if !ok || operation.Scope.Priority() > existing.Scope.Priority() {
			collapsed[operation.Target] = operation
			continue
		}
		if operation.Scope.Priority() == existing.Scope.Priority() && operation.SkillID != existing.SkillID {
			return domain.Plan{}, fmt.Errorf("multiple Skills target %s at the same scope: %s and %s", operation.Target, existing.SkillID, operation.SkillID)
		}
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
				SkillID:     operation.SkillID,
				Name:        operation.Name,
				Agent:       operation.Agent,
				Scope:       operation.Scope,
				ProjectRoot: operation.ProjectRoot,
				Target:      operation.Target,
				SourcePath:  operation.SourcePath,
				Mode:        operation.Mode,
				Hash:        operation.Hash,
				UpdatedAt:   e.Now().UTC(),
			})
		}
	}
	return e.Store.SaveState(*state)
}

func (e *Engine) AddInstallations(state *domain.State, skills []domain.Skill, scope domain.Scope, projectRoot string, agents []domain.Agent, mode domain.LinkMode) {
	for _, value := range skills {
		installation := domain.Installation{
			SkillID:   value.ID,
			Name:      value.Name,
			Scope:     scope,
			Agents:    uniqueAgents(agents),
			Mode:      mode,
			UpdatedAt: e.Now().UTC(),
		}
		if scope == domain.ScopeProject {
			installation.ProjectRoot = projectRoot
		}
		state.Installations = upsertInstallation(state.Installations, installation)
	}
}

func (e *Engine) Unlink(state *domain.State, skillIDs map[string]struct{}, scope domain.Scope, projectRoot string, agents map[domain.Agent]struct{}, force bool) error {
	keptDeployments := state.Deployments[:0]
	for _, deployment := range state.Deployments {
		if !matchesSelection(deployment.SkillID, deployment.Scope, deployment.ProjectRoot, deployment.Agent, skillIDs, scope, projectRoot, agents) {
			keptDeployments = append(keptDeployments, deployment)
			continue
		}
		expected, err := adapter.Target(deployment.Agent, deployment.Scope, e.Store.Paths.UserHome, deployment.ProjectRoot, deployment.Name)
		if err != nil || filepath.Clean(expected) != filepath.Clean(deployment.Target) {
			return fmt.Errorf("deployment target failed safety check: %s", deployment.Target)
		}
		if err := removeManaged(deployment, force); err != nil {
			return err
		}
	}
	state.Deployments = keptDeployments
	keptInstallations := state.Installations[:0]
	for _, installation := range state.Installations {
		if _, selected := skillIDs[installation.SkillID]; !selected || installation.Scope != scope || (scope == domain.ScopeProject && installation.ProjectRoot != projectRoot) {
			keptInstallations = append(keptInstallations, installation)
			continue
		}
		if len(agents) == 0 {
			continue
		}
		remaining := installation.Agents[:0]
		for _, agentName := range installation.Agents {
			if _, selected := agents[agentName]; !selected {
				remaining = append(remaining, agentName)
			}
		}
		if len(remaining) > 0 {
			installation.Agents = remaining
			keptInstallations = append(keptInstallations, installation)
		}
	}
	state.Installations = keptInstallations
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

func upsertInstallation(values []domain.Installation, value domain.Installation) []domain.Installation {
	for i := range values {
		if values[i].SkillID == value.SkillID && values[i].Scope == value.Scope && values[i].ProjectRoot == value.ProjectRoot {
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

func matchesSelection(skillID string, deploymentScope domain.Scope, deploymentProject string, agentName domain.Agent, ids map[string]struct{}, scope domain.Scope, projectRoot string, agents map[domain.Agent]struct{}) bool {
	if _, ok := ids[skillID]; !ok || deploymentScope != scope {
		return false
	}
	if scope == domain.ScopeProject && deploymentProject != projectRoot {
		return false
	}
	if len(agents) == 0 {
		return true
	}
	_, ok := agents[agentName]
	return ok
}
