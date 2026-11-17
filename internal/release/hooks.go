package release

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
)

// HookEnv is injected into hook processes.
type HookEnv struct {
	Project     string
	ProjectPath string
	ReleasePath string
	CurrentPath string
	SharedPath  string
	Branch      string
	Ref         string
	ReleaseID   string
	CLIPHP      string
	Extra       []string
}

func (e HookEnv) Environ() []string {
	env := append(os.Environ(), e.Extra...)
	set := func(k, v string) {
		if v == "" {
			return
		}
		env = append(env, k+"="+v)
	}
	set("ABSTRAX_PROJECT", e.Project)
	set("ABSTRAX_PROJECT_PATH", e.ProjectPath)
	set("ABSTRAX_RELEASE_PATH", e.ReleasePath)
	set("ABSTRAX_CURRENT_PATH", e.CurrentPath)
	set("ABSTRAX_SHARED_PATH", e.SharedPath)
	set("ABSTRAX_BRANCH", e.Branch)
	set("ABSTRAX_REF", e.Ref)
	set("ABSTRAX_RELEASE_ID", e.ReleaseID)
	set("ABSTRAX_CLI_PHP", e.CLIPHP)
	return env
}

// Phase names.
const (
	PhaseAfterClone     = "after_clone"
	PhaseBeforeActivate = "before_activate"
	PhaseAfterActivate  = "after_activate"
)

// HooksFor returns the hook command list for a phase.
func HooksFor(cfg *config.Config, phase string) ([]string, error) {
	switch phase {
	case PhaseAfterClone:
		return cfg.Hooks.AfterClone, nil
	case PhaseBeforeActivate:
		return cfg.Hooks.BeforeActivate, nil
	case PhaseAfterActivate:
		return cfg.Hooks.AfterActivate, nil
	default:
		return nil, fmt.Errorf("unknown hook phase %q", phase)
	}
}

// RunPhase executes a phase: multi-line script if set, otherwise the command list.
// Scripts run with bash -e as the project user (unless they invoke abstrax),
// cwd = releasePath, with the same environment as command hooks.
func RunPhase(ctx context.Context, cfg *config.Config, phase, releasePath string, env HookEnv, runAsUser string) error {
	script, err := cfg.Hooks.ScriptFor(phase)
	if err != nil {
		return err
	}
	if strings.TrimSpace(script) != "" {
		return RunScript(ctx, script, releasePath, env, runAsUser)
	}
	hooks, err := HooksFor(cfg, phase)
	if err != nil {
		return err
	}
	return RunHooks(ctx, hooks, releasePath, env, runAsUser)
}

// RunHooks executes hook commands with bash -lc in releasePath as optional user.
// Hooks that invoke the Abstrax CLI (abstrax …) run as root when deploy is root,
// so system-installed plugins under /usr/local/lib/abstrax/plugins are found.
// Those hooks should pass --project / --user so tools like Composer drop privileges.
func RunHooks(ctx context.Context, hooks []string, releasePath string, env HookEnv, runAsUser string) error {
	for i, hook := range hooks {
		hook = strings.TrimSpace(hook)
		if hook == "" {
			continue
		}
		user := runAsUser
		if hookInvokesAbstrax(hook) {
			user = ""
		}
		if err := runHook(ctx, hook, releasePath, env, user); err != nil {
			return fmt.Errorf("hook %d (%s): %w", i+1, truncate(hook, 60), err)
		}
	}
	return nil
}

// RunScript executes a multi-line deploy script with bash -e in releasePath.
func RunScript(ctx context.Context, script, releasePath string, env HookEnv, runAsUser string) error {
	script = strings.TrimRight(script, "\n") + "\n"
	user := runAsUser
	if hookInvokesAbstrax(script) {
		user = ""
	}

	tmp, err := os.CreateTemp(releasePath, ".abstrax-hook-script-*.sh")
	if err != nil {
		return fmt.Errorf("creating hook script: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if _, err := tmp.WriteString(script); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing hook script: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o700); err != nil {
		return err
	}

	var cmd *exec.Cmd
	if user != "" && user != "root" && os.Geteuid() == 0 {
		cmd = exec.CommandContext(ctx, "runuser", "-u", user, "--", "bash", "-e", tmpPath)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-e", tmpPath)
	}
	cmd.Dir = releasePath
	cmd.Env = env.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		out := strings.TrimSpace(stdout.String())
		errOut := strings.TrimSpace(stderr.String())
		msg := combineHookOutput(out, errOut)
		if msg == "" {
			msg = err.Error()
		}
		msg = tailLines(msg, 40)
		return fmt.Errorf("hook script (%s): %s", filepath.Base(tmpPath), msg)
	}
	return nil
}

func runHook(ctx context.Context, hook, cwd string, env HookEnv, runAsUser string) error {
	var cmd *exec.Cmd
	if runAsUser != "" && runAsUser != "root" && os.Geteuid() == 0 {
		cmd = exec.CommandContext(ctx, "runuser", "-u", runAsUser, "--", "bash", "-lc", hook)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-lc", hook)
	}
	cmd.Dir = cwd
	cmd.Env = env.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		out := strings.TrimSpace(stdout.String())
		errOut := strings.TrimSpace(stderr.String())
		msg := combineHookOutput(out, errOut)
		if msg == "" {
			msg = err.Error()
		}
		// Keep the useful tail when Composer progress dominates the buffer.
		msg = tailLines(msg, 40)
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// hookInvokesAbstrax reports whether the hook shell string calls the Abstrax CLI
// or a plugin binary (abstrax-*). Those must run with the installing user's view
// of plugin paths (typically root → /usr/local/lib/abstrax/plugins).
func hookInvokesAbstrax(hook string) bool {
	for _, field := range strings.Fields(hook) {
		// Strip simple env assignments: FOO=bar abstrax …
		if i := strings.IndexByte(field, '='); i > 0 && !strings.Contains(field[:i], "/") {
			continue
		}
		base := field
		if j := strings.LastIndexByte(field, '/'); j >= 0 {
			base = field[j+1:]
		}
		if base == "abstrax" || strings.HasPrefix(base, "abstrax-") {
			return true
		}
	}
	return false
}

func combineHookOutput(stdout, stderr string) string {
	switch {
	case stdout != "" && stderr != "":
		return stderr + "\n" + stdout
	case stderr != "":
		return stderr
	default:
		return stdout
	}
}

func tailLines(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
