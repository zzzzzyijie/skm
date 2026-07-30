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
		Short: "Initialize the personal Skill Library",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			if withProject {
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
			}
			data := map[string]any{"home": storage.Paths.Home, "project": storage.Paths.ProjectRoot, "projectInitialized": withProject}
			return a.emit("init", data, func() error {
				_, err := fmt.Fprintf(a.Out, "Initialized personal Library at %s\n", storage.Paths.Home)
				return err
			})
		},
	}
	command.Flags().BoolVar(&withProject, "with-project", false, "also initialize the current project's .skm directory")
	return command
}

func (a *App) newAddCommand() *cobra.Command {
	var tagValues []string
	var sourceName string
	command := &cobra.Command{
		Use:   "add <skill-path>",
		Short: "Add a local Skill to the personal Library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Skill
			err = withLock(storage, func() error {
				value, err = catalog.New(storage).AddLocal(args[0], sourceName, tagValues)
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("add", value, func() error {
				_, err := fmt.Fprintf(a.Out, "Added %s to Library; tags=%s\n", value.ID, strings.Join(value.Tags, ","))
				return err
			})
		},
	}
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "personal Library tag (repeatable)")
	command.Flags().StringVar(&sourceName, "source", "", "local source name")
	return command
}

func (a *App) newListCommand() *cobra.Command {
	var tagValues []string
	command := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Skills in the personal Library",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			values, err := catalog.New(storage).List(domain.LocationLibrary, tagValues)
			if err != nil {
				return err
			}
			return a.emit("list", values, func() error {
				writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
				_, _ = fmt.Fprintln(writer, "ID\tTAGS\tREVISION\tDESCRIPTION")
				for _, value := range values {
					_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", value.ID, strings.Join(value.Tags, ","), shortRevision(value.Revision), singleLine(value.Description))
				}
				return writer.Flush()
			})
		},
	}
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "require a personal Library tag (repeatable, AND semantics)")
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
				_, err := fmt.Fprintf(a.Out, "ID: %s\nName: %s\nDescription: %s\nLocation: %s\nTags: %s\nSource: %s\nRevision: %s\nHash: %s\nPath: %s\n",
					value.ID, value.Name, value.Description, value.Location, strings.Join(value.Tags, ", "), value.Source, value.Revision, value.Hash, value.Path)
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
		Short: "Remove a disabled Skill from the personal Library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var removed domain.Skill
			err = withLock(storage, func() error {
				manager := catalog.New(storage)
				value, err := manager.ResolveLibrary(args[0])
				if err != nil {
					return err
				}
				state, err := storage.LoadState()
				if err != nil {
					return err
				}
				for _, activation := range state.Activations {
					if activation.SkillID == value.ID {
						return fmt.Errorf("skill %s is enabled; run skm disable first", value.ID)
					}
				}
				project, err := storage.LoadProjectCatalog()
				if err != nil {
					return err
				}
				for _, dependency := range project.Dependencies {
					if dependency.ID == value.ID {
						return fmt.Errorf("skill %s is required by the current project; run skm project remove first", value.ID)
					}
				}
				removed, err = manager.Remove(value.ID)
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("remove", removed, func() error {
				_, err := fmt.Fprintf(a.Out, "Removed %s from Library\n", removed.ID)
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
