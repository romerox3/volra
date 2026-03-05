package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/romerox3/volra/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestModel() tuiModel {
	return initialTUIModel()
}

func sendKey(m tuiModel, key string) tuiModel {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated.(tuiModel)
}

func sendSpecial(m tuiModel, keyType tea.KeyType) tuiModel {
	updated, _ := m.Update(tea.KeyMsg{Type: keyType})
	return updated.(tuiModel)
}

func TestTUI_InitialState(t *testing.T) {
	m := newTestModel()

	assert.Equal(t, stepSelect, m.step)
	assert.Equal(t, 0, m.cursor)
	assert.False(t, m.filtering)
	assert.NotEmpty(t, m.templates)
	assert.NotEmpty(t, m.groups)
}

func TestTUI_NavigateDown(t *testing.T) {
	m := newTestModel()
	assert.Equal(t, 0, m.cursor)

	m = sendKey(m, "j")
	assert.Equal(t, 1, m.cursor)

	m = sendKey(m, "j")
	assert.Equal(t, 2, m.cursor)
}

func TestTUI_NavigateUp(t *testing.T) {
	m := newTestModel()
	m = sendKey(m, "j")
	m = sendKey(m, "j")
	assert.Equal(t, 2, m.cursor)

	m = sendKey(m, "k")
	assert.Equal(t, 1, m.cursor)
}

func TestTUI_NavigateUpAtTop(t *testing.T) {
	m := newTestModel()
	assert.Equal(t, 0, m.cursor)

	m = sendKey(m, "k")
	assert.Equal(t, 0, m.cursor, "cursor should not go below 0")
}

func TestTUI_NavigateDownAtBottom(t *testing.T) {
	m := newTestModel()
	total := len(m.flatFiltered())

	for i := 0; i < total+5; i++ {
		m = sendKey(m, "j")
	}
	assert.Equal(t, total-1, m.cursor, "cursor should not exceed template count")
}

func TestTUI_EnterSelectsTemplate(t *testing.T) {
	m := newTestModel()
	flat := m.flatFiltered()
	require.NotEmpty(t, flat)

	m = sendSpecial(m, tea.KeyEnter)

	assert.Equal(t, stepName, m.step)
	assert.Equal(t, flat[0].Name, m.chosen.Name)
}

func TestTUI_EnterSecondTemplate(t *testing.T) {
	m := newTestModel()
	flat := m.flatFiltered()
	require.True(t, len(flat) > 1)

	m = sendKey(m, "j")
	m = sendSpecial(m, tea.KeyEnter)

	assert.Equal(t, stepName, m.step)
	assert.Equal(t, flat[1].Name, m.chosen.Name)
}

func TestTUI_EscFromNameGoesBack(t *testing.T) {
	m := newTestModel()
	m = sendSpecial(m, tea.KeyEnter) // → stepName
	assert.Equal(t, stepName, m.step)

	m = sendSpecial(m, tea.KeyEscape) // → back to stepSelect
	assert.Equal(t, stepSelect, m.step)
}

func TestTUI_FilterMode(t *testing.T) {
	m := newTestModel()

	// Enter filter mode.
	m = sendKey(m, "/")
	assert.True(t, m.filtering)

	// Type "rag".
	for _, c := range "rag" {
		m = sendKey(m, string(c))
	}
	assert.Equal(t, "rag", m.filter)

	flat := m.flatFiltered()
	for _, tmpl := range flat {
		assert.Contains(t, tmpl.Name+tmpl.Description, "rag",
			"filtered templates should contain 'rag'")
	}
}

func TestTUI_FilterEscClears(t *testing.T) {
	m := newTestModel()
	totalBefore := len(m.flatFiltered())

	m = sendKey(m, "/")
	m = sendKey(m, "x")
	m = sendKey(m, "x")
	m = sendKey(m, "x")
	assert.True(t, m.filtering)

	m = sendSpecial(m, tea.KeyEscape)
	assert.False(t, m.filtering)
	assert.Equal(t, "", m.filter)
	assert.Equal(t, totalBefore, len(m.flatFiltered()))
}

func TestTUI_DNSValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid simple", "my-agent", true},
		{"valid short", "a", true},
		{"starts with number", "1abc", false},
		{"uppercase", "MyAgent", false},
		{"underscore", "my_agent", false},
		{"empty", "", false},
		{"hyphen start", "-agent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, dnsNameRe.MatchString(tt.input))
		})
	}
}

func TestTUI_ViewSelectRenders(t *testing.T) {
	m := newTestModel()
	view := m.viewSelect()

	assert.Contains(t, view, "Volra Quickstart")
	assert.Contains(t, view, "Getting Started")
	assert.Contains(t, view, "basic")
	assert.Contains(t, view, "▸")
}

func TestTUI_ViewNameRenders(t *testing.T) {
	m := newTestModel()
	m.step = stepName
	m.chosen = templates.Template{Name: "basic"}
	view := m.viewName()

	assert.Contains(t, view, "Template: basic")
	assert.Contains(t, view, "Project name")
}

func TestTUI_ViewDoneSuccess(t *testing.T) {
	m := newTestModel()
	m.step = stepDone
	m.chosen = templates.Template{Name: "rag"}
	m.name = "test-project"
	view := m.viewDone()

	assert.Contains(t, view, "✓")
	assert.Contains(t, view, "rag")
	assert.Contains(t, view, "test-project")
	assert.Contains(t, view, "volra deploy")
}

func TestTUI_ViewDoneError(t *testing.T) {
	m := newTestModel()
	m.step = stepDone
	m.err = assert.AnError
	view := m.viewDone()

	assert.Contains(t, view, "✗")
}

func TestTUI_GroupByCategory(t *testing.T) {
	groups := groupByCategory(templates.Available())

	assert.NotEmpty(t, groups)
	assert.Equal(t, templates.CategoryGettingStarted, groups[0].Category)
}

func TestTUI_FilterTemplates(t *testing.T) {
	all := templates.Available()

	result := filterTemplates(all, "rag")
	assert.NotEmpty(t, result)
	for _, tmpl := range result {
		lower := tmpl.Name + " " + tmpl.Description
		assert.Contains(t, lower, "rag")
	}

	result = filterTemplates(all, "")
	assert.Equal(t, len(all), len(result))
}

func TestTUI_ScaffoldDoneMsg(t *testing.T) {
	m := newTestModel()
	m.step = stepScaffold

	updated, _ := m.Update(scaffoldDoneMsg{err: nil})
	fm := updated.(tuiModel)

	assert.Equal(t, stepDone, fm.step)
	assert.Nil(t, fm.err)
}
