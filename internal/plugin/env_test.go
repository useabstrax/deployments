package plugin_test

import (
	"testing"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/plugin"
)

func TestIsRunningAsPlugin(t *testing.T) {
	t.Setenv("ABSTRAX_PLUGIN", "1")
	if !plugin.IsRunningAsPlugin() {
		t.Fatal("expected true when ABSTRAX_PLUGIN=1")
	}

	t.Setenv("ABSTRAX_PLUGIN", "")
	if plugin.IsRunningAsPlugin() {
		t.Fatal("expected false when ABSTRAX_PLUGIN is empty")
	}
}

func TestHostVersion(t *testing.T) {
	t.Setenv("ABSTRAX_VERSION", "1.2.3")
	if got := plugin.HostVersion(); got != "1.2.3" {
		t.Fatalf("HostVersion() = %q, want 1.2.3", got)
	}
}

func TestHostBinary(t *testing.T) {
	t.Setenv("ABSTRAX_BINARY", "/usr/bin/abstrax")
	if got := plugin.HostBinary(); got != "/usr/bin/abstrax" {
		t.Fatalf("HostBinary() = %q", got)
	}
}
