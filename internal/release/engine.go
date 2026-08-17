package release

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/gitx"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/output"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/presets"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/userx"
)

// DeployOptions controls a deploy now run.
type DeployOptions struct {
	ProjectPath string
	Config      *config.Config
	Ref         string
	Keep        int
	SkipHooks   bool
	NoActivate  bool
	DryRun      bool
	RunAsUser   string
	Owner       string // project Linux user; release tree is chowned before hooks
	CLIPHP      string
	KnownHosts  string
	Now         time.Time
	Progress    output.ProgressEmitter
	Action      string
}

// DeployResult is returned on success.
type DeployResult struct {
	ReleaseID string   `json:"release_id"`
	Ref       string   `json:"ref"`
	SHA       string   `json:"sha,omitempty"`
	ShortSHA  string   `json:"short_sha,omitempty"`
	Activated bool     `json:"activated"`
	Pruned    []string `json:"pruned,omitempty"`
}

// Deploy runs the full zero-downtime pipeline.
func Deploy(ctx context.Context, opts DeployOptions) (*DeployResult, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if opts.ProjectPath == "" {
		return nil, fmt.Errorf("project path is required")
	}
	if opts.Action == "" {
		opts.Action = "deploy.now"
	}
	progress := opts.Progress
	if progress == nil {
		progress = output.NewPrinter(false, false, false, false, false)
	}
	cfg := opts.Config
	ref := opts.Ref
	if ref == "" {
		ref = cfg.Branch
	}
	keep := opts.Keep
	if keep < 1 {
		keep = cfg.KeepReleases
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	releaseID := layout.NewReleaseID(now)
	paths := layout.ForProject(opts.ProjectPath)
	releasePath := paths.ReleasePath(releaseID)

	step := func(name, msg string) {
		progress.Progress(opts.Action, name, msg)
	}

	if cfg.Repository == "" {
		return nil, fmt.Errorf("repository is not configured; run abstrax deploy configure")
	}
	if cfg.DeployKey == "" {
		return nil, fmt.Errorf("deploy key is not configured; run abstrax deploy key")
	}

	if opts.DryRun {
		progress.DryRun("would create release %s from %s@%s", releaseID, cfg.Repository, ref)
		progress.DryRun("would symlink shared paths, run hooks, health-check, activate, prune keep=%d", keep)
		return &DeployResult{ReleaseID: releaseID, Ref: ref, Activated: !opts.NoActivate}, nil
	}

	step("prepare", "Preparing release "+releaseID)
	if err := layout.EnsureScaffold(opts.ProjectPath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(releasePath, 0o755); err != nil {
		return nil, err
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(releasePath)
		}
	}()

	step("clone", "Cloning "+cfg.Repository+" @ "+ref)
	cloneRes, err := gitx.CloneFresh(ctx, gitx.CloneOptions{
		Repository: cfg.Repository,
		Ref:        ref,
		Dest:       releasePath,
		SSHKey:     cfg.DeployKey,
		KnownHosts: opts.KnownHosts,
	})
	if err != nil {
		return nil, fmt.Errorf("clone failed: %w", err)
	}

	meta := Meta{
		ID:        releaseID,
		Ref:       ref,
		SHA:       cloneRes.SHA,
		ShortSHA:  cloneRes.ShortSHA,
		Branch:    cfg.Branch,
		CreatedAt: now,
	}
	if err := WriteMeta(releasePath, meta); err != nil {
		return nil, err
	}
	if err := gitx.RemoveGitDir(releasePath); err != nil {
		return nil, err
	}

	step("shared", "Linking shared paths")
	if err := presets.EnsureSharedScaffold(opts.ProjectPath, cfg); err != nil {
		return nil, err
	}
	if err := LinkShared(opts.ProjectPath, releasePath, cfg.Shared); err != nil {
		return nil, err
	}

	// Clone/link run as root; hooks run as the project user and must be able to
	// write into the release (e.g. composer creating vendor/).
	if opts.Owner != "" {
		if err := userx.ChownPath(releasePath, opts.Owner); err != nil {
			return nil, fmt.Errorf("setting release ownership: %w", err)
		}
		if err := userx.ChownPath(paths.Shared, opts.Owner); err != nil {
			return nil, fmt.Errorf("setting shared ownership: %w", err)
		}
	}

	hookEnv := HookEnv{
		Project:     cfg.Project,
		ProjectPath: opts.ProjectPath,
		ReleasePath: releasePath,
		CurrentPath: paths.Current,
		SharedPath:  paths.Shared,
		Branch:      cfg.Branch,
		Ref:         ref,
		ReleaseID:   releaseID,
		CLIPHP:      opts.CLIPHP,
	}

	if !opts.SkipHooks {
		step("hooks_after_clone", "Running after_clone hooks")
		if err := RunHooks(ctx, cfg.Hooks.AfterClone, releasePath, hookEnv, opts.RunAsUser); err != nil {
			return nil, err
		}
		step("hooks_before_activate", "Running before_activate hooks")
		if err := RunHooks(ctx, cfg.Hooks.BeforeActivate, releasePath, hookEnv, opts.RunAsUser); err != nil {
			return nil, err
		}
	}

	step("health", "Running health check")
	if err := HealthCheck(releasePath, cfg.PublicDir); err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	activated := false
	if !opts.NoActivate {
		step("activate", "Activating release "+releaseID)
		if err := Activate(opts.ProjectPath, releaseID); err != nil {
			return nil, err
		}
		activated = true

		if !opts.SkipHooks {
			step("hooks_after_activate", "Running after_activate hooks")
			if err := RunHooks(ctx, cfg.Hooks.AfterActivate, releasePath, hookEnv, opts.RunAsUser); err != nil {
				return nil, err
			}
		}

		step("prune", fmt.Sprintf("Pruning old releases (keep %d)", keep))
		pruned, err := Prune(opts.ProjectPath, keep)
		if err != nil {
			return nil, err
		}
		cleanup = false
		return &DeployResult{
			ReleaseID: releaseID,
			Ref:       ref,
			SHA:       cloneRes.SHA,
			ShortSHA:  cloneRes.ShortSHA,
			Activated: activated,
			Pruned:    pruned,
		}, nil
	}

	cleanup = false
	return &DeployResult{
		ReleaseID: releaseID,
		Ref:       ref,
		SHA:       cloneRes.SHA,
		ShortSHA:  cloneRes.ShortSHA,
		Activated: activated,
	}, nil
}

// RollbackOptions controls rollback.
type RollbackOptions struct {
	ProjectPath string
	Config      *config.Config
	ReleaseID   string
	DryRun      bool
	SkipHooks   bool
	RunAsUser   string
	Owner       string
	CLIPHP      string
	Progress    output.ProgressEmitter
	Action      string
}

// Rollback points current at releaseID (or previous) and re-runs after_activate hooks.
func Rollback(ctx context.Context, opts RollbackOptions) (*DeployResult, error) {
	if opts.Action == "" {
		opts.Action = "deploy.rollback"
	}
	progress := opts.Progress
	if progress == nil {
		progress = output.NewPrinter(false, false, false, false, false)
	}
	cfg := opts.Config
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	id := opts.ReleaseID
	var err error
	if id == "" {
		id, err = PreviousReleaseID(opts.ProjectPath)
		if err != nil {
			return nil, err
		}
	}

	paths := layout.ForProject(opts.ProjectPath)
	releasePath := paths.ReleasePath(id)

	if opts.DryRun {
		progress.DryRun("would activate release %s and re-run after_activate hooks", id)
		return &DeployResult{ReleaseID: id, Activated: true}, nil
	}

	progress.Progress(opts.Action, "health", "Verifying release "+id)
	if err := HealthCheck(releasePath, cfg.PublicDir); err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	progress.Progress(opts.Action, "activate", "Activating release "+id)
	if err := Activate(opts.ProjectPath, id); err != nil {
		return nil, err
	}

	meta, _ := ReadMeta(releasePath)
	ref := ""
	sha := ""
	short := ""
	if meta != nil {
		ref = meta.Ref
		sha = meta.SHA
		short = meta.ShortSHA
	}

	if !opts.SkipHooks {
		progress.Progress(opts.Action, "hooks_after_activate", "Running after_activate hooks")
		hookEnv := HookEnv{
			Project:     cfg.Project,
			ProjectPath: opts.ProjectPath,
			ReleasePath: releasePath,
			CurrentPath: paths.Current,
			SharedPath:  paths.Shared,
			Branch:      cfg.Branch,
			Ref:         ref,
			ReleaseID:   id,
			CLIPHP:      opts.CLIPHP,
		}
		if err := RunHooks(ctx, cfg.Hooks.AfterActivate, releasePath, hookEnv, opts.RunAsUser); err != nil {
			return nil, err
		}
	}

	return &DeployResult{ReleaseID: id, Ref: ref, SHA: sha, ShortSHA: short, Activated: true}, nil
}
