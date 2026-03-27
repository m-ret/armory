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

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage skill teams",
	Long:  "Create, edit, list, and inspect teams that group skills together.",
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured teams",
	RunE:  runTeamList,
}

var teamCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new team with interactive skill picker",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamCreate,
}

var teamEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an existing team's skills",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamEdit,
}

var teamShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a team",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamShow,
}

func init() {
	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamCreateCmd)
	teamCmd.AddCommand(teamEditCmd)
	teamCmd.AddCommand(teamShowCmd)
	rootCmd.AddCommand(teamCmd)
}

// --- team list ------------------------------------------------------------

func runTeamList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Teams) == 0 {
		fmt.Println("No teams configured. Use 'armory team create <name>' to create one.")
		return nil
	}

	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	// Compute column widths.
	nameW := len("TEAM")
	descW := len("DESCRIPTION")
	for name, team := range cfg.Teams {
		if len(name) > nameW {
			nameW = len(name)
		}
		if len(team.Description) > descW {
			descW = len(team.Description)
		}
	}

	pad := 3
	fmtStr := fmt.Sprintf("%%-%ds%%-%ds%%s", nameW+pad, descW+pad)

	header := fmt.Sprintf(fmtStr, "TEAM", "DESCRIPTION", "SKILLS")
	fmt.Println(hdrStyle.Render(header))
	fmt.Println(hdrStyle.Render(strings.Repeat("─", nameW+descW+8+2*pad)))

	// Sort team names for stable output.
	names := make([]string, 0, len(cfg.Teams))
	for n := range cfg.Teams {
		names = append(names, n)
	}
	sortStrings(names)

	for _, name := range names {
		team := cfg.Teams[name]
		line := fmt.Sprintf(fmtStr,
			nameStyle.Render(name),
			dimStyle.Render(team.Description),
			dimStyle.Render(fmt.Sprintf("%d skills", len(team.Skills))),
		)
		fmt.Println(line)
	}

	return nil
}

// --- team create ----------------------------------------------------------

func runTeamCreate(cmd *cobra.Command, args []string) error {
	teamName := args[0]

	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Check for existing team.
	if _, exists := cfg.Teams[teamName]; exists {
		fmt.Printf("Team %q already exists. Overwrite? [y/N] ", teamName)
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
		fmt.Println("No skills selected. Team not created.")
		return nil
	}

	// Ask for description.
	fmt.Print("Description: ")
	reader := bufio.NewScanner(os.Stdin)
	reader.Scan()
	description := strings.TrimSpace(reader.Text())

	team := config.Team{
		Description:   description,
		Skills:        result.Selected,
		MissingAction: "prompt",
	}

	if cfg.Teams == nil {
		cfg.Teams = make(map[string]config.Team)
	}
	cfg.Teams[teamName] = team

	if err := saveToGlobalConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Team %q created with %d skills.\n", teamName, len(result.Selected))
	return nil
}

// --- team edit ------------------------------------------------------------

func runTeamEdit(cmd *cobra.Command, args []string) error {
	teamName := args[0]

	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	existing, err := cfg.GetTeam(teamName)
	if err != nil {
		return fmt.Errorf("team %q not found", teamName)
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
	cfg.Teams[teamName] = updated

	if err := saveToGlobalConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Team %q updated with %d skills.\n", teamName, len(result.Selected))
	return nil
}

// --- team show ------------------------------------------------------------

func runTeamShow(cmd *cobra.Command, args []string) error {
	teamName := args[0]

	cfg, err := config.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	team, err := cfg.GetTeam(teamName)
	if err != nil {
		return fmt.Errorf("team %q not found", teamName)
	}

	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	fmt.Println(hdrStyle.Render("Team: ") + nameStyle.Render(teamName))
	fmt.Println(hdrStyle.Render("Description: ") + dimStyle.Render(team.Description))
	fmt.Println(hdrStyle.Render("Missing action: ") + dimStyle.Render(team.MissingAction))
	fmt.Println(hdrStyle.Render(fmt.Sprintf("Skills (%d):", len(team.Skills))))

	for _, s := range team.Skills {
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
