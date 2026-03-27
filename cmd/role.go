package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/m-ret/armory/internal/config"
	"github.com/m-ret/armory/internal/scanner"
	"github.com/m-ret/armory/internal/tui"
	"github.com/spf13/cobra"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage skill roles",
	Long:  "Create, edit, list, and inspect roles that group skills together.",
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured roles",
	RunE:  runRoleList,
}

var roleCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new role with interactive skill picker",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleCreate,
}

var roleEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing role's skills",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleEdit,
}

var roleShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a role",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleShow,
}

func init() {
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleCreateCmd)
	roleCmd.AddCommand(roleEditCmd)
	roleCmd.AddCommand(roleShowCmd)
	rootCmd.AddCommand(roleCmd)
}

// --- role list ------------------------------------------------------------

func runRoleList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Roles) == 0 {
		fmt.Println("No roles configured. Use 'armory role create <name>' to create one.")
		return nil
	}

	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	// Compute column widths.
	nameW := len("ROLE")
	descW := len("DESCRIPTION")
	for name, role := range cfg.Roles {
		if len(name) > nameW {
			nameW = len(name)
		}
		if len(role.Description) > descW {
			descW = len(role.Description)
		}
	}

	pad := 3
	fmtStr := fmt.Sprintf("%%-%ds%%-%ds%%s", nameW+pad, descW+pad)

	header := fmt.Sprintf(fmtStr, "ROLE", "DESCRIPTION", "SKILLS")
	fmt.Println(hdrStyle.Render(header))
	fmt.Println(hdrStyle.Render(strings.Repeat("─", nameW+descW+8+2*pad)))

	// Sort role names for stable output.
	names := make([]string, 0, len(cfg.Roles))
	for n := range cfg.Roles {
		names = append(names, n)
	}
	sortStrings(names)

	for _, name := range names {
		role := cfg.Roles[name]
		line := fmt.Sprintf(fmtStr,
			nameStyle.Render(name),
			dimStyle.Render(role.Description),
			dimStyle.Render(fmt.Sprintf("%d skills", len(role.Skills))),
		)
		fmt.Println(line)
	}

	return nil
}

// --- role create ----------------------------------------------------------

func runRoleCreate(cmd *cobra.Command, args []string) error {
	roleName := args[0]

	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Check for existing role.
	if _, exists := cfg.Roles[roleName]; exists {
		fmt.Printf("Role %q already exists. Overwrite? [y/N] ", roleName)
		reader := bufio.NewScanner(os.Stdin)
		reader.Scan()
		answer := strings.TrimSpace(strings.ToLower(reader.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	skills, err := scanAllSkills(cfg)
	if err != nil {
		return err
	}

	result, err := tui.RunPicker(skills, nil)
	if err != nil {
		return fmt.Errorf("running picker: %w", err)
	}
	if result.Cancelled {
		fmt.Println("Cancelled.")
		return nil
	}

	if len(result.Selected) == 0 {
		fmt.Println("No skills selected. Role not created.")
		return nil
	}

	// Ask for description.
	fmt.Print("Description: ")
	reader := bufio.NewScanner(os.Stdin)
	reader.Scan()
	description := strings.TrimSpace(reader.Text())

	role := config.Role{
		Description:   description,
		Skills:        result.Selected,
		MissingAction: "prompt",
	}

	if cfg.Roles == nil {
		cfg.Roles = make(map[string]config.Role)
	}
	cfg.Roles[roleName] = role

	if err := saveToGlobalConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Role %q created with %d skills.\n", roleName, len(result.Selected))
	return nil
}

// --- role edit ------------------------------------------------------------

func runRoleEdit(cmd *cobra.Command, args []string) error {
	roleName := args[0]

	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	existing, err := cfg.GetRole(roleName)
	if err != nil {
		return fmt.Errorf("role %q not found", roleName)
	}

	skills, err := scanAllSkills(cfg)
	if err != nil {
		return err
	}

	result, err := tui.RunPicker(skills, existing.Skills)
	if err != nil {
		return fmt.Errorf("running picker: %w", err)
	}
	if result.Cancelled {
		fmt.Println("Cancelled.")
		return nil
	}

	updated := *existing
	updated.Skills = result.Selected
	cfg.Roles[roleName] = updated

	if err := saveToGlobalConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Role %q updated with %d skills.\n", roleName, len(result.Selected))
	return nil
}

// --- role show ------------------------------------------------------------

func runRoleShow(cmd *cobra.Command, args []string) error {
	roleName := args[0]

	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	role, err := cfg.GetRole(roleName)
	if err != nil {
		return fmt.Errorf("role %q not found", roleName)
	}

	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	fmt.Println(hdrStyle.Render("Role: ") + nameStyle.Render(roleName))
	fmt.Println(hdrStyle.Render("Description: ") + dimStyle.Render(role.Description))
	fmt.Println(hdrStyle.Render("Missing action: ") + dimStyle.Render(role.MissingAction))
	fmt.Println(hdrStyle.Render(fmt.Sprintf("Skills (%d):", len(role.Skills))))

	for _, s := range role.Skills {
		fmt.Println("  " + nameStyle.Render(s))
	}

	return nil
}

// --- helpers --------------------------------------------------------------

func scanAllSkills(cfg *config.Config) ([]scanner.Skill, error) {
	expanded := make([]string, len(cfg.SkillPaths))
	for i, p := range cfg.SkillPaths {
		expanded[i] = config.ExpandPath(p)
	}

	skills, err := scanner.ScanSkillPaths(expanded)
	if err != nil {
		return nil, fmt.Errorf("scanning skills: %w", err)
	}

	if len(skills) == 0 {
		return nil, fmt.Errorf("no skills found in configured paths")
	}

	return skills, nil
}

func saveToGlobalConfig(cfg *config.Config) error {
	globalPath, err := config.GlobalConfigPath()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	// Collapse skill paths back to tilde form for saving.
	home, _ := os.UserHomeDir()
	saveCfg := *cfg
	saveCfg.SkillPaths = make([]string, len(cfg.SkillPaths))
	for i, p := range cfg.SkillPaths {
		if home != "" && strings.HasPrefix(p, home) {
			saveCfg.SkillPaths[i] = "~" + p[len(home):]
		} else {
			saveCfg.SkillPaths[i] = p
		}
	}

	if err := config.SaveConfig(globalPath, &saveCfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	return nil
}

// sortStrings sorts a string slice in place. Avoids importing "sort" for a
// single trivial use.
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] < ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}
