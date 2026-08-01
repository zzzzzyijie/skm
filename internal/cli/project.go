package cli

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"github.com/zzzzzyijie/skm/internal/planner"
	"github.com/zzzzzyijie/skm/internal/skill"
	gitSource "github.com/zzzzzyijie/skm/internal/source"
	"github.com/zzzzzyijie/skm/internal/store"
)

type projectApplyResult struct {
	Plan            domain.Plan `json:"plan"`
	SatisfiedByUser []string    `json:"satisfiedByUser"`
}

func (a *App) newProjectCommand() *cobra.Command {
	command := &cobra.Command{Use: "project", Short: "Manage project Skill requirements and vendored Skills"}
	command.AddCommand(
		a.newProjectInitCommand(),
		a.newProjectListCommand(),
		a.newProjectSkillsCommand(),
		a.newProjectAddCommand(),
		a.newProjectShowCommand(),
		a.newProjectLinkCommand(),
		a.newProjectCopyCommand(),
		a.newProjectUnlinkCommand(),
		a.newProjectStatusCommand(),
		a.newProjectUnregisterCommand(),
		a.newProjectRequireCommand(),
		a.newProjectVendorCommand(),
		a.newProjectRemoveCommand(),
		a.newProjectApplyCommand(),
	)
	return command
}

func (a *App) newProjectInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize .skm project state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			err = withLock(storage, func() error {
				project, err := storage.LoadProjectCatalog()
				if err != nil {
					return err
				}
				if err := storage.SaveProjectCatalog(project); err != nil {
					return err
				}
				return storage.SaveProjectLock(project)
			})
			if err != nil {
				return err
			}
			return a.emit("project init", map[string]string{"project": storage.Paths.ProjectRoot}, func() error {
				_, err := fmt.Fprintf(a.Out, "Initialized project at %s\n", storage.Paths.ProjectRoot)
				return err
			})
		},
	}
}

func (a *App) newProjectSkillsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "skills",
		Short: "List required and vendored project Skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			project, err := storage.LoadProjectCatalog()
			if err != nil {
				return err
			}
			data := map[string]any{"dependencies": project.Dependencies, "vendored": project.Skills}
			return a.emit("project skills", data, func() error {
				writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
				_, _ = fmt.Fprintln(writer, "TYPE\tID\tAGENTS\tREVISION/HASH")
				for _, dependency := range project.Dependencies {
					_, _ = fmt.Fprintf(writer, "require\t%s\t%s\t%s\n", dependency.ID, joinAgents(dependency.Agents), shortRevision(dependency.Revision))
				}
				for _, value := range project.Skills {
					_, _ = fmt.Fprintf(writer, "vendor\t%s\t%s\t%s\n", value.ID, joinAgents(value.Agents), shortRevision(value.Hash))
				}
				return writer.Flush()
			})
		},
	}
}

