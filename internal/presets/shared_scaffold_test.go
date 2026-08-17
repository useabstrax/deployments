package presets_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/presets"
)

func TestEnsureSharedScaffoldLaravel(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default("app")
	if err := presets.Apply(&cfg, "laravel"); err != nil {
		t.Fatal(err)
	}
	if err := presets.EnsureSharedScaffold(root, &cfg); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		"shared/storage/framework/views",
		"shared/storage/framework/cache/data",
		"shared/storage/framework/sessions",
		"shared/storage/logs",
		"shared/storage/app/public",
		"shared/.env",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
	}

	env, err := os.ReadFile(filepath.Join(root, "shared/.env"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(env)
	if !strings.Contains(body, "APP_KEY=base64:") {
		t.Fatalf("missing APP_KEY in %s", body)
	}
	if !strings.Contains(body, "SESSION_DRIVER=file") {
		t.Fatalf("expected file session driver")
	}
}

func TestEnsureSharedScaffoldDoesNotOverwriteEnv(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default("app")
	_ = presets.Apply(&cfg, "laravel")
	shared := filepath.Join(root, "shared")
	_ = os.MkdirAll(shared, 0o755)
	path := filepath.Join(shared, ".env")
	if err := os.WriteFile(path, []byte("APP_KEY=existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := presets.EnsureSharedScaffold(root, &cfg); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "APP_KEY=existing\n" {
		t.Fatalf("env overwritten: %q", got)
	}
}
