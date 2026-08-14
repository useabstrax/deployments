package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <project>",
		Short: "List releases for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			p := printer()
			action := "deploy.list"

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

			entries, err := release.ListReleases(proj.Path)
			if err != nil {
				return err
			}

			if globals.JSON || globals.JSONStream {
				p.Print(output.Success(action, "", entries))
				return nil
			}

			if len(entries) == 0 {
				p.Line("No releases found.")
				return nil
			}

			p.Line("%-16s %-8s %-12s %-40s", "ID", "CURRENT", "SHA", "REF")
			for _, e := range entries {
				cur := ""
				if e.Current {
					cur = "*"
				}
				sha := e.ShortSHA
				if sha == "" {
					sha = "-"
				}
				ref := e.Ref
				if ref == "" {
					ref = "-"
				}
				p.Line("%-16s %-8s %-12s %-40s", e.ID, cur, sha, ref)
			}
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <project>",
		Short: "Show deploy status and current release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			p := printer()
			action := "deploy.status"

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

			paths := layout.ForProject(proj.Path)
			currentID, _ := paths.CurrentReleaseID()
			currentTarget, _ := paths.CurrentTarget()

			var currentMeta *release.Meta
			if currentID != "" {
				currentMeta, _ = release.ReadMeta(paths.ReleasePath(currentID))
			}

			data := map[string]interface{}{
				"project":        proj.Name,
				"path":           proj.Path,
				"repository":     cfg.Repository,
				"branch":         cfg.Branch,
				"provider":       cfg.Provider,
				"preset":         cfg.Preset,
				"public_dir":     cfg.PublicDir,
				"keep_releases":  cfg.KeepReleases,
				"deploy_key":     cfg.DeployKey,
				"current_id":     currentID,
				"current_target": currentTarget,
				"current_meta":   currentMeta,
			}

			if globals.JSON || globals.JSONStream {
				p.Print(output.Success(action, "", data))
				return nil
			}

			p.Line("Deploy status for %s", proj.Name)
			p.Line("  Path:           %s", proj.Path)
			p.Line("  Repository:     %s", emptyDash(cfg.Repository))
			p.Line("  Branch:         %s", cfg.Branch)
			p.Line("  Preset:         %s", cfg.Preset)
			p.Line("  Public dir:     %s", cfg.PublicDir)
			p.Line("  Keep releases:  %d", cfg.KeepReleases)
			p.Line("  Deploy key:     %s", emptyDash(cfg.DeployKey))
			p.Line("  Current:        %s", emptyDash(currentID))
			p.Line("  Symlink target: %s", emptyDash(currentTarget))
			if currentMeta != nil {
				p.Line("  Current ref:    %s", emptyDash(currentMeta.Ref))
				p.Line("  Current SHA:    %s", emptyDash(currentMeta.ShortSHA))
			}
			return nil
		},
	}
}