func (a *App) newProjectRequireCommand() *cobra.Command {
	var agentValues []string
	var modeValue string
	var noApply bool
	command := &cobra.Command{
		Use:   "require <skill>",
		Short: "Declare a reproducible Git-backed project dependency",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := domain.LinkMode(modeValue)
			if !mode.Valid() {
				return fmt.Errorf("invalid mode %q", modeValue)
			}
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var dependency domain.ProjectDependency
			var result projectApplyResult
			err = withLock(storage, func() error {
				config, err := storage.LoadConfig()
				if err != nil {
					return err
				}
				agents, err := parseAgents(agentValues, config.Defaults.Agents)
				if err != nil {
					return err
				}
				value, err := catalog.New(storage).ResolveLibrary(args[0])
				if err != nil {
					return err
				}
				source, err := findSource(storage, value.Source)
				if err != nil || !shareableGitURL(source.URL) {
					return fmt.Errorf("skill %s has no shareable Git source; use skm project vendor instead", value.ID)
				}
				if value.Revision == "" {
					return fmt.Errorf("skill %s has no Git revision; update its source before requiring it", value.ID)
				}
				dependency = domain.ProjectDependency{
					ID: value.ID, Name: value.Name, Source: value.Source, URL: source.URL,
					Ref: source.Ref, SourcePath: value.SourcePath, Revision: value.Revision,
					Hash: value.Hash, Tags: append([]string(nil), value.Tags...), Agents: agents, Mode: mode,
				}
				project, err := storage.LoadProjectCatalog()
				if err != nil {
					return err
				}
				if err := ensureProjectNameAvailable(project, dependency.Name, dependency.ID); err != nil {
					return err
				}
				project.Dependencies = upsertDependency(project.Dependencies, dependency)
				if err := storage.SaveProjectCatalog(project); err != nil {
					return err
				}
				if err := storage.SaveProjectLock(project); err != nil {
					return err
				}
				if noApply {
					return nil
				}
				result, err = applyProjectLocked(storage)
				return err
			})
			if err != nil {
				return err
			}
			data := map[string]any{"dependency": dependency, "apply": result, "applied": !noApply}
			return a.emit("project require", data, func() error {
				if _, err := fmt.Fprintf(a.Out, "Required %s at %s\n", dependency.ID, shortRevision(dependency.Revision)); err != nil {
					return err
				}
				if noApply {
					return nil
				}
				return printProjectApply(a.Out, result)
			})
		},
	}
	command.Flags().StringSliceVar(&agentValues, "agent", nil, "target agent: claude,codex (defaults to both)")
	command.Flags().StringVar(&modeValue, "mode", string(domain.ModeAuto), "deployment mode: auto, symlink, or copy")
	command.Flags().BoolVar(&noApply, "no-apply", false, "write project requirement without applying it")
	return command
}

func (a *App) newProjectVendorCommand() *cobra.Command {
	var agentValues []string
	var modeValue string
	var tagValues []string
	var noApply bool
	command := &cobra.Command{
		Use:   "vendor <skill>",
		Short: "Copy a Library Skill into the project for independent maintenance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := domain.LinkMode(modeValue)
			if !mode.Valid() {
				return fmt.Errorf("invalid mode %q", modeValue)
			}
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var vendored domain.Skill
			var result projectApplyResult
			err = withLock(storage, func() error {
				config, err := storage.LoadConfig()
				if err != nil {
					return err
				}
				agents, err := parseAgents(agentValues, config.Defaults.Agents)
				if err != nil {
					return err
				}
				value, err := catalog.New(storage).ResolveLibrary(args[0])
				if err != nil {
					return err
				}
				if err := checkUserNameConflict(storage, value.Name, "project/"+value.Name, value.Hash, agents); err != nil {
					return err
				}
				project, err := storage.LoadProjectCatalog()
				if err != nil {
					return err
				}
				if err := ensureProjectNameAvailable(project, value.Name, "project/"+value.Name); err != nil {
					return err
				}
				vendored, err = catalog.New(storage).Vendor(value, agents, mode, tagValues)
				if err != nil {
					return err
				}
				project, err = storage.LoadProjectCatalog()
				if err != nil {
					return err
				}
				if err := storage.SaveProjectLock(project); err != nil {
					return err
				}
				if noApply {
					return nil
				}
				result, err = applyProjectLocked(storage)
				return err
			})
			if err != nil {
				return err
			}
			data := map[string]any{"skill": vendored, "apply": result, "applied": !noApply}
			return a.emit("project vendor", data, func() error {
				if _, err := fmt.Fprintf(a.Out, "Vendored %s as %s; personal Library copy retained\n", vendored.ForkedFrom, vendored.ID); err != nil {
					return err
				}
				if noApply {
					return nil
				}
				return printProjectApply(a.Out, result)
			})
		},
	}
	command.Flags().StringSliceVar(&agentValues, "agent", nil, "target agent: claude,codex (defaults to both)")
	command.Flags().StringVar(&modeValue, "mode", string(domain.ModeAuto), "deployment mode: auto, symlink, or copy")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "project copy tag (repeatable; defaults to Library tags)")
	command.Flags().BoolVar(&noApply, "no-apply", false, "copy and record the Skill without applying it")
	return command
}

