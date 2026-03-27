package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/m-ret/armory/internal/config"
	"github.com/m-ret/armory/internal/registry"
	"github.com/m-ret/armory/internal/scanner"
	"github.com/m-ret/armory/internal/teams"
)

// --- public types -----------------------------------------------------------

// WizardResult holds the outcome of the setup wizard.
type WizardResult struct {
	Teams     map[string]config.Team
	Cancelled bool
}

// --- wizard screens ---------------------------------------------------------

type wizScreen int

const (
	wizWelcome wizScreen = iota
	wizTeamPick
	wizScanning
	wizInstallConfirm
	wizInstalling
	wizSummary
)

// --- custom messages --------------------------------------------------------

type wizScanDoneMsg struct {
	teamResults []teamScanResult
}

type wizInstallLineMsg string

type wizInstallDoneMsg struct {
	results []registry.InstallResult
	err     error
}

// --- scan result per team ---------------------------------------------------

type teamScanResult struct {
	presetName   string
	localSkills  []string
	remoteSkills []registry.SearchResult
	notFound     []string
	totalSkills  int
}

// --- styles -----------------------------------------------------------------

var (
	wizAccent    = lipgloss.NewStyle().Foreground(lipgloss.Color("#e94560")).Bold(true)
	wizGreen     = lipgloss.NewStyle().Foreground(lipgloss.Color("#27ae60"))
	wizDim       = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	wizNormal    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	wizBold      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
	wizCheckmark = wizGreen.Render("✓")
	wizBox       = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#e94560")).
			Padding(1, 3)
)

// --- wizard model -----------------------------------------------------------

type wizardModel struct {
	screen       wizScreen
	presets      []teams.Preset
	selected     map[int]bool
	cursor       int
	localSkills  []scanner.Skill
	localSet     map[string]bool
	teamResults  []teamScanResult
	installLines []string
	installCh    <-chan tea.Msg // channel for streaming install progress
	installCount int
	localCount   int
	remoteCount  int
	confirmYes   bool // true = user said yes to install, false = skipped
	width        int
	height       int
	cancelled    bool
	err          error
}

// --- public API -------------------------------------------------------------

// RunWizard launches the setup wizard and returns the configured teams.
func RunWizard(localSkills []scanner.Skill) (WizardResult, error) {
	presets := teams.AllPresets()

	localSet := make(map[string]bool, len(localSkills))
	for _, s := range localSkills {
		localSet[s.Name] = true
	}

	m := wizardModel{
		screen:      wizWelcome,
		presets:     presets,
		selected:    make(map[int]bool),
		localSkills: localSkills,
		localSet:    localSet,
		width:       80,
		height:      24,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return WizardResult{}, fmt.Errorf("running wizard: %w", err)
	}

	fm := finalModel.(wizardModel)
	if fm.cancelled {
		return WizardResult{Cancelled: true}, nil
	}

	// Build teams from selected presets.
	resultTeams := make(map[string]config.Team)
	for i, sel := range fm.selected {
		if !sel {
			continue
		}
		preset := fm.presets[i]
		resultTeams[preset.Name] = config.Team{
			Description:   preset.Description,
			Skills:        preset.Skills,
			MissingAction: "prompt",
		}
	}

	// Save config.
	cfgPath, pathErr := config.GlobalConfigPath()
	if pathErr != nil {
		return WizardResult{}, fmt.Errorf("getting config path: %w", pathErr)
	}

	cfg := config.DefaultConfig()
	cfg.Teams = resultTeams

	if saveErr := config.SaveConfig(cfgPath, cfg); saveErr != nil {
		return WizardResult{}, fmt.Errorf("saving config: %w", saveErr)
	}

	return WizardResult{Teams: resultTeams}, nil
}

// --- bubbletea interface ----------------------------------------------------

func (m wizardModel) Init() tea.Cmd {
	return nil
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case wizScanDoneMsg:
		m.teamResults = msg.teamResults
		// Calculate how many remote skills are available.
		var totalRemote int
		for _, tr := range m.teamResults {
			totalRemote += len(tr.remoteSkills)
		}
		if totalRemote > 0 {
			m.screen = wizInstallConfirm
			m.remoteCount = totalRemote
		} else {
			m.screen = wizSummary
			m.computeSummary()
		}
		return m, nil

	case wizInstallLineMsg:
		m.installLines = append(m.installLines, string(msg))
		return m, installRelayCmd(m.installCh)

	case wizInstallDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		var installed int
		for _, r := range msg.results {
			if r.Err == nil {
				installed++
			}
		}
		m.installCount = installed
		m.screen = wizSummary
		m.computeSummary()
		return m, nil
	}

	return m, nil
}

