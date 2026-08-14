package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("example.com")
	cfg.Repository = "git@github.com:acme/app.git"
	cfg.Shared = []string{".env", "storage"}
	cfg.Hooks.AfterClone = []string{"composer install"}

	if err := config.Save(dir, &cfg); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Repository != cfg.Repository {
		t.Fatalf("repository = %q", got.Repository)
	}
	if len(got.Shared) != 2 {
		t.Fatalf("shared = %#v", got.Shared)
	}
	if _, err := os.Stat(filepath.Join(dir, "deploy.json")); err != nil {
		t.Fatal(err)
	}
}

func TestValidateKeepReleases(t *testing.T) {
	cfg := config.Default("x")
	cfg.KeepReleases = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateUnsupportedProvider(t *testing.T) {
	cfg := config.Default("x")
	cfg.Provider = "gitlab"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}
