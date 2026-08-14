package abstrax_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax"
	"github.com/useabstrax/abstrax/plugins/deploy/sdk/abstrax/testutil"
)

func TestNewWithBinary(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}
	if client.Binary != fake {
		t.Fatalf("binary = %q, want %q", client.Binary, fake)
	}
}

func TestNewPrefersABSTRAXBinary(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	testutil.WithEnvBinary(t, fake)

	client, err := abstrax.New()
	if err != nil {
		t.Fatal(err)
	}
	if client.Binary != fake {
		t.Fatalf("binary = %q, want %q", client.Binary, fake)
	}
}

func TestNewBinaryNotFound(t *testing.T) {
	t.Setenv("ABSTRAX_BINARY", "")
	t.Setenv("PATH", t.TempDir())

	_, err := abstrax.New()
	if !errors.Is(err, abstrax.ErrBinaryNotFound) {
		t.Fatalf("expected ErrBinaryNotFound, got %v", err)
	}
}

func TestProjectSuccess(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Project(context.Background(), "exists")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Project.Name != "example" {
		t.Fatalf("name = %q", resp.Project.Name)
	}
	if resp.Project.Runtime.Version != "8.5" {
		t.Fatalf("runtime version = %q", resp.Project.Runtime.Version)
	}
}

func TestProjectNotFound(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Project(context.Background(), "missing")
	if !errors.Is(err, abstrax.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestProjectMalformedJSON(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Project(context.Background(), "badjson")
	if !errors.Is(err, abstrax.ErrMalformedJSON) {
		t.Fatalf("expected ErrMalformedJSON, got %v", err)
	}
}

func TestProjectUnsupportedAPIVersion(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Project(context.Background(), "oldapi")
	if !errors.Is(err, abstrax.ErrUnsupportedAPIVersion) {
		t.Fatalf("expected ErrUnsupportedAPIVersion, got %v", err)
	}
}

func TestProjectContextCancellation(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = client.Project(ctx, "slow")
	if err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestRestartProjectService(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.RestartProjectService(context.Background(), "example", "worker"); err != nil {
		t.Fatal(err)
	}
}

func TestReloadProjectService(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.ReloadProjectService(context.Background(), "example", "worker"); err != nil {
		t.Fatal(err)
	}
}

func TestServiceMissingArgs(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.RestartProjectService(context.Background(), "", "worker"); err == nil {
		t.Fatal("expected error for empty project")
	}
}

func TestNewWithBinaryInvalidPath(t *testing.T) {
	_, err := abstrax.NewWithBinary("/nonexistent/abstrax")
	if !errors.Is(err, abstrax.ErrBinaryNotFound) {
		t.Fatalf("expected ErrBinaryNotFound, got %v", err)
	}
}

func TestNewWithBinaryDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := abstrax.NewWithBinary(dir)
	if !errors.Is(err, abstrax.ErrBinaryNotFound) {
		t.Fatalf("expected ErrBinaryNotFound, got %v", err)
	}
}

func TestLookPathFallback(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "abstrax")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ABSTRAX_BINARY", "")
	t.Setenv("PATH", dir)

	client, err := abstrax.New()
	if err != nil {
		t.Fatal(err)
	}
	if client.Binary != fake {
		t.Fatalf("binary = %q, want %q", client.Binary, fake)
	}
}

func TestHostCommandErrorExitCode(t *testing.T) {
	fake := testutil.InstallFakeAbstrax(t)
	client, err := abstrax.NewWithBinary(fake)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Project(context.Background(), "missing")
	var hostErr *abstrax.HostCommandError
	if errors.As(err, &hostErr) {
		t.Fatal("expected ErrProjectNotFound, not HostCommandError")
	}
	if !errors.Is(err, abstrax.ErrProjectNotFound) {
		t.Fatalf("got %v", err)
	}
}