func (a *App) newProjectRemoveCommand() *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   "remove <skill>",
		Short: "Remove a project requirement or vendored Skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var removedID string
			var result projectApplyResult
			err = withLock(storage, func() error {
				project, err := storage.LoadProjectCatalog()
				if err != nil {
					return err
				}
				updated, vendoredPath, id, err := removeProjectEntry(project, args[0])
				if err != nil {
					return err
				}
				removedID = id
				if err := storage.SaveProjectCatalog(updated); err != nil {
					return err
				}
				if err := storage.SaveProjectLock(updated); err != nil {
					return err
				}
				result, err = applyProjectLockedWithForce(storage, force)
				if err != nil {
					return err
				}
				if vendoredPath != "" {
					root := filepath.Join(storage.Paths.ProjectRoot, ".skm", "skills")
					if !fsx.Within(root, vendoredPath) {
						return fmt.Errorf("refusing to remove vendored path outside %s", root)
					}
					if err := os.RemoveAll(vendoredPath); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			return a.emit("project remove", map[string]any{"id": removedID, "apply": result}, func() error {
				if _, err := fmt.Fprintf(a.Out, "Removed %s from project\n", removedID); err != nil {
					return err
				}
				return printProjectApply(a.Out, result)
			})
		},
	}
	command.Flags().BoolVar(&force, "force", false, "remove modified managed project targets")
	return command
}

func (a *App) newProjectApplyCommand() *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   "apply",
		Short: "Restore and apply project requirements and vendored Skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var result projectApplyResult
			err = withLock(storage, func() error {
				result, err = applyProjectLockedWithForce(storage, force)
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("project apply", result, func() error { return printProjectApply(a.Out, result) })
		},
	}
	command.Flags().BoolVar(&force, "force", false, "remove modified stale managed project targets")
	return command
}

func applyProjectLocked(storage *store.Store) (projectApplyResult, error) {
	return applyProjectLockedWithForce(storage, false)
}

func applyProjectLockedWithForce(storage *store.Store, force bool) (projectApplyResult, error) {
	project, err := storage.LoadProjectCatalog()
	if err != nil {
		return projectApplyResult{}, err
	}
	if err := refreshVendoredSkills(&project); err != nil {
		return projectApplyResult{}, err
	}
	pinned, err := ensureDependencySnapshots(storage, project.Dependencies)
	if err != nil {
		return projectApplyResult{}, err
	}
	state, err := storage.LoadState()
	if err != nil {
		return projectApplyResult{}, err
	}
	desired, satisfied, err := projectActivations(storage, project, pinned, state)
	if err != nil {
		return projectApplyResult{}, err
	}
	engine := planner.New(storage)
	if err := engine.SetProjectActivations(&state, storage.Paths.ProjectRoot, desired, force); err != nil {
		return projectApplyResult{}, err
	}
	if err := storage.SaveProjectCatalog(project); err != nil {
		return projectApplyResult{}, err
	}
	if err := storage.SaveProjectLock(project); err != nil {
		return projectApplyResult{}, err
	}
	skills, err := storage.LoadAllSkills()
	if err != nil {
		return projectApplyResult{}, err
	}
	plan, err := engine.Build(skills, state)
	if err != nil {
		return projectApplyResult{}, err
	}
	if err := engine.Apply(plan, &state); err != nil {
		return projectApplyResult{}, err
	}
	return projectApplyResult{Plan: plan, SatisfiedByUser: satisfied}, nil
}

func refreshVendoredSkills(project *domain.Catalog) error {
	for i := range project.Skills {
		document, err := skill.Validate(project.Skills[i].Path)
		if err != nil {
			return fmt.Errorf("validate vendored Skill %s: %w", project.Skills[i].ID, err)
		}
		project.Skills[i].Name = document.Name
		project.Skills[i].Description = document.Description
		project.Skills[i].Hash = document.Hash
		project.Skills[i].Metadata = document.Metadata
		project.Skills[i].Location = domain.LocationProject
	}
	return nil
}

