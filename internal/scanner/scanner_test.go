package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// helper creates a skill directory with a SKILL.md containing the given content.
func createSkillDir(t *testing.T, base, name, content string) {
	t.Helper()
	dir := filepath.Join(base, name)
	err := os.MkdirAll(dir, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644)
	assert.NoError(t, err)
}

func TestScanSkillPaths(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) []string
		wantLen  int
		validate func(t *testing.T, skills []Skill)
	}{
		{
			name: "happy path with 2 skills in 1 path",
			setup: func(t *testing.T) []string {
				base := t.TempDir()
				createSkillDir(t, base, "skill-a", "---\nname: skill-a\ndescription: First skill\n---\n# Skill A")
				createSkillDir(t, base, "skill-b", "---\nname: skill-b\ndescription: Second skill\n---\n# Skill B")
				return []string{base}
			},
			wantLen: 2,
			validate: func(t *testing.T, skills []Skill) {
				names := map[string]bool{}
				for _, s := range skills {
					names[s.Name] = true
				}
				assert.True(t, names["skill-a"])
				assert.True(t, names["skill-b"])
			},
		},
		{
			name: "skill dir missing SKILL.md is skipped",
			setup: func(t *testing.T) []string {
				base := t.TempDir()
				createSkillDir(t, base, "has-skill", "---\nname: has-skill\ndescription: present\n---\n")
				// Create a directory without SKILL.md
				err := os.MkdirAll(filepath.Join(base, "no-skill"), 0755)
				assert.NoError(t, err)
				return []string{base}
			},
			wantLen: 1,
			validate: func(t *testing.T, skills []Skill) {
				assert.Equal(t, "has-skill", skills[0].Name)
			},
		},
		{
			name: "nonexistent path is skipped without error",
			setup: func(t *testing.T) []string {
				return []string{"/nonexistent/path/that/does/not/exist"}
			},
			wantLen: 0,
			validate: func(t *testing.T, skills []Skill) {
				assert.Empty(t, skills)
			},
		},
		{
			name: "duplicate across paths uses first-match-wins",
			setup: func(t *testing.T) []string {
				path1 := t.TempDir()
				path2 := t.TempDir()
				createSkillDir(t, path1, "dup-skill", "---\nname: dup-skill\ndescription: from path1\n---\n")
				createSkillDir(t, path2, "dup-skill", "---\nname: dup-skill\ndescription: from path2\n---\n")
				return []string{path1, path2}
			},
			wantLen: 1,
			validate: func(t *testing.T, skills []Skill) {
				assert.Equal(t, "dup-skill", skills[0].Name)
				assert.Equal(t, "from path1", skills[0].Description)
			},
		},
		{
			name: "empty paths returns empty result",
			setup: func(t *testing.T) []string {
				return []string{}
			},
			wantLen: 0,
			validate: func(t *testing.T, skills []Skill) {
				assert.Empty(t, skills)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := tt.setup(t)
			skills, err := ScanSkillPaths(paths)
			assert.NoError(t, err)
			assert.Len(t, skills, tt.wantLen)
			if tt.validate != nil {
				tt.validate(t, skills)
			}
		})
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    SkillMeta
		wantErr bool
	}{
		{
			name:    "valid frontmatter with name and description",
			content: "---\nname: my-skill\ndescription: A useful skill\n---\n# Body",
			want:    SkillMeta{Name: "my-skill", Description: "A useful skill"},
			wantErr: false,
		},
		{
			name:    "frontmatter with explicit category",
			content: "---\nname: sec-tool\ndescription: Security tool\ncategory: security\n---\n",
			want:    SkillMeta{Name: "sec-tool", Description: "Security tool", Category: "security"},
			wantErr: false,
		},
		{
			name:    "no frontmatter markers returns error",
			content: "Just some markdown\nwithout frontmatter",
			want:    SkillMeta{},
			wantErr: true,
		},
		{
			name:    "missing name field returns error",
			content: "---\ndescription: No name here\n---\n",
			want:    SkillMeta{},
			wantErr: true,
		},
		{
			name:    "multi-line description",
			content: "---\nname: multi\ndescription: |\n  This is a long\n  multi-line description\n---\n",
			want:    SkillMeta{Name: "multi", Description: "This is a long\nmulti-line description\n"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFrontmatter([]byte(tt.content))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInferCategory(t *testing.T) {
	tests := []struct {
		name     string
		skill    Skill
		expected string
	}{
		{
			name:     "explicit category in frontmatter wins",
			skill:    Skill{Name: "some-tool", Category: "custom-cat"},
			expected: "custom-cat",
		},
		{
			name:     "keyword match security",
			skill:    Skill{Name: "codeql-scanner"},
			expected: "security",
		},
		{
			name:     "keyword match marketing",
			skill:    Skill{Name: "seo-optimizer"},
			expected: "marketing",
		},
		{
			name:     "keyword match dev",
			skill:    Skill{Name: "best-practices-checker"},
			expected: "dev",
		},
		{
			name:     "keyword match design",
			skill:    Skill{Name: "frontend-toolkit"},
			expected: "design",
		},
		{
			name:     "keyword match planning",
			skill:    Skill{Name: "brainstorming-helper"},
			expected: "planning",
		},
		{
			name:     "keyword match qa",
			skill:    Skill{Name: "benchmark-runner"},
			expected: "qa",
		},
		{
			name:     "no match returns other",
			skill:    Skill{Name: "random-thing"},
			expected: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InferCategory(tt.skill)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCacheSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")

	original := []Skill{
		{Name: "alpha", Description: "First", Category: "dev", Source: "/src1", Dir: "/src1/alpha"},
		{Name: "beta", Description: "Second", Category: "security", Source: "/src2", Dir: "/src2/beta"},
	}

	err := SaveCache(cachePath, original)
	assert.NoError(t, err)

	loaded, err := LoadCache(cachePath)
	assert.NoError(t, err)
	assert.Equal(t, original, loaded)
}

func TestIsCacheValid(t *testing.T) {
	t.Run("fresh cache is valid", func(t *testing.T) {
		dir := t.TempDir()
		skillPath := filepath.Join(dir, "skills")
		err := os.MkdirAll(skillPath, 0755)
		assert.NoError(t, err)

		// Wait briefly so cache file is strictly newer than the skill dir.
		time.Sleep(50 * time.Millisecond)

		cachePath := filepath.Join(dir, "cache.json")
		err = os.WriteFile(cachePath, []byte("[]"), 0644)
		assert.NoError(t, err)

		assert.True(t, IsCacheValid(cachePath, []string{skillPath}))
	})

	t.Run("stale cache is invalid", func(t *testing.T) {
		dir := t.TempDir()
		cachePath := filepath.Join(dir, "cache.json")

		// Create cache file first.
		err := os.WriteFile(cachePath, []byte("[]"), 0644)
		assert.NoError(t, err)

		// Wait briefly then create/touch the skill path so it is newer.
		time.Sleep(50 * time.Millisecond)

		skillPath := filepath.Join(dir, "skills")
		err = os.MkdirAll(skillPath, 0755)
		assert.NoError(t, err)

		// Explicitly set skill dir mtime to future to guarantee it's newer.
		future := time.Now().Add(1 * time.Second)
		err = os.Chtimes(skillPath, future, future)
		assert.NoError(t, err)

		assert.False(t, IsCacheValid(cachePath, []string{skillPath}))
	})
}
