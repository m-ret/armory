package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func sampleEntry(dir string) EquippedEntry {
	return EquippedEntry{
		Dir:             dir,
		Team:            "frontend",
		Skills:          []string{"seo", "performance"},
		ManagedSymlinks: []string{"seo", "performance"},
		EquippedAt:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestLoadStateFrom_FileDoesNotExist(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "nonexistent", "state.json")

	s, err := LoadStateFrom(path)

	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Empty(t, s.Equipped)
}

func TestLoadStateFrom_ValidJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.json")

	data := `{
  "equipped": [
    {
      "dir": "/tmp/project",
      "team": "backend",
      "skills": ["seo"],
      "managed_symlinks": ["seo"],
      "equipped_at": "2026-01-01T00:00:00Z"
    }
  ]
}`
	err := os.WriteFile(path, []byte(data), 0o644)
	assert.NoError(t, err)

	s, err := LoadStateFrom(path)

	assert.NoError(t, err)
	assert.Len(t, s.Equipped, 1)
	assert.Equal(t, "/tmp/project", s.Equipped[0].Dir)
	assert.Equal(t, "backend", s.Equipped[0].Team)
	assert.Equal(t, []string{"seo"}, s.Equipped[0].Skills)
}

func TestLoadStateFrom_CorruptedJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.json")

	err := os.WriteFile(path, []byte("{broken json!!!"), 0o644)
	assert.NoError(t, err)

	s, err := LoadStateFrom(path)

	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Empty(t, s.Equipped)
}

func TestSaveToAndLoadStateFrom_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "sub", "state.json")

	original := &State{
		Equipped: []EquippedEntry{sampleEntry("/home/user/project")},
	}

	err := original.SaveTo(path)
	assert.NoError(t, err)

	loaded, err := LoadStateFrom(path)
	assert.NoError(t, err)
	assert.Equal(t, original.Equipped, loaded.Equipped)
}

func TestAddEquipped_NewEntry(t *testing.T) {
	s := &State{}
	entry := sampleEntry("/home/user/project")

	s.AddEquipped(entry)

	assert.Len(t, s.Equipped, 1)
	assert.Equal(t, entry.Dir, s.Equipped[0].Dir)
}

func TestAddEquipped_UpdateExisting(t *testing.T) {
	s := &State{}
	entry1 := sampleEntry("/home/user/project")
	entry1.Team = "frontend"
	s.AddEquipped(entry1)

	entry2 := sampleEntry("/home/user/project")
	entry2.Team = "backend"
	entry2.Skills = []string{"api-design"}
	s.AddEquipped(entry2)

	assert.Len(t, s.Equipped, 1)
	assert.Equal(t, "backend", s.Equipped[0].Team)
	assert.Equal(t, []string{"api-design"}, s.Equipped[0].Skills)
}

func TestRemoveEquipped_Existing(t *testing.T) {
	s := &State{
		Equipped: []EquippedEntry{sampleEntry("/home/user/project")},
	}

	removed := s.RemoveEquipped("/home/user/project")

	assert.True(t, removed)
	assert.Empty(t, s.Equipped)
}

func TestRemoveEquipped_Nonexistent(t *testing.T) {
	s := &State{
		Equipped: []EquippedEntry{sampleEntry("/home/user/project")},
	}

	removed := s.RemoveEquipped("/nonexistent")

	assert.False(t, removed)
	assert.Len(t, s.Equipped, 1)
}

func TestFindByDir_Found(t *testing.T) {
	s := &State{
		Equipped: []EquippedEntry{sampleEntry("/home/user/project")},
	}

	found := s.FindByDir("/home/user/project")

	assert.NotNil(t, found)
	assert.Equal(t, "/home/user/project", found.Dir)
}

func TestFindByDir_NotFound(t *testing.T) {
	s := &State{
		Equipped: []EquippedEntry{sampleEntry("/home/user/project")},
	}

	found := s.FindByDir("/nonexistent")

	assert.Nil(t, found)
}

func TestVerifyEquipped_AllSymlinksExist(t *testing.T) {
	tmp := t.TempDir()
	skillsDir := filepath.Join(tmp, ".claude", "skills")
	err := os.MkdirAll(skillsDir, 0o755)
	assert.NoError(t, err)

	// Create real targets and symlinks
	for _, name := range []string{"seo", "performance"} {
		target := filepath.Join(tmp, "target-"+name)
		err := os.MkdirAll(target, 0o755)
		assert.NoError(t, err)
		err = os.Symlink(target, filepath.Join(skillsDir, name))
		assert.NoError(t, err)
	}

	entry := EquippedEntry{
		Dir:             tmp,
		ManagedSymlinks: []string{"seo", "performance"},
	}

	status := VerifyEquipped(entry)

	assert.Equal(t, StatusEquipped, status)
}

func TestVerifyEquipped_SomeSymlinksMissing(t *testing.T) {
	tmp := t.TempDir()
	skillsDir := filepath.Join(tmp, ".claude", "skills")
	err := os.MkdirAll(skillsDir, 0o755)
	assert.NoError(t, err)

	// Create only one of two expected symlinks
	target := filepath.Join(tmp, "target-seo")
	err = os.MkdirAll(target, 0o755)
	assert.NoError(t, err)
	err = os.Symlink(target, filepath.Join(skillsDir, "seo"))
	assert.NoError(t, err)

	entry := EquippedEntry{
		Dir:             tmp,
		ManagedSymlinks: []string{"seo", "performance"},
	}

	status := VerifyEquipped(entry)

	assert.Equal(t, StatusStale, status)
}

func TestVerifyEquipped_DirectoryDoesNotExist(t *testing.T) {
	entry := EquippedEntry{
		Dir:             "/nonexistent/path/that/does/not/exist",
		ManagedSymlinks: []string{"seo"},
	}

	status := VerifyEquipped(entry)

	assert.Equal(t, StatusBroken, status)
}
