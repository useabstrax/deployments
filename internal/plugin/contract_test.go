package plugin_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/plugin"
)

func TestDefaultMetadataShape(t *testing.T) {
	meta := plugin.DefaultMetadata()
	if meta.Name != "deploy" {
		t.Fatalf("name = %q", meta.Name)
	}
	if meta.ProtocolVersion != 1 {
		t.Fatalf("protocol = %d", meta.ProtocolVersion)
	}
	if len(meta.Commands) < 9 {
		t.Fatalf("commands = %d", len(meta.Commands))
	}
	for _, c := range meta.Commands {
		if c.Name == "release" {
			t.Fatal("release command must not be listed")
		}
	}
}

func TestReleaseManifestExampleMatchesFixtureShape(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	examplePath := filepath.Join(filepath.Dir(file), "..", "..", "plugin-manifest.example.json")
	example, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(example, &got); err != nil {
		t.Fatal(err)
	}
	required := []string{"name", "version", "protocol_version", "requires_abstrax", "channel", "platforms"}
	for _, key := range required {
		if _, ok := got[key]; !ok {
			t.Fatalf("example manifest missing %q", key)
		}
	}
	if got["name"] != "deploy" {
		t.Fatalf("name = %v", got["name"])
	}
}
