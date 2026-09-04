package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wagnerlim/hakuban/internal/task"
)

// modelWithFilters builds a model on a board with a simple filter (contas) and a
// static-options filter (sprint), one column, ready for the filters modal.
func modelWithFilters(t *testing.T) *Model {
	t.Helper()
	dir := t.TempDir()
	board := `---
name: PGM
key: PGM
columns:
    - Col
filters:
    contas: 'assignee = me'
    sprint:
        options:
            Atual: 'sprint in openSprints()'
            S1: 'sprint = 1'
actions:
    Col:
        jql: 'project = X'
---
`
	if err := os.MkdirAll(filepath.Join(dir, "boards"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "boards", "acme.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	for i, id := range m.open {
		if id == "acme" {
			m.active = i
		}
	}
	m.w, m.h = 100, 24
	m.reload()
	return m
}

// Level 1 lists filter names; a simple filter toggles; drilling into an options filter and
// toggling an option persists both to use_filters.
func TestLaneFiltersDrillAndSave(t *testing.T) {
	m := modelWithFilters(t)
	m.openLaneFilters(0)

	if !reflect.DeepEqual(m.laneFilterNames, []string{"contas", "sprint"}) || m.laneFilterDrill != "" {
		t.Fatalf("nível 1 inesperado: names=%v drill=%q", m.laneFilterNames, m.laneFilterDrill)
	}
	m.toggleSimple("contas")
	m.drillFilter("sprint")
	if m.laneFilterDrill != "sprint" || len(m.laneFilterOpts) != 2 {
		t.Fatalf("drill falhou: drill=%q opts=%+v", m.laneFilterDrill, m.laneFilterOpts)
	}
	m.toggleOption("sprint", "Atual")
	if !m.optionSelected("sprint", "Atual") || m.optionSelected("sprint", "S1") {
		t.Fatal("só Atual devia estar marcado")
	}
	m.saveLaneFilters()

	got := m.store.ColumnUseFilters("acme", "Col")
	want := map[string][]string{"contas": {}, "sprint": {"Atual"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("use_filters:\n got  %v\n want %v", got, want)
	}
	res := resolveSyncFilters(m.store, "acme", "Col")
	if !reflect.DeepEqual(res, map[string][]string{"contas": {"assignee = me"}, "sprint": {"sprint in openSprints()"}}) {
		t.Fatalf("resolveSyncFilters: %v", res)
	}
}

// Untoggling the last option drops the filter key entirely.
func TestLaneFiltersUntoggleDropsKey(t *testing.T) {
	m := modelWithFilters(t)
	m.openLaneFilters(0)
	m.toggleOption("sprint", "Atual")
	m.toggleOption("sprint", "Atual")
	if _, ok := m.laneFilterSel["sprint"]; ok {
		t.Fatalf("sprint devia sumir quando vazio: %v", m.laneFilterSel)
	}
}

// A single click on a filter row selects it; a double click on a row with options
// drills in (same as enter). Rows start at modalY+6 (frame 4 + header 2).
func TestLaneFilterClickDrill(t *testing.T) {
	m := modelWithFilters(t)
	m.openLaneFilters(0)
	m.modalPlaced, m.modalX, m.modalY, m.modalW = true, 0, 0, 40
	click := tea.MouseClickMsg{X: 10, Y: 7, Button: tea.MouseLeft} // row 1 = "sprint" (has options)

	m.updateMouse(click)
	if m.laneFilterCursor != 1 || m.laneFilterDrill != "" {
		t.Fatalf("um clique devia só selecionar: cur=%d drill=%q", m.laneFilterCursor, m.laneFilterDrill)
	}
	m.updateMouse(click) // second click = double → drill
	if m.laneFilterDrill != "sprint" {
		t.Fatalf("double click devia entrar em sprint: drill=%q", m.laneFilterDrill)
	}
	if m.menuRowAt(10, 99) != -1 { // outside the rows
		t.Fatal("clique fora das linhas devia ser -1")
	}
}

// filterHasOptions distinguishes simple filters from option filters (no exec).
func TestFilterHasOptions(t *testing.T) {
	m := modelWithFilters(t)
	if filterHasOptions(m.store, "acme", "contas") {
		t.Error("contas é simples")
	}
	if !filterHasOptions(m.store, "acme", "sprint") {
		t.Error("sprint tem opções")
	}
}
