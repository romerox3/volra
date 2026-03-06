package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/romerox3/volra/internal/templates"
)

// Steps in the quickstart TUI flow.
type tuiStep int

const (
	stepSelect   tuiStep = iota // Choose a template.
	stepName                    // Enter project name.
	stepScaffold                // Scaffolding in progress.
	stepDone                    // Finished.
)

// categoryOrder defines the display order for template categories.
var categoryOrder = []string{
	templates.CategoryGettingStarted,
	templates.CategoryUseCase,
	templates.CategoryFramework,
	templates.CategoryPlatform,
}

// scaffoldDoneMsg is sent when scaffolding completes.
type scaffoldDoneMsg struct{ err error }

// tuiModel is the Bubbletea model for the quickstart TUI.
type tuiModel struct {
	step      tuiStep
	templates []templates.Template
	groups    []categoryGroup
	cursor    int
	filter    string
	filtering bool
	input     textinput.Model
	spinner   spinner.Model
	chosen    templates.Template
	name      string
	err       error
	width     int
}

// categoryGroup groups templates under a category heading.
type categoryGroup struct {
	Category  string
	Templates []templates.Template
}

// --- Styles ---

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	categoryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("243"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	descStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	validStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	invalidStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
)

func initialTUIModel() tuiModel {
	ti := textinput.New()
	ti.Placeholder = "my-agent"
	ti.CharLimit = 63
	ti.Width = 40

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	available := templates.Available()
	groups := groupByCategory(available)

	return tuiModel{
		step:      stepSelect,
		templates: available,
		groups:    groups,
		input:     ti,
		spinner:   sp,
		width:     80,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case scaffoldDoneMsg:
		m.step = stepDone
		m.err = msg.err
		return m, tea.Quit

	case spinner.TickMsg:
		if m.step == stepScaffold {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Forward to textinput if in name step.
	if m.step == stepName {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m tuiModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case stepSelect:
		return m.handleSelectKey(msg)
	case stepName:
		return m.handleNameKey(msg)
	default:
		return m, nil
	}
}

func (m tuiModel) handleSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	flat := m.flatFiltered()

	if m.filtering {
		switch msg.Type {
		case tea.KeyEscape:
			m.filtering = false
			m.filter = ""
			m.cursor = 0
			m.groups = groupByCategory(m.templates)
			return m, nil
		case tea.KeyEnter:
			m.filtering = false
			return m, nil
		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.groups = groupByCategory(filterTemplates(m.templates, m.filter))
				m.cursor = 0
			}
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				m.filter += string(msg.Runes)
				m.groups = groupByCategory(filterTemplates(m.templates, m.filter))
				m.cursor = 0
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(flat)-1 {
			m.cursor++
		}
	case "/":
		m.filtering = true
		m.filter = ""
		return m, nil
	case "enter":
		if len(flat) > 0 {
			m.chosen = flat[m.cursor]
			m.step = stepName
			m.input.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m tuiModel) handleNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.step = stepSelect
		m.input.Blur()
		m.input.Reset()
		return m, nil
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEnter:
		name := strings.TrimSpace(m.input.Value())
		if !dnsNameRe.MatchString(name) {
			return m, nil
		}
		m.name = name
		m.step = stepScaffold
		return m, tea.Batch(m.spinner.Tick, doScaffold(m.chosen.Name, m.name))
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m tuiModel) View() string {
	switch m.step {
	case stepSelect:
		return m.viewSelect()
	case stepName:
		return m.viewName()
	case stepScaffold:
		return m.viewScaffold()
	case stepDone:
		return m.viewDone()
	default:
		return ""
	}
}

func (m tuiModel) viewSelect() string {
	var b strings.Builder
	flat := m.flatFiltered()

	b.WriteString(titleStyle.Render("Volra Quickstart"))
	b.WriteString("\n\n")

	idx := 0
	for _, g := range m.groups {
		b.WriteString(categoryStyle.Render(g.Category))
		b.WriteString("\n")
		for _, t := range g.Templates {
			prefix := "  "
			name := t.Name
			desc := descStyle.Render(t.Description)
			if idx == m.cursor {
				prefix = cursorStyle.Render("▸ ")
				name = selectedStyle.Render(t.Name)
			}
			fmt.Fprintf(&b, "%s%-20s %s\n", prefix, name, desc)
			idx++
		}
		b.WriteString("\n")
	}

	if len(flat) == 0 {
		b.WriteString(dimStyle.Render("  No templates match filter.\n\n"))
	}

	// Detail panel for selected template.
	if m.cursor < len(flat) {
		sel := flat[m.cursor]
		services := "—"
		if len(sel.Services) > 0 {
			services = strings.Join(sel.Services, ", ")
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Services: %s", services)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Framework: %s", sel.Framework)))
		b.WriteString("\n")
	}

	// Help line.
	b.WriteString("\n")
	if m.filtering {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  filter: %s█  (esc clear)", m.filter)))
	} else {
		b.WriteString(helpStyle.Render("  ↑/↓ navigate · enter select · / filter · esc quit"))
	}
	b.WriteString("\n")

	return b.String()
}

