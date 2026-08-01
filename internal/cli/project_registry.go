package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	"github.com/zzzzzyijie/skm/internal/store"
)

type registeredProjectRow struct {
	domain.Project
	Exists          bool `json:"exists"`
	ActivationCount int  `json:"activationCount"`
}

func (a *App) newProjectListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered local projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			projects, err := storage.LoadProjects()
			if err != nil {
				return err
			}
			state, err := storage.LoadState()
			if err != nil {
				return err
			}
			rows := registeredProjectRows(projects.Projects, state)
			return a.emit("project list", rows, func() error {
				writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
				_, _ = fmt.Fprintln(writer, "ID\tSTATUS\tACTIVATIONS\tPATH")
				for _, row := range rows {
					status := "missing"
					if row.Exists {
						status = "ready"
					}
					_, _ = fmt.Fprintf(writer, "%s\t%s\t%d\t%s\n", row.ID, status, row.ActivationCount, row.Path)
				}
				return writer.Flush()
			})
		},
	}
}

func (a *App) newProjectAddCommand() *cobra.Command {
	var projectID string
	command := &cobra.Command{
		Use:   "add <project-path>",
		Short: "Register a local project without changing its files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := canonicalProjectPath(args[0])
			if err != nil {
				return err
			}
			id := strings.TrimSpace(projectID)
			if id == "" {
				id = filepath.Base(path)
			}
			if err := validateProjectID(id); err != nil {
				return err
			}
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			value := domain.Project{ID: id, Path: path, AddedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
			err = withLock(storage, func() error {
				projects, err := storage.LoadProjects()
				if err != nil {
					return err
				}
				for _, existing := range projects.Projects {
					if existing.ID == value.ID {
						return fmt.Errorf("project %q is already registered", value.ID)
					}
					if filepath.Clean(existing.Path) == filepath.Clean(value.Path) {
						return fmt.Errorf("project path is already registered as %s", existing.ID)
					}
				}
				projects.Projects = append(projects.Projects, value)
				return storage.SaveProjects(projects)
			})
			if err != nil {
				return err
			}
			return a.emit("project add", value, func() error {
				_, err := fmt.Fprintf(a.Out, "Registered project %s at %s\n", value.ID, value.Path)
				return err
			})
		},
	}
	command.Flags().StringVar(&projectID, "name", "", "project ID (defaults to the directory name)")
	return command
}

func (a *App) newProjectShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <project>",
		Short: "Show a registered project and its managed Activations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			project, err := resolveRegisteredProject(storage, args[0])
			if err != nil {
				return err
			}
			state, err := storage.LoadState()
			if err != nil {
				return err
			}
			activations := registeredProjectActivations(state, project.Path)
			return a.emit("project show", map[string]any{"project": project, "activations": activations}, func() error {
				if _, err := fmt.Fprintf(a.Out, "ID: %s\nPath: %s\nActivations: %d\n", project.ID, project.Path, len(activations)); err != nil {
					return err
				}
				for _, activation := range activations {
					if _, err := fmt.Fprintf(a.Out, "%s\t%s\t%s\t%s\n", activation.SkillID, joinAgents(activation.Agents), activation.Mode, activation.Name); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
}

func (a *App) newProjectLinkCommand() *cobra.Command {
	return a.newProjectDeployCommand("link", domain.ModeSymlink)
}

func (a *App) newProjectCopyCommand() *cobra.Command {
	return a.newProjectDeployCommand("copy", domain.ModeCopy)
}

func (a *App) newProjectDeployCommand(commandName string, mode domain.LinkMode) *cobra.Command {
	var agentValues []string
	var dryRun bool
	command := &cobra.Command{
		Use:   commandName + " <project> <skill>",
		Short: "Deploy a Library Skill to a registered project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Skill
			var project domain.Project
			var plan domain.Plan
			err = withLock(storage, func() error {
				project, err = resolveRegisteredProject(storage, args[0])
				if err != nil {
					return err
				}
				config, err := storage.LoadConfig()
				if err != nil {
					return err
				}
				agents, err := parseAgents(agentValues, config.Defaults.Agents)
				if err != nil {
					return err
				}
				value, err = catalog.New(storage).ResolveLibrary(args[1])
				if err != nil {
					return err
				}
				state, err := storage.LoadState()
				if err != nil {
					return err
				}
				if err := ensureRegisteredProjectMode(state, project.Path, value.ID, mode); err != nil {
					return err
				}
				engine := planner.New(storage)
				engine.AddActivations(&state, []domain.Skill{value}, domain.PlacementProject, project.Path, agents, mode)
				skills, err := storage.LoadAllSkills()
				if err != nil {
					return err
				}
				plan, err = engine.Build(skills, state)
				if err != nil {
					return err
				}
				if dryRun {
					return nil
				}
				return engine.Apply(plan, &state)
			})
			if err != nil {
				return err
			}
			return a.emit("project "+commandName, map[string]any{"project": project, "skill": value, "plan": plan, "applied": !dryRun}, func() error {
				if _, err := fmt.Fprintf(a.Out, "%s %s to project %s\n", deploymentVerb(commandName), value.ID, project.ID); err != nil {
					return err
				}
				return printPlan(a.Out, plan)
			})
		},
	}
	command.Flags().StringSliceVar(&agentValues, "agent", nil, "target agent: claude,codex (defaults to both)")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "show the plan without changing files or state")
	return command
}

