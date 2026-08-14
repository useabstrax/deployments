package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/plugin"
)

func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Plugin protocol commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "metadata",
		Short: "Print plugin metadata JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return plugin.WriteMetadata(os.Stdout, plugin.DefaultMetadata())
		},
	})

	return cmd
}
