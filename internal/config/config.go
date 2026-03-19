// Package config loads and validates preflight configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// validProviders is the set of allowed provider values.
var validProviders = map[string]bool{
	"auto":   true,
	"claude": true,
	"codex":  true,
}

// validBlockOn is the set of allowed block_on values.
var validBlockOn = map[string]bool{
	"critical": true,
	"warning":  true,
}

// Config holds the resolved preflight settings for a run.
type Config struct {
	Provider     string        `yaml:"provider"`
	BlockOn      string        `yaml:"block_on"`
	Timeout      time.Duration `yaml:"timeout"`
	PromptExtra  string        `yaml:"prompt_extra"`
	MaxDiffBytes int           `yaml:"max_diff_bytes"`
}

// defaults returns a Config populated with default values.
func defaults() *Config {
	return &Config{
		Provider:     "auto",
		BlockOn:      "critical",
		Timeout:      60 * time.Second,
		PromptExtra:  "",
		MaxDiffBytes: 524288,
	}
}

// Load reads the project-level config (projectPath) and global config
// (globalPath), merging them so that project-level values override globals.
// Missing files are silently skipped (defaults are used).
func Load(projectPath, globalPath string) (*Config, error) {
	cfg := defaults()

	// Load global first, then project overrides.
	for _, path := range []string{globalPath, projectPath} {
		if path == "" {
			continue
		}
		if err := mergeFile(cfg, path); err != nil {
			return nil, err
		}
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// mergeFile reads the YAML file at path (if it exists) into cfg.
func mergeFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}
	return nil
}

// validate checks that all Config fields have legal values.
func validate(cfg *Config) error {
	if !validProviders[cfg.Provider] {
		return fmt.Errorf("config: invalid provider %q; must be one of auto, claude, codex", cfg.Provider)
	}
	if !validBlockOn[cfg.BlockOn] {
		return fmt.Errorf("config: invalid block_on %q; must be critical or warning", cfg.BlockOn)
	}
	if cfg.Timeout <= 0 {
		return errors.New("config: timeout must be greater than zero")
	}
	if cfg.MaxDiffBytes <= 0 {
		return errors.New("config: max_diff_bytes must be greater than zero")
	}
	return nil
}