func (a *App) newProjectUnlinkCommand() *cobra.Command {
	var agentValues []string
	var force bool
	command := &cobra.Command{
		Use:   "unlink <project> <skill>",
		Short: "Remove a Skill from a registered project's Agent directories",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var project domain.Project
			err = withLock(storage, func() error {
				project, err = resolveRegisteredProject(storage, args[0])
				if err != nil {
					return err
				}
				state, err := storage.LoadState()
				if err != nil {
					return err
				}
				skillID, err := resolveRegisteredProjectSkill(storage, state, project.Path, args[1])
				if err != nil {
					return err
				}
				agents := make(map[domain.Agent]struct{})
				if len(agentValues) > 0 {
					parsed, err := parseAgents(agentValues, nil)
					if err != nil {
						return err
					}
					for _, agentName := range parsed {
						agents[agentName] = struct{}{}
					}
				}
				return planner.New(storage).Disable(&state, map[string]struct{}{skillID: {}}, domain.PlacementProject, project.Path, agents, force)
			})
			if err != nil {
				return err
			}
			return a.emit("project unlink", map[string]string{"project": project.ID, "skill": args[1]}, func() error {
				_, err := fmt.Fprintf(a.Out, "Unlinked %s from project %s\n", args[1], project.ID)
				return err
			})
		},
	}
	command.Flags().StringSliceVar(&agentValues, "agent", nil, "only unlink selected agents")
	command.Flags().BoolVar(&force, "force", false, "remove a managed target even if it was modified")
	return command
}

func (a *App) newProjectStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status <project>",
		Short: "Show managed deployment status for a registered project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			project, err := resolveRegisteredProject(storage, args[0])
			if err != nil {
				return err
			}
			plan, err := buildCurrentPlan(storage)
			if err != nil {
				return err
			}
			filtered := plan
			filtered.Operations = nil
			for _, operation := range plan.Operations {
				if operation.Placement == domain.PlacementProject && filepath.Clean(operation.ProjectRoot) == filepath.Clean(project.Path) {
					filtered.Operations = append(filtered.Operations, operation)
				}
			}
			return a.emit("project status", map[string]any{"project": project, "plan": filtered}, func() error { return printPlan(a.Out, filtered) })
		},
	}
}

