package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/plugin"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display plugin version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Plugin version: %s\n", plugin.Version)
			fmt.Printf("Plugin protocol version: %d\n", plugin.ProtocolVersion)
			fmt.Printf("Build commit: %s\n", plugin.Commit)
			fmt.Printf("Build date: %s\n", plugin.BuildDate)
			if plugin.IsRunningAsPlugin() {
				fmt.Printf("Abstrax host version: %s\n", plugin.HostVersion())
			}
			return nil
		},
	}
}
