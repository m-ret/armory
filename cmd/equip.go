package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/m-ret/armory/internal/config"
	"github.com/m-ret/armory/internal/scanner"
	"github.com/m-ret/armory/internal/state"
	"github.com/spf13/cobra"
)

var equipDir string
var equipMerge bool

var equipCmd = &cobra.Command{
	Use:   "equip <role>",
	Short: "Symlink role skills into a target directory",
	Long: `Equip a directory with the skills defined in a role.

By default, existing armory-managed symlinks are replaced. Use --merge to
keep existing symlinks and only add missing ones.`,
	Args: cobra.ExactArgs(1),
	RunE: runEquip,
}

func init() {
	equipCmd.Flags().StringVar(&equipDir, "dir", "", "target directory (default: current working directory)")
	equipCmd.Flags().BoolVar(&equipMerge, "merge", false, "keep existing symlinks, only add missing ones")
	rootCmd.AddCommand(equipCmd)
}

func runEquip(cmd *cobra.Command, args []string) error {
	roleName := args[0]

	// Load config and resolve role.
	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	role, err := cfg.GetRole(roleName)
	if err != nil {
		return err
	}

	// Resolve target directory.
	targetDir := equipDir
	if targetDir == "" {
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
	}
	targetDir, err = filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolving target directory: %w", err)
	}

	// Overlap guard: ensure targetDir/.claude/skills/ does not resolve to a skill source path.
	skillsTarget := filepath.Join(targetDir, ".claude", "skills")
	resolvedTarget, _ := filepath.EvalSymlinks(skillsTarget)
	if resolvedTarget == "" {
		resolvedTarget = skillsTarget
	}
	resolvedTarget, _ = filepath.Abs(resolvedTarget)

	for _, sp := range cfg.SkillPaths {
		expanded := config.ExpandPath(sp)
		resolvedSource, _ := filepath.EvalSymlinks(expanded)
		if resolvedSource == "" {
			resolvedSource = expanded
		}
		resolvedSource, _ = filepath.Abs(resolvedSource)

		if resolvedTarget == resolvedSource || strings.HasPrefix(resolvedTarget, resolvedSource+string(filepath.Separator)) || strings.HasPrefix(resolvedSource, resolvedTarget+string(filepath.Separator)) {
			return fmt.Errorf("target directory overlaps with skill source path")
		}
	}

	// Scan skills.
	expanded := make([]string, len(cfg.SkillPaths))
	for i, p := range cfg.SkillPaths {
		expanded[i] = config.ExpandPath(p)
	}

	cachePath := config.ExpandPath("~/.armory/cache/skills.json")
	var skills []scanner.Skill
	if scanner.IsCacheValid(cachePath, expanded) {
		skills, err = scanner.LoadCache(cachePath)
		if err != nil {
			skills = nil
		}
	}
	if skills == nil {
		skills, err = scanner.ScanSkillPaths(expanded)
		if err != nil {
			return fmt.Errorf("scanning skills: %w", err)
		}
	}

	// Build lookup map.
	skillMap := make(map[string]scanner.Skill)
	for _, s := range skills {
		skillMap[s.Name] = s
	}

	// Match role skills against scanned skills.
	var found []scanner.Skill
	var missing []string
	for _, name := range role.Skills {
		if s, ok := skillMap[name]; ok {
			found = append(found, s)
		} else {
			missing = append(missing, name)
		}
	}

	// Handle missing skills.
	if len(missing) > 0 {
		action := role.MissingAction
		if action == "" {
			action = "skip"
		}

		switch action {
		case "error":
			return fmt.Errorf("missing skills: %s", strings.Join(missing, ", "))
		case "prompt":
			if isTerminal() {
				fmt.Printf("Missing skills: %s\n", strings.Join(missing, ", "))
				fmt.Print("[s]kip / [a]bort? ")
				reader := bufio.NewScanner(os.Stdin)
				reader.Scan()
				answer := strings.TrimSpace(strings.ToLower(reader.Text()))
				if answer == "a" || answer == "abort" {
					return fmt.Errorf("aborted by user")
				}
			}
			// Non-terminal or user chose skip: fall through.
		case "skip":
			// Fall through with warning below.
		}
	}

	// Create target skills directory.
	if err := os.MkdirAll(skillsTarget, 0o755); err != nil {
		return fmt.Errorf("creating skills directory: %w", err)
	}

	// Load state.
	st, err := state.LoadState()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	existing := st.FindByDir(targetDir)
	var oldRole string

	// If NOT --merge: remove previously managed symlinks.
	if !equipMerge && existing != nil {
		oldRole = existing.Role
		for _, name := range existing.ManagedSymlinks {
			link := filepath.Join(skillsTarget, name)
			_ = os.Remove(link)
		}
	}

	// Styling.
	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#2ecc71"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f39c12"))
	summaryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))

	// Create symlinks.
	var managedSymlinks []string
	var linkedSkills []string
	var skippedNames []string

	// Collect previously managed names for this dir (if merging).
	previousManaged := make(map[string]bool)
	if equipMerge && existing != nil {
		for _, name := range existing.ManagedSymlinks {
			previousManaged[name] = true
		}
		// Carry forward previous managed symlinks that still exist.
		for _, name := range existing.ManagedSymlinks {
			link := filepath.Join(skillsTarget, name)
			if _, err := os.Lstat(link); err == nil {
				managedSymlinks = append(managedSymlinks, name)
			}
		}
	}

	for _, skill := range found {
		link := filepath.Join(skillsTarget, skill.Name)

		// Check if symlink already exists.
		if _, err := os.Lstat(link); err == nil {
			// Already exists. Check if armory-managed.
			isManaged := false
			if existing != nil {
				for _, name := range existing.ManagedSymlinks {
					if name == skill.Name {
						isManaged = true
						break
					}
				}
			}

			if !isManaged {
				fmt.Println(warnStyle.Render("  ! skipping "+skill.Name+": already exists (not managed by armory)"))
				skippedNames = append(skippedNames, skill.Name)
				continue
			}

			// Managed by armory: remove and re-create.
			_ = os.Remove(link)
		}

		if err := os.Symlink(skill.Dir, link); err != nil {
			fmt.Println(warnStyle.Render(fmt.Sprintf("  ! failed to link %s: %v", skill.Name, err)))
			continue
		}

		fmt.Println(checkStyle.Render("  + " + skill.Name))
		managedSymlinks = append(managedSymlinks, skill.Name)
		linkedSkills = append(linkedSkills, skill.Name)
	}

	// Print warnings for missing skills.
	for _, name := range missing {
		fmt.Println(warnStyle.Render("  ? " + name + " (not found)"))
	}

	// Deduplicate managed symlinks.
	seen := make(map[string]bool)
	var deduped []string
	for _, name := range managedSymlinks {
		if !seen[name] {
			seen[name] = true
			deduped = append(deduped, name)
		}
	}
	managedSymlinks = deduped

	// Update state.
	entry := state.EquippedEntry{
		Dir:             targetDir,
		Role:            roleName,
		Skills:          role.Skills,
		ManagedSymlinks: managedSymlinks,
		EquippedAt:      time.Now(),
	}
	st.AddEquipped(entry)

	if err := st.Save(); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	// Role replacement warning.
	if oldRole != "" && oldRole != roleName {
		fmt.Println(warnStyle.Render(fmt.Sprintf("Replacing '%s' role with '%s'", oldRole, roleName)))
	}

	// Final summary.
	fmt.Println(summaryStyle.Render(fmt.Sprintf("\nEquipped '%s' role: %d skills loaded", roleName, len(linkedSkills))))

	return nil
}

// isTerminal checks whether stdin appears to be a terminal.
func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
