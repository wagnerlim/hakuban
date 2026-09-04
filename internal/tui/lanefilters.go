package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// The column filters modal is a two-level drill-down:
//   - level 1 (laneFilterDrill == ""): the board's filters. A simple filter is a checkbox;
//     a filter with options shows a count + "›" and drills in on →/enter.
//   - level 2 (laneFilterDrill == name): that filter's options (e.g. Atual + the sprints),
//     multi-select; ←/esc goes back to level 1.
// The selection is a working copy, persisted to the column's use_filters on close.

// openLaneFilters loads the board's filter names and opens the modal at level 1. Options
// are NOT fetched here — a dynamic filter's options_cmd runs only when you drill into it.
func (m *Model) openLaneFilters(i int) {
	if i < 0 || i >= len(m.columns) {
		return
	}
	m.laneIdx = i
	board, col := m.activeBoardID(), m.columns[i]
	m.laneFilterSel = map[string][]string{}
	for k, v := range m.store.ColumnUseFilters(board, col) {
		m.laneFilterSel[k] = append([]string(nil), v...)
	}
	m.laneFilterNames = m.store.BoardFilterNames(board)
	m.laneFilterDrill = ""
	m.laneFilterOpts = nil
	m.laneFilterCursor = 0
	m.modalPlaced = false
	m.mode = modeLaneFilters
}

// updateLaneFilters drives both levels.
func (m *Model) updateLaneFilters(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.laneFilterDrill == "" {
		return m.updateFilterList(k)
	}
	return m.updateFilterOptions(k)
}

// updateFilterList is level 1: navigate the filters; toggle a simple one; drill into one
// with options; esc saves and returns to the column menu.
func (m *Model) updateFilterList(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q":
		m.saveLaneFilters()
		m.laneCursor = 0
		m.mode = modeLaneConfig
	case "j", "down":
		if m.laneFilterCursor < len(m.laneFilterNames)-1 {
			m.laneFilterCursor++
		}
	case "k", "up":
		if m.laneFilterCursor > 0 {
			m.laneFilterCursor--
		}
	case "enter", "l", "right", " ":
		if m.laneFilterCursor >= len(m.laneFilterNames) {
			return m, nil
		}
		name := m.laneFilterNames[m.laneFilterCursor]
		if filterHasOptions(m.store, m.activeBoardID(), name) {
			if k.String() == " " { // space doesn't drill; only →/enter do
				return m, nil
			}
			m.drillFilter(name)
		} else {
			m.toggleSimple(name)
		}
	}
	return m, nil
}

// updateFilterOptions is level 2: navigate the drilled filter's options, toggle them, and
// go back to the filter list on ←/esc.
func (m *Model) updateFilterOptions(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q", "h", "left":
		m.laneFilterDrill = ""
		m.laneFilterOpts = nil
		m.laneFilterCursor = 0
	case "j", "down":
		if m.laneFilterCursor < len(m.laneFilterOpts)-1 {
			m.laneFilterCursor++
		}
	case "k", "up":
		if m.laneFilterCursor > 0 {
			m.laneFilterCursor--
		}
	case "enter", " ":
		if m.laneFilterCursor < len(m.laneFilterOpts) {
			m.toggleOption(m.laneFilterDrill, m.laneFilterOpts[m.laneFilterCursor].Label)
		}
	}
	return m, nil
}

// drillFilter enters a filter's options (runs its options_cmd if dynamic).
func (m *Model) drillFilter(name string) {
	m.laneFilterOpts = filterOptions(m.store, m.activeBoardID(), name)
	m.laneFilterDrill = name
	m.laneFilterCursor = 0
}

// toggleSimple flips a simple filter's on/off state.
func (m *Model) toggleSimple(name string) {
	if _, ok := m.laneFilterSel[name]; ok {
		delete(m.laneFilterSel, name)
	} else {
		m.laneFilterSel[name] = []string{}
	}
}

// toggleOption adds/removes an option label from a filter's selection (dropping the key
// when it goes empty).
func (m *Model) toggleOption(name, label string) {
	cur := m.laneFilterSel[name]
	for i, l := range cur {
		if l == label {
			cur = append(cur[:i], cur[i+1:]...)
			if len(cur) == 0 {
				delete(m.laneFilterSel, name)
			} else {
				m.laneFilterSel[name] = cur
			}
			return
		}
	}
	m.laneFilterSel[name] = append(cur, label)
}

// optionSelected reports whether a filter's option label is currently checked.
func (m *Model) optionSelected(name, label string) bool {
	for _, l := range m.laneFilterSel[name] {
		if l == label {
			return true
		}
	}
	return false
}

// saveLaneFilters persists the working selection to the column's use_filters.
func (m *Model) saveLaneFilters() {
	if m.laneIdx < 0 || m.laneIdx >= len(m.columns) {
		return
	}
	_ = m.store.SetColumnUseFilters(m.activeBoardID(), m.columns[m.laneIdx], m.laneFilterSel)
}

// laneFiltersBox renders the current level.
func (m *Model) laneFiltersBox() string {
	name := ""
	if m.laneIdx >= 0 && m.laneIdx < len(m.columns) {
		name = m.columns[m.laneIdx]
	}
	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
	var b strings.Builder

	if m.laneFilterDrill != "" { // level 2: the drilled filter's options
		b.WriteString(faint.Render(m.laneFilterDrill) + "\n\n")
		if len(m.laneFilterOpts) == 0 {
			b.WriteString(faint.Render(msg.noFilters))
		}
		for i, o := range m.laneFilterOpts {
			mark := "○"
			if m.optionSelected(m.laneFilterDrill, o.Label) {
				mark = "◉"
			}
			line := fmt.Sprintf("%s %s", mark, o.Label)
			if i == m.laneFilterCursor {
				line = accent.Render("› " + line)
			} else {
				line = "  " + line
			}
			b.WriteString(line + "\n")
		}
		return modalBox(msg.tLaneFilters, strings.TrimRight(b.String(), "\n"), msg.hLaneFilters)
	}

	// level 1: the board's filters
	b.WriteString(faint.Render(msg.laneLabel+" "+name) + "\n\n")
	if len(m.laneFilterNames) == 0 {
		b.WriteString(faint.Render(msg.noFilters))
	}
	for i, fn := range m.laneFilterNames {
		var line string
		if filterHasOptions(m.store, m.activeBoardID(), fn) {
			n := len(m.laneFilterSel[fn])
			suffix := " ›"
			if n > 0 {
				suffix = fmt.Sprintf(" (%d) ›", n)
			}
			line = fn + suffix
		} else {
			mark := "○"
			if _, ok := m.laneFilterSel[fn]; ok {
				mark = "◉"
			}
			line = mark + " " + fn
		}
		if i == m.laneFilterCursor {
			line = accent.Render("› " + line)
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	return modalBox(msg.tLaneFilters, strings.TrimRight(b.String(), "\n"), msg.hLaneList)
}
