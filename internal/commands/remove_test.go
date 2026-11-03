package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
)

func TestRemoveDeployArtifacts(t *testing.T) {
	t.Parallel()

	project := t.TempDir()
	paths := layout.ForProject(project)
	if err := layout.EnsureScaffold(project); err != nil {
		t.Fatal(err)
	}
	release := filepath.Join(paths.Releases, "20260101120000")
	if err := os.MkdirAll(release, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(release, "app.txt")
	if err := os.WriteFile(marker, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("releases/20260101120000", paths.Current); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "current.tmp"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.Shared, ".env"), []byte("APP_KEY=x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default("example.com")
	cfg.PreviousPublicDir = "public"
	if err := config.Save(project, &cfg); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(sshDir, "abstrax_deploy_example_com")
	if err := os.WriteFile(keyPath, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath+".pub", []byte("public"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, kept, err := removeDeployArtifacts(project, keyPath, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if pathExists(paths.Current) {
		t.Fatal("current symlink should be removed")
	}
	if pathExists(marker) {
		t.Fatal("release tree should be removed without following current")
	}
	if pathExists(config.Path(project)) {
		t.Fatal("deploy.json should be removed")
	}
	if !pathExists(paths.Shared) {
		t.Fatal("shared should be kept by default")
	}
	if !pathExists(keyPath) {
		t.Fatal("key should be kept by default")
	}
	if len(kept) != 2 {
		t.Fatalf("kept = %#v", kept)
	}
	if len(removed) < 3 {
		t.Fatalf("removed = %#v", removed)
	}

	// Idempotent second pass.
	if _, _, err := removeDeployArtifacts(project, keyPath, true, true); err != nil {
		t.Fatal(err)
	}
	if pathExists(paths.Shared) {
		t.Fatal("shared should be removed with flag")
	}
	if pathExists(keyPath) || pathExists(keyPath+".pub") {
		t.Fatal("key pair should be removed with flag")
	}
}

func TestValidateDeployKeyPath(t *testing.T) {
	t.Parallel()

	if err := validateDeployKeyPath("/var/www/.ssh/abstrax_deploy_app"); err != nil {
		t.Fatal(err)
	}
	if err := validateDeployKeyPath("/tmp/id_ed25519"); err == nil {
		t.Fatal("expected refusal for non-abstrax key name")
	}
	if err := validateDeployKeyPath("/tmp/abstrax_deploy_app"); err == nil {
		t.Fatal("expected refusal when not under .ssh")
	}
}

func TestCapturePreviousPublicDir(t *testing.T) {
	t.Parallel()

	cfg := config.Default("x")
	capturePreviousPublicDir(&cfg, "public")
	if cfg.PreviousPublicDir != "public" {
		t.Fatalf("got %q", cfg.PreviousPublicDir)
	}
	capturePreviousPublicDir(&cfg, "other")
	if cfg.PreviousPublicDir != "public" {
		t.Fatal("should not overwrite existing previous_public_dir")
	}

	cfg2 := config.Default("x")
	capturePreviousPublicDir(&cfg2, "current/public")
	if cfg2.PreviousPublicDir != "" {
		t.Fatalf("should ignore deploy-shaped public dir, got %q", cfg2.PreviousPublicDir)
	}
}

func TestIsDeployPublicDir(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"current":        true,
		"current/public": true,
		"public":         false,
		".":              false,
		"":               false,
	}
	for in, want := range cases {
		if got := isDeployPublicDir(in); got != want {
			t.Fatalf("isDeployPublicDir(%q)=%v want %v", in, got, want)
		}
	}
	if !isDeployPublicDir("current" + string(filepath.Separator) + "public") {
		t.Fatal("platform separator form should count as deploy public dir")
	}
}
