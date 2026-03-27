package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/m-ret/armory/internal/config"
	"github.com/m-ret/armory/internal/scanner"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "List available skills from all skill paths",
	RunE:  runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	expanded := make([]string, len(cfg.SkillPaths))
	for i, p := range cfg.SkillPaths {
		expanded[i] = config.ExpandPath(p)
	}

	cachePath := config.ExpandPath("~/.armory/cache/skills.json")

	var skills []scanner.Skill
	fromCache := false

	if scanner.IsCacheValid(cachePath, expanded) {
		skills, err = scanner.LoadCache(cachePath)
		if err != nil {
			return fmt.Errorf("loading cache: %w", err)
		}
		fromCache = true
	} else {
		skills, err = scanner.ScanSkillPaths(expanded)
		if err != nil {
			return fmt.Errorf("scanning skills: %w", err)
		}

		// Count duplicates: total directories minus unique skills returned.
		totalDirs := 0
		for _, p := range expanded {
			entries, dirErr := os.ReadDir(p)
			if dirErr != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					skillFile := filepath.Join(p, e.Name(), "SKILL.md")
					if _, statErr := os.Stat(skillFile); statErr == nil {
						totalDirs++
					}
				}
			}
		}
		duplicates := totalDirs - len(skills)
		if duplicates < 0 {
			duplicates = 0
		}

		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
			return fmt.Errorf("creating cache directory: %w", err)
		}
		if err := scanner.SaveCache(cachePath, skills); err != nil {
			return fmt.Errorf("saving cache: %w", err)
		}

		pathsDisplay := abbreviatePaths(expanded)
		fmt.Printf("Scanning %s... %d skills found (%d duplicates merged)\n",
			pathsDisplay, len(skills), duplicates)
	}

	if fromCache {
		fmt.Printf("Loaded %d skills from cache\n", len(skills))
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	printTable(skills)

	fmt.Printf("\n%d skills indexed | Cache: ~/.armory/cache/skills.json\n", len(skills))

	return nil
}

func printTable(skills []scanner.Skill) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	categoryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db"))
	sourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	nameW := len("SKILL")
	catW := len("CATEGORY")
	srcW := len("SOURCE")

	type row struct {
		name, category, source string
	}

	rows := make([]row, len(skills))
	for i, s := range skills {
		src := abbreviatePath(s.Source)
		rows[i] = row{name: s.Name, category: s.Category, source: src}
		if len(s.Name) > nameW {
			nameW = len(s.Name)
		}
		if len(s.Category) > catW {
			catW = len(s.Category)
		}
		if len(src) > srcW {
			srcW = len(src)
		}
	}

	pad := 3
	fmtStr := fmt.Sprintf("%%-%ds%%-%ds%%s", nameW+pad, catW+pad)

	header := fmt.Sprintf(fmtStr, "SKILL", "CATEGORY", "SOURCE")
	fmt.Println(headerStyle.Render(header))
	fmt.Println(headerStyle.Render(strings.Repeat("─", nameW+catW+srcW+2*pad)))

	for _, r := range rows {
		line := fmt.Sprintf(fmtStr,
			nameStyle.Render(r.name),
			categoryStyle.Render(r.category),
			sourceStyle.Render(r.source),
		)
		fmt.Println(line)
	}
}

func abbreviatePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

func abbreviatePaths(paths []string) string {
	abbr := make([]string, len(paths))
	for i, p := range paths {
		abbr[i] = abbreviatePath(p)
	}
	return strings.Join(abbr, ", ")
}
