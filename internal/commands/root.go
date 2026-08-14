package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/plugin"
)

// GlobalFlags are Abstrax-style globals shared across commands.
type GlobalFlags struct {
	JSON       bool
	JSONStream bool
	Yes        bool
	DryRun     bool
	Verbose    bool
	Quiet      bool
	NoColor    bool
}

var globals GlobalFlags

// NewRootCmd creates the root command for abstrax-deploy.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "abstrax-deploy",
		Short:         "Zero-downtime deployments for Abstrax projects",
		Long:          "Official Abstrax Deploy plugin: releases/, current symlink, shallow clones, and hooks.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if globals.JSON && globals.JSONStream {
				return fmt.Errorf("--json and --json-stream are mutually exclusive; use --json for a single result object or --json-stream for NDJSON progress")
			}
			return nil
		},
	}

	root.PersistentFlags().BoolVar(&globals.JSON, "json", false, "Output a single JSON result object")
	root.PersistentFlags().BoolVar(&globals.JSONStream, "json-stream", false, "Output NDJSON progress events and a final result line")
	root.PersistentFlags().BoolVar(&globals.Yes, "yes", false, "Skip confirmation prompts")
	root.PersistentFlags().BoolVar(&globals.DryRun, "dry-run", false, "Preview changes without applying them")
	root.PersistentFlags().BoolVar(&globals.Verbose, "verbose", false, "Verbose output")
	root.PersistentFlags().BoolVar(&globals.Quiet, "quiet", false, "Suppress informational messages")
	root.PersistentFlags().BoolVar(&globals.NoColor, "no-color", false, "Disable colored output")

	root.AddCommand(newPluginCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newSetupCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newConfigureCmd())
	root.AddCommand(newKeyCmd())
	root.AddCommand(newNowCmd())
	root.AddCommand(newRollbackCmd())
	root.AddCommand(newListCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newHooksCmd())

	return root
}

// Execute runs the plugin command tree.
func Execute() int {
	root := NewRootCmd()
	err := root.Execute()
	if err == nil {
		return plugin.ExitSuccess
	}
	if !globals.JSON && !globals.JSONStream {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	} else {
		p := printer()
		p.Print(output.Failure(guessAction(root), "command_error", err.Error()))
	}
	code := plugin.ExitCodeFor(err)
	if code == plugin.ExitInternal && isUsageError(err) {
		return plugin.ExitUsage
	}
	return code
}

func printer() *output.Printer {
	return output.NewPrinter(globals.JSON, globals.JSONStream, globals.Quiet, globals.Verbose, globals.NoColor)
}

func isUsageError(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unknown command") ||
		strings.Contains(s, "accepts") ||
		strings.Contains(s, "required") ||
		strings.Contains(s, "invalid argument") ||
		strings.Contains(s, "flag needs an argument")
}

func guessAction(root *cobra.Command) string {
	if root == nil {
		return "deploy"
	}
	cmd, _, _ := root.Find(os.Args[1:])
	if cmd == nil {
		return "deploy"
	}
	parts := strings.Split(cmd.CommandPath(), " ")
	if len(parts) >= 2 {
		return "deploy." + parts[len(parts)-1]
	}
	return "deploy"
}