func (m wizardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit.
	if key == "ctrl+c" {
		m.cancelled = true
		return m, tea.Quit
	}

	switch m.screen {
	case wizWelcome:
		if key == "enter" {
			m.screen = wizTeamPick
		} else if key == "esc" || key == "q" {
			m.cancelled = true
			return m, tea.Quit
		}

	case wizTeamPick:
		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.presets)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
			if m.countSelected() == 0 {
				return m, nil
			}
			m.screen = wizScanning
			return m, m.scanCmd()
		case "esc", "q":
			m.cancelled = true
			return m, tea.Quit
		}

	case wizInstallConfirm:
		switch key {
		case "y", "Y", "enter":
			m.confirmYes = true
			m.screen = wizInstalling
			cmd := m.startInstall()
			return m, cmd
		case "n", "N":
			m.confirmYes = false
			m.screen = wizSummary
			m.computeSummary()
		case "esc", "q":
			m.cancelled = true
			return m, tea.Quit
		}

	case wizInstalling:
		// No key input during install.
		return m, nil

	case wizSummary:
		if key == "enter" || key == "esc" || key == "q" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m wizardModel) View() string {
	switch m.screen {
	case wizWelcome:
		return m.viewWelcome()
	case wizTeamPick:
		return m.viewTeamPick()
	case wizScanning:
		return m.viewScanning()
	case wizInstallConfirm:
		return m.viewInstallConfirm()
	case wizInstalling:
		return m.viewInstalling()
	case wizSummary:
		return m.viewSummary()
	default:
		return ""
	}
}

// --- screen views -----------------------------------------------------------

func (m wizardModel) viewWelcome() string {
	content := wizAccent.Render("Welcome to armory") + "\n\n" +
		wizNormal.Render("Skill control plane for") + "\n" +
		wizNormal.Render("AI agents.") + "\n\n" +
		wizNormal.Render("Let's set up your teams.") + "\n\n" +
		wizDim.Render("Press Enter to continue...")

	box := wizBox.Render(content)
	return "\n" + centerHorizontally(box, m.width) + "\n"
}

func (m wizardModel) viewTeamPick() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + wizBold.Render("Select your teams") +
		wizDim.Render(" (space to toggle, enter to confirm)") + "\n\n")

	for i, preset := range m.presets {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		check := wizDim.Render("  ")
		if m.selected[i] {
			check = wizCheckmark + " "
		}

		name := wizNormal.Render(fmt.Sprintf("%-12s", preset.Name))
		if i == m.cursor {
			name = wizGreen.Render(fmt.Sprintf("%-12s", preset.Name))
		}

		desc := wizDim.Render(preset.Description)
		b.WriteString(cursor + check + name + desc + "\n")
	}

	b.WriteString("\n  " + wizAccent.Render(fmt.Sprintf("%d selected", m.countSelected())))
	return b.String()
}

func (m wizardModel) viewScanning() string {
	var b strings.Builder
	b.WriteString("\n  " + wizBold.Render("Setting up your teams...") + "\n\n")

	if len(m.teamResults) == 0 {
		b.WriteString("  " + wizDim.Render("Scanning skills...") + "\n")
		return b.String()
	}

	for _, tr := range m.teamResults {
		b.WriteString("  " + wizAccent.Render(tr.presetName+":") + "\n")
		renderTeamResult(&b, tr)
		b.WriteString("\n")
	}

	return b.String()
}

func (m wizardModel) viewInstallConfirm() string {
	var b strings.Builder
	b.WriteString("\n  " + wizBold.Render("Setting up your teams...") + "\n\n")

	for _, tr := range m.teamResults {
		b.WriteString("  " + wizAccent.Render(tr.presetName+":") + "\n")
		renderTeamResult(&b, tr)
		b.WriteString("\n")
	}

	b.WriteString("  " + wizBold.Render(
		fmt.Sprintf("Install %d skills from skills.sh? [Y/n]", m.remoteCount)) + "\n")

	return b.String()
}

func (m wizardModel) viewInstalling() string {
	var b strings.Builder
	b.WriteString("\n  " + wizBold.Render("Installing skills from skills.sh...") + "\n\n")

	for _, line := range m.installLines {
		b.WriteString("    " + wizGreen.Render("+ ") + wizNormal.Render(line) + "\n")
	}

	if m.screen == wizInstalling {
		b.WriteString("\n  " + wizDim.Render("Working...") + "\n")
	}

	return b.String()
}