func ensureDependencySnapshots(storage *store.Store, dependencies []domain.ProjectDependency) (map[string]domain.Skill, error) {
	result := make(map[string]domain.Skill, len(dependencies))
	for _, dependency := range dependencies {
		path := storage.ObjectPath(dependency.Hash, dependency.Name)
		if hash, err := fsx.HashDir(path); err == nil && hash == dependency.Hash {
			result[dependency.ID] = domain.Skill{ID: dependency.ID, Name: dependency.Name, Hash: dependency.Hash, Path: path}
			continue
		}
		if dependency.URL == "" || dependency.Revision == "" {
			return nil, fmt.Errorf("project dependency %s cannot be restored; Git URL or revision is missing", dependency.ID)
		}
		paths := []string(nil)
		if dependency.SourcePath != "" {
			paths = []string{dependency.SourcePath}
		}
		fetched, err := gitSource.NewGitManager(storage, catalog.New(storage)).FetchPinned(domain.Source{
			Name: dependency.Source, URL: dependency.URL, Ref: dependency.Revision, Paths: paths, Tags: dependency.Tags,
		})
		if err != nil {
			return nil, fmt.Errorf("restore project dependency %s: %w", dependency.ID, err)
		}
		matched := false
		for _, value := range fetched {
			if value.ID == dependency.ID && value.Hash == dependency.Hash {
				result[dependency.ID] = value
				matched = true
				break
			}
		}
		if !matched {
			return nil, fmt.Errorf("restored project dependency %s does not match locked hash %s", dependency.ID, dependency.Hash)
		}
	}
	return result, nil
}

func projectActivations(storage *store.Store, project domain.Catalog, pinned map[string]domain.Skill, state domain.State) ([]domain.Activation, []string, error) {
	library, err := storage.LoadCatalog()
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]domain.Skill, len(library.Skills))
	for _, value := range library.Skills {
		byID[value.ID] = value
	}
	var desired []domain.Activation
	var satisfied []string
	for _, dependency := range project.Dependencies {
		value, ok := pinned[dependency.ID]
		if !ok {
			return nil, nil, fmt.Errorf("missing pinned snapshot for %s", dependency.ID)
		}
		remaining, matched, err := remainingProjectAgents(dependency.ID, dependency.Name, dependency.Hash, dependency.Agents, state.Activations, byID)
		if err != nil {
			return nil, nil, err
		}
		for _, agentName := range matched {
			satisfied = append(satisfied, dependency.ID+":"+string(agentName))
		}
		if len(remaining) > 0 {
			desired = append(desired, domain.Activation{
				SkillID: dependency.ID, Name: dependency.Name, Placement: domain.PlacementProject,
				ProjectRoot: storage.Paths.ProjectRoot, Agents: remaining, Mode: dependency.Mode,
				PinnedHash: dependency.Hash, PinnedPath: value.Path,
			})
		}
	}
	for _, value := range project.Skills {
		remaining, _, err := remainingProjectAgents(value.ID, value.Name, value.Hash, value.Agents, state.Activations, byID)
		if err != nil {
			return nil, nil, err
		}
		if len(remaining) > 0 {
			desired = append(desired, domain.Activation{
				SkillID: value.ID, Name: value.Name, Placement: domain.PlacementProject,
				ProjectRoot: storage.Paths.ProjectRoot, Agents: remaining, Mode: value.Mode,
				PinnedHash: value.Hash, PinnedPath: value.Path,
			})
		}
	}
	sort.Strings(satisfied)
	return desired, satisfied, nil
}

func remainingProjectAgents(id, name, hash string, requested []domain.Agent, activations []domain.Activation, library map[string]domain.Skill) ([]domain.Agent, []domain.Agent, error) {
	var remaining []domain.Agent
	var satisfied []domain.Agent
	for _, agentName := range requested {
		matched := false
		for _, activation := range activations {
			if activation.Placement != domain.PlacementUser || !containsAgent(activation.Agents, agentName) {
				continue
			}
			activeSkill, ok := library[activation.SkillID]
			if !ok {
				return nil, nil, fmt.Errorf("enabled skill %s is missing from Library", activation.SkillID)
			}
			if activeSkill.Name != name {
				continue
			}
			matched = true
			if activeSkill.ID == id && activeSkill.Hash == hash {
				satisfied = append(satisfied, agentName)
				break
			}
			return nil, nil, fmt.Errorf("project Skill %s conflicts with enabled user Skill %s for %s; disable the user Skill or align versions", id, activeSkill.ID, agentName)
		}
		if !matched {
			remaining = append(remaining, agentName)
		}
	}
	return remaining, satisfied, nil
}

