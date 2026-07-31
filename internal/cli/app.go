package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/apperr"
	"github.com/zzzzzyijie/skm/internal/store"
)

var Version = "dev"

type App struct {
	Out       io.Writer
	Err       io.Writer
	Home      string
	UserHome  string
	Project   string
	JSON      bool
	NoColor   bool
	lastStore *store.Store
}

type envelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	Success       bool   `json:"success"`
	Data          any    `json:"data,omitempty"`
	Error         *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func Execute(args []string, out, errOut io.Writer) int {
	app := &App{Out: out, Err: errOut}
	root := app.RootCommand()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		app.emitError(root.CommandPath(), err)
		return 1
	}
	return 0
}

func (a *App) RootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "skm",
		Short:         "Manage and deploy AI Agent Skills",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(a.Out)
	root.SetErr(a.Err)
	root.PersistentFlags().StringVar(&a.Home, "home", "", "override SKM_HOME")
	root.PersistentFlags().StringVar(&a.UserHome, "user-home", "", "override user home used for Agent targets")
	root.PersistentFlags().StringVar(&a.Project, "project", "", "project root (defaults to current repository)")
	root.PersistentFlags().BoolVar(&a.JSON, "json", false, "write versioned JSON output")
	root.PersistentFlags().BoolVar(&a.NoColor, "no-color", false, "disable colored output")
	root.AddCommand(
		a.newInitCommand(),
		a.newAddCommand(),
		a.newListCommand(),
		a.newShowCommand(),
		a.newValidateCommand(),
		a.newRemoveCommand(),
		a.newPruneCommand(),
		a.newEnableCommand(),
		a.newDisableCommand(),
		a.newPlanCommand(),
		a.newApplyCommand(),
		a.newStatusCommand(),
		a.newDoctorCommand(),
		a.newSourceCommand(),
		a.newTagCommand(),
		a.newProjectCommand(),
		a.newSyncCommand(),
		a.newUICommand(),
		a.newCompletionCommand(root),
		&cobra.Command{
			Use:   "version",
			Short: "Print the skm version",
			RunE: func(cmd *cobra.Command, args []string) error {
				version := currentVersion()
				return a.emit("version", map[string]string{"version": version}, func() error {
					_, err := fmt.Fprintln(a.Out, version)
					return err
				})
			},
		},
	)
	return root
}

func currentVersion() string {
	if Version != "" && Version != "dev" {
		return strings.TrimPrefix(Version, "v")
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "dev"
}

func (a *App) openStore() (*store.Store, error) {
	paths, err := store.DefaultPathsWithUserHome(a.Home, a.UserHome, a.Project)
	if err != nil {
		return nil, err
	}
	storage, err := store.New(paths)
	if err != nil {
		return nil, err
	}
	if err := storage.Ensure(); err != nil {
		return nil, err
	}
	a.lastStore = storage
	return storage, nil
}

func (a *App) emit(command string, data any, text func() error) error {
	if a.JSON {
		return json.NewEncoder(a.Out).Encode(envelope{
			SchemaVersion: 1,
			Command:       command,
			Success:       true,
			Data:          data,
		})
	}
	return text()
}

func (a *App) emitError(command string, err error) {
	if a.JSON {
		value := envelope{SchemaVersion: 1, Command: command, Success: false}
		value.Error = &struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: apperr.Code(err), Message: err.Error()}
		_ = json.NewEncoder(a.Err).Encode(value)
		return
	}
	_, _ = fmt.Fprintln(a.Err, "Error:", err)
}

func withLock(storage *store.Store, fn func() error) (err error) {
	unlock, err := storage.Lock()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, unlock()) }()
	return fn()
}
