package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	gitSource "github.com/zzzzzyijie/skm/internal/source"
)

func (a *App) newSourceCommand() *cobra.Command {
	command := &cobra.Command{Use: "source", Short: "Manage custom Git Skill sources"}
	command.AddCommand(a.newSourceAddCommand(), a.newSourceListCommand(), a.newSourceUpdateCommand(), a.newSourceRemoveCommand())
	return command
}

func (a *App) newSourceAddCommand() *cobra.Command {
	var name string
	var ref string
	var paths []string
	var tagValues []string
	command := &cobra.Command{
		Use:   "add <git-url>",
		Short: "Bind and import a custom Git Skill source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var value domain.Source
			var imported []domain.Skill
			err = withLock(storage, func() error {
				manager := catalog.New(storage)
				value, imported, err = gitSource.NewGitManager(storage, manager).Add(domain.Source{
					Name: name, URL: args[0], Ref: ref, Paths: paths, Tags: tagValues,
				})
				return err
			})
			if err != nil {
				return err
			}
			data := map[string]any{"source": value, "skills": imported}
			return a.emit("source add", data, func() error {
				_, err := fmt.Fprintf(a.Out, "Added source %s at %s; imported %d Skill(s)\n", value.Name, shortRevision(value.Revision), len(imported))
				return err
			})
		},
	}
	command.Flags().StringVar(&name, "name", "", "unique source name")
	command.Flags().StringVar(&ref, "ref", "", "branch, tag, or commit to bind")
	command.Flags().StringArrayVar(&paths, "path", nil, "relative Skill directory to bind (repeatable; default scans repository)")
	command.Flags().StringArrayVar(&tagValues, "tag", nil, "default tag for imported Skills (repeatable)")
	_ = command.MarkFlagRequired("name")
	return command
}

func (a *App) newSourceListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List custom Git sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			values, err := storage.LoadSources()
			if err != nil {
				return err
			}
			return a.emit("source list", values.Sources, func() error {
				writer := tabwriter.NewWriter(a.Out, 0, 4, 2, ' ', 0)
				_, _ = fmt.Fprintln(writer, "NAME\tREF\tREVISION\tPATHS\tURL")
				for _, value := range values.Sources {
					_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", value.Name, value.Ref, shortRevision(value.Revision), strings.Join(value.Paths, ","), value.URL)
				}
				return writer.Flush()
			})
		},
	}
}

func (a *App) newSourceUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update [name...]",
		Short: "Refresh one or all custom Git sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var sources []domain.Source
			var imported []domain.Skill
			err = withLock(storage, func() error {
				sources, imported, err = gitSource.NewGitManager(storage, catalog.New(storage)).Update(args)
				return err
			})
			if err != nil {
				return err
			}
			data := map[string]any{"sources": sources, "skills": imported}
			return a.emit("source update", data, func() error {
				_, err := fmt.Fprintf(a.Out, "Updated %d source(s); imported %d Skill(s)\n", len(sources), len(imported))
				return err
			})
		},
	}
}

func (a *App) newSourceRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a Git binding and cached checkout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var removed gitSource.RemovalResult
			err = withLock(storage, func() error {
				removed, err = gitSource.NewGitManager(storage, catalog.New(storage)).Remove(args[0])
				return err
			})
			if err != nil {
				return err
			}
			return a.emit("source remove", removed, func() error {
				if !removed.BindingRemoved {
					_, err := fmt.Fprintf(a.Out, "Removed orphaned source checkout %s\n", removed.Name)
					return err
				}
				checkout := ""
				if removed.CheckoutRemoved {
					checkout = " and its checkout"
				}
				_, err := fmt.Fprintf(a.Out, "Removed source binding %s%s; imported Library Skills were retained\n", removed.Name, checkout)
				return err
			})
		},
	}
}

func (a *App) newSyncCommand() *cobra.Command {
	var sourceNames []string
	var noApply bool
	command := &cobra.Command{
		Use:   "sync",
		Short: "Update Git sources and reconcile managed deployments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			var updated []domain.Source
			var imported []domain.Skill
			var plan domain.Plan
			err = withLock(storage, func() error {
				updated, imported, err = gitSource.NewGitManager(storage, catalog.New(storage)).Update(sourceNames)
				if err != nil {
					return err
				}
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
				if noApply {
					return nil
				}
				if err := engine.Apply(plan, &state); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}
			data := map[string]any{"sources": updated, "skills": imported, "plan": plan, "applied": !noApply}
			return a.emit("sync", data, func() error {
				if _, err := fmt.Fprintf(a.Out, "Updated %d source(s), imported %d Skill(s)\n", len(updated), len(imported)); err != nil {
					return err
				}
				return printPlan(a.Out, plan)
			})
		},
	}
	command.Flags().StringArrayVar(&sourceNames, "source", nil, "only update this source (repeatable)")
	command.Flags().BoolVar(&noApply, "no-apply", false, "update sources and print the deployment plan without applying it")
	return command
}

func shortRevision(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
