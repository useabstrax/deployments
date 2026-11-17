package release

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// CancelResult is returned by CancelDeploy.
type CancelResult struct {
	Cancelled bool   `json:"cancelled"`
	ReleaseID string `json:"release_id,omitempty"`
	Step      string `json:"step,omitempty"`
	PID       int    `json:"pid,omitempty"`
	Message   string `json:"message,omitempty"`
}

// CancelOptions controls CancelDeploy.
type CancelOptions struct {
	ProjectPath string
	// Wait is how long to wait for the deploy process to exit after SIGTERM.
	Wait time.Duration
}

// CancelDeploy stops an in-flight deploy for the project, removes the unfinished
// release (never touching current), and clears the lock.
func CancelDeploy(opts CancelOptions) (*CancelResult, error) {
	if opts.ProjectPath == "" {
		return nil, fmt.Errorf("project path is required")
	}
	wait := opts.Wait
	if wait <= 0 {
		wait = 15 * time.Second
	}

	st, err := ReadLock(opts.ProjectPath)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return &CancelResult{
			Cancelled: false,
			Message:   "No deploy in progress.",
		}, nil
	}

	alive := ProcessAlive(st.PID)
	result := &CancelResult{
		ReleaseID: st.ReleaseID,
		Step:      st.Step,
		PID:       st.PID,
	}

	if alive {
		// Prefer process-group signal so hook children stop too (deploy sets its own pgid).
		_ = syscall.Kill(-st.PID, syscall.SIGTERM)
		_ = syscall.Kill(st.PID, syscall.SIGTERM)

		deadline := time.Now().Add(wait)
		for ProcessAlive(st.PID) && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		if ProcessAlive(st.PID) {
			_ = syscall.Kill(-st.PID, syscall.SIGKILL)
			_ = syscall.Kill(st.PID, syscall.SIGKILL)
			deadline = time.Now().Add(3 * time.Second)
			for ProcessAlive(st.PID) && time.Now().Before(deadline) {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	if err := RemoveIncompleteRelease(opts.ProjectPath, st.ReleaseID); err != nil {
		return nil, err
	}
	if err := ClearLock(opts.ProjectPath); err != nil {
		return nil, err
	}

	if !alive {
		result.Cancelled = false
		result.Message = fmt.Sprintf("No live deploy (cleared stale lock for release %s).", st.ReleaseID)
		return result, nil
	}

	result.Cancelled = true
	result.Message = fmt.Sprintf("Cancelled deploy of release %s at step %s.", st.ReleaseID, st.Step)
	return result, nil
}

// BecomeProcessGroupLeader makes the current process a process-group leader so
// cancel can signal the whole tree (hooks included).
func BecomeProcessGroupLeader() error {
	return syscall.Setpgid(0, 0)
}

// OwnLockPID reports whether the lock file belongs to this process.
func OwnLockPID(projectPath string) bool {
	st, err := ReadLock(projectPath)
	if err != nil || st == nil {
		return false
	}
	return st.PID == os.Getpid()
}
