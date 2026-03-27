package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/m-ret/armory/internal/config"
	"github.com/m-ret/armory/internal/scanner"
	"github.com/m-ret/armory/internal/state"
)

// shellVersion is set by the caller via the RunShell entrypoint.
// It defaults to "dev" but is overridden from cmd.Version.
var shellVersion = "dev"

// SetShellVersion sets the version string shown in the shell header.
func SetShellVersion(v string) {
	shellVersion = v
}

// --- shell screens ----------------------------------------------------------

type shellScreen int

const (
	shellScreenMenu shellScreen = iota
	shellScreenBoard
	shellScreenSettings
)

// --- shell model ------------------------------------------------------------

type shellModel struct {
	screen       shellScreen
	menuItems    []string
	menuDescs    []string
	cursor       int
	statusLine   string
	width        int
	height       int
	quitting     bool
	exitMsg      string // message to print after shell exits

	// Sub-models
	board       boardModel
	boardActive bool

	// Settings view
	settingsInfo string
}

// --- styles -----------------------------------------------------------------

var (
	shellTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	shellSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	shellDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	shellFooter   = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
)

// --- public API -------------------------------------------------------------

// RunShell launches the interactive shell. When firstRun is true, the setup
// wizard runs before showing the main menu.
func RunShell(firstRun bool) error {
	if firstRun {
		// Scan local skills for the wizard.
		cfg := config.DefaultConfig()
		expanded := make([]string, len(cfg.SkillPaths))
		for i, p := range cfg.SkillPaths {
			expanded[i] = config.ExpandPath(p)
		}

		localSkills, _ := scanner.ScanSkillPaths(expanded)

		result, err := RunWizard(localSkills)
		if err != nil {
			return fmt.Errorf("running wizard: %w", err)
		}
		// If wizard was cancelled, proceed to menu anyway.
		_ = result
	}

	m := newShellModel()

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	fm := finalModel.(shellModel)
	if fm.exitMsg != "" {
		fmt.Println(fm.exitMsg)
	}

	return nil
}

// --- model construction -----------------------------------------------------

func newShellModel() shellModel {
	m := shellModel{
		screen: shellScreenMenu,
		menuItems: []string{
			"Equip a terminal",
			"Browse & install skills",
			"Manage teams",
			"Board",
			"Settings",
		},
		menuDescs: []string{
			"Symlink team skills into a directory",
			"Scan and install available skills",
			"Create, edit, and list teams",
			"Dashboard of equipped directories",
			"View paths and configuration",
		},
		width:  80,
		height: 24,
	}

	m.statusLine = computeStatusLine()
	m.settingsInfo = computeSettingsInfo()
	return m
}

func computeStatusLine() string {
	var teamCount, skillCount, equippedCount int

	// Count teams from config.
	cfg, err := config.LoadConfig(".")
	if err == nil {
		teamCount = len(cfg.Teams)
	}

	// Count skills from scanner cache.
	cachePath := config.ExpandPath("~/.armory/cache/skills.json")
	if cfg != nil {
		expanded := make([]string, len(cfg.SkillPaths))
		for i, p := range cfg.SkillPaths {
			expanded[i] = config.ExpandPath(p)
		}
		if scanner.IsCacheValid(cachePath, expanded) {
			if skills, err := scanner.LoadCache(cachePath); err == nil {
				skillCount = len(skills)
			}
		}
	}

	// Count equipped from state.
	st, err := state.LoadState()
	if err == nil {
		equippedCount = len(st.Equipped)
	}

	skillStr := "?"
	if skillCount > 0 {
		skillStr = fmt.Sprintf("%d", skillCount)
	}

	return fmt.Sprintf("%d teams · %s skills · %d equipped", teamCount, skillStr, equippedCount)
}

func computeSettingsInfo() string {
	var b strings.Builder

	cfgPath, err := config.GlobalConfigPath()
	if err != nil {
		cfgPath = "(unknown)"
	}

	b.WriteString(fmt.Sprintf("  Config:      %s\n", cfgPath))
	b.WriteString(fmt.Sprintf("  Skill paths: %s\n", strings.Join(config.DefaultSkillPaths(), ", ")))
	b.WriteString(fmt.Sprintf("  Version:     %s\n", shellVersion))

	return b.String()
}

// --- bubbletea interface ----------------------------------------------------

func (m shellModel) Init() tea.Cmd {
	return nil
}

func (m shellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When the board sub-screen is active, delegate everything to it.
	if m.screen == shellScreenBoard {
		return m.updateBoard(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Settings screen: any key returns to menu.
		if m.screen == shellScreenSettings {
			m.screen = shellScreenMenu
			return m, nil
		}
		return m.handleMenuKey(msg)
	}

	return m, nil
}

func (m shellModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.screen {
	case shellScreenMenu:
		return m.viewMenu()
	case shellScreenBoard:
		return m.board.View()
	case shellScreenSettings:
		return m.viewSettings()
	default:
		return ""
	}
}

// --- menu key handling ------------------------------------------------------

func (m shellModel) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.quitting = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.menuItems)-1 {
			m.cursor++
		}

	case "enter":
		return m.selectMenuItem()
	}

	return m, nil
}

func (m shellModel) selectMenuItem() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0: // Equip a terminal
		m.quitting = true
		cfg, err := config.LoadConfig(".")
		var teamNames []string
		if err == nil {
			for name := range cfg.Teams {
				teamNames = append(teamNames, name)
			}
		}
		teams := "none configured"
		if len(teamNames) > 0 {
			teams = strings.Join(teamNames, ", ")
		}
		m.exitMsg = fmt.Sprintf("Run: armory equip <team> --dir <path>\nAvailable teams: %s", teams)
		return m, tea.Quit

	case 1: // Browse & install skills
		m.quitting = true
		m.exitMsg = "Run: armory scan"
		return m, tea.Quit

	case 2: // Manage teams
		m.quitting = true
		m.exitMsg = "Run: armory team list"
		return m, tea.Quit

	case 3: // Board
		m.screen = shellScreenBoard
		m.board = boardModel{width: m.width, height: m.height}
		m.board.loadData()
		m.boardActive = true
		return m, nil

	case 4: // Settings
		m.screen = shellScreenSettings
		return m, nil
	}

	return m, nil
}

// --- board sub-screen -------------------------------------------------------

func (m shellModel) updateBoard(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.board.Update(msg)
	m.board = updated.(boardModel)

	// When board wants to quit, return to menu instead of quitting the shell.
	if m.board.quitting {
		m.board.quitting = false
		m.screen = shellScreenMenu
		m.boardActive = false
		return m, nil
	}

	return m, cmd
}

// --- views ------------------------------------------------------------------

func (m shellModel) viewMenu() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + shellTitle.Render(fmt.Sprintf("ARMORY v%s", shellVersion)) + "\n")
	b.WriteString("\n")

	for i, item := range m.menuItems {
		if i == m.cursor {
			b.WriteString("  " + shellSelected.Render("> "+item) + "\n")
		} else {
			b.WriteString("    " + shellDim.Render(item) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString("  " + shellDim.Render(m.statusLine) + "\n")
	b.WriteString("\n")
	b.WriteString("  " + shellFooter.Render("[j/k] navigate  [enter] select  [q] quit") + "\n")

	return b.String()
}

func (m shellModel) viewSettings() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + shellTitle.Render("Settings") + "\n")
	b.WriteString("\n")
	b.WriteString(m.settingsInfo)
	b.WriteString("\n")
	b.WriteString("  " + shellFooter.Render("Press any key to return...") + "\n")

	return b.String()
}
