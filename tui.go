package main

import (
	"fmt"
	"reflect"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	sTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7B61FF")).
		Padding(0, 2)

	sPersonName = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#C4A1FF"))

	sFieldName = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA"))

	sBold = lipgloss.NewStyle().Bold(true)

	sSubtle = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	sHelp = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	sSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#73F59F"))

	sConflictDone = lipgloss.NewStyle().Foreground(lipgloss.Color("#73F59F"))
	sConflictCur  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7DD6F4"))
	sConflictPend = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	sBoxSelected = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7DD6F4")).
			Foreground(lipgloss.Color("#FAFAFA")).
			Padding(0, 1)

	sBoxNormal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Foreground(lipgloss.Color("#777777")).
			Padding(0, 1)
)

// ── Model ─────────────────────────────────────────────────────────────────────

type tuiPhase int

const (
	phaseOverview  tuiPhase = iota
	phaseResolving tuiPhase = iota
	phaseDone      tuiPhase = iota
)

type tuiModel struct {
	conflicts   []Conflict
	merged      []Person
	personIndex map[string]Person

	phase   tuiPhase
	current int   // index of current conflict being resolved
	choice  int   // 0 = mine, 1 = theirs
	choices []int // one entry per conflict
	scroll  int   // scroll offset for overview

	width  int
	height int
}

func newTUIModel(conflicts []Conflict, merged []Person) tuiModel {
	idx := make(map[string]Person, len(merged))
	for _, p := range merged {
		idx[p.ID] = p
	}
	return tuiModel{
		conflicts:   conflicts,
		merged:      merged,
		personIndex: idx,
		choices:     make([]int, len(conflicts)),
	}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch m.phase {
		case phaseOverview:
			switch msg.String() {
			case "up", "k":
				if m.scroll > 0 {
					m.scroll--
				}
			case "down", "j":
				m.scroll++
			case "enter", " ":
				m.phase = phaseResolving
			case "q", "ctrl+c":
				return m, tea.Quit
			}

		case phaseResolving:
			switch msg.String() {
			case "left", "h", "m":
				m.choice = 0
			case "right", "l", "t":
				m.choice = 1
			case "enter", " ":
				m.choices[m.current] = m.choice
				m.current++
				m.choice = 0
				if m.current >= len(m.conflicts) {
					m.phase = phaseDone
				}
			case "q", "ctrl+c":
				return m, tea.Quit
			}

		case phaseDone:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m tuiModel) View() string {
	switch m.phase {
	case phaseOverview:
		return m.viewOverview()
	case phaseResolving:
		return m.viewResolve()
	case phaseDone:
		return m.viewDone()
	}
	return ""
}

// ── Overview ──────────────────────────────────────────────────────────────────

func (m tuiModel) viewOverview() string {
	// Group conflicts by person, preserving order.
	var order []string
	grouped := make(map[string][]Conflict)
	for _, c := range m.conflicts {
		if _, ok := grouped[c.PersonID]; !ok {
			order = append(order, c.PersonID)
		}
		grouped[c.PersonID] = append(grouped[c.PersonID], c)
	}

	// Build all scrollable lines (no trailing newlines inside entries).
	var lines []string
	for _, id := range order {
		p := m.personIndex[id]
		lines = append(lines, "  "+sPersonName.Render(fullName(p)))
		for _, c := range grouped[id] {
			lines = append(lines, sSubtle.Render(fmt.Sprintf(
				"    %-30s  %q  →  %q", c.Field, c.Mine, c.Theirs)))
		}
		lines = append(lines, "") // blank separator between persons
	}

	// Fixed header: \n + title\n + \n + summary\n + \n = 5 lines.
	header := "\n" +
		sTitle.Render(" Family Tree Merger — Conflict Resolution ") + "\n\n" +
		sBold.Render(fmt.Sprintf("  %d conflict(s) across %d person(s)", len(m.conflicts), len(order))) + "\n\n"

	// Fixed footer: \n + help\n = 2 lines.
	const headerLines = 5
	const footerLines = 2

	visibleH := m.height - headerLines - footerLines
	if visibleH < 5 || m.height == 0 {
		visibleH = len(lines) // show all when terminal size not yet known
	}

	// Clamp scroll offset.
	scroll := m.scroll
	if maxScroll := len(lines) - visibleH; scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + visibleH
	if end > len(lines) {
		end = len(lines)
	}
	content := strings.Join(lines[scroll:end], "\n")

	scrollHint := ""
	if len(lines) > visibleH {
		scrollHint = fmt.Sprintf("   [%d–%d / %d]", scroll+1, end, len(lines))
	}
	footer := "\n" + sHelp.Render("  ↑ ↓ to scroll   Enter to start resolving   q to quit"+scrollHint) + "\n"

	return header + content + footer
}

// ── Resolve ───────────────────────────────────────────────────────────────────

func (m tuiModel) viewResolve() string {
	c := m.conflicts[m.current]
	p := m.personIndex[c.PersonID]

	// Collect conflict indices for this person (to show progress within person).
	var personIdx []int
	for i, cc := range m.conflicts {
		if cc.PersonID == c.PersonID {
			personIdx = append(personIdx, i)
		}
	}

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(sTitle.Render(fmt.Sprintf(" Conflict %d / %d ", m.current+1, len(m.conflicts))))
	b.WriteString("\n\n")

	b.WriteString("  " + sPersonName.Render(fullName(p)) + "\n\n")

	// Conflict list for this person
	b.WriteString(sSubtle.Render("  Conflicts for this person:\n"))
	for _, idx := range personIdx {
		cc := m.conflicts[idx]
		const ind = "    "
		switch {
		case idx < m.current:
			choice := "mine"
			if m.choices[idx] == 1 {
				choice = "theirs"
			}
			b.WriteString(ind + sConflictDone.Render("✓") + "  " + fmt.Sprintf("%-30s", cc.Field) + "  " + sSubtle.Render("("+choice+")") + "\n")
		case idx == m.current:
			b.WriteString(ind + sConflictCur.Render("→") + "  " + cc.Field + "\n")
		default:
			b.WriteString(ind + sConflictPend.Render("·") + "  " + cc.Field + "\n")
		}
	}
	b.WriteString("\n")

	// Field label
	b.WriteString("  Field: " + sFieldName.Render(c.Field) + "\n\n")

	// Mine / Theirs boxes side by side
	boxW := 34

	var mineLabel, theirsLabel string
	mineBox := sBoxNormal.Width(boxW).Render(c.Mine)
	theirsBox := sBoxNormal.Width(boxW).Render(c.Theirs)

	if m.choice == 0 {
		mineBox = sBoxSelected.Width(boxW).Render(c.Mine)
		mineLabel = sBold.Render("◀ Mine")
		theirsLabel = sSubtle.Render("  Theirs")
	} else {
		theirsBox = sBoxSelected.Width(boxW).Render(c.Theirs)
		mineLabel = sSubtle.Render("  Mine")
		theirsLabel = sBold.Render("Theirs ▶")
	}

	mineCol := lipgloss.JoinVertical(lipgloss.Left, "  "+mineLabel, mineBox)
	theirsCol := lipgloss.JoinVertical(lipgloss.Left, "  "+theirsLabel, theirsBox)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, mineCol, "  ", theirsCol))
	b.WriteString("\n\n")

	b.WriteString(sHelp.Render("  ← / m  Mine     Theirs  → / t     Enter to confirm   q to quit"))
	b.WriteString("\n")
	return b.String()
}

