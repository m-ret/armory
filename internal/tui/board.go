package tui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/m-ret/armory/internal/state"
)

// boardRow holds pre-computed display data for one equipped directory.
type boardRow struct {
	dir       string
	team      string
	skills    string // e.g. "5/6"
	status    state.SymlinkStatus
	since     string
	entry     state.EquippedEntry
	missing   []string // skills in entry.Skills but not in ManagedSymlinks
	staleInfo []string // symlinks that no longer resolve
}

type boardModel struct {
	rows     []boardRow
	cursor   int
	width    int
	height   int
	quitting bool
}

// --- styles -----------------------------------------------------------------

var (
	boardHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	boardTableHdr    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	boardSelectedBg  = lipgloss.NewStyle().Background(lipgloss.Color("#333333"))
	boardNormal      = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	boardDim         = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	statusEquipped = lipgloss.NewStyle().Foreground(lipgloss.Color("#27ae60"))
	statusStale    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f39c12"))
	statusBroken   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e74c3c"))

	boardFooter = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	detailLabel = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#3498db"))
)

// --- public API -------------------------------------------------------------

// RunBoard launches the board TUI dashboard.
func RunBoard() error {
	m := boardModel{width: 80, height: 24}
	m.loadData()

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// --- data loading -----------------------------------------------------------

func (m *boardModel) loadData() {
	st, err := state.LoadState()
	if err != nil || len(st.Equipped) == 0 {
		m.rows = nil
		return
	}

	home, _ := os.UserHomeDir()
	rows := make([]boardRow, 0, len(st.Equipped))

	for _, entry := range st.Equipped {
		status := state.VerifyEquipped(entry)

		// Compute skills fraction: managed / total in role.
		managed := len(entry.ManagedSymlinks)
		total := len(entry.Skills)
		skillsFrac := fmt.Sprintf("%d/%d", managed, total)

		// Determine missing skills.
		managedSet := make(map[string]bool, managed)
		for _, s := range entry.ManagedSymlinks {
			managedSet[s] = true
		}
		var missing []string
		for _, s := range entry.Skills {
			if !managedSet[s] {
				missing = append(missing, s)
			}
		}

		// Determine stale symlinks (managed but broken on disk).
		var staleInfo []string
		if status == state.StatusStale {
			skillsDir := filepath.Join(entry.Dir, ".claude", "skills")
			for _, name := range entry.ManagedSymlinks {
				link := filepath.Join(skillsDir, name)
				fi, linkErr := os.Lstat(link)
				if linkErr != nil || fi.Mode()&os.ModeSymlink == 0 {
					staleInfo = append(staleInfo, name)
				}
			}
		}

		rows = append(rows, boardRow{
			dir:       abbreviateDir(entry.Dir, home),
			team:      entry.Team,
			skills:    skillsFrac,
			status:    status,
			since:     relativeTime(entry.EquippedAt),
			entry:     entry,
			missing:   missing,
			staleInfo: staleInfo,
		})
	}
	m.rows = rows
	if m.cursor >= len(m.rows) {
		m.cursor = max(0, len(m.rows)-1)
	}
}

// --- bubbletea interface ----------------------------------------------------

func (m boardModel) Init() tea.Cmd {
	return nil
}

func (m boardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case "r":
			m.loadData()
		}
	}
	return m, nil
}

