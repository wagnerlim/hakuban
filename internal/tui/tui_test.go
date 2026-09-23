package tui

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/wagnerlim/hakuban/internal/task"
)

// pointerShape: dragging → closed hand, focused input → I-beam, card hover →
// open hand, otherwise default. Dragging beats a focused input.
func TestPointerShape(t *testing.T) {
	s, err := task.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	if got := m.pointerShape(); got != ptrDefault {
		t.Fatalf("clean state: want %q, got %q", ptrDefault, got)
	}
	m.input.Focus()
	if got := m.pointerShape(); got != ptrText {
		t.Fatalf("focused input: want %q, got %q", ptrText, got)
	}
	m.draggingCard = true // dragging beats the focused input
	if got := m.pointerShape(); got != ptrGrabbing {
		t.Fatalf("dragging: want %q, got %q", ptrGrabbing, got)
	}

	// hovering a card on the board → open hand; off the card → default.
	m.draggingCard = false
	m.input.Blur()
	m.mode = modeBoard
	m.colW = 20
	m.cardRegions = []cardHit{{col: 0, y0: 5, y1: 8}}
	m.mouseX, m.mouseY = 3, 6
	if got := m.pointerShape(); got != ptrGrab {
		t.Fatalf("hover card: want %q, got %q", ptrGrab, got)
	}
	m.mouseY = 20 // below the card
	if got := m.pointerShape(); got != ptrDefault {
		t.Fatalf("off card: want %q, got %q", ptrDefault, got)
	}
}

// cycle: wraps both directions and a value outside the set starts from the first.
func TestCycle(t *testing.T) {
	opts := []string{"auto", "dark", "light"}
	cases := []struct {
		cur  string
		dir  int
		want string
	}{
		{"auto", 1, "dark"},
		{"light", 1, "auto"},  // wrap forward
		{"auto", -1, "light"}, // wrap backward
		{"xxx", 1, "dark"},    // outside the set → i=0, +1
	}
	for _, c := range cases {
		if got := cycle(opts, c.cur, c.dir); got != c.want {
			t.Errorf("cycle(%q, %d) = %q, quer %q", c.cur, c.dir, got, c.want)
		}
	}
}

// truncate cuts by rune (not byte) and only adds an ellipsis when it overflows.
func TestTruncate(t *testing.T) {
	if got := truncate("café", 10); got != "café" {
		t.Errorf("sem corte: %q", got)
	}
	if got := truncate("café", 3); got != "ca…" {
		t.Errorf("corte por runa: %q", got)
	}
}

// Settings menu + theme picker: 's' opens, 'j' descends to the theme, enter opens
// the list, 'j' moves (live preview) and enter selects — persisting to config.yml
// (rereads from the store to confirm it saved, not just changed in memory).
func TestThemePickerSelects(t *testing.T) {
	s, err := task.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	key := func(k string) { m.Update(tea.KeyPressMsg{Text: k, Code: []rune(k)[0]}) }
	enter := func() { m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) }

	m.prefixArmed = true // 's' is a command → requires the prefix armed
	key("s")
	key("j") // cursor on "Theme"
	enter()  // opens the picker
	if m.mode != modePicker {
		t.Fatalf("enter on the theme did not open the picker: mode=%d", m.mode)
	}
	// cursor starts on the current theme (themeNames[0]); descend 1 and select
	key("j")
	if m.cfg.Theme == themeNames[1] {
		t.Fatal("the preview should not have saved before enter")
	}
	enter() // select
	if m.mode != modeSettings {
		t.Fatalf("enter on the picker did not go back to settings: mode=%d", m.mode)
	}
	if m.cfg.Theme != themeNames[1] {
		t.Fatalf("theme not selected: %q", m.cfg.Theme)
	}
	if got := s.LoadConfig(); got.Theme != themeNames[1] {
		t.Fatalf("theme did not persist to disk: %q", got.Theme)
	}
}

// Preview pane on: the whole board still fits in h lines and the pane shows the
// title of the card under the cursor. Reinforces the height math
// (bodyH = h-7-paneH) and the previewPane wiring.
func TestPreviewPane(t *testing.T) {
	s, err := task.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&task.Task{Title: "comprar café", Status: task.StatusBacklog, Notes: "moído"}); err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.w, m.h = 110, 30
	m.cfg.PreviewPane = true
	m.reload()

	out := m.boardView()
	if got := len(strings.Split(out, "\n")); got != m.h {
		t.Fatalf("the board with the panel does not fit in h: %d lines, want %d", got, m.h)
	}
	if !strings.Contains(out, "comprar café") {
		t.Fatal("the panel did not show the selected card's title")
	}
}

