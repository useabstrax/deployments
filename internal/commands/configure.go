package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/presets"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/provider"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newConfigureCmd() *cobra.Command {
	var (
		repository string
		branch     string
		preset     string
		publicDir  string
		keep       int
		shared     string
		edit       bool
	)

	cmd := &cobra.Command{
		Use:   "configure <project>",
		Short: "Show or update deploy configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			p := printer()
			action := "deploy.configure"

			client, err := abstrax.New()
			if err != nil {
				return err
			}
			resp, err := client.Project(cmd.Context(), projectName)
			if err != nil {
				return err
			}
			proj := resp.Project

			changing := cmd.Flags().Changed("repository") ||
				cmd.Flags().Changed("branch") ||
				cmd.Flags().Changed("preset") ||
				cmd.Flags().Changed("public-dir") ||
				cmd.Flags().Changed("keep") ||
				cmd.Flags().Changed("shared") ||
				edit

			if changing {
				if err := userx.RequireRoot(); err != nil {
					return err
				}
			}

			if !config.Exists(proj.Path) {
				return fmt.Errorf("deploy.json not found; run abstrax deploy init %s first", projectName)
			}
			cfg, err := config.Load(proj.Path)
			if err != nil {
				return err
			}

			if !changing {
				if globals.JSON || globals.JSONStream {
					p.Print(output.Success(action, "", cfg))
					return nil
				}
				printConfig(p, cfg, proj.Path)
				return nil
			}

			if edit {
				p.Info("Opening %s in $EDITOR is not automated in v1; edit the file directly or use flags.", config.Path(proj.Path))
			}

			if cmd.Flags().Changed("preset") {
				if err := presets.Apply(cfg, preset); err != nil {
					return err
				}
				maybeWarnComposerPlugin(p, cfg.Preset)
			}
			if cmd.Flags().Changed("repository") {
				prov, err := provider.Lookup(cfg.Provider)
				if err != nil {
					return err
				}
				norm, err := prov.NormalizeRepository(repository)
				if err != nil {
					return err
				}
				cfg.Repository = norm
			}
			if cmd.Flags().Changed("branch") {
				cfg.Branch = branch
			}
			if cmd.Flags().Changed("public-dir") {
				cfg.PublicDir = publicDir
			}
			if cmd.Flags().Changed("keep") {
				cfg.KeepReleases = keep
			}
			if cmd.Flags().Changed("shared") {
				cfg.Shared = splitCSV(shared)
			}

			if globals.DryRun {
				p.DryRun("would write updated deploy.json")
				p.Print(output.Success(action, "Dry run complete.", cfg))
				return nil
			}

			if err := config.Save(proj.Path, cfg); err != nil {
				return err
			}
			_ = userx.ChownPath(config.Path(proj.Path), proj.User)

			if cmd.Flags().Changed("public-dir") || cmd.Flags().Changed("preset") {
				docRoot := layout.DocumentRootRel(cfg.PublicDir)
				if err := client.SetProjectPublicDir(cmd.Context(), proj.Name, docRoot); err != nil {
					p.Warn("could not update project public-dir: %v", err)
				}
			}

			p.Print(output.Success(action, fmt.Sprintf("Deploy config updated for %s.", proj.Name), cfg))
			return nil
		},
	}

	cmd.Flags().StringVar(&repository, "repository", "", "Git repository URL")
	cmd.Flags().StringVar(&branch, "branch", "", "Default branch")
	cmd.Flags().StringVar(&preset, "preset", "", "Apply a preset")
	cmd.Flags().StringVar(&publicDir, "public-dir", "", "Public directory inside each release")
	cmd.Flags().IntVar(&keep, "keep", 5, "Number of releases to keep")
	cmd.Flags().StringVar(&shared, "shared", "", "Comma-separated shared paths")
	cmd.Flags().BoolVar(&edit, "edit", false, "Escape hatch reminder to edit deploy.json manually")
	return cmd
}

func printConfig(p *output.Printer, cfg *config.Config, projectPath string) {
	p.Line("Deploy config (%s):", config.Path(projectPath))
	p.Line("  Project:       %s", cfg.Project)
	p.Line("  Repository:    %s", emptyDash(cfg.Repository))
	p.Line("  Branch:        %s", cfg.Branch)
	p.Line("  Provider:      %s", cfg.Provider)
	p.Line("  Preset:        %s", cfg.Preset)
	p.Line("  Public dir:    %s", cfg.PublicDir)
	p.Line("  Keep releases: %d", cfg.KeepReleases)
	p.Line("  Shared:        %s", emptyDash(strings.Join(cfg.Shared, ", ")))
	p.Line("  Deploy key:    %s", emptyDash(cfg.DeployKey))
	p.Line("  Hooks:")
	p.Line("    after_clone:      %s", formatHooks(cfg.Hooks.AfterClone))
	p.Line("    before_activate:  %s", formatHooks(cfg.Hooks.BeforeActivate))
	p.Line("    after_activate:   %s", formatHooks(cfg.Hooks.AfterActivate))
}

func formatHooks(hooks []string) string {
	if len(hooks) == 0 {
		return "(none)"
	}
	return strings.Join(hooks, " | ")
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// confirm asks unless --yes.
func confirm(prompt string) (bool, error) {
	if globals.Yes {
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	var ans string
	_, _ = fmt.Scanln(&ans)
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes", nil
}
