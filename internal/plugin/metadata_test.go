package plugin_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/plugin"
)

func TestDefaultMetadataJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := plugin.WriteMetadata(&buf, plugin.DefaultMetadata()); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}

	required := []string{
		"protocol_version", "name", "display_name", "description",
		"version", "requires_abstrax", "commands",
	}
	for _, key := range required {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing required field %q", key)
		}
	}

	if raw["protocol_version"].(float64) != float64(plugin.ProtocolVersion) {
		t.Errorf("protocol_version = %v, want %d", raw["protocol_version"], plugin.ProtocolVersion)
	}
	if raw["name"] != plugin.PluginName {
		t.Errorf("name = %v, want %q", raw["name"], plugin.PluginName)
	}

	if commands, ok := raw["commands"].([]any); !ok || len(commands) < 9 {
		t.Errorf("commands = %v, want at least 9 entries", raw["commands"])
	} else {
		foundNow := false
		for _, c := range commands {
			m := c.(map[string]any)
			if m["name"] == "now" {
				foundNow = true
				if m["action"] != "plugin.deploy.now" {
					t.Errorf("now action = %v, want plugin.deploy.now", m["action"])
				}
			}
		}
		if !foundNow {
			t.Error("missing now command")
		}
	}
}

func TestWriteMetadataNoTrailingGarbage(t *testing.T) {
	var buf bytes.Buffer
	if err := plugin.WriteMetadata(&buf, plugin.DefaultMetadata()); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("output is not valid JSON: %s", buf.String())
	}
}
