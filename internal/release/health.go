package release

import (
	"fmt"
	"os"
	"path/filepath"
)

// HealthCheck verifies the release is ready to activate.
func HealthCheck(releasePath, publicDir string) error {
	info, err := os.Stat(releasePath)
	if err != nil {
		return fmt.Errorf("release path missing: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("release path is not a directory: %s", releasePath)
	}
	pub := filepath.Join(releasePath, publicDir)
	info, err = os.Stat(pub)
	if err != nil {
		return fmt.Errorf("public_dir %q missing under release: %w", publicDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("public_dir %q is not a directory", publicDir)
	}
	return nil
}
