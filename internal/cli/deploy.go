package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/adapter"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
	"github.com/zzzzzyijie/skm/internal/planner"
	"github.com/zzzzzyijie/skm/internal/store"
	"github.com/zzzzzyijie/skm/internal/tags"
)

func (a *App) newEnableCommand() *cobra.Command {
	var agentValues []string
	var tagValues []string
	var modeValue string
	var dryRun bool
	command := &cobra.Command{
		Use:     "enable [skill...]",
		Aliases: []string{"link"},
		Short:   "Enable personal Library Skills for Claude Code or Codex",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := domain.LinkMode(modeValue)
			if !mode.Valid() {
				return fmt.Errorf("invalid mode %q", modeValue)
			}
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var plan domain.Plan
			err = withLock(storage, func() error {
				config, err := storage.LoadConfig()
				if err != nil {
					return err
				}
				agents, err := parseAgents(agentValues, config.Defaults.Agents)
				if err != nil {
					return err
				}
				selected, err := selectLibrarySkills(catalog.New(storage), args, tagValues)
				if err != nil {
					return err
				}
				state, err := storage.LoadState()
				if err != nil {
					return err
				}
				engine := planner.New(storage)
				engine.AddActivations(&state, selected, domain.PlacementUser, "", agents, mode)
				skills, err := storage.LoadAllSkills()
				if err != nil {
					return err
				}
				// Validate the complete desired state so a targeted enable cannot
				// leave conflicting same-name Activations behind.
				if _, err := engine.Build(skills, state); err != nil {
					return err
				}
				selectedIDs := make(map[string]struct{}, len(selected))
				for _, value := range selected {
					selectedIDs[value.ID] = struct{}{}
				}
				planState := domain.State{Version: state.Version, Deployments: state.Deployments}
				for _, activation := range state.Activations {
					if _, ok := selectedIDs[activation.SkillID]; ok && activation.Placement == domain.PlacementUser {
						planState.Activations = append(planState.Activations, activation)
					}
				}
				plan, err = engine.Build(skills, planState)
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
			return a.emit("enable", plan, func() error { return printPlan(a.Out, plan) })
		},
	}
	command.Flags().StringSliceVar(&agentValues, "agent", nil, "target agent: claude,codex (defaults to both)")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "select personal Library Skills by tag (repeatable, AND semantics)")
	command.Flags().StringVar(&modeValue, "mode", string(domain.ModeAuto), "deployment mode: auto, symlink, or copy")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "show the plan without changing files or state")
	return command
}

func (a *App) newDisableCommand() *cobra.Command {
	var agentValues []string
	var tagValues []string
	var force bool
	command := &cobra.Command{
		Use:     "disable [skill...]",
		Aliases: []string{"unlink"},
		Short:   "Disable personal Library Skills without removing them",
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var selected []domain.Skill
			disabledCount := 0
			projectActivations := make(map[string][]string)
			err = withLock(storage, func() error {
				selected, err = selectLibrarySkills(catalog.New(storage), args, tagValues)
				if err != nil {
					return err
				}
				ids := make(map[string]struct{}, len(selected))
				for _, value := range selected {
					ids[value.ID] = struct{}{}
				}
				agents := make(map[domain.Agent]struct{})
				if len(agentValues) > 0 {
					parsed, err := parseAgents(agentValues, nil)
					if err != nil {
						return err
					}
					for _, value := range parsed {
						agents[value] = struct{}{}
					}
				}
				state, err := storage.LoadState()
				if err != nil {
					return err
				}
				for _, activation := range state.Activations {
					if _, ok := ids[activation.SkillID]; !ok {
						continue
					}
					if activation.Placement == domain.PlacementProject {
						projectActivations[activation.ProjectRoot] = append(projectActivations[activation.ProjectRoot], activation.SkillID)
						continue
					}
					if activationMatchesAgents(activation, agents) {
						disabledCount++
					}
				}
				return planner.New(storage).Disable(&state, ids, domain.PlacementUser, "", agents, force)
			})
			if err != nil {
				return err
			}
			return a.emit("disable", selected, func() error {
				if _, err := fmt.Fprintf(a.Out, "Disabled user Activation(s) for %d Library Skill(s)\n", disabledCount); err != nil {
					return err
				}
				if disabledCount != 0 || len(projectActivations) == 0 {
					return nil
				}
				roots := make([]string, 0, len(projectActivations))
				for root := range projectActivations {
					roots = append(roots, root)
				}
				sort.Strings(roots)
				for _, root := range roots {
					skillIDs := uniqueSorted(projectActivations[root])
					for _, skillID := range skillIDs {
						if _, err := fmt.Fprintf(a.Out, "%s remains enabled by project %s; remove it with: skm --project %q project remove %s\n", skillID, root, root, skillID); err != nil {
							return err
						}
					}
				}
				return nil
			})
		},
	}
	command.Flags().StringSliceVar(&agentValues, "agent", nil, "only disable selected agents")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "select personal Library Skills by tag")
	command.Flags().BoolVar(&force, "force", false, "remove a managed target even if it was modified")
	return command
}

func activationMatchesAgents(activation domain.Activation, agents map[domain.Agent]struct{}) bool {
	if len(agents) == 0 {
		return true
	}
	for _, agentName := range activation.Agents {
		if _, ok := agents[agentName]; ok {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (a *App) newPlanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show changes needed to reach the desired activation state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			plan, err := buildCurrentPlan(storage)
			if err != nil {
				return err
			}
			return a.emit("plan", plan, func() error { return printPlan(a.Out, plan) })
		},
	}
}