func (m boardModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	if len(m.rows) == 0 {
		b.WriteString("\n")
		b.WriteString(boardHeaderStyle.Render("  ARMORY BOARD") + "\n\n")
		b.WriteString(boardDim.Render("  No equipped directories. Run 'armory equip <team>' to get started.") + "\n")
		b.WriteString("\n" + boardFooter.Render("  [q] quit") + "\n")
		return b.String()
	}

	// Header.
	b.WriteString("\n")
	title := fmt.Sprintf("  ARMORY BOARD  (%d equipped)", len(m.rows))
	b.WriteString(boardHeaderStyle.Render(title) + "\n\n")

	// Compute column widths.
	dirW := len("DIRECTORY")
	teamW := len("TEAM")
	skillsW := len("SKILLS")
	statusW := len("STATUS")
	sinceW := len("SINCE")

	for _, r := range m.rows {
		if len(r.dir) > dirW {
			dirW = len(r.dir)
		}
		if len(r.team) > teamW {
			teamW = len(r.team)
		}
		if len(r.skills) > skillsW {
			skillsW = len(r.skills)
		}
		rendered := statusText(r.status)
		if len(rendered) > statusW {
			statusW = len(rendered)
		}
		if len(r.since) > sinceW {
			sinceW = len(r.since)
		}
	}

	pad := 2
	hdrFmt := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s",
		dirW+pad, teamW+pad, skillsW+pad, statusW+pad)

	headerLine := fmt.Sprintf(hdrFmt, "DIRECTORY", "TEAM", "SKILLS", "STATUS", "SINCE")
	b.WriteString(boardTableHdr.Render(headerLine) + "\n")
	sep := strings.Repeat("─", dirW+teamW+skillsW+statusW+sinceW+5*(pad+2)+2)
	b.WriteString(boardDim.Render("  "+sep) + "\n")

	// Table rows.
	for i, r := range m.rows {
		dirCell := boardNormal.Render(r.dir)
		teamCell := boardNormal.Render(r.team)
		skillsCell := boardNormal.Render(r.skills)
		statusCell := renderStatus(r.status)
		sinceCell := boardDim.Render(r.since)

		rowFmt := fmt.Sprintf("  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s",
			dirW+pad, teamW+pad, skillsW+pad, statusW+pad)

		line := fmt.Sprintf(rowFmt, dirCell, teamCell, skillsCell, statusCell, sinceCell)

		if i == m.cursor {
			line = boardSelectedBg.Render(line)
		}
		b.WriteString(line + "\n")
	}

	// Detail pane for selected row.
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		r := m.rows[m.cursor]
		b.WriteString("\n")
		b.WriteString(detailLabel.Render("  Team: ") + boardNormal.Render(r.team) + "\n")
		b.WriteString(detailLabel.Render("  Dir:  ") + boardDim.Render(r.entry.Dir) + "\n")
		b.WriteString(detailLabel.Render("  Skills: ") + boardNormal.Render(strings.Join(r.entry.Skills, ", ")) + "\n")
		if len(r.missing) > 0 {
			b.WriteString(statusBroken.Render("  Missing: "+strings.Join(r.missing, ", ")) + "\n")
		}
		if len(r.staleInfo) > 0 {
			b.WriteString(statusStale.Render("  Stale: "+strings.Join(r.staleInfo, ", ")) + "\n")
		}
	}

	// Footer.
	b.WriteString("\n" + boardFooter.Render("  [j/k] navigate  [r] refresh  [q] quit") + "\n")

	return b.String()
}

// --- helpers ----------------------------------------------------------------

// abbreviateDir replaces the home prefix with ~ and shortens long paths to the
// last two components.
func abbreviateDir(dir, home string) string {
	d := dir
	if home != "" && strings.HasPrefix(d, home) {
		d = "~" + d[len(home):]
	}
	parts := strings.Split(d, string(filepath.Separator))
	if len(parts) > 3 {
		prefix := ""
		if strings.HasPrefix(d, "~") {
			prefix = "~/"
			parts = parts[1:] // drop the "~" element
		}
		if len(parts) > 2 {
			d = prefix + ".../" + strings.Join(parts[len(parts)-2:], string(filepath.Separator))
		}
	}
	return d
}

// relativeTime formats a timestamp as a human-friendly relative duration.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(math.Round(d.Seconds())))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}

func statusText(s state.SymlinkStatus) string {
	switch s {
	case state.StatusEquipped:
		return "equipped"
	case state.StatusStale:
		return "stale"
	case state.StatusBroken:
		return "broken"
	default:
		return "unknown"
	}
}

func renderStatus(s state.SymlinkStatus) string {
	switch s {
	case state.StatusEquipped:
		return statusEquipped.Render("equipped")
	case state.StatusStale:
		return statusStale.Render("stale")
	case state.StatusBroken:
		return statusBroken.Render("broken")
	default:
		return boardDim.Render("unknown")
	}
}
