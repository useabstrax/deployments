package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
)

func newHooksCmd() *cobra.Command {
	var (
		setValue    string
		appendValue string
		clear       bool
	)

	cmd := &cobra.Command{
		Use:   "hooks <project> [phase]",
		Short: "List or edit deploy hooks",
		Long: `List or edit hooks for after_clone, before_activate, or after_activate.

Phases:
  after_clone       Run in the release directory after clone and shared linking
  before_activate   Run after after_clone, before health check / activate
  after_activate    Run after current is flipped (also re-run on rollback)

Hooks run with bash -lc as the project user (when isolated), cwd = release path.
`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			phase := ""
			if len(args) > 1 {
				phase = args[1]
			}
			p := printer()
			action := "deploy.hooks"

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

			mutating := cmd.Flags().Changed("set") || cmd.Flags().Changed("append") || clear
			if mutating {
				if err := userx.RequireRoot(); err != nil {
					return err
				}
				if phase == "" {
					return fmt.Errorf("phase is required when modifying hooks")
				}
			}

			if phase != "" {
				if _, err := release.HooksFor(cfg, phase); err != nil {
					return err
				}
			}

			if !mutating {
				if globals.JSON || globals.JSONStream {
					if phase == "" {
						p.Print(output.Success(action, "", cfg.Hooks))
					} else {
						hooks, _ := release.HooksFor(cfg, phase)
						p.Print(output.Success(action, "", map[string]interface{}{"phase": phase, "hooks": hooks}))
					}
					return nil
				}
				if phase == "" {
					p.Line("after_clone:")
					printHookList(p, cfg.Hooks.AfterClone)
					p.Line("before_activate:")
					printHookList(p, cfg.Hooks.BeforeActivate)
					p.Line("after_activate:")
					printHookList(p, cfg.Hooks.AfterActivate)
					return nil
				}
				hooks, _ := release.HooksFor(cfg, phase)
				p.Line("%s:", phase)
				printHookList(p, hooks)
				return nil
			}

			if clear {
				if err := setHooks(cfg, phase, nil); err != nil {
					return err
				}
			} else if cmd.Flags().Changed("set") {
				var hooks []string
				if strings.TrimSpace(setValue) != "" {
					hooks = []string{setValue}
				}
				if err := setHooks(cfg, phase, hooks); err != nil {
					return err
				}
			} else if cmd.Flags().Changed("append") {
				existing, _ := release.HooksFor(cfg, phase)
				existing = append(existing, appendValue)
				if err := setHooks(cfg, phase, existing); err != nil {
					return err
				}
			}

			if globals.DryRun {
				p.DryRun("would update %s hooks", phase)
				p.Print(output.Success(action, "Dry run complete.", cfg.Hooks))
				return nil
			}

			if err := config.Save(proj.Path, cfg); err != nil {
				return err
			}
			_ = userx.ChownPath(config.Path(proj.Path), proj.User)

			hooks, _ := release.HooksFor(cfg, phase)
			p.Print(output.Success(action, fmt.Sprintf("Updated %s hooks.", phase), map[string]interface{}{
				"phase": phase,
				"hooks": hooks,
			}))
			return nil
		},
	}

	cmd.Flags().StringVar(&setValue, "set", "", "Replace hooks for the phase with a single command")
	cmd.Flags().StringVar(&appendValue, "append", "", "Append a hook command to the phase")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear all hooks for the phase")
	return cmd
}

func printHookList(p *output.Printer, hooks []string) {
	if len(hooks) == 0 {
		p.Line("  (none)")
		return
	}
	for i, h := range hooks {
		p.Line("  %d. %s", i+1, h)
	}
}

func setHooks(cfg *config.Config, phase string, hooks []string) error {
	if hooks == nil {
		hooks = []string{}
	}
	switch phase {
	case release.PhaseAfterClone:
		cfg.Hooks.AfterClone = hooks
	case release.PhaseBeforeActivate:
		cfg.Hooks.BeforeActivate = hooks
	case release.PhaseAfterActivate:
		cfg.Hooks.AfterActivate = hooks
	default:
		return fmt.Errorf("unknown hook phase %q", phase)
	}
	return nil
}
