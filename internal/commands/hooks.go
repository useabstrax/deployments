package commands

import (
	"encoding/json"
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
		setValues   []string
		setJSON     string
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

Replace a phase with one or more commands using repeatable --set, or pass an
ordered JSON array with --set-json. --set, --set-json, --append, and --clear
are mutually exclusive.
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

			setChanged := cmd.Flags().Changed("set")
			setJSONChanged := cmd.Flags().Changed("set-json")
			appendChanged := cmd.Flags().Changed("append")
			mutating := setChanged || setJSONChanged || appendChanged || clear
			if mutating {
				if err := userx.RequireRoot(); err != nil {
					return err
				}
				if phase == "" {
					return fmt.Errorf("phase is required when modifying hooks")
				}
				if err := mutuallyExclusiveHookMutators(setChanged, setJSONChanged, appendChanged, clear); err != nil {
					return err
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

			var hooks []string
			switch {
			case clear:
				hooks = nil
			case setJSONChanged:
				hooks, err = parseSetJSON(setJSON)
				if err != nil {
					return err
				}
			case setChanged:
				hooks = normalizeSetValues(setValues)
			case appendChanged:
				existing, _ := release.HooksFor(cfg, phase)
				hooks = append(append([]string(nil), existing...), appendValue)
			}
			if err := setHooks(cfg, phase, hooks); err != nil {
				return err
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

			saved, _ := release.HooksFor(cfg, phase)
			p.Print(output.Success(action, fmt.Sprintf("Updated %s hooks.", phase), map[string]interface{}{
				"phase": phase,
				"hooks": saved,
			}))
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&setValues, "set", nil, "Replace hooks for the phase (repeatable for multiple commands)")
	cmd.Flags().StringVar(&setJSON, "set-json", "", "Replace hooks for the phase with a JSON array of commands")
	cmd.Flags().StringVar(&appendValue, "append", "", "Append a hook command to the phase")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear all hooks for the phase")
	return cmd
}

func mutuallyExclusiveHookMutators(set, setJSON, appendFlag, clear bool) error {
	n := 0
	if set {
		n++
	}
	if setJSON {
		n++
	}
	if appendFlag {
		n++
	}
	if clear {
		n++
	}
	if n > 1 {
		return fmt.Errorf("--set, --set-json, --append, and --clear are mutually exclusive")
	}
	return nil
}

// parseSetJSON decodes a JSON array of strings for --set-json.
func parseSetJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}
	var hooks []string
	if err := json.Unmarshal([]byte(raw), &hooks); err != nil {
		return nil, fmt.Errorf("--set-json must be a JSON array of strings: %w", err)
	}
	if hooks == nil {
		hooks = []string{}
	}
	return hooks, nil
}

// normalizeSetValues turns repeatable --set values into the stored hook list.
// A single whitespace-only value clears the phase (same as historical --set "").
func normalizeSetValues(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	if len(values) == 1 && strings.TrimSpace(values[0]) == "" {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out
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
