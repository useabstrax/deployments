package presets

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
)

var laravelStorageDirs = []string{
	"app",
	"app/public",
	"framework",
	"framework/cache",
	"framework/cache/data",
	"framework/sessions",
	"framework/views",
	"logs",
}

// EnsureSharedScaffold creates preset-specific shared paths under project/shared.
// Safe to call repeatedly: existing non-empty .env files are left untouched.
func EnsureSharedScaffold(projectPath string, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	sharedRoot := filepath.Join(projectPath, "shared")
	if err := os.MkdirAll(sharedRoot, 0o755); err != nil {
		return err
	}

	preset := Name(strings.ToLower(strings.TrimSpace(cfg.Preset)))
	if preset == Laravel || containsShared(cfg.Shared, "storage") {
		if err := ensureLaravelStorage(sharedRoot); err != nil {
			return err
		}
	}
	if preset == Laravel || containsShared(cfg.Shared, ".env") {
		if err := ensureLaravelEnv(filepath.Join(sharedRoot, ".env")); err != nil {
			return err
		}
	}
	return nil
}

func containsShared(shared []string, name string) bool {
	for _, s := range shared {
		if strings.TrimSpace(s) == name {
			return true
		}
	}
	return false
}

func ensureLaravelStorage(sharedRoot string) error {
	storageRoot := filepath.Join(sharedRoot, "storage")
	for _, rel := range laravelStorageDirs {
		dir := filepath.Join(storageRoot, rel)
		if err := os.MkdirAll(dir, 0o775); err != nil {
			return fmt.Errorf("creating shared storage %s: %w", rel, err)
		}
	}
	// gitkeep-style placeholders so empty dirs survive some deploy tools
	for _, rel := range []string{"app/public", "framework/cache/data", "framework/sessions", "framework/views", "logs"} {
		keep := filepath.Join(storageRoot, rel, ".gitignore")
		if _, err := os.Stat(keep); os.IsNotExist(err) {
			_ = os.WriteFile(keep, []byte("*\n!.gitignore\n"), 0o644)
		}
	}
	return nil
}

func ensureLaravelEnv(path string) error {
	if info, err := os.Stat(path); err == nil {
		if info.Size() > 0 {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	key, err := generateAppKey()
	if err != nil {
		return err
	}

	content := minimalLaravelEnv(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o640)
}

func generateAppKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "base64:" + base64.StdEncoding.EncodeToString(buf), nil
}

func minimalLaravelEnv(appKey string) string {
	return strings.TrimSpace(`
APP_NAME=Laravel
APP_ENV=production
APP_KEY=`+appKey+`
APP_DEBUG=false
APP_URL=http://localhost

APP_LOCALE=en
APP_FALLBACK_LOCALE=en

LOG_CHANNEL=stack
LOG_LEVEL=error

# Replace with your database settings before migrate / go-live.
DB_CONNECTION=sqlite
# DB_HOST=127.0.0.1
# DB_PORT=3306
# DB_DATABASE=laravel
# DB_USERNAME=root
# DB_PASSWORD=

SESSION_DRIVER=file
CACHE_STORE=file
QUEUE_CONNECTION=sync
FILESYSTEM_DISK=local

BROADCAST_CONNECTION=log
`) + "\n"
}
