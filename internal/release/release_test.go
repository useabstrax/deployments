package release_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
)

func TestLinkShared(t *testing.T) {
	root := t.TempDir()
	releasePath := filepath.Join(root, "releases", "1")
	if err := os.MkdirAll(filepath.Join(releasePath, "storage"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(releasePath, ".env"), []byte("OLD=1"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := release.LinkShared(root, releasePath, []string{".env", "storage"}); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{".env", "storage"} {
		link := filepath.Join(releasePath, name)
		fi, err := os.Lstat(link)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s is not a symlink", name)
		}
		shared := filepath.Join(root, "shared", name)
		if _, err := os.Stat(shared); err != nil {
			t.Fatalf("shared target missing: %v", err)
		}
	}
}

func TestActivateAtomic(t *testing.T) {
	root := t.TempDir()
	rel1 := filepath.Join(root, "releases", "20260101000001")
	rel2 := filepath.Join(root, "releases", "20260101000002")
	if err := os.MkdirAll(rel1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rel2, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := release.Activate(root, "20260101000001"); err != nil {
		t.Fatal(err)
	}
	if err := release.Activate(root, "20260101000002"); err != nil {
		t.Fatal(err)
	}

	target, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(target) != "20260101000002" {
		t.Fatalf("target = %q", target)
	}
	if _, err := os.Stat(filepath.Join(root, "current.tmp")); !os.IsNotExist(err) {
		t.Fatalf("tmp symlink should be gone: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	root := t.TempDir()
	rel := filepath.Join(root, "releases", "1")
	if err := os.MkdirAll(filepath.Join(rel, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := release.HealthCheck(rel, "public"); err != nil {
		t.Fatal(err)
	}
	if err := release.HealthCheck(rel, "missing"); err == nil {
		t.Fatal("expected health check failure")
	}
}

func TestPruneKeepN(t *testing.T) {
	root := t.TempDir()
	ids := []string{"20260101000001", "20260101000002", "20260101000003", "20260101000004"}
	for _, id := range ids {
		if err := os.MkdirAll(filepath.Join(root, "releases", id), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := release.Activate(root, "20260101000004"); err != nil {
		t.Fatal(err)
	}

	removed, err := release.Prune(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed = %#v", removed)
	}
	// current must remain
	if _, err := os.Stat(filepath.Join(root, "releases", "20260101000004")); err != nil {
		t.Fatal("current release pruned")
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "20260101000003")); err != nil {
		t.Fatal("newest non-current should be kept")
	}
}

func TestHookEnvInjection(t *testing.T) {
	env := release.HookEnv{
		Project:     "app",
		ProjectPath: "/var/www/app",
		ReleasePath: "/var/www/app/releases/1",
		CurrentPath: "/var/www/app/current",
		SharedPath:  "/var/www/app/shared",
		Branch:      "main",
		Ref:         "main",
		ReleaseID:   "1",
		CLIPHP:      "/usr/bin/php8.5",
	}
	vars := map[string]string{}
	for _, e := range env.Environ() {
		// last wins for duplicates from os.Environ
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				vars[e[:i]] = e[i+1:]
				break
			}
		}
	}
	checks := map[string]string{
		"ABSTRAX_PROJECT":      "app",
		"ABSTRAX_PROJECT_PATH": "/var/www/app",
		"ABSTRAX_RELEASE_PATH": "/var/www/app/releases/1",
		"ABSTRAX_CURRENT_PATH": "/var/www/app/current",
		"ABSTRAX_SHARED_PATH":  "/var/www/app/shared",
		"ABSTRAX_BRANCH":       "main",
		"ABSTRAX_REF":          "main",
		"ABSTRAX_RELEASE_ID":   "1",
		"ABSTRAX_CLI_PHP":      "/usr/bin/php8.5",
	}
	for k, want := range checks {
		if vars[k] != want {
			t.Fatalf("%s = %q, want %q", k, vars[k], want)
		}
	}
}
