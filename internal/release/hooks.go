package release

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
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

// HooksFor returns the hook list for a phase.
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

// RunHooks executes hook commands with bash -lc in releasePath as optional user.
func RunHooks(ctx context.Context, hooks []string, releasePath string, env HookEnv, runAsUser string) error {
	for i, hook := range hooks {
		hook = strings.TrimSpace(hook)
		if hook == "" {
			continue
		}
		if err := runHook(ctx, hook, releasePath, env, runAsUser); err != nil {
			return fmt.Errorf("hook %d (%s): %w", i+1, truncate(hook, 60), err)
		}
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
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
