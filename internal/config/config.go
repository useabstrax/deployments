package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Filename is the deploy config file name under the project path.
	Filename = "deploy.json"
	Version  = 1
)

// Hooks holds shell commands run at deploy phases.
type Hooks struct {
	AfterClone     []string `json:"after_clone"`
	BeforeActivate []string `json:"before_activate"`
	AfterActivate  []string `json:"after_activate"`
}

// Config is the plugin-owned deploy.json schema.
type Config struct {
	Version      int      `json:"version"`
	Project      string   `json:"project"`
	Repository   string   `json:"repository"`
	Branch       string   `json:"branch"`
	Provider     string   `json:"provider"`
	KeepReleases int      `json:"keep_releases"`
	PublicDir    string   `json:"public_dir"`
	Preset       string   `json:"preset"`
	Shared       []string `json:"shared"`
	Hooks        Hooks    `json:"hooks"`
	DeployKey    string   `json:"deploy_key"`
}

// Path returns the deploy.json path for a project root.
func Path(projectPath string) string {
	return filepath.Join(projectPath, Filename)
}

// Default returns a minimal config for a project.
func Default(projectName string) Config {
	return Config{
		Version:      Version,
		Project:      projectName,
		Repository:   "",
		Branch:       "main",
		Provider:     "github",
		KeepReleases: 5,
		PublicDir:    "public",
		Preset:       "none",
		Shared:       []string{},
		Hooks: Hooks{
			AfterClone:     []string{},
			BeforeActivate: []string{},
			AfterActivate:  []string{},
		},
		DeployKey: "",
	}
}

// Load reads and validates deploy.json from projectPath.
func Load(projectPath string) (*Config, error) {
	path := Path(projectPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid deploy.json: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Exists reports whether deploy.json is present.
func Exists(projectPath string) bool {
	_, err := os.Stat(Path(projectPath))
	return err == nil
}

// Save writes deploy.json atomically.
func Save(projectPath string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Version == 0 {
		cfg.Version = Version
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := Path(projectPath)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Validate checks required fields and sane defaults.
func (c *Config) Validate() error {
	if c.Version != Version {
		return fmt.Errorf("unsupported deploy.json version %d (want %d)", c.Version, Version)
	}
	if strings.TrimSpace(c.Project) == "" {
		return fmt.Errorf("deploy.json: project is required")
	}
	if c.KeepReleases < 1 {
		return fmt.Errorf("deploy.json: keep_releases must be >= 1")
	}
	if c.Provider == "" {
		c.Provider = "github"
	}
	if c.Provider != "github" {
		return fmt.Errorf("deploy.json: unsupported provider %q (v1 supports github)", c.Provider)
	}
	if c.Branch == "" {
		c.Branch = "main"
	}
	if c.PublicDir == "" {
		c.PublicDir = "public"
	}
	if c.Preset == "" {
		c.Preset = "none"
	}
	if c.Shared == nil {
		c.Shared = []string{}
	}
	if c.Hooks.AfterClone == nil {
		c.Hooks.AfterClone = []string{}
	}
	if c.Hooks.BeforeActivate == nil {
		c.Hooks.BeforeActivate = []string{}
	}
	if c.Hooks.AfterActivate == nil {
		c.Hooks.AfterActivate = []string{}
	}
	return nil
}

// Clone returns a deep copy.
func (c *Config) Clone() *Config {
	cp := *c
	cp.Shared = append([]string(nil), c.Shared...)
	cp.Hooks.AfterClone = append([]string(nil), c.Hooks.AfterClone...)
	cp.Hooks.BeforeActivate = append([]string(nil), c.Hooks.BeforeActivate...)
	cp.Hooks.AfterActivate = append([]string(nil), c.Hooks.AfterActivate...)
	return &cp
}
