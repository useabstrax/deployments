package presets_test

import (
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/presets"
)

func TestLaravelPreset(t *testing.T) {
	cfg := config.Default("app")
	if err := presets.Apply(&cfg, "laravel"); err != nil {
		t.Fatal(err)
	}
	if cfg.PublicDir != "public" {
		t.Fatalf("public_dir = %q", cfg.PublicDir)
	}
	if len(cfg.Shared) != 2 {
		t.Fatalf("shared = %#v", cfg.Shared)
	}
	if len(cfg.Hooks.AfterClone) != 1 || len(cfg.Hooks.BeforeActivate) != 1 {
		t.Fatalf("hooks = %#v", cfg.Hooks)
	}
}

func TestUnknownPreset(t *testing.T) {
	cfg := config.Default("app")
	if err := presets.Apply(&cfg, "django"); err == nil {
		t.Fatal("expected error")
	}
}