// ── Done ──────────────────────────────────────────────────────────────────────

func (m tuiModel) viewDone() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(sTitle.Render(" Family Tree Merger — Done "))
	b.WriteString("\n\n")

	mineCount, theirsCount := 0, 0
	for _, ch := range m.choices {
		if ch == 0 {
			mineCount++
		} else {
			theirsCount++
		}
	}

	b.WriteString(sSuccess.Render(fmt.Sprintf("  ✓  All %d conflict(s) resolved\n\n", len(m.conflicts))))
	b.WriteString(fmt.Sprintf("  Kept mine:   %s\n", sBold.Render(fmt.Sprint(mineCount))))
	b.WriteString(fmt.Sprintf("  Kept theirs: %s\n\n", sBold.Render(fmt.Sprint(theirsCount))))
	b.WriteString(sHelp.Render("  Press any key to save and exit"))
	b.WriteString("\n")
	return b.String()
}

// ── Run ───────────────────────────────────────────────────────────────────────

// runInteractive launches the Bubble Tea TUI for interactive conflict resolution.
// Returns the updated merged persons and any conflicts the user kept as "mine".
func runInteractive(conflicts []Conflict, merged []Person) ([]Person, []Conflict, error) {
	m := newTUIModel(conflicts, merged)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	result, err := prog.Run()
	if err != nil {
		return nil, nil, err
	}
	fm := result.(tuiModel)

	updatedMerged := applyChoices(fm.merged, fm.conflicts, fm.choices)

	var remaining []Conflict
	for i, c := range fm.conflicts {
		if fm.choices[i] == 0 {
			remaining = append(remaining, c)
		}
	}
	return updatedMerged, remaining, nil
}

// ── Apply choices ─────────────────────────────────────────────────────────────

func applyChoices(merged []Person, conflicts []Conflict, choices []int) []Person {
	result := make([]Person, len(merged))
	copy(result, merged)
	for i, c := range conflicts {
		if choices[i] == 1 {
			applyTheirsChoice(result, c)
		}
	}
	return result
}

func applyTheirsChoice(merged []Person, c Conflict) {
	parts := strings.SplitN(c.Field, ".", 2)
	if len(parts) != 2 {
		return
	}
	section, field := parts[0], parts[1]
	for i := range merged {
		if merged[i].ID != c.PersonID {
			continue
		}
		switch section {
		case "data":
			setPersonDataField(&merged[i].Data, field, c.Theirs)
		case "rels":
			switch field {
			case "father":
				merged[i].Rels.Father = c.Theirs
			case "mother":
				merged[i].Rels.Mother = c.Theirs
			}
		}
		return
	}
}

// setPersonDataField sets a PersonData field by its JSON tag name.
func setPersonDataField(data *PersonData, jsonKey, value string) {
	t := reflect.TypeOf(*data)
	v := reflect.ValueOf(data).Elem()
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == jsonKey {
			v.Field(i).SetString(value)
			return
		}
	}
}
