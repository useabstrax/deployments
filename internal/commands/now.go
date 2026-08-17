package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/runtimephp"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newNowCmd() *cobra.Command {
	var (
		ref        string
		keep       int
		skipHooks  bool
		noActivate bool
		force      bool
	)

	cmd := &cobra.Command{
		Use:   "now <project>",
		Short: "Deploy a new release with zero downtime",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := userx.RequireRoot(); err != nil {
				return err
			}
			projectName := args[0]
			p := printer()
			action := "deploy.now"

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

			if !force && !globals.Yes && !globals.DryRun {
				ok, err := confirm(fmt.Sprintf("Deploy %s from %s?", proj.Name, emptyDash(cfg.Repository)))
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

			knownHosts := ""
			if cfg.DeployKey != "" {
				knownHosts = filepath.Join(filepath.Dir(cfg.DeployKey), "known_hosts")
			}

			result, err := release.Deploy(cmd.Context(), release.DeployOptions{
				ProjectPath: proj.Path,
				Config:      cfg,
				Ref:         ref,
				Keep:        keep,
				SkipHooks:   skipHooks,
				NoActivate:  noActivate,
				DryRun:      globals.DryRun,
				RunAsUser:   userx.RunAs(proj.User),
				Owner:       proj.User,
				CLIPHP:      phpCLI,
				KnownHosts:  knownHosts,
				Progress:    p,
				Action:      action,
			})
			if err != nil {
				return err
			}

			summary := fmt.Sprintf("Deployed release %s for %s.", result.ReleaseID, proj.Name)
			if !result.Activated {
				summary = fmt.Sprintf("Prepared release %s for %s (not activated).", result.ReleaseID, proj.Name)
			}
			p.Print(output.Success(action, summary, result))
			return nil
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "Git ref to deploy (branch, tag, or SHA; default: configured branch)")
	cmd.Flags().IntVar(&keep, "keep", 0, "Override keep_releases for this deploy")
	cmd.Flags().BoolVar(&skipHooks, "skip-hooks", false, "Skip all hooks")
	cmd.Flags().BoolVar(&noActivate, "no-activate", false, "Prepare the release without flipping current")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}
