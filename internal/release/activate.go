package release

import (
	"fmt"
	"os"
	"path/filepath"
)

// Activate atomically points current at releaseID under projectPath.
// Uses: ln -sfn releases/{id} current.tmp && mv -Tf current.tmp current
func Activate(projectPath, releaseID string) error {
	releasesRel := filepath.Join("releases", releaseID)
	releaseAbs := filepath.Join(projectPath, releasesRel)
	info, err := os.Stat(releaseAbs)
	if err != nil {
		return fmt.Errorf("release %s not found: %w", releaseID, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("release %s is not a directory", releaseID)
	}

	current := filepath.Join(projectPath, "current")
	tmp := filepath.Join(projectPath, "current.tmp")

	_ = os.Remove(tmp)
	if err := os.Symlink(releasesRel, tmp); err != nil {
		return fmt.Errorf("creating temporary current symlink: %w", err)
	}
	if err := os.Rename(tmp, current); err != nil {
		// Some platforms cannot rename over an existing symlink; replace explicitly.
		_ = os.Remove(current)
		if err2 := os.Rename(tmp, current); err2 != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("activating current symlink: %w", err2)
		}
	}
	return nil
}