func (m tuiModel) viewName() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Volra Quickstart"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "  Template: %s\n\n", selectedStyle.Render(m.chosen.Name))
	b.WriteString("  Project name (DNS-safe):\n")
	fmt.Fprintf(&b, "  %s\n\n", m.input.View())

	// Real-time DNS validation.
	val := strings.TrimSpace(m.input.Value())
	if val == "" {
		b.WriteString(dimStyle.Render("  Type a name to continue"))
	} else if dnsNameRe.MatchString(val) {
		b.WriteString(validStyle.Render("  ✓ valid name"))
	} else {
		b.WriteString(invalidStyle.Render("  ✗ must be lowercase, start with letter, a-z 0-9 hyphens only"))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  enter confirm · esc back"))
	b.WriteString("\n")

	return b.String()
}

func (m tuiModel) viewScaffold() string {
	return fmt.Sprintf("\n  %s Scaffolding %s project %q...\n",
		m.spinner.View(),
		selectedStyle.Render(m.chosen.Name),
		m.name,
	)
}

func (m tuiModel) viewDone() string {
	if m.err != nil {
		return fmt.Sprintf("\n  %s %s\n",
			invalidStyle.Render("✗"),
			m.err.Error(),
		)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s Created %s project in %s/\n\n",
		successStyle.Render("✓"),
		selectedStyle.Render(m.chosen.Name),
		m.name,
	)
	b.WriteString("  Next steps:\n")
	fmt.Fprintf(&b, "    cd %s\n", m.name)
	b.WriteString("    volra deploy\n\n")
	return b.String()
}

// --- Helpers ---

// groupByCategory sorts templates into ordered category groups.
func groupByCategory(tmpls []templates.Template) []categoryGroup {
	byCategory := make(map[string][]templates.Template)
	for _, t := range tmpls {
		byCategory[t.Category] = append(byCategory[t.Category], t)
	}

	var groups []categoryGroup
	for _, cat := range categoryOrder {
		if ts, ok := byCategory[cat]; ok && len(ts) > 0 {
			groups = append(groups, categoryGroup{Category: cat, Templates: ts})
		}
	}

	// Include any unknown categories at the end.
	known := make(map[string]bool)
	for _, c := range categoryOrder {
		known[c] = true
	}
	var extra []string
	for cat := range byCategory {
		if !known[cat] {
			extra = append(extra, cat)
		}
	}
	sort.Strings(extra)
	for _, cat := range extra {
		groups = append(groups, categoryGroup{Category: cat, Templates: byCategory[cat]})
	}

	return groups
}

// filterTemplates returns templates whose name or description contains the query (case-insensitive).
func filterTemplates(all []templates.Template, query string) []templates.Template {
	if query == "" {
		return all
	}
	q := strings.ToLower(query)
	var result []templates.Template
	for _, t := range all {
		if strings.Contains(strings.ToLower(t.Name), q) ||
			strings.Contains(strings.ToLower(t.Description), q) {
			result = append(result, t)
		}
	}
	return result
}

// flatFiltered returns the current filtered templates in display order.
func (m tuiModel) flatFiltered() []templates.Template {
	var flat []templates.Template
	for _, g := range m.groups {
		flat = append(flat, g.Templates...)
	}
	return flat
}

// doScaffold returns a tea.Cmd that performs the scaffold operation.
func doScaffold(templateName, name string) tea.Cmd {
	return func() tea.Msg {
		err := templates.Scaffold(templateName, name, name)
		return scaffoldDoneMsg{err: err}
	}
}

// runQuickstartTUI launches the Bubbletea quickstart program.
func runQuickstartTUI() error {
	p := tea.NewProgram(initialTUIModel(), tea.WithoutCatchPanics())
	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if scaffolding produced an error.
	if fm, ok := m.(tuiModel); ok && fm.err != nil {
		return fm.err
	}

	return nil
}
