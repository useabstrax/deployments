package presets

import (
	"fmt"
	"strings"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/config"
)

// Name identifies a built-in preset.
type Name string

const (
	Laravel Name = "laravel"
	Node    Name = "node"
	Ruby    Name = "ruby"
	Static  Name = "static"
	None    Name = "none"
)

// All returns supported preset names.
func All() []string {
	return []string{string(Laravel), string(Node), string(Ruby), string(Static), string(None)}
}

// Apply mutates cfg with preset defaults for shared paths and hooks.
// Repository, branch, project, and deploy_key are left unchanged.
func Apply(cfg *config.Config, preset string) error {
	name := Name(strings.ToLower(strings.TrimSpace(preset)))
	switch name {
	case Laravel:
		cfg.Preset = string(Laravel)
		cfg.PublicDir = "public"
		cfg.Shared = []string{".env", "storage"}
		cfg.Hooks.AfterClone = []string{`abstrax composer run --project="$ABSTRAX_PROJECT" --path="$ABSTRAX_RELEASE_PATH" install --no-dev --optimize-autoloader`}
		cfg.Hooks.BeforeActivate = []string{"$ABSTRAX_CLI_PHP artisan migrate --force"}
		cfg.Hooks.AfterActivate = []string{}
	case Node:
		cfg.Preset = string(Node)
		cfg.PublicDir = "."
		cfg.Shared = []string{}
		cfg.Hooks.AfterClone = []string{"npm ci && npm run build"}
		cfg.Hooks.BeforeActivate = []string{}
		cfg.Hooks.AfterActivate = []string{}
	case Ruby:
		cfg.Preset = string(Ruby)
		cfg.PublicDir = "."
		cfg.Shared = []string{}
		cfg.Hooks.AfterClone = []string{"bundle install --deployment --without development test"}
		cfg.Hooks.BeforeActivate = []string{}
		cfg.Hooks.AfterActivate = []string{}
	case Static:
		cfg.Preset = string(Static)
		cfg.PublicDir = "."
		cfg.Shared = []string{}
		cfg.Hooks.AfterClone = []string{}
		cfg.Hooks.BeforeActivate = []string{}
		cfg.Hooks.AfterActivate = []string{}
	case None, "":
		cfg.Preset = string(None)
		cfg.PublicDir = "."
		cfg.Shared = []string{}
		cfg.Hooks.AfterClone = []string{}
		cfg.Hooks.BeforeActivate = []string{}
		cfg.Hooks.AfterActivate = []string{}
	default:
		return fmt.Errorf("unknown preset %q (supported: %s)", preset, strings.Join(All(), ", "))
	}
	return nil
}

// Valid reports whether preset is known.
func Valid(preset string) bool {
	switch Name(strings.ToLower(strings.TrimSpace(preset))) {
	case Laravel, Node, Ruby, Static, None, "":
		return true
	default:
		return false
	}
}
