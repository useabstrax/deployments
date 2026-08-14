package layout_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
)

func TestNewReleaseID(t *testing.T) {
	ts := time.Date(2026, 8, 14, 21, 30, 45, 0, time.UTC)
	got := layout.NewReleaseID(ts)
	if got != "20260814213045" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureScaffold(t *testing.T) {
	dir := t.TempDir()
	if err := layout.EnsureScaffold(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"releases", "shared"} {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || !st.IsDir() {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestDocumentRootRel(t *testing.T) {
	if got := layout.DocumentRootRel("public"); got != "current/public" {
		t.Fatalf("got %q", got)
	}
}
