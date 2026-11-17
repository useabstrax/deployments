package release

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
)

func TestCombineHookOutputIncludesStdout(t *testing.T) {
	got := combineHookOutput("artisan error here", "composer progress")
	if got != "composer progress\nartisan error here" {
		t.Fatalf("got %q", got)
	}
}

func TestTailLines(t *testing.T) {
	in := "a\nb\nc\nd\ne"
	got := tailLines(in, 3)
	if got != "c\nd\ne" {
		t.Fatalf("got %q", got)
	}
}

func TestHookInvokesAbstrax(t *testing.T) {
	cases := map[string]bool{
		`abstrax composer run --project="$ABSTRAX_PROJECT" install`: true,
		`/usr/bin/abstrax composer run install`:                     true,
		`abstrax-composer run install`:                              true,
		`FOO=1 abstrax composer run install`:                        true,
		`$ABSTRAX_CLI_PHP artisan migrate --force`:                  false,
		`composer install --no-dev`:                                 false,
		`npm ci && npm run build`:                                   false,
	}
	for hook, want := range cases {
		if got := hookInvokesAbstrax(hook); got != want {
			t.Fatalf("%q: got %v want %v", hook, got, want)
		}
	}
}

func TestRunScriptMultilineAndFailFast(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	marker := filepath.Join(dir, "ok.txt")
	script := "FOO=bar\necho \"$FOO\" > ok.txt\nif [ -f ok.txt ]; then\n  echo yes >> ok.txt\nfi\n"
	env := HookEnv{ReleasePath: dir, Project: "app"}
	if err := RunScript(context.Background(), script, dir, env, ""); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "bar") || !strings.Contains(got, "yes") {
		t.Fatalf("script output = %q", got)
	}

	failScript := "echo before\nfalse\necho after\n"
	err = RunScript(context.Background(), failScript, dir, env, "")
	if err == nil {
		t.Fatal("expected failure from bash -e")
	}
}

func TestRunPhasePrefersScriptOverCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config.Default("app")
	cfg.Hooks.AfterClone = []string{"echo commands-ran > from-commands.txt"}
	cfg.Hooks.AfterCloneScript = "echo script-ran > from-script.txt\n"
	env := HookEnv{ReleasePath: dir, Project: "app"}
	if err := RunPhase(context.Background(), &cfg, PhaseAfterClone, dir, env, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "from-script.txt")); err != nil {
		t.Fatal("expected script to run")
	}
	if _, err := os.Stat(filepath.Join(dir, "from-commands.txt")); !os.IsNotExist(err) {
		t.Fatal("commands should not run when script is set")
	}
}

func TestConfigScriptRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config.Default("example.com")
	script := "cd \"$ABSTRAX_RELEASE_PATH\"\nif [ -f artisan ]; then\n  echo migrate\nfi\n"
	if err := cfg.Hooks.SetScript(PhaseAfterClone, script); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(dir, &cfg); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hooks.AfterCloneScript != script {
		t.Fatalf("script round-trip mismatch:\n got %q\nwant %q", got.Hooks.AfterCloneScript, script)
	}
}