// clipLines cuts at n lines and marks the cut (doesn't overflow the pane height).
func TestClipLines(t *testing.T) {
	in := "a\nb\nc\nd\ne"
	got := strings.Split(clipLines(in, 3), "\n")
	if len(got) != 3 {
		t.Fatalf("clipping to 3 gave %d lines", len(got))
	}
	if !strings.Contains(got[2], "enter") {
		t.Fatalf("última linha devia marcar o corte, veio %q", got[2])
	}
	if clipLines("a\nb", 5) != "a\nb" {
		t.Fatal("content shorter than n should not change")
	}
}

// Language picker: same flow as the theme ("Language" row → enter opens list →
// select), and '<' goes back to settings. Guards the langRow index and the lang commit.
func TestLangPickerSelects(t *testing.T) {
	s, err := task.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	key := func(k string) { m.Update(tea.KeyPressMsg{Text: k, Code: []rune(k)[0]}) }
	enter := func() { m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) }

	m.prefixArmed = true // 's' is a command → requires the prefix armed
	key("s")
	key("j")
	key("j")
	key("j") // cursor on the "Language" row (index 3)
	enter()  // opens the picker
	if m.mode != modePicker {
		t.Fatalf("enter on the language did not open the picker: mode=%d", m.mode)
	}
	key("j") // move to the next language (preview)
	key("<") // go back without selecting → language unchanged
	if m.mode != modeSettings {
		t.Fatalf("'<' did not go back to settings: mode=%d", m.mode)
	}
	if m.cfg.Lang != langNames[0] {
		t.Fatalf("'<' should not have switched the language: %q", m.cfg.Lang)
	}
	// reopen and actually select
	enter()
	key("j")
	enter()
	if m.cfg.Lang != langNames[1] {
		t.Fatalf("language not selected: %q", m.cfg.Lang)
	}
	if got := s.LoadConfig(); got.Lang != langNames[1] {
		t.Fatalf("language did not persist: %q", got.Lang)
	}
}