func checkUserNameConflict(storage *store.Store, name, id, hash string, agents []domain.Agent) error {
	state, err := storage.LoadState()
	if err != nil {
		return err
	}
	library, err := storage.LoadCatalog()
	if err != nil {
		return err
	}
	byID := make(map[string]domain.Skill, len(library.Skills))
	for _, value := range library.Skills {
		byID[value.ID] = value
	}
	_, _, err = remainingProjectAgents(id, name, hash, agents, state.Activations, byID)
	return err
}

func findSource(storage *store.Store, name string) (domain.Source, error) {
	sources, err := storage.LoadSources()
	if err != nil {
		return domain.Source{}, err
	}
	for _, value := range sources.Sources {
		if value.Name == name {
			return value, nil
		}
	}
	return domain.Source{}, fmt.Errorf("Git source %q is not configured", name)
}

func shareableGitURL(value string) bool {
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme != "" && parsed.Scheme != "file" {
		return true
	}
	return strings.Contains(value, "@") && strings.Contains(value, ":") && !filepath.IsAbs(value)
}

func ensureProjectNameAvailable(project domain.Catalog, name, replacingID string) error {
	for _, dependency := range project.Dependencies {
		if dependency.Name == name && dependency.ID != replacingID {
			return fmt.Errorf("project already contains same-name Skill %s", dependency.ID)
		}
	}
	for _, value := range project.Skills {
		if value.Name == name && value.ID != replacingID {
			return fmt.Errorf("project already contains same-name Skill %s", value.ID)
		}
	}
	return nil
}

func upsertDependency(values []domain.ProjectDependency, value domain.ProjectDependency) []domain.ProjectDependency {
	for i := range values {
		if values[i].ID == value.ID {
			values[i] = value
			return values
		}
	}
	return append(values, value)
}

func removeProjectEntry(project domain.Catalog, query string) (domain.Catalog, string, string, error) {
	var matches []string
	for _, dependency := range project.Dependencies {
		if dependency.ID == query || (!strings.Contains(query, "/") && dependency.Name == query) {
			matches = append(matches, dependency.ID)
		}
	}
	for _, value := range project.Skills {
		if value.ID == query || (!strings.Contains(query, "/") && value.Name == query) {
			matches = append(matches, value.ID)
		}
	}
	if len(matches) == 0 {
		return project, "", "", fmt.Errorf("project Skill %q not found", query)
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		return project, "", "", fmt.Errorf("project Skill %q is ambiguous; use one of: %s", query, strings.Join(matches, ", "))
	}
	id := matches[0]
	dependencies := project.Dependencies[:0]
	for _, dependency := range project.Dependencies {
		if dependency.ID != id {
			dependencies = append(dependencies, dependency)
		}
	}
	project.Dependencies = dependencies
	var vendoredPath string
	skills := project.Skills[:0]
	for _, value := range project.Skills {
		if value.ID == id {
			vendoredPath = value.Path
			continue
		}
		skills = append(skills, value)
	}
	project.Skills = skills
	return project, vendoredPath, id, nil
}

func containsAgent(values []domain.Agent, target domain.Agent) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func joinAgents(values []domain.Agent) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = string(value)
	}
	return strings.Join(parts, ",")
}

func printProjectApply(out interface{ Write([]byte) (int, error) }, result projectApplyResult) error {
	if len(result.SatisfiedByUser) > 0 {
		if _, err := fmt.Fprintf(out, "Satisfied by user activation: %s\n", strings.Join(result.SatisfiedByUser, ", ")); err != nil {
			return err
		}
	}
	return printPlan(out, result.Plan)
}
