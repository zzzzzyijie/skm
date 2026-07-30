package cli

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/tags"
)

type tagCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func (a *App) newTagCommand() *cobra.Command {
	command := &cobra.Command{Use: "tag", Short: "Manage personal Library tags"}
	command.AddCommand(a.newTagListCommand(), a.newTagAddCommand(), a.newTagRemoveCommand(), a.newTagRenameCommand())
	return command
}

func (a *App) newTagListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List personal Library tags and Skill counts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			library, err := storage.LoadCatalog()
			if err != nil {
				return err
			}
			counts := aggregateTags(library.Skills)
			result := make([]tagCount, 0, len(counts))
			for name, count := range counts {
				result = append(result, tagCount{Name: name, Count: count})
			}
			sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
			return a.emit("tag list", result, func() error {
				writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
				_, _ = fmt.Fprintln(writer, "TAG\tSKILLS")
				for _, value := range result {
					_, _ = fmt.Fprintf(writer, "%s\t%d\n", value.Name, value.Count)
				}
				return writer.Flush()
			})
		},
	}
}

func (a *App) newTagAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <skill> <tag...>",
		Short: "Add one or more tags to a personal Library Skill",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Skill
			err = withLock(storage, func() error {
				value, err = catalog.New(storage).UpdateTags(args[0], func(current []string) []string {
					return append(current, args[1:]...)
				})
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("tag add", value, func() error {
				_, err := fmt.Fprintf(a.Out, "%s tags=%s\n", value.ID, strings.Join(value.Tags, ","))
				return err
			})
		},
	}
}

func (a *App) newTagRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <skill> <tag...>",
		Short: "Remove tags; the configured default is restored if none remain",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			remove := make(map[string]struct{}, len(args)-1)
			for _, value := range args[1:] {
				remove[strings.ToLower(value)] = struct{}{}
			}
			var value domain.Skill
			err = withLock(storage, func() error {
				value, err = catalog.New(storage).UpdateTags(args[0], func(current []string) []string {
					result := current[:0]
					for _, tag := range current {
						if _, ok := remove[tag]; !ok {
							result = append(result, tag)
						}
					}
					return result
				})
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("tag remove", value, func() error {
				_, err := fmt.Fprintf(a.Out, "%s tags=%s\n", value.ID, strings.Join(value.Tags, ","))
				return err
			})
		},
	}
}

func (a *App) newTagRenameCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a tag in the personal Library",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			validated, err := tags.Normalize([]string{args[1]}, nil)
			if err != nil {
				return err
			}
			oldName := strings.ToLower(args[0])
			newName := validated[0]
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			changed := 0
			err = withLock(storage, func() error {
				library, err := storage.LoadCatalog()
				if err != nil {
					return err
				}
				for _, value := range library.Skills {
					found := false
					for i, tag := range value.Tags {
						if tag == oldName {
							value.Tags[i] = newName
							found = true
						}
					}
					if !found {
						continue
					}
					value.Tags, err = tags.Normalize(value.Tags, nil)
					if err != nil {
						return err
					}
					if err := storage.UpsertSkill(value); err != nil {
						return err
					}
					changed++
				}
				if changed == 0 {
					return fmt.Errorf("tag %q not found", oldName)
				}
				return nil
			})
			if err != nil {
				return err
			}
			data := map[string]any{"old": oldName, "new": newName, "skillsChanged": changed}
			return a.emit("tag rename", data, func() error {
				_, err := fmt.Fprintf(a.Out, "Renamed %s to %s on %d Skills\n", oldName, newName, changed)
				return err
			})
		},
	}
}
