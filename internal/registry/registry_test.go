package registry

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchByNames_FoundInMarketing(t *testing.T) {
	found, notFound := SearchByNames([]string{"copywriting"})

	assert.Len(t, found, 1)
	assert.Empty(t, notFound)
	assert.Equal(t, "copywriting", found[0].Name)
	assert.Equal(t, "coreyhaines31/marketingskills", found[0].Collection)
	assert.Equal(t, "https://github.com/coreyhaines31/marketingskills", found[0].RepoURL)
}

func TestSearchByNames_FoundInPresetNoCollection(t *testing.T) {
	found, notFound := SearchByNames([]string{"best-practices"})

	assert.Empty(t, found)
	assert.Len(t, notFound, 1)
	assert.Equal(t, "best-practices", notFound[0])
}

func TestSearchByNames_NotFoundAnywhere(t *testing.T) {
	found, notFound := SearchByNames([]string{"nonexistent-skill"})

	assert.Empty(t, found)
	assert.Len(t, notFound, 1)
	assert.Equal(t, "nonexistent-skill", notFound[0])
}

func TestSearchByNames_Mixed(t *testing.T) {
	found, notFound := SearchByNames([]string{"copywriting", "best-practices", "fake"})

	assert.Len(t, found, 1)
	assert.Equal(t, "copywriting", found[0].Name)

	assert.Len(t, notFound, 2)
	assert.Contains(t, notFound, "best-practices")
	assert.Contains(t, notFound, "fake")
}

func TestHasNpx(t *testing.T) {
	// Just verify it returns a bool without panicking.
	result := HasNpx()
	assert.IsType(t, true, result)
}

func TestDefaultInstallDir(t *testing.T) {
	dir := DefaultInstallDir()
	assert.True(t, strings.HasSuffix(dir, ".agents/skills"),
		"expected dir to end with .agents/skills, got %s", dir)
}

func TestSearchByNames_Empty(t *testing.T) {
	found, notFound := SearchByNames([]string{})

	assert.Empty(t, found)
	assert.Empty(t, notFound)
}
