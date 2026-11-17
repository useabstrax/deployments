package release_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
	"github.com/useabstrax/abstrax/plugins/deploy/internal/release"
)

func TestAcquireLockAndStaleClear(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	st, err := release.AcquireLock(dir, "20260101010101", "clone")
	if err != nil {
		t.Fatal(err)
	}
	if st.PID != os.Getpid() {
		t.Fatalf("pid = %d", st.PID)
	}

	_, err = release.AcquireLock(dir, "20260101010102", "clone")
	if err == nil {
		t.Fatal("expected deploy in progress")
	}

	// Simulate stale lock from a dead PID.
	dead := &release.LockState{
		PID:       1<<30 - 3, // unlikely to be alive
		ReleaseID: "20260101010101",
		Step:      "hooks_after_clone",
		StartedAt: time.Now().UTC(),
	}
	if release.ProcessAlive(dead.PID) {
		t.Skip("unlikely PID is alive; skip stale test")
	}
	if err := release.WriteLock(dir, dead); err != nil {
		t.Fatal(err)
	}
	st2, err := release.AcquireLock(dir, "20260101010103", "prepare")
	if err != nil {
		t.Fatal(err)
	}
	if st2.ReleaseID != "20260101010103" {
		t.Fatalf("release = %q", st2.ReleaseID)
	}
	_ = release.ClearLock(dir)
}

func TestRemoveIncompleteReleaseLeavesCurrent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	paths := layout.ForProject(dir)
	if err := layout.EnsureScaffold(dir); err != nil {
		t.Fatal(err)
	}
	currentID := "20260101000000"
	otherID := "20260101000001"
	if err := os.MkdirAll(paths.ReleasePath(currentID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.ReleasePath(otherID), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(layout.ReleasesDir, currentID), paths.Current); err != nil {
		t.Fatal(err)
	}

	if err := release.RemoveIncompleteRelease(dir, currentID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.ReleasePath(currentID)); err != nil {
		t.Fatal("current release must not be removed")
	}

	if err := release.RemoveIncompleteRelease(dir, otherID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.ReleasePath(otherID)); !os.IsNotExist(err) {
		t.Fatal("incomplete release should be removed")
	}
}

func TestCancelDeployNothingRunning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	res, err := release.CancelDeploy(release.CancelOptions{ProjectPath: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Cancelled {
		t.Fatal("expected not cancelled")
	}
}

func TestCancelDeployClearsStaleLockAndRelease(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	paths := layout.ForProject(dir)
	if err := layout.EnsureScaffold(dir); err != nil {
		t.Fatal(err)
	}
	id := "20260101121212"
	if err := os.MkdirAll(paths.ReleasePath(id), 0o755); err != nil {
		t.Fatal(err)
	}
	deadPID := 1<<30 - 7
	if release.ProcessAlive(deadPID) {
		t.Skip("unlikely PID alive")
	}
	if err := release.WriteLock(dir, &release.LockState{
		PID:       deadPID,
		ReleaseID: id,
		Step:      "hooks_after_clone",
		StartedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	res, err := release.CancelDeploy(release.CancelOptions{ProjectPath: dir})
	if err != nil {
		t.Fatal(err)
	}
	if res.Cancelled {
		t.Fatal("stale lock should not report cancelled=true")
	}
	if _, err := os.Stat(paths.ReleasePath(id)); !os.IsNotExist(err) {
		t.Fatal("stale incomplete release should be removed")
	}
	if st, _ := release.ReadLock(dir); st != nil {
		t.Fatal("lock should be cleared")
	}
}
