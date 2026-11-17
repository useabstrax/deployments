package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
		scriptFile  string
		scriptStdin bool
	)

	cmd := &cobra.Command{
		Use:   "hooks <project> [phase]",
		Short: "List or edit deploy hooks",
		Long: `List or edit hooks for after_clone, before_activate, or after_activate.

Phases:
  after_clone       Run in the release directory after clone and shared linking
  before_activate   Run after after_clone, before health check / activate
  after_activate    Run after current is flipped (also re-run on rollback)

Command hooks run with bash -lc as the project user (when isolated), cwd = release path.
A phase may instead hold a multi-line script (--script-file / --script) which runs with
bash -e under the same user, cwd, and environment.

Environment available to hooks and scripts:
  ABSTRAX_PROJECT, ABSTRAX_PROJECT_PATH, ABSTRAX_RELEASE_PATH, ABSTRAX_CURRENT_PATH,
  ABSTRAX_SHARED_PATH, ABSTRAX_BRANCH, ABSTRAX_REF, ABSTRAX_RELEASE_ID, ABSTRAX_CLI_PHP

Replace a phase with one or more commands using repeatable --set, or pass an
ordered JSON array with --set-json. Use --script-file or --script (stdin) for a
multi-line script. Setting commands clears any script for that phase, and setting
a script clears the command list. --set, --set-json, --append, --clear,
--script-file, and --script are mutually exclusive.
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
			scriptFileChanged := cmd.Flags().Changed("script-file")
			mutating := setChanged || setJSONChanged || appendChanged || clear || scriptFileChanged || scriptStdin
			if mutating {
				if err := userx.RequireRoot(); err != nil {
					return err
				}
				if phase == "" {
					return fmt.Errorf("phase is required when modifying hooks")
				}
				if err := mutuallyExclusiveHookMutators(setChanged, setJSONChanged, appendChanged, clear, scriptFileChanged, scriptStdin); err != nil {
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
						p.Print(output.Success(action, "", phaseView(cfg, phase)))
					}
					return nil
				}
				if phase == "" {
					printPhaseHuman(p, "after_clone", cfg.Hooks.AfterClone, cfg.Hooks.AfterCloneScript)
					printPhaseHuman(p, "before_activate", cfg.Hooks.BeforeActivate, cfg.Hooks.BeforeActivateScript)
					printPhaseHuman(p, "after_activate", cfg.Hooks.AfterActivate, cfg.Hooks.AfterActivateScript)
					return nil
				}
				hooks, _ := release.HooksFor(cfg, phase)
				script, _ := cfg.Hooks.ScriptFor(phase)
				printPhaseHuman(p, phase, hooks, script)
				return nil
			}

			switch {
			case clear:
				if err := setHooks(cfg, phase, nil); err != nil {
					return err
				}
				if err := cfg.Hooks.SetScript(phase, ""); err != nil {
					return err
				}
			case scriptFileChanged || scriptStdin:
				script, err := readHookScript(scriptStdin, scriptFile)
				if err != nil {
					return err
				}
				if err := setHooks(cfg, phase, nil); err != nil {
					return err
				}
				if err := cfg.Hooks.SetScript(phase, script); err != nil {
					return err
				}
			case setJSONChanged:
				hooks, err := parseSetJSON(setJSON)
				if err != nil {
					return err
				}
				if err := cfg.Hooks.SetScript(phase, ""); err != nil {
					return err
				}
				if err := setHooks(cfg, phase, hooks); err != nil {
					return err
				}
			case setChanged:
				if err := cfg.Hooks.SetScript(phase, ""); err != nil {
					return err
				}
				if err := setHooks(cfg, phase, normalizeSetValues(setValues)); err != nil {
					return err
				}
			case appendChanged:
				if script, _ := cfg.Hooks.ScriptFor(phase); strings.TrimSpace(script) != "" {
					return fmt.Errorf("cannot --append to a phase that has a script; clear or replace the script first")
				}
				existing, _ := release.HooksFor(cfg, phase)
				hooks := append(append([]string(nil), existing...), appendValue)
				if err := setHooks(cfg, phase, hooks); err != nil {
					return err
				}
			}

			if globals.DryRun {
				p.DryRun("would update %s hooks", phase)
				p.Print(output.Success(action, "Dry run complete.", phaseView(cfg, phase)))
				return nil
			}

			if err := config.Save(proj.Path, cfg); err != nil {
				return err
			}
			_ = userx.ChownPath(config.Path(proj.Path), proj.User)

			p.Print(output.Success(action, fmt.Sprintf("Updated %s hooks.", phase), phaseView(cfg, phase)))
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&setValues, "set", nil, "Replace hooks for the phase (repeatable for multiple commands)")
	cmd.Flags().StringVar(&setJSON, "set-json", "", "Replace hooks for the phase with a JSON array of commands")
	cmd.Flags().StringVar(&appendValue, "append", "", "Append a hook command to the phase")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear all hooks and any script for the phase")
	cmd.Flags().StringVar(&scriptFile, "script-file", "", "Replace the phase with a multi-line script from a file")
	cmd.Flags().BoolVar(&scriptStdin, "script", false, "Replace the phase with a multi-line script read from stdin")
	return cmd
}

func mutuallyExclusiveHookMutators(set, setJSON, appendFlag, clear, scriptFile, scriptStdin bool) error {
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
	if scriptFile {
		n++
	}
	if scriptStdin {
		n++
	}
	if n > 1 {
		return fmt.Errorf("--set, --set-json, --append, --clear, --script-file, and --script are mutually exclusive")
	}
	return nil
}

func readHookScript(fromStdin bool, filePath string) (string, error) {
	if fromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading script from stdin: %w", err)
		}
		if len(data) == 0 {
			return "", fmt.Errorf("--script requires a non-empty script on stdin")
		}
		return string(data), nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading --script-file: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("--script-file is empty")
	}
	return string(data), nil
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

func printPhaseHuman(p *output.Printer, phase string, hooks []string, script string) {
	p.Line("%s:", phase)
	if strings.TrimSpace(script) != "" {
		p.Line("  (script)")
		for _, line := range strings.Split(strings.TrimRight(script, "\n"), "\n") {
			p.Line("  %s", line)
		}
		return
	}
	printHookList(p, hooks)
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

func phaseView(cfg *config.Config, phase string) map[string]interface{} {
	hooks, _ := release.HooksFor(cfg, phase)
	script, _ := cfg.Hooks.ScriptFor(phase)
	out := map[string]interface{}{
		"phase": phase,
		"hooks": hooks,
	}
	if strings.TrimSpace(script) != "" {
		out["script"] = script
	}
	return out
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
