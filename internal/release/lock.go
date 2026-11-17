package release

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/layout"
)

// LockFilename is the per-project deploy lock under the project path.
const LockFilename = ".abstrax-deploy.lock"

// ErrDeployInProgress is returned when another live deploy holds the lock.
var ErrDeployInProgress = errors.New("deploy already in progress")

// ErrCancelled is returned when a deploy is stopped by deploy cancel or signal.
var ErrCancelled = errors.New("deploy cancelled")

// IsCancelled reports whether err is or wraps ErrCancelled.
func IsCancelled(err error) bool {
	return errors.Is(err, ErrCancelled)
}

// LockState records the in-flight deploy for cancel and status.
type LockState struct {
	PID       int       `json:"pid"`
	ReleaseID string    `json:"release_id"`
	Step      string    `json:"step"`
	StartedAt time.Time `json:"started_at"`
}

// LockPath returns the lock file path for a project root.
func LockPath(projectPath string) string {
	return filepath.Join(projectPath, LockFilename)
}

// ReadLock loads the lock file, or nil if missing.
func ReadLock(projectPath string) (*LockState, error) {
	data, err := os.ReadFile(LockPath(projectPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var st LockState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("invalid deploy lock: %w", err)
	}
	return &st, nil
}

// WriteLock writes the lock file atomically.
func WriteLock(projectPath string, st *LockState) error {
	if st == nil {
		return fmt.Errorf("lock state is nil")
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := LockPath(projectPath)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ClearLock removes the lock file if present.
func ClearLock(projectPath string) error {
	err := os.Remove(LockPath(projectPath))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ProcessAlive reports whether pid appears to be running.
func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

// AcquireLock claims the deploy lock for this process.
// Stale locks (dead PID) are cleared first. Returns ErrDeployInProgress if a live deploy holds it.
func AcquireLock(projectPath string, releaseID, step string) (*LockState, error) {
	existing, err := ReadLock(projectPath)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if ProcessAlive(existing.PID) {
			return nil, fmt.Errorf("%w (pid %d, release %s, step %s)", ErrDeployInProgress, existing.PID, existing.ReleaseID, existing.Step)
		}
		// Stale lock: drop it (and leave release cleanup to cancel / next failure path).
		_ = ClearLock(projectPath)
	}
	st := &LockState{
		PID:       os.Getpid(),
		ReleaseID: releaseID,
		Step:      step,
		StartedAt: time.Now().UTC(),
	}
	if err := WriteLock(projectPath, st); err != nil {
		return nil, err
	}
	return st, nil
}

// UpdateLockStep rewrites the lock with a new step name.
func UpdateLockStep(projectPath, step string) error {
	st, err := ReadLock(projectPath)
	if err != nil {
		return err
	}
	if st == nil {
		return nil
	}
	if st.PID != os.Getpid() {
		return nil
	}
	st.Step = step
	return WriteLock(projectPath, st)
}

// RemoveIncompleteRelease deletes releaseID under releases/ when it is not current.
func RemoveIncompleteRelease(projectPath, releaseID string) error {
	if releaseID == "" {
		return nil
	}
	paths := layout.ForProject(projectPath)
	currentID, err := paths.CurrentReleaseID()
	if err != nil {
		return err
	}
	if currentID != "" && currentID == releaseID {
		return nil
	}
	rel := paths.ReleasePath(releaseID)
	if err := os.RemoveAll(rel); err != nil {
		return fmt.Errorf("removing incomplete release %s: %w", releaseID, err)
	}
	return nil
}
