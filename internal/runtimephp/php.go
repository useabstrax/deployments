package runtimephp

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveCLI returns the best PHP CLI binary for a project runtime version.
// Prefers versioned binaries on Debian/Ubuntu and Remi paths on RHEL-family.
// Falls back to "php" on PATH when nothing matches.
func ResolveCLI(version string) string {
	version = strings.TrimSpace(version)
	candidates := []string{}
	if version != "" {
		compact := strings.ReplaceAll(version, ".", "")
		candidates = append(candidates,
			"php"+version,
			filepath.Join("/opt/remi", "php"+compact, "root/usr/bin/php"),
			filepath.Join("/usr/bin", "php"+version),
		)
	}
	candidates = append(candidates, "php")

	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
		if strings.Contains(c, "/") {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
		}
	}
	return "php"
}
