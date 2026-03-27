package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/m-ret/armory/internal/scanner"
)

// PickerResult holds the outcome of the interactive picker.
type PickerResult struct {
	Selected  []string
	Cancelled bool
}

// item wraps a skill with its selection state and whether it is a category header.
type item struct {
	skill      scanner.Skill
	selected   bool
	isHeader   bool
	headerName string
}

type model struct {
	items    []item   // all rows (headers + skills)
	filtered []int    // indices into items that match the current filter
	cursor   int      // position within filtered
	filter   string   // current search text
	done     bool
	cancel   bool
	height   int
}

// --- styles ---------------------------------------------------------------

var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e94560"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db")).Bold(true)
	checkMark     = selectedStyle.Render("[x]")
	emptyMark     = dimStyle.Render("[ ]")
)

// --- public API -----------------------------------------------------------

// RunPicker launches the interactive skill picker and returns the result.
// preSelected contains skill names that should start checked.
func RunPicker(skills []scanner.Skill, preSelected []string) (PickerResult, error) {
	preMap := make(map[string]bool, len(preSelected))
	for _, n := range preSelected {
		preMap[n] = true
	}

	// Group skills by category.
	groups := make(map[string][]scanner.Skill)
	var categories []string
	for _, s := range skills {
		cat := s.Category
		if cat == "" {
			cat = "other"
		}
		if _, ok := groups[cat]; !ok {
			categories = append(categories, cat)
		}
		groups[cat] = append(groups[cat], s)
	}
	sort.Strings(categories)

	// Build flat item list with headers.
	var items []item
	for _, cat := range categories {
		items = append(items, item{isHeader: true, headerName: cat})
		catSkills := groups[cat]
		sort.Slice(catSkills, func(i, j int) bool {
			return catSkills[i].Name < catSkills[j].Name
		})
		for _, s := range catSkills {
			items = append(items, item{skill: s, selected: preMap[s.Name]})
		}
	}

	// Initial filtered list = everything; cursor starts at first non-header.
	filtered := make([]int, len(items))
	for i := range items {
		filtered[i] = i
	}
	startCursor := 0
	for i, idx := range filtered {
		if !items[idx].isHeader {
			startCursor = i
			break
		}
	}

	m := model{
		items:    items,
		filtered: filtered,
		cursor:   startCursor,
		height:   20,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return PickerResult{}, fmt.Errorf("running picker: %w", err)
	}

	fm := finalModel.(model)
	if fm.cancel {
		return PickerResult{Cancelled: true}, nil
	}

	var selected []string
	for _, it := range fm.items {
		if !it.isHeader && it.selected {
			selected = append(selected, it.skill.Name)
		}
	}
	sort.Strings(selected)
	return PickerResult{Selected: selected}, nil
}

// --- bubbletea interface --------------------------------------------------

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height - 4 // leave room for status + filter
		if m.height < 5 {
			m.height = 5
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			// Only quit/cancel when not filtering.
			if m.filter == "" {
				m.cancel = true
				m.done = true
				return m, tea.Quit
			}
			// If filtering, clear filter instead.
			if msg.String() == "esc" {
				m.filter = ""
				m.refilter()
				return m, nil
			}
			// q while filtering is just a character.
			if msg.String() == "q" {
				m.filter += "q"
				m.refilter()
				return m, nil
			}
			m.cancel = true
			m.done = true
			return m, tea.Quit

		case "enter":
			m.done = true
			return m, tea.Quit

		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)

		case " ":
			m.toggleCurrent()

		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.refilter()
			}

		default:
			// Single printable character goes to filter.
			if len(msg.String()) == 1 && msg.String() >= " " && msg.String() != " " {
				m.filter += msg.String()
				m.refilter()
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	// Filter line.
	if m.filter != "" {
		b.WriteString(statusStyle.Render("Filter: ") + m.filter + "\n")
	} else {
		b.WriteString(dimStyle.Render("Type to filter | Space: toggle | Enter: confirm | Esc: cancel") + "\n")
	}

	// Determine visible window.
	start := 0
	end := len(m.filtered)
	if end > m.height {
		half := m.height / 2
		start = m.cursor - half
		if start < 0 {
			start = 0
		}
		end = start + m.height
		if end > len(m.filtered) {
			end = len(m.filtered)
			start = end - m.height
			if start < 0 {
				start = 0
			}
		}
	}

	for vi := start; vi < end; vi++ {
		idx := m.filtered[vi]
		it := m.items[idx]

		cursor := "  "
		if vi == m.cursor {
			cursor = "> "
		}

		if it.isHeader {
			b.WriteString(headerStyle.Render("  "+strings.ToUpper(it.headerName)) + "\n")
		} else {
			check := emptyMark
			if it.selected {
				check = checkMark
			}
			name := normalStyle.Render(it.skill.Name)
			if vi == m.cursor {
				name = selectedStyle.Render(it.skill.Name)
			}
			desc := ""
			if it.skill.Description != "" {
				desc = " " + dimStyle.Render(it.skill.Description)
			}
			b.WriteString(cursor + check + " " + name + desc + "\n")
		}
	}

	// Status bar.
	count := 0
	for _, it := range m.items {
		if !it.isHeader && it.selected {
			count++
		}
	}
	b.WriteString("\n" + statusStyle.Render(fmt.Sprintf("%d selected", count)))

	return b.String()
}

// --- helpers --------------------------------------------------------------

func (m *model) refilter() {
	if m.filter == "" {
		m.filtered = make([]int, len(m.items))
		for i := range m.items {
			m.filtered[i] = i
		}
		m.clampCursor()
		return
	}

	lower := strings.ToLower(m.filter)
	var filtered []int

	// Track which categories have matching skills.
	matchingCats := make(map[string]bool)
	for i, it := range m.items {
		if it.isHeader {
			continue
		}
		if fuzzyMatch(strings.ToLower(it.skill.Name), lower) {
			matchingCats[it.skill.Category] = true
			filtered = append(filtered, i)
		}
	}

	// Re-insert headers for categories that have matches.
	var withHeaders []int
	lastCat := ""
	for _, idx := range filtered {
		it := m.items[idx]
		cat := it.skill.Category
		if cat != lastCat {
			// Find the header for this category.
			for hi, h := range m.items {
				if h.isHeader && h.headerName == cat {
					withHeaders = append(withHeaders, hi)
					break
				}
			}
			lastCat = cat
		}
		withHeaders = append(withHeaders, idx)
	}

	m.filtered = withHeaders
	m.clampCursor()

	// Skip header if cursor lands on one.
	if m.cursor < len(m.filtered) {
		if m.items[m.filtered[m.cursor]].isHeader {
			m.moveCursor(1)
		}
	}
}

func (m *model) moveCursor(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.cursor += delta
	m.clampCursor()

	// Skip headers.
	for m.cursor >= 0 && m.cursor < len(m.filtered) && m.items[m.filtered[m.cursor]].isHeader {
		m.cursor += delta
	}
	m.clampCursor()
}

func (m *model) clampCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *model) toggleCurrent() {
	if m.cursor >= len(m.filtered) {
		return
	}
	idx := m.filtered[m.cursor]
	if m.items[idx].isHeader {
		return
	}
	m.items[idx].selected = !m.items[idx].selected
}

// fuzzyMatch checks if all characters in pattern appear in s in order.
func fuzzyMatch(s, pattern string) bool {
	pi := 0
	for i := 0; i < len(s) && pi < len(pattern); i++ {
		if s[i] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}
