package commands_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "abstrax-deploy")

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "build", "-o", out, "./cmd/abstrax-deploy")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, output)
	}
	return out
}

func TestPluginMetadata(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "plugin", "metadata")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() > 0 {
		t.Fatalf("stderr should be empty, got %q", stderr.String())
	}
	var meta map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["name"] != "deploy" {
		t.Fatalf("name = %v", meta["name"])
	}
	commands, _ := meta["commands"].([]any)
	names := map[string]bool{}
	for _, c := range commands {
		m := c.(map[string]any)
		names[m["name"].(string)] = true
	}
	for _, want := range []string{"setup", "init", "configure", "key", "now", "rollback", "list", "status", "hooks"} {
		if !names[want] {
			t.Fatalf("missing command %q in metadata", want)
		}
	}
	if names["release"] {
		t.Fatal("deploy release must not exist")
	}
}

func TestVersion(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Plugin version:") {
		t.Fatalf("output = %q", got)
	}
}

func TestUnknownCommandExitCode(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "unknown-cmd")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error")
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if ee.ExitCode() != 2 {
		t.Fatalf("exit code = %d, want 2", ee.ExitCode())
	}
}

func TestJSONStreamMutuallyExclusive(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "version", "--json", "--json-stream")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error")
	}
}