func (m wizardModel) viewSummary() string {
	var b strings.Builder
	b.WriteString("\n  " + wizAccent.Render("Setup complete!") + "\n\n")

	numTeams := m.countSelected()
	totalSkills := m.localCount + m.installCount

	b.WriteString(fmt.Sprintf("    %s teams configured\n",
		wizBold.Render(fmt.Sprintf("%d", numTeams))))

	detail := fmt.Sprintf("%d local", m.localCount)
	if m.installCount > 0 {
		detail += fmt.Sprintf(" + %d installed", m.installCount)
	}
	b.WriteString(fmt.Sprintf("    %s skills ready (%s)\n",
		wizBold.Render(fmt.Sprintf("%d", totalSkills)), detail))

	if m.err != nil {
		b.WriteString("\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#e74c3c")).
			Render("Warning: some installs failed: "+m.err.Error()) + "\n")
	}

	b.WriteString("\n  " + wizDim.Render("Press Enter to continue...") + "\n")

	return b.String()
}

// --- commands ---------------------------------------------------------------

func (m wizardModel) scanCmd() tea.Cmd {
	return func() tea.Msg {
		var results []teamScanResult

		for i, preset := range m.presets {
			if !m.selected[i] {
				continue
			}

			var local []string
			var missing []string

			for _, skillName := range preset.Skills {
				if m.localSet[skillName] {
					local = append(local, skillName)
				} else {
					missing = append(missing, skillName)
				}
			}

			var remote []registry.SearchResult
			var notFound []string
			if len(missing) > 0 {
				remote, notFound = registry.SearchByNames(missing)
			}

			results = append(results, teamScanResult{
				presetName:   preset.Name,
				localSkills:  local,
				remoteSkills: remote,
				notFound:     notFound,
				totalSkills:  len(preset.Skills),
			})
		}

		return wizScanDoneMsg{teamResults: results}
	}
}

func (m *wizardModel) startInstall() tea.Cmd {
	// Gather all remote skills to install.
	var toInstall []registry.SearchResult
	for _, tr := range m.teamResults {
		toInstall = append(toInstall, tr.remoteSkills...)
	}

	ch := make(chan tea.Msg, 64)
	m.installCh = ch

	go func() {
		progress := func(msg string) {
			ch <- wizInstallLineMsg(msg)
		}

		results, err := registry.InstallSkills(
			context.Background(), toInstall, progress,
		)

		// Send individual result lines for display.
		for _, r := range results {
			if r.Err == nil {
				ch <- wizInstallLineMsg(
					fmt.Sprintf("%s -> %s", r.Name, r.Dir))
			}
		}

		ch <- wizInstallDoneMsg{results: results, err: err}
		close(ch)
	}()

	return installRelayCmd(ch)
}

// installRelayCmd returns a command that reads from a channel and returns
// the next message. This creates a chain of commands for streaming output.
func installRelayCmd(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

// --- helpers ----------------------------------------------------------------

func (m wizardModel) countSelected() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

func (m *wizardModel) computeSummary() {
	var local int
	for _, tr := range m.teamResults {
		local += len(tr.localSkills)
	}
	m.localCount = local
	// installCount is set by wizInstallDoneMsg handler.
}

func renderTeamResult(b *strings.Builder, tr teamScanResult) {
	// Local skills line.
	if len(tr.localSkills) > 0 {
		names := strings.Join(tr.localSkills, ", ")
		b.WriteString(fmt.Sprintf("    %s %s (%d/%d)\n",
			wizGreen.Render("Found locally:"),
			wizNormal.Render(names),
			len(tr.localSkills), tr.totalSkills))
	}

	// Remote skills line.
	if len(tr.remoteSkills) > 0 {
		const maxShow = 2
		var names []string
		for i, r := range tr.remoteSkills {
			if i >= maxShow {
				break
			}
			names = append(names, r.Name)
		}
		line := strings.Join(names, ", ")
		if len(tr.remoteSkills) > maxShow {
			line += fmt.Sprintf(" + %d more", len(tr.remoteSkills)-maxShow)
		}
		b.WriteString(fmt.Sprintf("    %s %s (%d more)\n",
			wizGreen.Render("Available from skills.sh:"),
			wizNormal.Render(line),
			len(tr.remoteSkills)))
	}

	// Not found line.
	if len(tr.notFound) > 0 {
		b.WriteString(fmt.Sprintf("    %s %s\n",
			wizDim.Render("Not available:"),
			wizDim.Render(strings.Join(tr.notFound, ", "))))
	}
}

func centerHorizontally(s string, width int) string {
	lines := strings.Split(s, "\n")
	maxLen := 0
	for _, l := range lines {
		w := lipgloss.Width(l)
		if w > maxLen {
			maxLen = w
		}
	}

	pad := (width - maxLen) / 2
	if pad < 0 {
		pad = 0
	}

	prefix := strings.Repeat(" ", pad)
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(prefix + l)
	}
	return b.String()
}