// Change the data dir via the UI: settings → browser → "use this folder" (empty
// destination) → confirm move → data moved and store repoints. HOME redirected
// to tmp keeps the pointer (root.yml) out of the real config.
func TestDataDirMove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	from := t.TempDir()
	s, err := task.Open(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&task.Task{Title: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveConfig(task.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	m := New(s)
	enter := func() { m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) }

	to := filepath.Join(t.TempDir(), "vault") // doesn't exist yet → no collision
	m.setCursor = dirRow
	m.openDirBrowser()
	if m.mode != modeDirBrowser {
		t.Fatalf("the browser did not open: mode=%d", m.mode)
	}
	m.browsePath, m.browseCursor = to, 0 // pretends it navigated to the destination, cursor on "use this"
	enter()                              // select → asks for confirmation
	if m.mode != modeConfirmMove {
		t.Fatalf("it did not ask to confirm the move: mode=%d", m.mode)
	}
	enter() // confirm move
	if m.store.Dir() != to {
		t.Fatalf("the store did not repoint: %q", m.store.Dir())
	}
	if !task.HasData(to) || task.HasData(from) {
		t.Fatalf("the data did not migrate: to=%v from=%v", task.HasData(to), task.HasData(from))
	}
}

// File watch: an external process creates a .md → the tick detects the signature
// change and refresh() brings in the new card without reopening the app.
func TestLiveReload(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.w, m.h = 110, 30
	m.reload()
	before := len(m.cols[0]) // backlog

	// external "AI": another store writes a card in the same dir
	ext, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ext.Save(&task.Task{Title: "externo", Status: task.StatusBacklog}); err != nil {
		t.Fatal(err)
	}

	// signature changed → the tick handler reloads
	if dirSig(dir) == m.lastSig {
		t.Fatal("assinatura devia ter mudado após o write externo")
	}
	m.Update(reloadTickMsg{})
	if got := len(m.cols[0]); got != before+1 {
		t.Fatalf("the external card did not show up: before=%d after=%d", before, got)
	}
}

// Browser filter: '/' enters the mode, typing filters (case-insensitive substring),
// esc clears. filteredDirs is the heart of the feature.
func TestBrowseFilter(t *testing.T) {
	m := &Model{browseDirs: []string{"Documents", "Downloads", "Music", "dev"}}

	m.browseFilter = "do"
	if got := m.filteredDirs(); len(got) != 2 { // Documents, Downloads
		t.Fatalf("filtro 'do' → %v", got)
	}
	m.browseFilter = ""
	if len(m.filteredDirs()) != 4 {
		t.Fatal("sem filtro deve devolver tudo")
	}

	// key flow: '/' activates, 'd' types, esc clears
	m.updateDirBrowser(tea.KeyPressMsg{Text: "/", Code: '/'})
	if !m.browseFiltering {
		t.Fatal("'/' did not activate the filter")
	}
	m.updateBrowseFilter(tea.KeyPressMsg{Text: "d", Code: 'd'})
	if m.browseFilter != "d" || len(m.filteredDirs()) != 3 { // Documents, Downloads, dev
		t.Fatalf("digitar 'd' → filtro=%q dirs=%v", m.browseFilter, m.filteredDirs())
	}
	m.updateBrowseFilter(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.browseFilter != "" || m.browseFiltering {
		t.Fatal("esc did not clear the filter")
	}
}

// Clicks on the tab bar: left switches, middle closes, + opens the modal. Covers
// the per-column routing and the index adjustment when closing a tab.
func TestMouseTabs(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveBoard(&task.Board{ID: "work", Name: "Work"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveBoard(&task.Board{ID: "home", Name: "Home"}); err != nil {
		t.Fatal(err)
	}
	m := New(s) // open = [home, work] (no Inbox)
	m.w, m.h = 100, 20

	region := func(match func(idx int, id string) bool) clickRegion {
		t.Helper()
		m.render() // repopulates the hitboxes with the current layout
		for _, r := range m.tabRegions {
			if r.kind == "new" && match(-1, "+") {
				return r
			}
			if r.kind == "tab" && match(r.idx, m.open[r.idx]) {
				return r
			}
		}
		t.Fatal("region not found")
		return clickRegion{}
	}
	click := func(r clickRegion, b tea.MouseButton) {
		m.Update(tea.MouseClickMsg{X: r.x0 + 1, Y: 0, Button: b})
	}

	click(region(func(_ int, id string) bool { return id == "work" }), tea.MouseLeft)
	if m.activeBoardID() != "work" {
		t.Fatalf("left did not activate work: active=%q", m.activeBoardID())
	}

	n := len(m.open)
	click(region(func(_ int, id string) bool { return id == "home" }), tea.MouseMiddle)
	if len(m.open) != n-1 || m.isOpen("home") {
		t.Fatalf("middle did not close home: open=%v", m.open)
	}

	click(region(func(idx int, _ string) bool { return idx == -1 }), tea.MouseLeft)
	if m.mode != modeNewBoard {
		t.Fatalf("clicking + did not open a new board: mode=%d", m.mode)
	}
}

// Dragging a card to another column changes the status and persists to disk.
func TestCardDrag(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&task.Task{Title: "x", Status: task.StatusBacklog}); err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.w, m.h = 120, 24
	m.render() // populates colW and cardRegions

	if len(m.cardRegions) == 0 {
		t.Fatal("nenhuma hitbox de card")
	}
	r := m.cardRegions[0] // card in the backlog (col 0)
	id := r.id
	m.Update(tea.MouseClickMsg{X: r.col*m.colW + 2, Y: r.y0, Button: tea.MouseLeft})
	if !m.draggingCard || m.dragCardID != id {
		t.Fatalf("the card was not grabbed: dragging=%v id=%q", m.draggingCard, m.dragCardID)
	}
	doingX := 1*m.colW + 2 // doing column
	m.Update(tea.MouseMotionMsg{X: doingX, Y: 8, Button: tea.MouseLeft})
	if m.dropCol != 1 {
		t.Fatalf("dropCol=%d, esperado 1 (doing)", m.dropCol)
	}
	m.Update(tea.MouseReleaseMsg{X: doingX, Y: 8})
	if m.draggingCard {
		t.Fatal("release did not end the drag")
	}
	s2, err := task.Open(dir) // reopen from disk
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.All()[0].Status; got != task.StatusDoing {
		t.Fatalf("status no disco: quero doing, veio %q", got)
	}
}

// Closed tabs must stay closed on reopen (persistence in state.yml).
func TestTabsPersist(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveBoard(&task.Board{ID: "work", Name: "Work"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveBoard(&task.Board{ID: "home", Name: "Home"}); err != nil {
		t.Fatal(err)
	}
	m := New(s) // open = [home, work] (no Inbox)
	for i, id := range m.open {
		if id == "home" {
			m.closeTabAt(i)
		}
	}
	if m.isOpen("home") {
		t.Fatal("home devia estar fechada")
	}

	s2, err := task.Open(dir) // reopen the app from scratch
	if err != nil {
		t.Fatal(err)
	}
	m2 := New(s2)
	if m2.isOpen("home") {
		t.Fatalf("persistência falhou: home reabriu; open=%v", m2.open)
	}
	if !m2.isOpen("work") {
		t.Fatalf("work devia ter continuado aberta; open=%v", m2.open)
	}
}

// ✕ buttons: close the tab (on the bar) and the modal (title corner).
func TestMouseCloseButtons(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveBoard(&task.Board{ID: "work", Name: "Work"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveBoard(&task.Board{ID: "home", Name: "Home"}); err != nil {
		t.Fatal(err)
	}
	m := New(s) // open = [home, work] (≥2 tabs → ✕ visible)
	m.w, m.h = 100, 24
	m.render()

	var closeR clickRegion
	found := false
	for _, r := range m.tabRegions {
		if r.kind == "close" && m.open[r.idx] == "work" {
			closeR, found = r, true
		}
	}
	if !found {
		t.Fatal("the 'work' tab has no close region (✕)")
	}
	n := len(m.open)
	m.Update(tea.MouseClickMsg{X: closeR.x0, Y: 0, Button: tea.MouseLeft})
	if len(m.open) != n-1 || m.isOpen("work") {
		t.Fatalf("the tab's ✕ did not close it: open=%v", m.open)
	}

	m.startNewBoard()
	m.render()
	// red dot: on the top border, after the gap (modalX+3, modalY)
	m.Update(tea.MouseClickMsg{X: m.modalX + 3, Y: m.modalY, Button: tea.MouseLeft})
	if m.mode != modeBoard {
		t.Fatalf("the red dot did not close the modal: mode=%d", m.mode)
	}
}

// Dragging the modal: grab on the title → move → drop. And a click outside the
// title doesn't start a drag.
func TestMouseDrag(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.w, m.h = 100, 30
	m.startNewBoard()
	m.render() // centraliza e define modalX/Y/W/H

	x0, y0 := m.modalX, m.modalY
	gx, gy := x0+m.modalW/2, y0+2                                    // title bar, away from the dots
	m.Update(tea.MouseClickMsg{X: gx, Y: gy, Button: tea.MouseLeft}) // grab
	if !m.dragging {
		t.Fatal("grabbing the title bar did not start the drag")
	}
	m.Update(tea.MouseMotionMsg{X: gx + 5, Y: gy + 3, Button: tea.MouseLeft}) // move +5,+3
	if m.modalX != x0+5 || m.modalY != y0+3 {
		t.Fatalf("the modal did not follow: (%d,%d), expected (%d,%d)", m.modalX, m.modalY, x0+5, y0+3)
	}
	m.Update(tea.MouseReleaseMsg{X: x0 + 7, Y: y0 + 3})
	if m.dragging {
		t.Fatal("release did not stop the drag")
	}

	m.render()
	m.Update(tea.MouseClickMsg{X: m.modalX + 2, Y: m.modalY + m.modalH - 1, Button: tea.MouseLeft})
	if m.dragging {
		t.Fatal("a click outside the title (the footer) should not start a drag")
	}
}

// While in the detail, esc must go back to the board (regression of "won't exit").
func TestDetailEscExits(t *testing.T) {
	dir := t.TempDir()
	s, _ := task.Open(dir)
	if err := s.Save(&task.Task{Title: "x"}); err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.mode = modeDetail
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.mode != modeBoard {
		t.Fatalf("esc devia voltar pro board, mode=%d", m.mode)
	}
}

// groupByStatus must drop each task in the right column and send unknown/empty
// status to the backlog (otherwise the card would vanish from the screen).
func TestGroupByStatus(t *testing.T) {
	ts := []*task.Task{
		{ID: "a", Status: task.StatusBacklog},
		{ID: "b", Status: task.StatusDoing},
		{ID: "c", Status: task.StatusDone},
		{ID: "d", Status: "lixo"}, // unknown → backlog
		{ID: "e", Status: ""},     // empty → backlog
	}
	cols := groupByStatus(task.DefaultColumns, ts)
	if len(cols[0]) != 3 {
		t.Errorf("backlog: quero 3 (a,d,e), veio %d", len(cols[0]))
	}
	if len(cols[1]) != 1 || cols[1][0].ID != "b" {
		t.Errorf("doing errado: %v", cols[1])
	}
	if len(cols[2]) != 1 || cols[2][0].ID != "c" {
		t.Errorf("done errado: %v", cols[2])
	}
}

// recomputeFilter matches the query on the title OR the id (case-insensitive), and
// focusCard puts the cursor on the right card. Covers the example TESTE-2 → TESTE-2, TESTE-21.
func TestFilter(t *testing.T) {
	s, err := task.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.cols = [][]*task.Task{
		{{ID: "TESTE-2", Title: "alpha"}, {ID: "TESTE-21", Title: "beta"}},
		{{ID: "CASA-1", Title: "comprar teste"}},
	}
	cases := []struct {
		q    string
		want int
	}{
		{"", 3},        // empty lists everything
		{"TESTE-2", 2}, // id: TESTE-2 and TESTE-21 (not CASA-1)
		{"teste", 3},   // matches the TESTE-* ids and the title "comprar teste"
		{"alpha", 1},   // title only
		{"inexistente", 0},
	}
	for _, c := range cases {
		m.input.SetValue(c.q)
		m.recomputeFilter()
		if len(m.filterHits) != c.want {
			t.Errorf("query %q: quero %d hits, veio %d", c.q, c.want, len(m.filterHits))
		}
	}

	m.focusCard("TESTE-21")
	if m.col != 0 || m.row[0] != 1 {
		t.Errorf("focusCard(TESTE-21): quero col=0 row=1, veio col=%d row=%d", m.col, m.row[0])
	}
}

// Compound filter in the same overlay: picking the tag in the dropdown (tab → ↓ → enter)
// narrows the cards, and the text refines by title WITHIN the tag (AND).
func TestFilterTagCompound(t *testing.T) {
	s, err := task.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.cfg.Tags = []task.TagDef{{Name: "FRONTEND"}, {Name: "BACKEND"}}
	m.cols = [][]*task.Task{{
		{ID: "A-1", Title: "login", Tags: []string{"FRONTEND"}},
		{ID: "A-2", Title: "login backend", Tags: []string{"BACKEND"}},
		{ID: "A-3", Title: "logout", Tags: []string{"FRONTEND"}},
	}}
	m.startFilter()
	if len(m.filterHits) != 3 {
		t.Fatalf("sem filtro devia listar 3, veio %d", len(m.filterHits))
	}

	m.updateFilter(tea.KeyPressMsg{Code: tea.KeyTab}) // opens the dropdown
	if !m.filterDropOpen {
		t.Fatal("tab did not open the dropdown")
	}
	m.updateFilter(tea.KeyPressMsg{Code: tea.KeyDown})  // all → FRONTEND
	m.updateFilter(tea.KeyPressMsg{Code: tea.KeyEnter}) // select
	if m.filterTag != "FRONTEND" || m.filterDropOpen || len(m.filterHits) != 2 {
		t.Fatalf("escolher FRONTEND: tag=%q aberto=%v hits=%d", m.filterTag, m.filterDropOpen, len(m.filterHits))
	}

	// refine by title within the tag
	m.input.SetValue("out")
	m.recomputeFilter()
	if len(m.filterHits) != 1 || m.filterHits[0].t.ID != "A-3" {
		t.Fatalf("tag FRONTEND + texto 'out' devia dar só A-3, veio %d", len(m.filterHits))
	}

	// going back to "all" clears the tag filter
	m.updateFilter(tea.KeyPressMsg{Code: tea.KeyTab})
	m.updateFilter(tea.KeyPressMsg{Code: tea.KeyUp}) // FRONTEND → all
	m.updateFilter(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.input.SetValue("")
	m.recomputeFilter()
	if m.filterTag != "" || len(m.filterHits) != 3 {
		t.Fatalf("back to all: tag=%q hits=%d", m.filterTag, len(m.filterHits))
	}
}

// Keymap: rebind persists only the override, conflict and reserved key don't save,
// and going back to default clears the config.
func TestKeymap(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	if m.actionFor("a") != kaAdd || m.actionFor("/") != kaFind {
		t.Fatalf("defaults errados: a=%q /=%q", m.actionFor("a"), m.actionFor("/"))
	}

	reopen := func() task.Config {
		s2, err := task.Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		return s2.LoadConfig()
	}

	m.rebind(kaAdd, "n") // remaps new task to 'n'
	if m.keys[kaAdd] != "n" || m.actionFor("n") != kaAdd {
		t.Fatalf("the rebind did not apply: %q", m.keys[kaAdd])
	}
	if got := reopen().Keys["add"]; got != "n" {
		t.Fatalf("the override did not persist to disk: %q", got)
	}

	m.rebind(kaFind, "n") // 'n' already belongs to add → conflict, doesn't save
	if m.keys[kaFind] != "/" || m.keymapConflict != kaAdd {
		t.Fatalf("the conflict was not blocked: find=%q conflict=%q", m.keys[kaFind], m.keymapConflict)
	}

	m.rebind(kaAdd, "esc") // reserved → doesn't save
	if m.keys[kaAdd] != "n" || m.keymapConflict != "reserved" {
		t.Fatalf("the reserved key was not blocked: add=%q conflict=%q", m.keys[kaAdd], m.keymapConflict)
	}

	m.rebind(kaAdd, defaultKeyFor(kaAdd)) // back to default → config cleared
	if m.keys[kaAdd] != "a" {
		t.Fatalf("reset did not go back to the default: %q", m.keys[kaAdd])
	}
	if got := reopen().Keys; len(got) != 0 {
		t.Fatalf("the config should be left with no overrides, got %v", got)
	}

	// render smoke test: open modal + capturing must not panic and show the label
	m.startKeymap()
	if out := m.keymapBox(); !strings.Contains(out, msg.keyLabels[kaAdd]) {
		t.Fatal("the shortcuts modal did not render the first action's label")
	}
	m.keymapCapturing = true
	if out := m.keymapBox(); !strings.Contains(out, msg.keymapPress) {
		t.Fatal("modal capturando devia mostrar o prompt de tecla")
	}
}

// Tag catalog: add with case-insensitive dedup, color cycle, delete, and the
// color resolution (catalog = suggestion; outside it falls back to default). Persists everything.
func TestTags(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	press := func(k string) { m.updateTags(tea.KeyPressMsg{Text: k, Code: []rune(k)[0]}) }
	reopen := func() task.Config {
		s2, err := task.Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		return s2.LoadConfig()
	}

	// create "urgente" via the form, cycling a preset (mauve→blue) in the color field
	m.openTagForm(-1)
	m.input.SetValue("urgente")
	m.updateTagForm(tea.KeyPressMsg{Code: tea.KeyTab})   // focus the color field (comes as "mauve")
	m.updateTagForm(tea.KeyPressMsg{Code: tea.KeyRight}) // mauve → blue
	m.updateTagForm(enter)
	if len(m.cfg.Tags) != 1 || m.cfg.Tags[0].Color != "blue" {
		t.Fatalf("creation through the form failed: %+v", m.cfg.Tags)
	}
	if got := reopen().Tags; len(got) != 1 || got[0].Name != "urgente" || got[0].Color != "blue" {
		t.Fatalf("the tag did not persist: %+v", got)
	}

	// dedup: "URGENTE" (different case) is ignored
	m.openTagForm(-1)
	m.input.SetValue("URGENTE")
	m.updateTagForm(enter)
	if len(m.cfg.Tags) != 1 {
		t.Fatalf("dedup falhou, veio %d tags", len(m.cfg.Tags))
	}

	// edit: prefill selects the color swatch (blue), no hex in the custom slot
	m.tagCursor = 0
	m.openTagForm(0)
	if m.input.Value() != "urgente" || m.tagColorSel != hueIndex("blue") || m.colorInput.Value() != "" {
		t.Fatalf("editing did not prefill: name=%q sel=%d hex=%q", m.input.Value(), m.tagColorSel, m.colorInput.Value())
	}
	// typing in the color field jumps to the hex slot; complete it and save along with the description
	m.tagFieldFocus(1)
	m.updateTagForm(tea.KeyPressMsg{Text: "#", Code: '#'})
	if m.tagColorSel != len(tagHues) {
		t.Fatalf("digitar no campo cor devia saltar pro slot hex, sel=%d", m.tagColorSel)
	}
	m.colorInput.SetValue("#ff8800")
	m.descInput.SetValue("prioridade máxima")
	m.updateTagForm(enter)
	if td := m.cfg.Tags[0]; td.Color != "#ff8800" || td.Desc != "prioridade máxima" || td.Name != "urgente" {
		t.Fatalf("editing did not save correctly: %+v", td)
	}

	// color resolution: hex passes through; hue matches; outside the catalog falls back to mauve
	if hueColor("#ff8800") != "#ff8800" || !isHexColor("#ff8800") || isHexColor("blue") {
		t.Fatal("the hex did not resolve correctly")
	}
	if tagColor("URGENTE", m.cfg.Tags) != "#ff8800" {
		t.Fatal("tagColor devia casar case-insensitive")
	}
	if tagColor("qualquer", m.cfg.Tags) != "" || hueColor("") != pal.mauve {
		t.Fatal("tag fora do catálogo deve cair no default mauve")
	}

	// render smoke test: list and form don't panic and show the content
	m.startTags()
	if !strings.Contains(m.tagsBox(), "urgente") {
		t.Fatal("tagsBox did not render the tag")
	}
	m.openTagForm(0)
	if !strings.Contains(m.tagFormBox(), msg.fTagName) {
		t.Fatal("tagFormBox did not render the Name field")
	}

	// delete empties the catalog and clears the config
	press("d")
	if len(m.cfg.Tags) != 0 {
		t.Fatalf("delete falhou, sobrou %d", len(m.cfg.Tags))
	}
	if !strings.Contains(m.tagsBox(), msg.tagsEmpty) {
		t.Fatal("tagsBox vazio devia mostrar o empty-state")
	}
	if got := reopen().Tags; len(got) != 0 {
		t.Fatalf("the config should be left with no tags, got %+v", got)
	}
}

// When a card's tags overflow the width and wrap to the 2nd line, the pill on
// the bottom line must start at the same column as the one above (the box can't
// trim its left padding). Regression of the reported misalignment.
func TestCardTagWrapAlign(t *testing.T) {
	cat := []task.TagDef{
		{Name: "FRONTEND", Color: "blue"},
		{Name: "BACKEND", Color: "green"},
		{Name: "INFRA", Color: "mauve"},
	}
	tk := &task.Task{ID: "X-1", Title: "t", Tags: []string{"FRONTEND", "BACKEND", "INFRA"}}
	// contentW=20: FRONTEND+BACKEND fit on the 1st line, INFRA wraps to the 2nd
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	clean := ansi.ReplaceAllString(renderCard(tk, false, 20, "", cat), "")

	var frontLine, infraLine string
	for _, ln := range strings.Split(clean, "\n") {
		if strings.Contains(ln, "FRONTEND") {
			frontLine = ln
		} else if strings.Contains(ln, "INFRA") {
			infraLine = ln
		}
	}
	if frontLine == "" || infraLine == "" {
		t.Fatalf("the tags did not wrap into 2 lines:\n%s", clean)
	}
	if a, b := strings.Index(frontLine, "FRONTEND"), strings.Index(infraLine, "INFRA"); a != b {
		t.Errorf("pílula desalinhada na 2ª linha: FRONTEND col %d, INFRA col %d", a, b)
	}
}

// Assign a tag to a card via the TUI: the checklist toggles the catalog tag on the
// card, persists to the .md, and tags outside the catalog show up so they can be removed.
func TestCardTags(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&task.Task{Title: "x", Status: task.StatusBacklog}); err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.cfg.Tags = []task.TagDef{{Name: "FRONTEND", Color: "blue"}, {Name: "BACKEND", Color: "green"}}
	cur := m.current()
	if cur == nil {
		t.Fatal("sem card selecionado")
	}
	id := cur.ID

	m.startCardTags()
	if m.mode != modeCardTags || m.cardTagID != id {
		t.Fatalf("startCardTags: mode=%d id=%q", m.mode, m.cardTagID)
	}
	if rows := m.cardTagRows(); len(rows) != 2 {
		t.Fatalf("rows do catálogo: %v", rows)
	}

	toggle := tea.KeyPressMsg{Code: tea.KeyEnter} // enter also toggles
	m.updateCardTags(toggle)                      // checks FRONTEND (cursor 0)
	if got := m.store.Get(id).Tags; len(got) != 1 || got[0] != "FRONTEND" {
		t.Fatalf("toggle on: %v", got)
	}
	if s2, _ := task.Open(dir); len(s2.Get(id).Tags) != 1 { // persisted to disk?
		t.Fatal("the tag did not persist in the .md")
	}

	m.updateCardTags(toggle) // unchecks FRONTEND
	if got := m.store.Get(id).Tags; len(got) != 0 {
		t.Fatalf("toggle off: %v", got)
	}

	// a card tag outside the catalog becomes a row (so it can be unchecked)
	tk := m.store.Get(id)
	tk.Tags = []string{"LEGADO"}
	m.store.Save(tk)
	m.reload()
	if rows := m.cardTagRows(); len(rows) != 3 || rows[2] != "LEGADO" {
		t.Fatalf("extra fora do catálogo: %v", rows)
	}
	if !strings.Contains(m.cardTagsBox(), "FRONTEND") {
		t.Fatal("cardTagsBox did not render")
	}
}

// move must change the card's status, persist to disk and regroup into the
// columns. Reopens the store from scratch to prove it really went to the .md.
func TestMovePersists(t *testing.T) {
	dir := t.TempDir()
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&task.Task{Title: "x"}); err != nil { // starts in backlog
		t.Fatal(err)
	}
	m := New(s)
	m.move(1) // backlog → doing

	if m.col != 1 {
		t.Errorf("the cursor should follow the card to column 1, it stayed at %d", m.col)
	}
	if len(m.cols[0]) != 0 || len(m.cols[1]) != 1 {
		t.Errorf("the board did not regroup: backlog=%d doing=%d", len(m.cols[0]), len(m.cols[1]))
	}
	s2, _ := task.Open(dir) // reopen from disk
	if got := s2.All()[0].Status; got != task.StatusDoing {
		t.Errorf("status no disco: quero doing, veio %q", got)
	}
}

// A double click on a config menu row activates it (replays enter): clicking
// "Renomear board" (row 1) enters rename mode. Rows start at modalY+6.
func TestBoardConfigClickRename(t *testing.T) {
	m := modelWithFilters(t)
	m.mode = modeBoardConfig
	m.modalPlaced, m.modalX, m.modalY, m.modalW = true, 0, 0, 40
	click := tea.MouseClickMsg{X: 10, Y: 7, Button: tea.MouseLeft} // row 1 = "Renomear board"

	m.updateMouse(click)
	if m.menuCursor != miRenameBoard {
		t.Fatalf("um clique devia selecionar rename: cur=%d", m.menuCursor)
	}
	m.updateMouse(click) // double click → enters rename
	if m.mode != modeRenameBoard {
		t.Fatalf("double click devia entrar em rename: mode=%v", m.mode)
	}
}

// The wheel scrolls the hovered overflowing column by moving its cursor (scroll also
// selects). Boundary rule: at the top "up" is ignored, at the bottom "down" is ignored.
// A column that fits returns false so the wheel falls back to horizontal pan.
func TestScrollColBy(t *testing.T) {
	m := &Model{
		cols:     [][]*task.Task{{{}, {}, {}}, {{}, {}, {}, {}, {}}}, // col0: 3, col1: 5
		row:      []int{0, 0},
		colTotal: []int{4, 40}, // rendered lines per column
		bodyH:    10,           // col0 fits (4<=10), col1 overflows (40>10)
	}
	if m.scrollColBy(0, 1) { // col0 fits → no scroll, falls back to pan
		t.Fatal("a column that fits should not scroll")
	}
	if !m.scrollColBy(1, 1) || m.col != 1 || m.row[1] != 1 {
		t.Fatalf("col1 devia rolar/selecionar e ativar: col=%d row=%d", m.col, m.row[1])
	}
	m.row[1] = 0 // at the top: "up" ignored
	if !m.scrollColBy(1, -1) || m.row[1] != 0 {
		t.Fatalf("no topo, subir devia ser ignorado: row=%d", m.row[1])
	}
	m.row[1] = 4 // at the bottom: "down" ignored
	if !m.scrollColBy(1, 1) || m.row[1] != 4 {
		t.Fatalf("no fim, descer devia ser ignorado: row=%d", m.row[1])
	}
	if m.scrollColBy(9, 1) { // out of range
		t.Fatal("coluna fora do range devia ser false")
	}
}

// cardAnnotation: a barra do % aparece no meio do caminho e SOME em 100. Quem zera o
// Progress is cleared by the move; an agent stamping 100 after moving used to leave a full
// bar stuck on the card forever (ACME-1234, 2026-08-03).
func TestCardAnnotationHidesFullBar(t *testing.T) {
	m, id := boardWithAction(t, `"true"`) // Model completo: loaderStyle precisa do store
	card := m.store.Get(id)

	// baseline = the annotation with no bar at all. Compare against it instead of looking for
	// "40%" in the text: loaderStyle's showPct is per column, so the number may not even appear.
	card.Progress = 0
	baseline := m.cardAnnotation(card, 20)

	card.Progress = 40
	if got := m.cardAnnotation(card, 20); got == baseline {
		t.Errorf("40%% should draw the bar, got the normal annotation %q", got)
	}

	card.Progress = 100
	if got := m.cardAnnotation(card, 20); got != baseline {
		t.Errorf("100%% should not draw a bar: expected %q, got %q", baseline, got)
	}
}
