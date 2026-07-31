package cli

import (
	"github.com/spf13/cobra"
	"github.com/zzzzzyijie/skm/internal/server"
)

func (a *App) newUICommand() *cobra.Command {
	var port int
	var noBrowser bool
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Start the web UI for managing Skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			server.Version = Version
			storage, err := a.openStore()
			if err != nil {
				return err
			}
			return server.New(storage).Run(port, !noBrowser)
		},
	}
	cmd.Flags().IntVar(&port, "port", 9527, "HTTP server port")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "don't auto-open browser")
	return cmd
}