func (a *App) newProjectUnregisterCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unregister <project>",
		Short: "Remove a project from the local registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var removed domain.Project
			err = withLock(storage, func() error {
				projects, err := storage.LoadProjects()
				if err != nil {
					return err
				}
				value, err := resolveRegisteredProjectFromList(projects.Projects, args[0])
				if err != nil {
					return err
				}
				state, err := storage.LoadState()
				if err != nil {
					return err
				}
				if len(registeredProjectActivations(state, value.Path)) > 0 {
					return fmt.Errorf("project %s still has managed Activations; unlink its Skills first", value.ID)
				}
				for _, deployment := range state.Deployments {
					if deployment.Placement == domain.PlacementProject && filepath.Clean(deployment.ProjectRoot) == filepath.Clean(value.Path) {
						return fmt.Errorf("project %s still has managed Deployments; check project status and unlink its Skills first", value.ID)
					}
				}
				removed = value
				remaining := projects.Projects[:0]
				for _, existing := range projects.Projects {
					if existing.ID != value.ID {
						remaining = append(remaining, existing)
					}
				}
				projects.Projects = remaining
				return storage.SaveProjects(projects)
			})
			if err != nil {
				return err
			}
			return a.emit("project unregister", removed, func() error {
				_, err := fmt.Fprintf(a.Out, "Unregistered project %s; project files were retained\n", removed.ID)
				return err
			})
		},
	}
}

func registeredProjectRows(projects []domain.Project, state domain.State) []registeredProjectRow {
	counts := make(map[string]int)
	for _, activation := range state.Activations {
		if activation.Placement == domain.PlacementProject {
			counts[filepath.Clean(activation.ProjectRoot)]++
		}
	}
	rows := make([]registeredProjectRow, 0, len(projects))
	for _, project := range projects {
		_, err := os.Stat(project.Path)
		rows = append(rows, registeredProjectRow{Project: project, Exists: err == nil, ActivationCount: counts[filepath.Clean(project.Path)]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

func registeredProjectActivations(state domain.State, projectRoot string) []domain.Activation {
	var result []domain.Activation
	for _, activation := range state.Activations {
		if activation.Placement == domain.PlacementProject && filepath.Clean(activation.ProjectRoot) == filepath.Clean(projectRoot) {
			result = append(result, activation)
		}
	}
	return result
}

func resolveRegisteredProject(storage *store.Store, query string) (domain.Project, error) {
	projects, err := storage.LoadProjects()
	if err != nil {
		return domain.Project{}, err
	}
	return resolveRegisteredProjectFromList(projects.Projects, query)
}

func resolveRegisteredProjectFromList(projects []domain.Project, query string) (domain.Project, error) {
	query = strings.TrimSpace(query)
	for _, project := range projects {
		if project.ID == query || filepath.Clean(project.Path) == filepath.Clean(query) {
			return project, nil
		}
	}
	return domain.Project{}, fmt.Errorf("registered project %q not found", query)
}

func resolveRegisteredProjectSkill(storage *store.Store, state domain.State, projectRoot, query string) (string, error) {
	if value, err := catalog.New(storage).ResolveLibrary(query); err == nil {
		return value.ID, nil
	}
	var matches []string
	for _, activation := range registeredProjectActivations(state, projectRoot) {
		if activation.SkillID == query || activation.Name == query {
			matches = append(matches, activation.SkillID)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		sort.Strings(matches)
		return "", fmt.Errorf("skill %q is ambiguous; use one of: %s", query, strings.Join(matches, ", "))
	}
	return "", fmt.Errorf("project Skill %q not found", query)
}

func canonicalProjectPath(value string) (string, error) {
	path, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("project path %q: %w", value, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project path %q is not a directory", value)
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(canonical), nil
}

func validateProjectID(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("project name is required")
	}
	if strings.ContainsAny(value, `/\\`) {
		return fmt.Errorf("project name %q must not contain path separators", value)
	}
	return nil
}

func ensureRegisteredProjectMode(state domain.State, projectRoot, skillID string, mode domain.LinkMode) error {
	for _, activation := range registeredProjectActivations(state, projectRoot) {
		if activation.SkillID == skillID && activation.Mode.Effective() != mode.Effective() {
			return fmt.Errorf("project Skill %s already uses mode %s; unlink it before switching to %s", skillID, activation.Mode.Effective(), mode.Effective())
		}
	}
	return nil
}

func deploymentVerb(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
