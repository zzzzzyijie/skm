package cli

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/skill"
)

func (a *App) newInitCommand() *cobra.Command {
	var withProject bool
	command := &cobra.Command{
		Use:   "init",
		Short: "Initialize skm storage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			if withProject {
				if err := storage.EnsureProject(); err != nil {
					return err
				}
			}
			data := map[string]any{"home": storage.Paths.Home, "project": storage.Paths.ProjectRoot, "projectInitialized": withProject}
			return a.emit("init", data, func() error {
				_, err := fmt.Fprintf(a.Out, "Initialized skm at %s\n", storage.Paths.Home)
				return err
			})
		},
	}
	command.Flags().BoolVar(&withProject, "with-project", false, "also initialize the current project's .skm directory")
	return command
}

func (a *App) newAddCommand() *cobra.Command {
	var scopeValue string
	var tagValues []string
	var sourceName string
	command := &cobra.Command{
		Use:   "add <skill-path>",
		Short: "Validate and add a local Skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scope := domain.Scope(scopeValue)
			if !scope.Valid() {
				return fmt.Errorf("invalid scope %q", scopeValue)
			}
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Skill
			err = withLock(storage, func() error {
				value, err = catalog.New(storage).AddLocal(args[0], sourceName, scope, tagValues)
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("add", value, func() error {
				_, err := fmt.Fprintf(a.Out, "Added %s [%s] tags=%s\n", value.ID, value.Scope, strings.Join(value.Tags, ","))
				return err
			})
		},
	}
	command.Flags().StringVar(&scopeValue, "scope", string(domain.ScopePersonal), "catalog scope: global, personal, or project")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "scenario tag (repeatable)")
	command.Flags().StringVar(&sourceName, "source", "", "local source name")
	return command
}

func (a *App) newListCommand() *cobra.Command {
	var scopeValue string
	var tagValues []string
	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Skills in the effective catalog",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var scope domain.Scope
			if scopeValue != "" {
				scope = domain.Scope(scopeValue)
				if !scope.Valid() {
					return fmt.Errorf("invalid scope %q", scopeValue)
				}
			}
			values, err := catalog.New(storage).List(scope, tagValues)
			if err != nil {
				return err
			}
			return a.emit("list", values, func() error {
				writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
				_, _ = fmt.Fprintln(writer, "ID\tSCOPE\tTAGS\tDESCRIPTION")
				for _, value := range values {
					_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", value.ID, value.Scope, strings.Join(value.Tags, ","), singleLine(value.Description))
				}
				return writer.Flush()
			})
		},
	}
	command.Flags().StringVar(&scopeValue, "scope", "", "filter by catalog scope")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "require a scenario tag (repeatable, AND semantics)")
	return command
}

func (a *App) newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <skill>",
		Short: "Show Skill metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			value, err := catalog.New(storage).Resolve(args[0])
			if err != nil {
				return err
			}
			return a.emit("show", value, func() error {
				_, err := fmt.Fprintf(a.Out, "ID: %s\nName: %s\nDescription: %s\nScope: %s\nTags: %s\nSource: %s\nRevision: %s\nHash: %s\nPath: %s\n",
					value.ID, value.Name, value.Description, value.Scope, strings.Join(value.Tags, ", "), value.Source, value.Revision, value.Hash, value.Path)
				return err
			})
		},
	}
}

func (a *App) newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <skill-path>",
		Short: "Validate a SKILL.md directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := skill.Validate(args[0])
			if err != nil {
				return err
			}
			document.Body = ""
			return a.emit("validate", document, func() error {
				_, err := fmt.Fprintf(a.Out, "Valid Skill: %s (%d files, %d bytes)\n", document.Name, document.Files, document.TotalSize)
				return err
			})
		},
	}
}

func (a *App) newRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <skill>",
		Short: "Remove an unlinked Skill from the catalog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var removed domain.Skill
			err = withLock(storage, func() error {
				manager := catalog.New(storage)
				value, err := manager.Resolve(args[0])
				if err != nil {
					return err
				}
				state, err := storage.LoadState()
				if err != nil {
					return err
				}
				for _, installation := range state.Installations {
					if installation.SkillID == value.ID {
						return fmt.Errorf("skill %s is linked; run skm unlink first", value.ID)
					}
				}
				removed, err = manager.Remove(value.ID)
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("remove", removed, func() error {
				_, err := fmt.Fprintf(a.Out, "Removed %s\n", removed.ID)
				return err
			})
		},
	}
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func aggregateTags(skills []domain.Skill) map[string]int {
	result := make(map[string]int)
	for _, value := range skills {
		for _, tag := range value.Tags {
			result[tag]++
		}
	}
	return result
}

func sortedKeys[T any](values map[string]T) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