func (a *App) newApplyCommand() *cobra.Command {
	var expectedDigest string
	command := &cobra.Command{
		Use:   "apply",
		Short: "Apply the current personal and project activation plan",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var plan domain.Plan
			err = withLock(storage, func() error {
				state, err := storage.LoadState()
				if err != nil {
					return err
				}
				skills, err := storage.LoadAllSkills()
				if err != nil {
					return err
				}
				engine := planner.New(storage)
				plan, err = engine.Build(skills, state)
				if err != nil {
					return err
				}
				if expectedDigest != "" && expectedDigest != plan.Digest {
					return fmt.Errorf("plan digest changed: expected %s, got %s", expectedDigest, plan.Digest)
				}
				return engine.Apply(plan, &state)
			})
			if err != nil {
				return err
			}
			return a.emit("apply", plan, func() error { return printPlan(a.Out, plan) })
		},
	}
	command.Flags().StringVar(&expectedDigest, "digest", "", "only apply a plan with this digest")
	return command
}

func (a *App) newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show managed activation status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			plan, err := buildCurrentPlan(storage)
			if err != nil {
				return err
			}
			return a.emit("status", plan, func() error { return printPlan(a.Out, plan) })
		},
	}
}

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (a *App) newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check skm, optional Git, Library, and Agent paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			checks := []doctorCheck{{Name: "skm-home", Status: "ok", Message: storage.Paths.Home}}
			sources, err := storage.LoadSources()
			if err != nil {
				return err
			}
			if path, err := exec.LookPath("git"); err != nil {
				status := "optional"
				if len(sources.Sources) > 0 {
					status = "error"
				}
				checks = append(checks, doctorCheck{Name: "git", Status: status, Message: "required only for configured Git sources"})
			} else {
				checks = append(checks, doctorCheck{Name: "git", Status: "ok", Message: path})
			}
			for _, agentName := range []domain.Agent{domain.AgentClaude, domain.AgentCodex} {
				target, _ := adapter.Target(agentName, domain.PlacementUser, storage.Paths.UserHome, storage.Paths.ProjectRoot, "probe")
				directory := filepath.Dir(target)
				status := "not-created"
				if _, err := os.Stat(directory); err == nil {
					status = "ok"
				}
				checks = append(checks, doctorCheck{Name: string(agentName), Status: status, Message: directory})
			}
			values, err := storage.LoadAllSkills()
			if err != nil {
				return err
			}
			for _, value := range values {
				hash, hashErr := fsx.HashDir(value.Path)
				if hashErr != nil || hash != value.Hash {
					checks = append(checks, doctorCheck{Name: value.ID, Status: "error", Message: "Skill content is missing or modified"})
				}
			}
			return a.emit("doctor", checks, func() error {
				writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
				_, _ = fmt.Fprintln(writer, "CHECK\tSTATUS\tDETAIL")
				for _, check := range checks {
					_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", check.Name, check.Status, check.Message)
				}
				return writer.Flush()
			})
		},
	}
}

func selectLibrarySkills(manager *catalog.Manager, args, tagValues []string) ([]domain.Skill, error) {
	if len(args) == 0 && len(tagValues) == 0 {
		return nil, fmt.Errorf("provide at least one Skill or --tag")
	}
	if len(args) == 0 {
		values, err := manager.List(domain.LocationLibrary, tagValues)
		if err != nil {
			return nil, err
		}
		if len(values) == 0 {
			return nil, fmt.Errorf("no Library Skills match the selected tags")
		}
		return values, nil
	}
	var required []string
	var err error
	if len(tagValues) > 0 {
		required, err = tags.Normalize(tagValues, nil)
		if err != nil {
			return nil, err
		}
	}
	seen := make(map[string]struct{})
	result := make([]domain.Skill, 0, len(args))
	for _, query := range args {
		value, err := manager.ResolveLibrary(query)
		if err != nil {
			return nil, err
		}
		if !tags.MatchAll(value.Tags, required) {
			continue
		}
		if _, ok := seen[value.ID]; ok {
			continue
		}
		seen[value.ID] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no Library Skills match the selection")
	}
	return result, nil
}

func parseAgents(values []string, defaults []domain.Agent) ([]domain.Agent, error) {
	if len(values) == 0 {
		if len(defaults) == 0 {
			return nil, fmt.Errorf("at least one agent is required")
		}
		return defaults, nil
	}
	seen := make(map[domain.Agent]struct{})
	var result []domain.Agent
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			value := domain.Agent(strings.ToLower(strings.TrimSpace(part)))
			if !value.Valid() {
				return nil, fmt.Errorf("unsupported agent %q: use claude or codex", part)
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result, nil
}

func buildCurrentPlan(storage *store.Store) (domain.Plan, error) {
	state, err := storage.LoadState()
	if err != nil {
		return domain.Plan{}, err
	}
	skills, err := storage.LoadAllSkills()
	if err != nil {
		return domain.Plan{}, err
	}
	return planner.New(storage).Build(skills, state)
}

func printPlan(out interface{ Write([]byte) (int, error) }, plan domain.Plan) error {
	writer := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "STATUS\tAGENT\tPLACEMENT\tSKILL\tTARGET")
	for _, operation := range plan.Operations {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", operation.Status, operation.Agent, operation.Placement, operation.SkillID, operation.Target)
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Plan digest: %s\n", plan.Digest)
	return err
}
