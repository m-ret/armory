package scanner

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a parsed skill from a skill directory.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Source      string `json:"source"`
	Dir         string `json:"dir"`
}

// SkillMeta holds the YAML frontmatter fields from a SKILL.md file.
type SkillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Category    string `yaml:"category"`
}

// categoryKeywords maps category names to keyword slices used for inference.
var categoryKeywords = map[string][]string{
	"security": {
		"cso", "codeql", "semgrep", "vulnerability", "security",
		"seatbelt", "yara", "zeroize", "burpsuite", "firebase-apk",
	},
	"marketing": {
		"copywriting", "marketing", "seo", "copy-editing",
		"social", "content-strategy", "email-sequence", "popup-cro",
	},
	"dev": {
		"best-practices", "performance", "test", "debug", "review",
		"ship", "investigate", "systematic", "verification", "tdd",
	},
	"design": {
		"design", "frontend", "accessibility", "core-web-vitals",
		"awwwards", "gsap", "animation", "threejs", "scroll",
	},
	"planning": {
		"office-hours", "plan", "writing-plans", "brainstorming", "retro",
	},
	"qa": {
		"qa", "browse", "benchmark",
	},
}

// ScanSkillPaths walks each path, finds directories containing SKILL.md,
// parses frontmatter, infers category, and returns the collected skills.
// First-match-wins for duplicate skill names across paths.
func ScanSkillPaths(paths []string) ([]Skill, error) {
	seen := make(map[string]bool)
	var skills []Skill

	for _, base := range paths {
		info, err := os.Stat(base)
		if err != nil || !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(base)
		if err != nil {
			log.Printf("warning: cannot read directory %s: %v", base, err)
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillDir := filepath.Join(base, entry.Name())
			skillFile := filepath.Join(skillDir, "SKILL.md")

			content, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}

			meta, err := ParseFrontmatter(content)
			if err != nil {
				// If frontmatter parsing fails but we have a directory, use dir name as fallback.
				log.Printf("warning: malformed YAML in %s: %v", skillFile, err)
				meta = SkillMeta{Name: entry.Name()}
			}

			// Fallback: if name is empty after parsing, use directory name.
			if meta.Name == "" {
				meta.Name = entry.Name()
			}

			if seen[meta.Name] {
				continue
			}
			seen[meta.Name] = true

			skill := Skill{
				Name:        meta.Name,
				Description: meta.Description,
				Category:    meta.Category,
				Source:      base,
				Dir:         skillDir,
			}
			skill.Category = InferCategory(skill)
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// ParseFrontmatter splits content on "---" markers and parses the YAML block.
// Returns an error if no frontmatter markers are found or if name is missing.
func ParseFrontmatter(content []byte) (SkillMeta, error) {
	delim := []byte("---")
	trimmed := bytes.TrimSpace(content)

	if !bytes.HasPrefix(trimmed, delim) {
		return SkillMeta{}, errors.New("no frontmatter markers found")
	}

	// Find the closing ---
	rest := trimmed[len(delim):]
	idx := bytes.Index(rest, delim)
	if idx < 0 {
		return SkillMeta{}, errors.New("no closing frontmatter marker found")
	}

	yamlBlock := rest[:idx]

	var meta SkillMeta
	if err := yaml.Unmarshal(yamlBlock, &meta); err != nil {
		return SkillMeta{}, err
	}

	if meta.Name == "" {
		return SkillMeta{}, errors.New("frontmatter missing required 'name' field")
	}

	return meta, nil
}

// InferCategory returns the category for a skill. If the skill already has an
// explicit category from frontmatter, that value is returned. Otherwise the
// skill name is matched against known keyword lists using substring matching.
func InferCategory(skill Skill) string {
	if skill.Category != "" {
		return skill.Category
	}

	lower := strings.ToLower(skill.Name)
	for category, keywords := range categoryKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return category
			}
		}
	}

	return "other"
}

// SaveCache writes the skills slice to path as JSON.
func SaveCache(path string, skills []Skill) error {
	data, err := json.MarshalIndent(skills, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadCache reads a JSON skills cache from path.
func LoadCache(path string) ([]Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	if err := json.Unmarshal(data, &skills); err != nil {
		return nil, err
	}

	return skills, nil
}

// IsCacheValid compares the cache file's modification time against the
// modification time of each skill_path directory. If any directory is newer
// than the cache, returns false.
func IsCacheValid(cachePath string, skillPaths []string) bool {
	cacheStat, err := os.Stat(cachePath)
	if err != nil {
		return false
	}
	cacheTime := cacheStat.ModTime()

	for _, p := range skillPaths {
		dirStat, err := os.Stat(p)
		if err != nil {
			continue
		}
		if dirStat.ModTime().After(cacheTime) {
			return false
		}
	}

	return true
}
