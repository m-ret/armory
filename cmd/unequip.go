package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/m-ret/armory/internal/state"
	"github.com/spf13/cobra"
)

var unequipDir string

var unequipCmd = &cobra.Command{
	Use:   "unequip",
	Short: "Remove armory-managed symlinks from a directory",
	Long:  "Unequip a directory by removing all symlinks that armory previously created.",
	RunE:  runUnequip,
}

func init() {
	unequipCmd.Flags().StringVar(&unequipDir, "dir", "", "target directory (default: current working directory)")
	rootCmd.AddCommand(unequipCmd)
}

func runUnequip(cmd *cobra.Command, args []string) error {
	// Resolve target directory.
	targetDir := unequipDir
	var err error
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

	// Load state.
	st, err := state.LoadState()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	entry := st.FindByDir(targetDir)
	if entry == nil {
		fmt.Printf("Nothing to unequip in %s\n", targetDir)
		return nil
	}

	roleName := entry.Role
	skillsDir := filepath.Join(targetDir, ".claude", "skills")

	// Remove each managed symlink.
	removed := 0
	for _, name := range entry.ManagedSymlinks {
		link := filepath.Join(skillsDir, name)
		if err := os.Remove(link); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Printf("  warning: failed to remove %s: %v\n", name, err)
			continue
		}
		removed++
	}

	// Remove entry from state and save.
	st.RemoveEquipped(targetDir)
	if err := st.Save(); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	summaryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	fmt.Println(summaryStyle.Render(fmt.Sprintf("Unequipped '%s' role: %d symlinks removed", roleName, removed)))

	return nil
}
