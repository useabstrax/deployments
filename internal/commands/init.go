package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/presets"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newInitCmd() *cobra.Command {
	var (
		repository string
		branch     string
		preset     string
		publicDir  string
		keep       int
	)

	cmd := &cobra.Command{
		Use:   "init <project>",
		Short: "Scaffold deploy directories and deploy.json",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := userx.RequireRoot(); err != nil {
				return err
			}
			projectName := args[0]
			p := printer()
			action := "deploy.init"

			client, err := abstrax.New()
			if err != nil {
				return err
			}
			resp, err := client.Project(cmd.Context(), projectName)
			if err != nil {
				return err
			}
			proj := resp.Project

			if globals.DryRun {
				p.DryRun("would scaffold releases/ shared/ and write deploy.json under %s", proj.Path)
				p.DryRun("would set project public-dir to %s", layout.DocumentRootRel(defaultPublic(publicDir, preset)))
				p.Print(output.Success(action, "Dry run complete.", map[string]string{"path": proj.Path}))
				return nil
			}

			p.Progress(action, "directories", "Creating deploy layout")
			if err := layout.EnsureScaffold(proj.Path); err != nil {
				return err
			}

			cfg := config.Default(proj.Name)
			if branch != "" {
				cfg.Branch = branch
			}
			if repository != "" {
				cfg.Repository = repository
			}
			if keep > 0 {
				cfg.KeepReleases = keep
			}
			if preset == "" {
				preset = inferPreset(proj.Runtime.Type)
			}
			if err := presets.Apply(&cfg, preset); err != nil {
				return err
			}
			if publicDir != "" {
				cfg.PublicDir = publicDir
			}

			if config.Exists(proj.Path) && !globals.Yes {
				return fmt.Errorf("deploy.json already exists (pass --yes to overwrite)")
			}

			p.Progress(action, "config", "Writing deploy.json")
			if err := config.Save(proj.Path, &cfg); err != nil {
				return err
			}
			_ = userx.ChownPath(config.Path(proj.Path), proj.User)
			_ = userx.ChownPath(layout.ForProject(proj.Path).Releases, proj.User)
			_ = userx.ChownPath(layout.ForProject(proj.Path).Shared, proj.User)

			docRoot := layout.DocumentRootRel(cfg.PublicDir)
			p.Progress(action, "web_root", "Setting project public-dir to "+docRoot)
			if err := client.SetProjectPublicDir(cmd.Context(), proj.Name, docRoot); err != nil {
				p.Warn("could not update project public-dir via Abstrax: %v", err)
			}

			p.Print(output.Success(action, fmt.Sprintf("Deploy initialized for %s.", proj.Name), map[string]interface{}{
				"path":       proj.Path,
				"public_dir": cfg.PublicDir,
				"preset":     cfg.Preset,
				"web_root":   docRoot,
			}))
			return nil
		},
	}

	cmd.Flags().StringVar(&repository, "repository", "", "Git repository URL")
	cmd.Flags().StringVar(&branch, "branch", "", "Default branch")
	cmd.Flags().StringVar(&preset, "preset", "", "Preset: laravel, node, ruby, static, none")
	cmd.Flags().StringVar(&publicDir, "public-dir", "", "Public directory inside each release")
	cmd.Flags().IntVar(&keep, "keep", 0, "Number of releases to keep")
	return cmd
}

func defaultPublic(publicDir, preset string) string {
	if publicDir != "" {
		return publicDir
	}
	if strings.EqualFold(preset, "laravel") || preset == "" {
		return "public"
	}
	return "public"
}

func inferPreset(runtimeType string) string {
	switch strings.ToLower(runtimeType) {
	case "php":
		return "laravel"
	case "node":
		return "node"
	case "ruby":
		return "ruby"
	case "static":
		return "static"
	default:
		return "none"
	}
}
