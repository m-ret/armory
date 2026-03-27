package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigDir = ".armory"
	ConfigFileName   = "armory.yaml"
)

// Role defines a named set of skills and how to handle missing ones.
type Role struct {
	Description   string   `yaml:"description"`
	Skills        []string `yaml:"skills"`
	MissingAction string   `yaml:"missing_action"`
}

// Config holds the full armory configuration.
type Config struct {
	Version    int             `yaml:"version"`
	SkillPaths []string        `yaml:"skill_paths"`
	Roles      map[string]Role `yaml:"roles,omitempty"`
}

// DefaultSkillPaths returns the default directories where skills are found.
func DefaultSkillPaths() []string {
	return []string{"~/.claude/skills", "~/.agents/skills"}
}

// ExpandPath replaces a leading ~ with the user's home directory.
func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// GlobalConfigPath returns the path to ~/.armory/armory.yaml.
func GlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, DefaultConfigDir, ConfigFileName), nil
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Version:    1,
		SkillPaths: DefaultSkillPaths(),
		Roles:      map[string]Role{},
	}
}

// LoadConfig loads configuration with project-over-global precedence.
//
// Precedence:
//  1. projectDir/armory.yaml
//  2. ~/.armory/armory.yaml
//  3. Built-in defaults (version 1, default skill paths, empty roles)
//
// When both project and global configs exist, project-level roles fully
// replace global roles. If the project config has no roles section the
// global roles are used instead.
func LoadConfig(projectDir string) (*Config, error) {
	globalPath, err := GlobalConfigPath()
	if err != nil {
		return nil, err
	}

	projectPath := filepath.Join(projectDir, ConfigFileName)

	globalCfg, globalErr := loadFile(globalPath)
	projectCfg, projectErr := loadFile(projectPath)

	// If both fail to parse (not just missing), surface the project error.
	if projectErr != nil && !os.IsNotExist(projectErr) {
		return nil, fmt.Errorf("loading project config: %w", projectErr)
	}
	if globalErr != nil && !os.IsNotExist(globalErr) {
		return nil, fmt.Errorf("loading global config: %w", globalErr)
	}

	// Neither file exists — return defaults.
	if projectCfg == nil && globalCfg == nil {
		return DefaultConfig(), nil
	}

	// Only global exists.
	if projectCfg == nil {
		expandSkillPaths(globalCfg)
		return globalCfg, nil
	}

	// Only project exists.
	if globalCfg == nil {
		expandSkillPaths(projectCfg)
		return projectCfg, nil
	}

	// Both exist. Project roles fully replace global roles when present.
	merged := projectCfg
	if len(merged.Roles) == 0 {
		merged.Roles = globalCfg.Roles
	}
	expandSkillPaths(merged)
	return merged, nil
}

// SaveConfig writes cfg as YAML to path, creating parent directories.
func SaveConfig(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// GetRole returns the role with the given name, or an error if not found.
func (c *Config) GetRole(name string) (*Role, error) {
	role, ok := c.Roles[name]
	if !ok {
		return nil, fmt.Errorf("role %q not found", name)
	}
	return &role, nil
}

// loadFile reads and parses a single YAML config file. Returns nil config
// and an os.IsNotExist-compatible error when the file does not exist.
func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// expandSkillPaths expands ~ in every skill path entry.
func expandSkillPaths(cfg *Config) {
	for i, p := range cfg.SkillPaths {
		cfg.SkillPaths[i] = ExpandPath(p)
	}
}
