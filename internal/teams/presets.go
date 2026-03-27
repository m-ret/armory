package teams

import "strings"

// Preset defines a hardcoded team preset with skills.sh collection mappings.
type Preset struct {
	Name        string
	Description string
	Skills      []string
	Collection  string // skills.sh collection slug (empty if no collection)
	RepoURL     string // GitHub clone URL fallback (empty if no repo)
}

// AllPresets returns all 10 team presets.
func AllPresets() []Preset {
	return []Preset{
		{
			Name:        "Dev",
			Description: "Software development and code quality",
			Skills: []string{
				"best-practices", "performance", "test-driven-development",
				"systematic-debugging", "verification-before-completion",
				"ship", "review", "investigate",
			},
		},
		{
			Name:        "Marketing",
			Description: "Content creation, SEO, and brand work",
			Skills: []string{
				"copywriting", "marketing-psychology", "seo", "content-strategy",
				"email-sequence", "social-content", "popup-cro", "ai-seo",
				"cold-email", "ad-creative", "lead-magnets", "copy-editing",
				"pricing-strategy", "marketing-ideas",
			},
			Collection: "coreyhaines31/marketingskills",
			RepoURL:    "https://github.com/coreyhaines31/marketingskills",
		},
		{
			Name:        "QA",
			Description: "Quality assurance and testing",
			Skills: []string{
				"qa", "browse", "benchmark", "web-quality-audit",
				"code-maturity-assessor",
			},
		},
		{
			Name:        "Design",
			Description: "UI/UX design and visual quality",
			Skills: []string{
				"frontend-design", "accessibility", "core-web-vitals",
				"awwwards-animations", "gsap", "threejs", "scroll-storyteller",
			},
		},
		{
			Name:        "Security",
			Description: "Security auditing and vulnerability scanning",
			Skills: []string{
				"cso", "codeql", "semgrep", "seatbelt-sandboxer",
				"yara-rule-authoring", "zeroize-audit", "burpsuite-project-parser",
			},
		},
		{
			Name:        "Planning",
			Description: "Architecture and project planning",
			Skills: []string{
				"office-hours", "plan-eng-review", "plan-ceo-review",
				"plan-design-review", "brainstorming",
			},
		},
		{
			Name:        "DevOps",
			Description: "Infrastructure, CI/CD, and deployment",
			Skills: []string{
				"land-and-deploy", "setup-deploy", "canary", "guard", "careful",
			},
		},
		{
			Name:        "Frontend",
			Description: "Frontend development and React",
			Skills: []string{
				"frontend-design", "gsap", "vercel-react-best-practices",
				"vercel-composition-patterns", "remotion-best-practices",
				"core-web-vitals", "accessibility",
			},
		},
		{
			Name:        "Backend",
			Description: "Backend development and APIs",
			Skills: []string{
				"best-practices", "performance", "systematic-debugging",
				"test-driven-development", "verification-before-completion",
			},
		},
		{
			Name:        "Data",
			Description: "Data analysis and analytics",
			Skills: []string{
				"analytics-tracking", "ab-test-setup", "programmatic-seo",
			},
		},
	}
}

// PresetNames returns just the names of all presets.
func PresetNames() []string {
	presets := AllPresets()
	names := make([]string, len(presets))
	for i, p := range presets {
		names[i] = p.Name
	}
	return names
}

// GetPreset returns a preset by name (case-insensitive), or nil if not found.
func GetPreset(name string) *Preset {
	lower := strings.ToLower(name)
	for _, p := range AllPresets() {
		if strings.ToLower(p.Name) == lower {
			return &p
		}
	}
	return nil
}
