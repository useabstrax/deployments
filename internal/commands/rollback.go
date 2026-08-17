package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/runtimephp"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newRollbackCmd() *cobra.Command {
	var skipHooks bool

	cmd := &cobra.Command{
		Use:   "rollback <project> [release-id]",
		Short: "Point current at a previous release",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := userx.RequireRoot(); err != nil {
				return err
			}
			projectName := args[0]
			releaseID := ""
			if len(args) > 1 {
				releaseID = args[1]
			}
			p := printer()
			action := "deploy.rollback"

			client, err := abstrax.New()
			if err != nil {
				return err
			}
			resp, err := client.Project(cmd.Context(), projectName)
			if err != nil {
				return err
			}
			proj := resp.Project

			if !config.Exists(proj.Path) {
				return fmt.Errorf("deploy.json not found; run abstrax deploy init %s first", projectName)
			}
			cfg, err := config.Load(proj.Path)
			if err != nil {
				return err
			}

			target := releaseID
			if target == "" {
				prev, err := release.PreviousReleaseID(proj.Path)
				if err != nil {
					return err
				}
				target = prev
			}

			if !globals.Yes && !globals.DryRun {
				ok, err := confirm(fmt.Sprintf("Rollback %s to release %s? (re-runs after_activate hooks)", proj.Name, target))
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted")
				}
			}

			phpCLI := ""
			if proj.Runtime.Type == "php" {
				phpCLI = runtimephp.ResolveCLI(proj.Runtime.Version)
			}

			result, err := release.Rollback(cmd.Context(), release.RollbackOptions{
				ProjectPath: proj.Path,
				Config:      cfg,
				ReleaseID:   target,
				DryRun:      globals.DryRun,
				SkipHooks:   skipHooks,
				RunAsUser:   userx.RunAs(proj.User),
				Owner:       proj.User,
				CLIPHP:      phpCLI,
				Progress:    p,
				Action:      action,
			})
			if err != nil {
				return err
			}

			p.Print(output.Success(action, fmt.Sprintf("Rolled back %s to %s.", proj.Name, result.ReleaseID), result))
			return nil
		},
	}

	cmd.Flags().BoolVar(&skipHooks, "skip-hooks", false, "Skip after_activate hooks on rollback")
	return cmd
}
