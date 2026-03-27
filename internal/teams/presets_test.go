package teams

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllPresets_NoDuplicateNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, p := range AllPresets() {
		assert.False(t, seen[p.Name], "duplicate preset name: %s", p.Name)
		seen[p.Name] = true
	}
}

func TestAllPresets_NonEmpty(t *testing.T) {
	for _, p := range AllPresets() {
		assert.NotEmpty(t, p.Skills, "preset %s has no skills", p.Name)
		assert.NotEmpty(t, p.Description, "preset %s has no description", p.Name)
	}
}

func TestAllPresets_Count(t *testing.T) {
	assert.Len(t, AllPresets(), 10)
}

func TestGetPreset_Existing(t *testing.T) {
	p := GetPreset("Marketing")
	assert.NotNil(t, p)
	assert.Equal(t, "Marketing", p.Name)
	assert.Equal(t, "Content creation, SEO, and brand work", p.Description)
	assert.Equal(t, "coreyhaines31/marketingskills", p.Collection)
	assert.Equal(t, "https://github.com/coreyhaines31/marketingskills", p.RepoURL)
}

func TestGetPreset_CaseInsensitive(t *testing.T) {
	p := GetPreset("marketing")
	assert.NotNil(t, p)
	assert.Equal(t, "Marketing", p.Name)
}

func TestGetPreset_Missing(t *testing.T) {
	p := GetPreset("nonexistent")
	assert.Nil(t, p)
}

func TestPresetNames_MatchesAllPresets(t *testing.T) {
	names := PresetNames()
	presets := AllPresets()
	assert.Len(t, names, len(presets))

	presetNames := make(map[string]bool)
	for _, p := range presets {
		presetNames[p.Name] = true
	}
	for _, n := range names {
		assert.True(t, presetNames[n], "name %s not found in presets", n)
	}
}
