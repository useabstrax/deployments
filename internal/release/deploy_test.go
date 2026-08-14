package release_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/key"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
)

func TestDeployShallowCloneRemovesGitAndActivates(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	root := t.TempDir()
	repo := filepath.Join(root, "repo.git")
	work := filepath.Join(root, "work")
	project := filepath.Join(root, "project")
	keyPath := filepath.Join(root, "id_ed25519")

	mustRun(t, root, "git", "init", "--bare", repo)
	mustRun(t, root, "git", "clone", repo, work)
	mustRun(t, work, "git", "config", "user.email", "test@example.com")
	mustRun(t, work, "git", "config", "user.name", "Test")
	if err := os.MkdirAll(filepath.Join(work, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "public", "index.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, work, "git", "add", ".")
	mustRun(t, work, "git", "commit", "-m", "init")
	mustRun(t, work, "git", "branch", "-M", "main")
	mustRun(t, work, "git", "push", "-u", "origin", "main")

	if err := key.Generate(keyPath, "test"); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default("app")
	cfg.Repository = repo
	cfg.Branch = "main"
	cfg.PublicDir = "public"
	cfg.DeployKey = keyPath
	cfg.Shared = []string{}
	cfg.Hooks = config.Hooks{}

	// Local path repos don't use SSH; CloneFresh always sets GIT_SSH_COMMAND.
	// For file:// repos this is fine — ssh isn't invoked.
	cfg.Repository = "file://" + repo

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	result, err := release.Deploy(context.Background(), release.DeployOptions{
		ProjectPath: project,
		Config:      &cfg,
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ReleaseID != "20260102030405" {
		t.Fatalf("release id = %q", result.ReleaseID)
	}
	relPath := filepath.Join(project, "releases", result.ReleaseID)
	if _, err := os.Stat(filepath.Join(relPath, ".git")); !os.IsNotExist(err) {
		t.Fatal(".git should be removed")
	}
	if _, err := os.Stat(filepath.Join(relPath, ".abstrax-release.json")); err != nil {
		t.Fatal("release metadata missing")
	}
	if _, err := os.Stat(filepath.Join(relPath, "public", "index.html")); err != nil {
		t.Fatal("release tree missing")
	}
	target, err := os.Readlink(filepath.Join(project, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(target) != result.ReleaseID {
		t.Fatalf("current = %q", target)
	}
}

func TestDeployFailureCleansIncompleteRelease(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	cfg := config.Default("app")
	cfg.Repository = "file:///no/such/repo"
	cfg.DeployKey = filepath.Join(root, "missing_key")
	cfg.PublicDir = "public"

	// Create a dummy key file so clone fails on repo, not key missing check...
	// Actually Deploy checks key path only as string; clone will fail.
	if err := os.WriteFile(cfg.DeployKey, []byte("dummy"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := release.Deploy(context.Background(), release.DeployOptions{
		ProjectPath: project,
		Config:      &cfg,
		Now:         time.Date(2026, 2, 2, 2, 2, 2, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	rel := filepath.Join(project, "releases", "20260202020202")
	if _, err := os.Stat(rel); !os.IsNotExist(err) {
		t.Fatal("incomplete release should be removed")
	}
	if _, err := os.Lstat(filepath.Join(project, "current")); !os.IsNotExist(err) {
		t.Fatal("current must not be created on failure")
	}
}

func TestHealthCheckBlocksActivateMissingPublic(t *testing.T) {
	root := t.TempDir()
	rel := filepath.Join(root, "releases", "1")
	if err := os.MkdirAll(rel, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := release.HealthCheck(rel, "public"); err == nil {
		t.Fatal("expected failure")
	}
}

func mustRun(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
