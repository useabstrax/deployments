package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// InstallFakeAbstrax writes a fake abstrax executable and returns its path.
func InstallFakeAbstrax(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "abstrax")
	script := `#!/bin/sh
set -eu

if [ "$1" = "project" ] && [ "$2" = "inspect" ]; then
  name="$3"
  json="$4"
  if [ "$json" != "--json" ]; then
    echo "expected --json" >&2
    exit 1
  fi
  case "$name" in
    exists)
      cat <<'EOF'
{
  "api_version": "v1",
  "project": {
    "name": "example",
    "path": "/var/www/example.com",
    "user": "example",
    "runtime": {"type": "php", "version": "8.5"},
    "domains": ["example.com"],
    "services": [{"name": "example-worker", "type": "worker"}]
  }
}
EOF
      exit 0
      ;;
    missing)
      echo 'project "missing" not found' >&2
      exit 1
      ;;
    badjson)
      echo 'not json'
      exit 0
      ;;
    oldapi)
      cat <<'EOF'
{"api_version": "v0", "project": {"name": "old"}}
EOF
      exit 0
      ;;
    slow)
      sleep 30
      exit 0
      ;;
    *)
      echo "unknown project $name" >&2
      exit 1
      ;;
  esac
fi

if [ "$1" = "project" ] && [ "$2" = "modify" ]; then
  name="$3"
  # Accept --public-dir=... and --json in any order after name
  echo "{\"status\":\"success\",\"action\":\"project.modify\",\"summary\":\"Project ${name} updated.\"}"
  exit 0
fi

if [ "$1" = "project" ] && [ "$2" = "service" ]; then
  action="$3"
  project="$4"
  service="$5"
  json="$6"
  if [ "$json" != "--json" ]; then
    echo "expected --json" >&2
    exit 1
  fi
  if [ -z "$project" ] || [ -z "$service" ]; then
    echo "project and service required" >&2
    exit 1
  fi
  case "$action" in
    restart)
      cat <<EOF
{"status":"success","action":"project.service.restart","summary":"Restarted service ${service} for project ${project}."}
EOF
      exit 0
      ;;
    reload)
      cat <<EOF
{"status":"success","action":"project.service.reload","summary":"Reloaded service ${service} for project ${project}."}
EOF
      exit 0
      ;;
    fail)
      cat <<EOF
{"status":"error","action":"","error_code":"command_error","message":"service restart failed"}
EOF
      exit 1
      ;;
    *)
      echo "unknown action $action" >&2
      exit 1
      ;;
  esac
fi

echo "unsupported invocation: $*" >&2
exit 1
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake abstrax: %v", err)
	}
	return path
}

// WithEnvBinary sets ABSTRAX_BINARY for the duration of the test.
func WithEnvBinary(t *testing.T, path string) {
	t.Helper()
	t.Setenv("ABSTRAX_BINARY", path)
}

// AssertContains fails if s does not contain substr.
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected %q to contain %q", s, substr)
	}
}

// AssertErrorIs fails if err does not wrap target.
func AssertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %v, got nil", target)
	}
	if target != nil && !strings.Contains(err.Error(), target.Error()) {
		t.Fatalf("expected error wrapping %v, got %v", target, err)
	}
}

// FormatRuntime returns a display runtime string for tests.
func FormatRuntime(runtimeType, version string) string {
	return fmt.Sprintf("%s %s", strings.ToUpper(runtimeType), version)
}
