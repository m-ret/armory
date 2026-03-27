package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoadConfig_NoConfigExists(t *testing.T) {
	tmp := t.TempDir()
	cfg, err := loadFromDirs(tmp, t.TempDir())
	require.NoError(t, err)

	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, DefaultSkillPaths(), cfg.SkillPaths)
	assert.Empty(t, cfg.Roles)
}

func TestLoadConfig_GlobalOnly(t *testing.T) {
	globalDir := t.TempDir()
	writeYAML(t, globalDir, ConfigFileName, `
version: 2
skill_paths:
  - /global/skills
roles:
  devops:
    description: DevOps engineer
    skills: [docker, k8s]
    missing_action: warn
`)

	cfg, err := loadFromDirs(t.TempDir(), globalDir)
	require.NoError(t, err)

	assert.Equal(t, 2, cfg.Version)
	assert.Equal(t, []string{"/global/skills"}, cfg.SkillPaths)
	assert.Contains(t, cfg.Roles, "devops")
	assert.Equal(t, "DevOps engineer", cfg.Roles["devops"].Description)
}

func TestLoadConfig_ProjectOnly(t *testing.T) {
	projectDir := t.TempDir()
	writeYAML(t, projectDir, ConfigFileName, `
version: 3
skill_paths:
  - /project/skills
roles:
  frontend:
    description: Frontend dev
    skills: [react, css]
    missing_action: skip
`)

	cfg, err := loadFromDirs(projectDir, t.TempDir())
	require.NoError(t, err)

	assert.Equal(t, 3, cfg.Version)
	assert.Equal(t, []string{"/project/skills"}, cfg.SkillPaths)
	assert.Contains(t, cfg.Roles, "frontend")
}

func TestLoadConfig_BothExist_ProjectRolesReplaceGlobal(t *testing.T) {
	globalDir := t.TempDir()
	writeYAML(t, globalDir, ConfigFileName, `
version: 1
skill_paths:
  - /global/skills
roles:
  devops:
    description: DevOps engineer
    skills: [docker]
    missing_action: warn
`)

	projectDir := t.TempDir()
	writeYAML(t, projectDir, ConfigFileName, `
version: 1
skill_paths:
  - /project/skills
roles:
  frontend:
    description: Frontend dev
    skills: [react]
    missing_action: skip
`)

	cfg, err := loadFromDirs(projectDir, globalDir)
	require.NoError(t, err)

	assert.Contains(t, cfg.Roles, "frontend", "project role should be present")
	assert.NotContains(t, cfg.Roles, "devops", "global role should be replaced")
}

func TestLoadConfig_ProjectNoRoles_FallsBackToGlobal(t *testing.T) {
	globalDir := t.TempDir()
	writeYAML(t, globalDir, ConfigFileName, `
version: 1
skill_paths:
  - /global/skills
roles:
  devops:
    description: DevOps engineer
    skills: [docker]
    missing_action: warn
`)

	projectDir := t.TempDir()
	writeYAML(t, projectDir, ConfigFileName, `
version: 2
skill_paths:
  - /project/skills
`)

	cfg, err := loadFromDirs(projectDir, globalDir)
	require.NoError(t, err)

	assert.Equal(t, 2, cfg.Version, "version from project config")
	assert.Equal(t, []string{"/project/skills"}, cfg.SkillPaths)
	assert.Contains(t, cfg.Roles, "devops", "should fall back to global roles")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	projectDir := t.TempDir()
	writeYAML(t, projectDir, ConfigFileName, `{{{invalid yaml!!!`)

	_, err := loadFromDirs(projectDir, t.TempDir())
	assert.Error(t, err)
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "sub", ConfigFileName)

	original := &Config{
		Version:    1,
		SkillPaths: []string{"/a", "/b"},
		Roles: map[string]Role{
			"backend": {
				Description:   "Backend dev",
				Skills:        []string{"go", "sql"},
				MissingAction: "error",
			},
		},
	}

	require.NoError(t, SaveConfig(path, original))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "backend")

	loaded, err := loadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original.Version, loaded.Version)
	assert.Equal(t, original.SkillPaths, loaded.SkillPaths)
	assert.Equal(t, original.Roles["backend"].Description, loaded.Roles["backend"].Description)
	assert.Equal(t, original.Roles["backend"].Skills, loaded.Roles["backend"].Skills)
}

func TestGetRole_Existing(t *testing.T) {
	cfg := &Config{
		Roles: map[string]Role{
			"admin": {Description: "Admin role", Skills: []string{"all"}},
		},
	}

	role, err := cfg.GetRole("admin")
	require.NoError(t, err)
	assert.Equal(t, "Admin role", role.Description)
	assert.Equal(t, []string{"all"}, role.Skills)
}

func TestGetRole_Missing(t *testing.T) {
	cfg := &Config{Roles: map[string]Role{}}

	_, err := cfg.GetRole("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestExpandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	expanded := ExpandPath("~/some/dir")
	assert.Equal(t, filepath.Join(home, "some/dir"), expanded)
}

func TestExpandPath_NoTilde(t *testing.T) {
	assert.Equal(t, "/absolute/path", ExpandPath("/absolute/path"))
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, DefaultSkillPaths(), cfg.SkillPaths)
	assert.NotNil(t, cfg.Roles)
	assert.Empty(t, cfg.Roles)
}

// loadFromDirs is a test helper that loads config using specific directories
// instead of relying on os.UserHomeDir for the global path. It places a
// symlink-like setup: the global config lives under globalDir/armory.yaml
// and project config under projectDir/armory.yaml.
func loadFromDirs(projectDir, globalDir string) (*Config, error) {
	globalPath := filepath.Join(globalDir, ConfigFileName)
	projectPath := filepath.Join(projectDir, ConfigFileName)

	globalCfg, globalErr := loadFile(globalPath)
	projectCfg, projectErr := loadFile(projectPath)

	if projectErr != nil && !os.IsNotExist(projectErr) {
		return nil, projectErr
	}
	if globalErr != nil && !os.IsNotExist(globalErr) {
		return nil, globalErr
	}

	if projectCfg == nil && globalCfg == nil {
		return DefaultConfig(), nil
	}

	if projectCfg == nil {
		expandSkillPaths(globalCfg)
		return globalCfg, nil
	}

	if globalCfg == nil {
		expandSkillPaths(projectCfg)
		return projectCfg, nil
	}

	merged := projectCfg
	if len(merged.Roles) == 0 {
		merged.Roles = globalCfg.Roles
	}
	expandSkillPaths(merged)
	return merged, nil
}
