package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ReleasesDir = "releases"
	SharedDir   = "shared"
	CurrentLink = "current"
	ReleaseMeta = ".abstrax-release.json"
)

// Paths holds resolved filesystem locations for a project deploy layout.
type Paths struct {
	Project  string
	Releases string
	Shared   string
	Current  string
}

// ForProject returns layout paths under projectPath.
func ForProject(projectPath string) Paths {
	return Paths{
		Project:  projectPath,
		Releases: filepath.Join(projectPath, ReleasesDir),
		Shared:   filepath.Join(projectPath, SharedDir),
		Current:  filepath.Join(projectPath, CurrentLink),
	}
}

// EnsureScaffold creates releases/ and shared/ directories.
func EnsureScaffold(projectPath string) error {
	p := ForProject(projectPath)
	for _, dir := range []string{p.Releases, p.Shared} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	return nil
}

// NewReleaseID returns a UTC timestamp release id YYYYMMDDHHMMSS.
func NewReleaseID(now time.Time) string {
	return now.UTC().Format("20060102150405")
}

// ReleasePath returns the absolute path for a release id.
func (p Paths) ReleasePath(id string) string {
	return filepath.Join(p.Releases, id)
}

// CurrentTarget resolves the current symlink target, or empty if missing.
func (p Paths) CurrentTarget() (string, error) {
	target, err := os.Readlink(p.Current)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(p.Project, target)
	}
	return filepath.Clean(target), nil
}

// CurrentReleaseID returns the release directory name current points at.
func (p Paths) CurrentReleaseID() (string, error) {
	target, err := p.CurrentTarget()
	if err != nil || target == "" {
		return "", err
	}
	return filepath.Base(target), nil
}

// DocumentRootRel returns the public_dir path relative to project root via current.
// Example: current/public
func DocumentRootRel(publicDir string) string {
	return filepath.Join(CurrentLink, publicDir)
}
