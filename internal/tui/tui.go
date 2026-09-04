// Package tui is the hakuban Kanban board: backlog/doing/done columns over
// the .md files in internal/task. Moving a card = change status + Save. It's just
// a view; the disk stays the source of truth (re-reads on every change).
package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/wagnerlim/hakuban/internal/task"
)

type mode int

const (
	modeBoard mode = iota
	modeDetail
	modeAdd
	modeNewBoard
	modeBoardList
	modeSettings
	modePicker
	modeDirBrowser
	modeConfirmMove
	modeNotice
	modeBoardConfig // main menu of the board config
	modeColumns     // sub-screen: column list (reorder/add/delete)
	modeRenameBoard // input to rename the board (reuses m.input)
	modeAddColumn   // input to name a new column (reuses m.input)
	modeLaneConfig  // menu of one column (☰ in the header): Filtros / Renomear / Apagar
	modeLaneFilters // multi-select of the column's filters (from the board's registry)
	modeRenameColumn
	modeConfirmDelete // yes/no confirmation for deleting a board
	modeRenameKey     // input to edit the board's key (prefix)
	modeAddSubtask    // input to create a subtask (child) of the selected card
	modeFilter        // card search (palette overlay): input + list, enter jumps the cursor
	modeKeymap        // shortcut config: lists action→key, captures a new key
	modeTags          // tag catalog: lists tag (color+desc), add/edit/delete
	modeTagForm       // tag form: name + color (←→) + description, create/edit
	modeCardTags      // checklist: check/uncheck catalog tags on the selected card
)

// gearY is the screen row of the gear (column title): tabs(1)+blank(1)+
// top-border(1) → title on row 3. Derived from boardCardsTop so it stays aligned.
const gearY = boardCardsTop - 2

// clickRegion is a clickable strip on the tab-bar row (columns [x0,x1)).
type clickRegion struct {
	x0, x1 int
	kind   string // "tab" | "close" | "new"
	idx    int    // tab index (only for kind "tab"/"close")
}

// cardHit is the on-screen area of a visible card (rows [y0,y1) in column col).
type cardHit struct {
	col    int
	id     string
	y0, y1 int
}

// boardCardsTop is the screen row where a column's cards start:
// tabs(1) + blank(1) + column top-border(1) + title(1) + blank(1).
const boardCardsTop = 5

// orphanBoardID is the slug of the board that adopts tasks without a project at
// boot (the display name comes from i18n, msg.defaultBoard).
const orphanBoardID = "geral"

// Model is the board state. cols[i] are the tasks of column i; row[i] is the
// cursor within it; col is the selected column.
type Model struct {
	store *task.Store

	boards []*task.Board // all boards (Inbox + the rest), reloaded on reload
	open   []string      // ids of the open tabs, in order (browser style)
	active int           // index in `open` of the active tab

	columns      []string // columns of the active board (status, in order), resolved on reload
	cols         [][]*task.Task
	col          int
	hOff         int // horizontal scroll in CELLS (not columns), for smooth sliding
	viewW        int // visible board width at the last render (cells; for clamping hOff)
	row          []int
	mode         mode
	menuCursor   int  // cursor of the config main menu (modeBoardConfig)
	cfgCursor    int  // cursor of the column list (modeColumns)
	cfgGrab      bool // column "grabbed" to reposition with the arrows
	renderedMode mode // last drawn mode — changes → re-centers the modal
	input        textinput.Model
	descInput    textinput.Model // "description" field of the tag form
	colorInput   textinput.Model // "color" field of the tag form (hue or hex)
	vp           viewport.Model  // scroll of the detail
	listCursor   int             // cursor of the board-list modal
	setCursor    int             // cursor of the settings menu
	picker       *picker         // active generic picker (theme/language)

	// directory browser (F13 data dir) + move confirmation
	browsePath      string   // folder being browsed
	browseCursor    int      // cursor in the list (index in the filtered view)
	browseDirs      []string // subfolders of browsePath (sorted, unfiltered)
	browseFilter    string   // live filter text (/)
	browseFiltering bool     // typing the filter?
	moveTarget      string   // chosen destination, awaiting confirmation
	notice          string   // message of the notice modal
	noticeBack      mode     // mode the notice returns to when dismissed

	cfg         task.Config   // user preferences (config.yml)
	lastSig     string        // dir signature at the last poll (file watch)
	tabRegions  []clickRegion // tab hitboxes (filled on render, read on click)
	cardRegions []cardHit     // hitboxes of the visible cards (idem)
	colW        int           // column width at the last render (for x → column)
	bodyH       int           // cards area height at the last render (for wheel scroll)
	colTotal    []int         // total card lines per column at the last render (overflow test)

	// horizontal scroll track (last-render hitbox + drag state)
	trackY, trackW int
	draggingTrack  bool

	// per-column (lane) config: gear in the header + index of the focused lane
	gearRegions []clickRegion // hitboxes of the visible gears (idx = column)
	syncRegions []clickRegion // hitboxes of the footer buttons (columns that have one; idx = column)
	syncY       int           // screen row of the footer buttons (0 = no footer)
	syncAllReg  clickRegion   // hitbox of the board-wide batch button on the top bar (y=0)
	// button menu: the popover a button opens when it has several actions behind it. The
	// first button always sits NEXT TO the anchor — a column's footer menu grows UP (first
	// at the bottom), the top bar's grows DOWN (first at the top).
	menuOpen            bool // popover visible
	menuCol             int  // column it belongs to, or boardMenuCol for the top bar
	menuSel             int  // selected BUTTON index (screen row depends on menuUp)
	menuUp              bool // grows upward (column footer) vs downward (top bar)
	menuX, menuY, menuW int  // top-left of the popover's CONTENT + its width (mouse hit test)
	menuRows            int  // rows drawn (= number of buttons)
	laneIdx             int  // target column of the lane config modal
	laneCursor          int  // cursor of the lane menu (modeLaneConfig)
	// column filters modal (modeLaneFilters): two-level drill-down.
	laneFilterNames  []string            // level 1: the board's filter names
	laneFilterDrill  string              // "" = filter list; else the filter drilled into
	laneFilterOpts   []filterOption      // level 2: options of the drilled filter
	laneFilterCursor int                 // cursor within the current level
	laneFilterSel    map[string][]string // edited use_filters, persisted on close
	subParent        string              // id of the parent card when creating a subtask

	// card search (modeFilter): live results + cursor in the list
	filterHits     []filterHit
	filterCursor   int
	filterTag      string // tag selected in the dropdown ("" = all); text filters within it
	filterDropOpen bool   // tag-selection dropdown open
	filterDropCur  int    // cursor in the dropdown (0 = all · i+1 = catalog[i])

	// transition action in progress (F16/S3): a progress loader over a card, in
	// memory and disposable (closing midway = card returns to its origin).
	action    *cardAction
	actionCh  chan actionEvent
	syncQueue []btnRef       // remaining buttons of a batch (chained one at a time)
	progress  progress.Model // progress bar (bubbles) reused by the action loader

	// configurable shortcuts (modeKeymap): action→key resolved (default + overrides)
	keys            map[string]string
	prefixArmed     bool // ctrl+t armed the prefix → the next key fires a command
	keymapCursor    int
	keymapCapturing bool          // waiting for the next key to rebind the action under the cursor
	keymapConflict  string        // "" none · id of the conflicting action · "reserved" (esc/ctrl+c)
	tagCursor       int           // cursor of the tag catalog list (modeTags)
	tagField        int           // focused field in the form: 0 name · 1 color · 2 description
	tagColorSel     int           // selection in the color field: 0..len(tagHues)-1 preset · len(tagHues) = custom hex
	tagEditIdx      int           // -1 = creating · >=0 = editing the tag at this index
	cardTagCursor   int           // cursor of the card's tag checklist (modeCardTags)
	cardTagID       string        // id of the card being tagged (robust to reload)
	detailStack     []detailFrame // detail stack (drill-down parent→subtask); top = displayed card
	detailRows      []detailRow   // navigable rows (parent/subtask) of the detail: id + line in the content
	detailHits      []detailHit   // hitboxes of the navigable rows on screen (id + y), for clicking
	detailLinks     []detailLink  // clickable URLs of the detail: the issue id + every URL in the body/comments

	// yes/no buttons of the delete-board confirmation (hitboxes relative to the modal)
	confirmNo, confirmYes clickRegion // x0/x1 relative to the modal's top-left
	confirmRow            int         // button row, relative to the modal top
	hoverBtn              int         // selected button (keyboard+mouse): 0 no, 1 yes (always one)

	// dragging a card between columns
	draggingCard bool
	dragCardID   string
	dropCol      int    // column under the cursor during the drag (highlight)
	dragX, dragY int    // current cursor position (for the ghost to follow)
	dragGhost    string // rendered card that follows the cursor

	// draggable modal: position/size at the last render + drag state
	modalX, modalY int
	modalW, modalH int
	modalPlaced    bool // already centered? (re-centers on each open)
	dragging       bool
	grabDX, grabDY int // offset from the grabbed point to the modal's top-left

	pointer        string // last mouse pointer shape sent (OSC 22); avoids re-sending on every event
	mouseX, mouseY int    // last cursor position (so hover can decide the pointer shape)

	// double-click detection (no native click count in bubbletea): a second click on
	// the same target within the window is a double click.
	lastClickKind string
	lastClickAt   time.Time

	w, h int

	glamStyle string                // Glamour style (dark/light), from the active theme
	renderer  *glamour.TermRenderer // cached; rebuild only if the width changes
	rendererW int
}

// New opens the board over the given store.
func New(s *task.Store) *Model {
	ti := textinput.New()
	m := &Model{store: s, row: make([]int, len(task.DefaultColumns)), input: ti, descInput: textinput.New(), colorInput: textinput.New(), cfg: s.LoadConfig()}
	m.loadKeymap()
	applyLang(m.cfg.Lang)
	m.applyTheme()
	m.store.AdoptOrphans(orphanBoardID, msg.defaultBoard, task.DefaultColumns) // migrates legacy tasks without a project
	m.restoreTabs()
	m.reload()
	m.maybeConfigEmpty() // active board without columns → open the config right away (no-op if there's no board: empty-state + "+")
	m.lastSig = dirSig(s.Dir())
	return m
}

// applyTheme applies the theme saved in the config. Called in New() and on restore.
func (m *Model) applyTheme() { m.previewTheme(m.cfg.Theme) }

// previewTheme rebuilds the styles (palette) of the named theme and aligns the
// Glamour style (dark/light) — without touching the config. The picker uses this
// to preview as the cursor moves; saving only happens on enter.
func (m *Model) previewTheme(name string) {
	p := themeByName(name)
	applyPalette(p)
	m.glamStyle = "light"
	if p.isDark {
		m.glamStyle = "dark"
	}
	m.renderer = nil           // forces Glamour to rebuild in the new theme
	m.progress = newProgress() // action progress bar recolored for the new theme
}

// restoreTabs recovers the open tabs from the persisted state (filtering out boards
// that no longer exist); with no valid state, it opens all boards.
func (m *Model) restoreTabs() {
	valid := map[string]bool{}
	for _, b := range m.store.Boards() {
		valid[b.ID] = true
	}
	st := m.store.LoadState()
	for _, id := range st.Open {
		if valid[id] {
			m.open = append(m.open, id)
		}
	}
	if len(m.open) == 0 {
		for _, b := range m.store.Boards() {
			m.open = append(m.open, b.ID)
		}
	}
	if m.active = st.Active; m.active < 0 || m.active >= len(m.open) {
		m.active = 0
	}
}

// persist saves the open tabs + the active one to disk.
func (m *Model) persist() {
	m.store.SaveState(task.State{Open: m.open, Active: m.active})
}

// reload re-reads the store, refreshes the board list and regroups the active
// board's tasks into columns, keeping cursors and active tab within range.
func (m *Model) reload() {
	m.boards = m.store.Boards()
	if m.active >= len(m.open) {
		m.active = max(0, len(m.open)-1)
	}
	m.columns = m.store.ColumnsFor(m.activeBoardID())
	if len(m.row) != len(m.columns) {
		m.row = make([]int, len(m.columns))
	}
	if m.col >= len(m.columns) {
		m.col = max(0, len(m.columns)-1)
	}
	m.cols = groupByStatus(m.columns, m.store.BoardTasks(m.activeBoardID()))
	m.applyFieldFilters(time.Now())
	for i := range m.row {
		if m.row[i] > len(m.cols[i])-1 {
			m.row[i] = max(0, len(m.cols[i])-1)
		}
	}
}

// applyFieldFilters drops, per column, the cards that fail a FIELD filter the column opts into
// via use_filters (Filter.Keep). Field filters are the core-evaluated, OFFLINE half of the
// filter system: they work on any card — local or mirror — unlike tracker filters, which only
// shape the sync query. Display-only: the cards stay on disk, they're just hidden here. Runs
// after groupByStatus and before the cursor clamp, so a shrunk column keeps a valid cursor.
func (m *Model) applyFieldFilters(now time.Time) {
	board := m.activeBoardID()
	for i, col := range m.columns {
		use := m.store.ColumnUseFilters(board, col)
		for name := range use {
			f, ok := m.store.FilterDef(board, name)
			if !ok || !f.IsField() {
				continue // tracker filter (sync's job) or unknown → not ours
			}
			kept := make([]*task.Task, 0, len(m.cols[i]))
			for _, t := range m.cols[i] {
				if f.Keep(t, now) {
					kept = append(kept, t)
				}
			}
			m.cols[i] = kept
		}
	}
}

// maybeConfigEmpty opens the board config menu when it has no columns (a board is
// born empty and needs to be configured before use).
func (m *Model) maybeConfigEmpty() {
	if m.activeBoardID() != task.InboxID && len(m.columns) == 0 {
		m.menuCursor = 0
		m.modalPlaced = false
		m.mode = modeBoardConfig
	}
}

// activeBoardID is the id of the active tab's board (Inbox if there is no tab).
func (m *Model) activeBoardID() string {
	if m.active < 0 || m.active >= len(m.open) {
		return task.InboxID
	}
	return m.open[m.active]
}

func (m *Model) isOpen(id string) bool {
	for _, o := range m.open {
		if o == id {
			return true
		}
	}
	return false
}

func (m *Model) resetCursor() {
	m.col, m.hOff = 0, 0
	for i := range m.row {
		m.row[i] = 0
	}
}

// switchTab switches the active tab (dir=±1, with wrap) and reloads the board.
func (m *Model) switchTab(dir int) {
	if len(m.open) < 2 {
		return
	}
	m.active = (m.active + dir + len(m.open)) % len(m.open)
	m.resetCursor()
	m.reload()
	m.persist()
	m.maybeConfigEmpty()
}

// activateTab makes tab i the active one.
func (m *Model) activateTab(i int) {
	if i < 0 || i >= len(m.open) {
		return
	}
	m.active = i
	m.resetCursor()
	m.reload()
	m.persist()
	m.maybeConfigEmpty()
}

// closeTabAt closes tab i (doesn't delete the board; it just disappears from the
// bar). Keeps at least one tab open and adjusts the active index.
func (m *Model) closeTabAt(i int) {
	if len(m.open) <= 1 || i < 0 || i >= len(m.open) {
		return
	}
	m.open = append(m.open[:i], m.open[i+1:]...)
	if i < m.active || m.active >= len(m.open) {
		m.active = max(0, m.active-1)
	}
	m.resetCursor()
	m.reload()
	m.persist()
}

// doubleClickWindow is how close two clicks on the same target must be to count as a
// double click. 400ms is the common desktop default.
const doubleClickWindow = 400 * time.Millisecond

// isDoubleClick reports whether this click is the second on the same target (kind)
// within the window. It records the click, so a third click starts a fresh pair.
func (m *Model) isDoubleClick(kind string) bool {
	now := time.Now()
	double := kind == m.lastClickKind && now.Sub(m.lastClickAt) <= doubleClickWindow
	m.lastClickAt = now
	if double {
		m.lastClickKind = "" // consume the pair so a triple click doesn't re-fire
	} else {
		m.lastClickKind = kind
	}
	return double
}

// currentMenu describes the clickable-row geometry of the menu modal in the current
// mode: n rows and headerLines (body lines — a label + blank — before the first row).
// ok is false when the mode isn't a row-based menu modal.
func (m *Model) currentMenu() (n, headerLines int, ok bool) {
	switch m.mode {
	case modeBoardConfig:
		return miCount, 2, true
	case modeLaneConfig:
		return len(m.laneMenu()), 2, true
	case modeSettings:
		return len(m.settingRows()), 0, true
	case modeLaneFilters:
		n = len(m.laneFilterNames)
		if m.laneFilterDrill != "" {
			n = len(m.laneFilterOpts)
		}
		return n, 2, true
	case modePicker:
		if m.picker == nil {
			return 0, 0, false
		}
		return len(m.picker.items), 0, true
	case modeBoardList:
		return len(m.boards), 0, true
	}
	return 0, 0, false
}

// menuRowAt maps a click (x,y) to the row index inside the current menu modal, or -1.
// The frame above the body is border(1)+pad(1)+title(1)+blank(1) = 4 rows; headerLines
// is the body lines before the first row.
func (m *Model) menuRowAt(x, y int) int {
	n, headerLines, ok := m.currentMenu()
	if !ok || !m.modalPlaced || x < m.modalX || x >= m.modalX+m.modalW {
		return -1
	}
	i := y - (m.modalY + 4 + headerLines)
	if i < 0 || i >= n {
		return -1
	}
	return i
}

// menuEnter is the current menu mode's key handler; replaying an Enter to it activates
// the selected row, reusing the same dispatch the keyboard uses.
func (m *Model) menuEnter() func(tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeBoardConfig:
		return m.updateBoardConfig
	case modeLaneConfig:
		return m.updateLaneConfig
	case modeSettings:
		return m.updateSettings
	case modeLaneFilters:
		return m.updateLaneFilters
	case modePicker:
		return m.updatePicker
	case modeBoardList:
		return m.updateBoardList
	}
	return nil
}

// menuCursorPtr is a pointer to the current menu mode's cursor field (so a click can
// move the selection there).
func (m *Model) menuCursorPtr() *int {
	switch m.mode {
	case modeBoardConfig:
		return &m.menuCursor
	case modeLaneConfig:
		return &m.laneCursor
	case modeSettings:
		return &m.setCursor
	case modeLaneFilters:
		return &m.laneFilterCursor
	case modePicker:
		return &m.picker.cursor
	case modeBoardList:
		return &m.listCursor
	}
	return nil
}

// clickMenuRow selects the row under (x,y); a double click activates it by replaying
// Enter to the mode's handler (same effect as pressing enter on that row).
func (m *Model) clickMenuRow(x, y int) (tea.Model, tea.Cmd) {
	i := m.menuRowAt(x, y)
	if i < 0 {
		return m, nil
	}
	*m.menuCursorPtr() = i
	if m.mode == modePicker { // live preview follows the highlight, like keyboard nav
		m.picker.preview(m.picker.items[i])
	}
	if m.isDoubleClick(fmt.Sprintf("menu:%d:%d", m.mode, i)) {
		return m.menuEnter()(tea.KeyPressMsg{Code: tea.KeyEnter})
	}
	return m, nil
}

// linkAt returns the URL under (x,y) in the detail, "" if there is none. y is a screen
// row of the viewport, so the scroll offset maps it back to the content line — no
// per-render bookkeeping needed.
func (m *Model) linkAt(x, y int) string {
	if y < 0 || y >= m.vp.Height() {
		return ""
	}
	line := y + m.vp.YOffset()
	for _, l := range m.detailLinks {
		if l.line == line && x >= l.x0 && x < l.x1 {
			return l.url
		}
	}
	return ""
}

// openURL opens a URL in the user's default browser (fire-and-forget; a failure is
// silent — worst case the click does nothing). Per-OS launcher.
func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// updateMouse handles the pressed button: on the board, a click on the tab bar; on
// a modal, grabs the title bar to drag.
func (m *Model) updateMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	e := msg.Mouse()
	if m.mode == modeDetail && e.Button == tea.MouseLeft {
		if url := m.linkAt(e.X, e.Y); url != "" {
			openURL(url) // click a link (id or URL in the body) → open it in the browser
			return m, nil
		}
		for _, h := range m.detailHits { // click on a navigable row (parent/subtask)
			if e.Y == h.y {
				m.openDetailID(h.id)
				break
			}
		}
		return m, nil
	}
	if m.mode == modeBoard {
		if m.menuOpen { // popover open: it swallows the board's mouse until it closes
			if n := m.menuHit(e.X, e.Y); n >= 0 {
				m.menuSel = n // hover follows the pointer
				if e.Button == tea.MouseLeft {
					return m, m.fireMenu(n)
				}
			} else if e.Button == tea.MouseLeft {
				m.menuOpen = false // click outside closes
			}
			return m, nil
		}
		if e.Y == 0 {
			if e.Button == tea.MouseLeft && e.X >= m.syncAllReg.x0 && e.X < m.syncAllReg.x1 && m.syncAllReg.x1 > 0 {
				return m, m.activateBoard() // board-wide button (fires, or opens its menu)
			}
			return m.clickTabBar(e)
		}
		if e.Button == tea.MouseLeft && e.Y == gearY { // gear in the column header
			for _, g := range m.gearRegions {
				if e.X >= g.x0 && e.X < g.x1 {
					m.openLaneConfig(g.idx)
					return m, nil
				}
			}
		}
		if e.Button == tea.MouseLeft && m.syncY > 0 && e.Y == m.syncY { // button in the footer
			for _, r := range m.syncRegions {
				if e.X >= r.x0 && e.X < r.x1 {
					return m, m.activateColumn(r.idx)
				}
			}
		}
		if e.Button == tea.MouseLeft && m.trackY > 0 && e.Y == m.trackY && e.X < m.trackW {
			m.draggingTrack = true // click/drag on the track scrolls the board
			m.scrollTrackTo(e.X)
			return m, nil
		}
		if e.Button == tea.MouseLeft {
			if hit, ok := m.cardAt(e.X, e.Y); ok {
				m.focusCard(hit.id)                    // single click selects the card
				if m.isDoubleClick("card:" + hit.id) { // double click opens it (no drag)
					m.openDetail()
					return m, nil
				}
				if t := m.store.Get(hit.id); t != nil { // grabs it to drag; mirror also drags: position is the user's (only the content is read-only)
					m.draggingCard, m.dragCardID, m.dropCol = true, hit.id, hit.col
					m.dragX, m.dragY = e.X, e.Y
					m.grabDX = e.X - (hit.col*m.colW - m.hOff + 2) // 2 = border+padding; screen position of the column
					m.grabDY = e.Y - hit.y0
					m.dragGhost = renderCard(t, true, m.colW-10, m.cardAnnotation(t, m.colW-10), m.cfg.Tags) // "lifted" ghost (same card width as the board)
				}
			}
		}
		return m, nil
	}
	if m.mode == modeConfirmDelete && e.Button == tea.MouseLeft {
		switch m.hitConfirmBtn(e.X, e.Y) {
		case 1: // Yes → deletes
			m.deleteActiveBoard()
			return m, nil
		case 0: // No → back to config
			m.modalPlaced = false
			m.mode = modeBoardConfig
			return m, nil
		}
		// outside the buttons: falls through to the modal handling (close/drag) below
	}
	if m.inModal() && e.Button == tea.MouseLeft {
		switch {
		case m.onModalClose(e.X, e.Y): // red dot
			if m.mode == modeLaneFilters {
				m.saveLaneFilters() // persist the working selection, like esc
			}
			m.closeModal()
		case m.onModalLights(e.X, e.Y): // yellow/green: decorative, they don't drag
		case m.onModalHandle(e.X, e.Y):
			m.dragging = true
			m.grabDX, m.grabDY = e.X-m.modalX, e.Y-m.modalY
		default: // click a menu row: select; double click activates (replays enter)
			return m.clickMenuRow(e.X, e.Y)
		}
	}
	return m, nil
}

// topBar is the y=0 row: the tabs plus the board-wide button, placed left or right per the
// board's sync_button. What the button says and does comes from the board (boardButtons);
// while a board-wide action runs, the button IS the loader. It shifts the tab hitboxes when
// the button sits on the left, and registers the button's own hitbox (syncAllReg). Returns
// just the tabs when the button is off / the board declares none.
func (m *Model) topBar() string {
	tabs := m.tabBar() // fills tabRegions from x=0
	m.syncAllReg = clickRegion{}
	bs := m.boardButtons()
	pos := m.store.SyncButtonPos(m.activeBoardID())
	if pos == "off" || len(bs) == 0 {
		return tabs
	}
	label := m.boardButtonLabel(bs)
	bw := lipgloss.Width(label) + 4 // btn padding (0,2) = +4; hover-independent
	// a board-wide cmd/agent has no column to anchor its loader → the button becomes the bar
	// (and widens the top row while it runs; the tabs shift with it, hitboxes included).
	render := func(hover bool) string { return btn(label, hover, false) }
	if m.action != nil && m.action.board {
		bar := m.actionBar(m.action, barCells+6)
		bw = lipgloss.Width(bar)
		render = func(bool) string { return bar }
	}
	const gap = 1
	if pos == "top-left" {
		for i := range m.tabRegions { // tabs move right to make room for the button
			m.tabRegions[i].x0 += bw + gap
			m.tabRegions[i].x1 += bw + gap
		}
		m.syncAllReg = clickRegion{x0: 0, x1: bw, kind: "syncall"}
		return render(m.hoverSyncAll(0, bw)) + strings.Repeat(" ", gap) + tabs
	}
	tw := lipgloss.Width(tabs) // default: top-right
	pad := max(gap, m.viewW-tw-bw)
	x0 := tw + pad
	m.syncAllReg = clickRegion{x0: x0, x1: x0 + bw, kind: "syncall"}
	return tabs + strings.Repeat(" ", pad) + render(m.hoverSyncAll(x0, x0+bw))
}

// hoverSyncAll is true while the mouse hovers the sync-all hitbox (highlight), or a
// batch is running (so the button stays lit through the sync).
func (m *Model) hoverSyncAll(x0, x1 int) bool {
	return m.syncing() || (m.mouseY == 0 && m.mouseX >= x0 && m.mouseX < x1)
}

// boardButtonLabel is what the top bar button reads: the single button's label (+ icon), or
// the board's button_label (default: the first one's) plus the ▾ that says "opens a menu".
func (m *Model) boardButtonLabel(bs []colButton) string {
	if len(bs) == 0 { // no button → no label (topBar already returns early; keeps it total)
		return ""
	}
	if len(bs) == 1 {
		return strings.TrimSpace(bs[0].Label + " " + bs[0].Icon)
	}
	label := m.store.BoardButtonLabel(m.activeBoardID())
	if label == "" {
		label = bs[0].Label
	}
	return label + " ▾"
}

// syncing reports whether a board-wide action (a batch, or a top-bar cmd/agent) is running —
// keeps the top bar button lit through it.
func (m *Model) syncing() bool {
	return m.action != nil && (m.action.col != "" || m.action.board) && !m.action.done && !m.action.failed
}

// clickTabBar handles a click on the tab bar (y==0): left switches the tab / opens
// the "+", middle closes the tab (browser gesture).
func (m *Model) clickTabBar(e tea.Mouse) (tea.Model, tea.Cmd) {
	if e.Y != 0 {
		return m, nil
	}
	for _, r := range m.tabRegions {
		if e.X < r.x0 || e.X >= r.x1 {
			continue
		}
		switch {
		case r.kind == "new":
			m.startNewBoard()
			return m, textinput.Blink
		case r.kind == "close", e.Button == tea.MouseMiddle:
			m.closeTabAt(r.idx)
		default:
			m.activateTab(r.idx)
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) inModal() bool {
	return m.mode == modeAdd || m.mode == modeNewBoard || m.mode == modeBoardList ||
		m.mode == modeSettings || m.mode == modePicker ||
		m.mode == modeDirBrowser || m.mode == modeConfirmMove || m.mode == modeNotice ||
		m.mode == modeBoardConfig || m.mode == modeColumns || m.mode == modeRenameBoard || m.mode == modeAddColumn ||
		m.mode == modeLaneConfig || m.mode == modeLaneFilters || m.mode == modeRenameColumn || m.mode == modeConfirmDelete ||
		m.mode == modeRenameKey || m.mode == modeAddSubtask || m.mode == modeFilter ||
		m.mode == modeKeymap || m.mode == modeTags || m.mode == modeTagForm ||
		m.mode == modeCardTags
}

// onModalHandle: (x,y) is on the modal's title bar (top: border + padding + title
// line), the drag area.
func (m *Model) onModalHandle(x, y int) bool {
	return m.modalPlaced &&
		x >= m.modalX && x < m.modalX+m.modalW &&
		y >= m.modalY && y < m.modalY+3 // border (dots) + padding + title line
}

// onModalClose: is (x,y) on the red dot? It sits on the top border (modalY), in
// column modalX+3 (after the " " of slack). Slack of ±1.
func (m *Model) onModalClose(x, y int) bool {
	return m.modalPlaced &&
		x >= m.modalX+2 && x <= m.modalX+4 &&
		y >= m.modalY && y <= m.modalY+1
}

// onModalLights: is (x,y) on the strip of the 3 dots (top border, col +3..+7)?
// Lets yellow/green consume the click without starting a drag.
func (m *Model) onModalLights(x, y int) bool {
	return m.modalPlaced && y == m.modalY && x >= m.modalX+3 && x <= m.modalX+7
}

// closeModal returns to the board (cancels the open modal).
func (m *Model) closeModal() {
	m.dragging = false
	m.mode = modeBoard
}

// openBoard ensures the board id is open and makes it the active tab.
func (m *Model) openBoard(id string) {
	if !m.isOpen(id) {
		m.open = append(m.open, id)
	}
	for i, o := range m.open {
		if o == id {
			m.active = i
		}
	}
	m.resetCursor()
	m.reload()
	m.persist()
	m.maybeConfigEmpty()
}

// startNewBoard opens the input to name a new board.
func (m *Model) startNewBoard() {
	m.input.SetValue("")
	m.input.Placeholder = msg.phBoard
	m.input.SetWidth(40)
	m.input.Focus()
	m.modalPlaced = false
	m.mode = modeNewBoard
}

// startAdd opens the input to create a task on the active board. With no columns
// you can't create one (a task needs a status) — it falls into the config so the
// user can create one.
func (m *Model) startAdd() {
	if len(m.columns) == 0 {
		m.maybeConfigEmpty()
		return
	}
	m.input.SetValue("")
	m.input.Placeholder = msg.phTask
	m.input.SetWidth(40)
	m.input.Focus()
	m.modalPlaced = false
	m.mode = modeAdd
}

// startAddSubtask opens the input to create a subtask (child of m.subParent).
func (m *Model) startAddSubtask() {
	m.input.SetValue("")
	m.input.Placeholder = msg.phTask
	m.input.SetWidth(40)
	m.input.Focus()
	m.modalPlaced = false
	m.mode = modeAddSubtask
}

// filterHit is a search result: the matched task + the column it's in (only for
// the result's breadcrumb). The cursor jump re-finds it by ID.
type filterHit struct {
	t   *task.Task
	col int
}

// startFilter opens the card-search overlay (command-palette style): input at the
// top, live list below. An empty query lists all cards of the active board.
func (m *Model) startFilter() {
	m.input.SetValue("")
	m.input.Placeholder = msg.phFilter
	m.input.SetWidth(40)
	m.input.Focus()
	m.filterCursor = 0
	m.filterTag = ""
	m.filterDropOpen = false
	m.filterDropCur = 0
	m.modalPlaced = false
	m.recomputeFilter()
	m.mode = modeFilter
}

// recomputeFilter rebuilds the results on every key: a card passes if it has the
// selected tag (if any) AND the text matches the title/id. Composite filter: pick
// the tag in the dropdown, then refine by title. Source = m.cols → every hit is locatable.
func (m *Model) recomputeFilter() {
	q := strings.ToLower(strings.TrimSpace(m.input.Value()))
	m.filterHits = m.filterHits[:0]
	for ci, col := range m.cols {
		for _, t := range col {
			if m.matchesFilter(t, q) {
				m.filterHits = append(m.filterHits, filterHit{t, ci})
			}
		}
	}
	if m.filterCursor >= len(m.filterHits) {
		m.filterCursor = max(0, len(m.filterHits)-1)
	}
}

// matchesFilter: has the selected tag (if any) AND (empty text OR matches the
// title/id). q already arrives lowercased.
func (m *Model) matchesFilter(t *task.Task, q string) bool {
	if m.filterTag != "" && !hasTag(t, m.filterTag) {
		return false
	}
	return q == "" || strings.Contains(strings.ToLower(t.Title), q) || strings.Contains(strings.ToLower(t.ID), q)
}

// hasTag reports whether the task has the tag (case-insensitive).
func hasTag(t *task.Task, name string) bool {
	for _, tg := range t.Tags {
		if strings.EqualFold(tg, name) {
			return true
		}
	}
	return false
}

// focusCard puts the board cursor on the card with the given id, rescanning m.cols
// (robust to a disk reload between opening the search and pressing enter). No-op if
// not found.
func (m *Model) focusCard(id string) {
	for ci, col := range m.cols {
		for ri, t := range col {
			if t.ID == id {
				m.col, m.row[ci] = ci, ri
				m.ensureColVisible()
				return
			}
		}
	}
}

// updateFilter handles the search overlay keys: esc closes, ↑↓ navigate the list,
// enter jumps the cursor to the selected card; the rest goes to the input and refilters.
func (m *Model) updateFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// dropdown open: ↑↓ navigate the options, enter chooses, esc closes without changing
	if m.filterDropOpen {
		n := len(m.cfg.Tags) + 1 // "all" + catalog
		switch msg.String() {
		case "esc":
			m.filterDropOpen = false
		case "up", "k", "ctrl+k":
			if m.filterDropCur > 0 {
				m.filterDropCur--
			}
		case "down", "j", "ctrl+j":
			if m.filterDropCur < n-1 {
				m.filterDropCur++
			}
		case "enter", " ":
			if m.filterDropCur == 0 {
				m.filterTag = ""
			} else {
				m.filterTag = m.cfg.Tags[m.filterDropCur-1].Name
			}
			m.filterDropOpen = false
			m.recomputeFilter()
		}
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.modalPlaced = false
		m.mode = modeBoard
		return m, nil
	case "tab": // opens the tag dropdown (cursor on the current selection)
		if len(m.cfg.Tags) > 0 {
			m.filterDropOpen = true
			m.filterDropCur = 0
			for i, td := range m.cfg.Tags {
				if td.Name == m.filterTag {
					m.filterDropCur = i + 1
				}
			}
		}
		return m, nil
	case "up", "ctrl+k":
		if m.filterCursor > 0 {
			m.filterCursor--
		}
		return m, nil
	case "down", "ctrl+j":
		if m.filterCursor < len(m.filterHits)-1 {
			m.filterCursor++
		}
		return m, nil
	case "enter":
		if m.filterCursor < len(m.filterHits) {
			m.focusCard(m.filterHits[m.filterCursor].t.ID)
		}
		m.modalPlaced = false
		m.mode = modeBoard
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.recomputeFilter()
	return m, cmd
}

// slugify turns a name into a filesystem-safe id ([a-z0-9-]).
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// groupByStatus splits the tasks into the board columns, in the order of `cols`. An
// unknown/empty status (e.g. a renamed or deleted column) falls into column 0 — it
// never disappears from the screen. A board with no columns returns an empty slice
// (no columns).
func groupByStatus(cols []string, tasks []*task.Task) [][]*task.Task {
	idx := map[string]int{}
	for i, c := range cols {
		idx[c] = i
	}
	out := make([][]*task.Task, len(cols))
	if len(cols) == 0 {
		return out
	}
	for _, t := range tasks {
		i, ok := idx[t.Status]
		if !ok {
			i = 0
		}
		out[i] = append(out[i], t)
	}
	return out
}

// current is the task under the cursor (nil if there's no column or the column is empty).
func (m *Model) current() *task.Task {
	if m.col < 0 || m.col >= len(m.cols) {
		return nil
	}
	c := m.cols[m.col]
	if len(c) == 0 {
		return nil
	}
	return c[m.row[m.col]]
}

// reloadTickMsg triggers the periodic poll for disk changes.
type reloadTickMsg struct{}

// pollInterval is the frequency of the file watch (external change to a .md → reload).
const pollInterval = time.Second

func tickCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return reloadTickMsg{} })
}

func (m *Model) Init() tea.Cmd { return tea.Batch(tickCmd(), m.syncOnOpen()) }

// dirSig is a cheap signature of the data dir (name+size+mtime of the files,
// without parsing) — it changes when any .md is created/edited/removed externally.
// Includes jira/ (F16 mirrors): an external/AI sync touching it triggers the refresh.
func dirSig(dir string) string {
	var b strings.Builder
	for _, sub := range []string{"tasks", "boards", "archive", "jira"} {
		entries, _ := os.ReadDir(filepath.Join(dir, sub))
		for _, e := range entries {
			if info, err := e.Info(); err == nil {
				fmt.Fprintf(&b, "%s:%d:%d;", e.Name(), info.Size(), info.ModTime().UnixNano())
			}
		}
	}
	return b.String()
}

// refresh re-reads the store from disk (picks up external changes) and regroups the
// columns, keeping tabs and cursors. No-op if Open fails (e.g. a file being written).
func (m *Model) refresh() {
	st, err := task.Open(m.store.Dir())
	if err != nil {
		return
	}
	m.store = st
	m.lastSig = dirSig(st.Dir())
	m.reload()
}

// debugMouse logs every mouse event to /tmp/charm-mouse.log when HAKUBAN_DEBUG_MOUSE
// is set. Temporary — to diagnose the terminal's wheel.
func debugMouse(msg tea.Msg) {
	if os.Getenv("HAKUBAN_DEBUG_MOUSE") == "" {
		return // real gate: without the env, no-op (doesn't open a file on every event)
	}
	mm, ok := msg.(tea.MouseMsg)
	if !ok {
		return
	}
	e := mm.Mouse()
	f, err := os.OpenFile("/tmp/charm-mouse.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	fmt.Fprintf(f, "%T button=%q mod=%v x=%d y=%d\n", msg, e.Button.String(), e.Mod, e.X, e.Y)
	f.Close()
}

// Mouse pointer shapes (OSC 22, CSS cursor names understood by Ghostty/kitty).
const (
	ptrDefault  = "default"
	ptrText     = "text"     // over a focused input field → I-beam
	ptrPointer  = "pointer"  // over a clickable control (☰ filter menu) → hand
	ptrGrab     = "grab"     // over a card on the board → open hand ("draggable")
	ptrGrabbing = "grabbing" // dragging something (card/mirror, modal, track) → closed hand
)

// Update is a thin wrapper: it runs the real update and then adjusts the mouse
// pointer shape based on the resulting state (drag, focused input, hover…). The
// OSC 22 sequence is only emitted when the shape changes — otherwise it would
// spam on every MouseMotion.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.update(msg)
	if want := m.pointerShape(); want != m.pointer {
		m.pointer = want
		cmd = tea.Batch(cmd, tea.Raw(ansi.SetPointerShape(want)))
	}
	return model, cmd
}

// pointerShape decides the pointer shape from the current state. Dragging beats
// input (you're not typing while dragging); hover over a card hints "draggable".
func (m *Model) pointerShape() string {
	switch {
	case m.dragging || m.draggingCard || m.draggingTrack:
		return ptrGrabbing
	case m.input.Focused() || m.descInput.Focused() || m.colorInput.Focused():
		return ptrText
	case m.mode == modeBoard:
		if m.menuHit(m.mouseX, m.mouseY) >= 0 { // hovering a row of the open column menu
			return ptrPointer
		}
		if m.mouseY == 0 && m.syncAllReg.x1 > 0 && m.mouseX >= m.syncAllReg.x0 && m.mouseX < m.syncAllReg.x1 {
			return ptrPointer // hovering the "sync all" button
		}
		if m.mouseY == gearY { // hovering the ☰ filter menu: signals it's clickable
			for _, g := range m.gearRegions {
				if m.mouseX >= g.x0 && m.mouseX < g.x1 {
					return ptrPointer
				}
			}
		}
		if m.syncY > 0 && m.mouseY == m.syncY { // hovering a sync button: clickable
			for _, r := range m.syncRegions {
				if m.mouseX >= r.x0 && m.mouseX < r.x1 {
					return ptrPointer
				}
			}
		}
		if _, ok := m.cardAt(m.mouseX, m.mouseY); ok {
			return ptrGrab // hovering a card: signals it can be dragged
		}
	case m.mode == modeDetail:
		if m.linkAt(m.mouseX, m.mouseY) != "" { // hovering a link (issue id or URL in the body)
			return ptrPointer
		}
	default:
		if m.menuRowAt(m.mouseX, m.mouseY) >= 0 { // hovering a clickable menu row
			return ptrPointer
		}
	}
	return ptrDefault
}

func (m *Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	debugMouse(msg) // no-op without HAKUBAN_DEBUG_MOUSE
	switch msg := msg.(type) {
	case reloadTickMsg:
		// file watch: reload if the disk changed (AI/external edit), except while
		// dragging a card (don't pull the rug out from under the gesture).
		if !m.draggingCard {
			if sig := dirSig(m.store.Dir()); sig != m.lastSig {
				m.refresh()
			}
		}
		return m, tickCmd()
	case actionEvent:
		return m.onActionEvent(msg)
	case progress.FrameMsg: // animation frame: the progress animates, and the loader (shown)
		// follows the target (pct) in the same beat → the block bar fills gradually
		// (5%→25% shows 1→5 filling) instead of jumping straight to the target.
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		if m.action != nil {
			target := float64(m.action.pct) / 100
			if d := target - m.action.shown; d > 0.004 || d < -0.004 {
				m.action.shown += d * 0.2 // exponential easing toward the target
			} else {
				m.action.shown = target
			}
		}
		return m, cmd
	case actionClearMsg:
		m.action = nil // clears the ✓ of success
		m.reload()
		return m, nil
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		if m.mode == modeDetail {
			m.resizeDetail()
		}
		return m, nil
	case tea.MouseClickMsg:
		return m.updateMouse(msg)
	case tea.MouseMotionMsg:
		e := msg.Mouse()
		m.mouseX, m.mouseY = e.X, e.Y // so hover can decide the pointer shape
		switch {
		case m.menuOpen: // open column menu: hover moves the selection
			if n := m.menuHit(e.X, e.Y); n >= 0 {
				m.menuSel = n
			}
		case m.dragging: // dragging the modal
			m.modalX = max(0, min(e.X-m.grabDX, m.w-m.modalW))
			m.modalY = max(0, min(e.Y-m.grabDY, m.h-m.modalH))
		case m.draggingCard: // dragging a card → ghost follows + highlights the column
			m.dragX, m.dragY = e.X, e.Y
			m.dropCol = m.colAt(e.X)
		case m.draggingTrack: // dragging the track → scrolls the board
			m.scrollTrackTo(e.X)
		case m.mode == modeConfirmDelete: // mouse hover moves the selection; outside the buttons keeps the last one
			if b := m.hitConfirmBtn(e.X, e.Y); b >= 0 {
				m.hoverBtn = b
			}
		}
		return m, nil
	case tea.MouseReleaseMsg:
		var cmd tea.Cmd
		if m.draggingCard {
			cmd = m.dropCard(m.colAt(msg.Mouse().X))
			m.draggingCard, m.dragCardID = false, ""
		}
		m.dragging = false
		m.draggingTrack = false
		return m, cmd
	case tea.MouseWheelMsg:
		if m.mode == modeDetail { // mouse wheel scrolls the detail
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
		if m.mode == modeBoard {
			const step = 6 // cells per pan tick (smooth slide)
			e := msg.Mouse()
			if col := m.colAt(e.X); m.colOverflows(col) {
				// Pointer over a scrollable column: dedicate the wheel to it. Ignore the
				// horizontal-axis events a trackpad emits alongside a vertical scroll,
				// otherwise the board would drift sideways while scrolling the column.
				switch e.Button {
				case tea.MouseWheelUp:
					m.scrollColBy(col, -1)
				case tea.MouseWheelDown:
					m.scrollColBy(col, 1)
				}
				return m, nil
			}
			// Column fits entirely → the wheel pans the board sideways. Any axis pans, so
			// terminals that only emit vertical wheel (e.g. Terminal.app) can still pan.
			switch e.Button {
			case tea.MouseWheelUp, tea.MouseWheelLeft:
				m.hOff -= step
			case tea.MouseWheelDown, tea.MouseWheelRight:
				m.hOff += step
			}
			m.clampHOff()
		}
		return m, nil
	case tea.KeyPressMsg:
		switch m.mode {
		case modeAdd, modeNewBoard, modeRenameBoard, modeAddColumn, modeRenameColumn, modeRenameKey, modeAddSubtask:
			return m.updateInput(msg)
		case modeBoardConfig:
			return m.updateBoardConfig(msg)
		case modeColumns:
			return m.updateColumns(msg)
		case modeLaneConfig:
			return m.updateLaneConfig(msg)
		case modeLaneFilters:
			return m.updateLaneFilters(msg)
		case modeBoardList:
			return m.updateBoardList(msg)
		case modeSettings:
			return m.updateSettings(msg)
		case modePicker:
			return m.updatePicker(msg)
		case modeDirBrowser:
			return m.updateDirBrowser(msg)
		case modeConfirmMove:
			return m.updateConfirmMove(msg)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		case modeNotice:
			return m.updateNotice(msg)
		case modeDetail:
			return m.updateDetail(msg)
		case modeFilter:
			return m.updateFilter(msg)
		case modeKeymap:
			return m.updateKeymap(msg)
		case modeTags:
			return m.updateTags(msg)
		case modeTagForm:
			return m.updateTagForm(msg)
		case modeCardTags:
			return m.updateCardTags(msg)
		default:
			return m.updateBoard(msg)
		}
	}
	return m, nil
}

// Rebindable board actions. The id is stable (goes in config.yml + i18n); the order
// of keyActions is the display order in the modal (most-used first).
const (
	kaAdd        = "add"
	kaFind       = "find"
	kaOpen       = "open"
	kaSubtask    = "subtask"
	kaMoveLeft   = "move_left"
	kaMoveRight  = "move_right"
	kaLeft       = "nav_left"
	kaRight      = "nav_right"
	kaDown       = "nav_down"
	kaUp         = "nav_up"
	kaNextBoard  = "next_board"
	kaPrevBoard  = "prev_board"
	kaNewBoard   = "new_board"
	kaCloseBoard = "close_board"
	kaBoardList  = "board_list"
	kaBoardCfg   = "board_cfg"
	kaLaneCfg    = "lane_cfg"
	kaSettings   = "settings"
	kaKeymap     = "keymap"
	kaTags       = "tags"
	kaCardTags   = "card_tags"
	kaSync       = "sync"
	kaQuit       = "quit"
)

// prefixKey arms the tmux-style prefix; the next key fires a command.
const prefixKey = "ctrl+t"

// action groups: nav fires directly (no prefix); cmd requires the prefix. They're
// also the sections of the keybind modal (grpNav → grpCmd, in that order).
const (
	grpNav = "nav"
	grpCmd = "cmd"
)

var keyActions = []struct{ id, def, grp string }{
	// navigation — direct keys, no prefix
	{kaLeft, "h", grpNav}, {kaDown, "j", grpNav}, {kaUp, "k", grpNav}, {kaRight, "l", grpNav},
	{kaMoveLeft, "H", grpNav}, {kaMoveRight, "L", grpNav},
	{kaNextBoard, "tab", grpNav}, {kaPrevBoard, "shift+tab", grpNav}, {kaOpen, "enter", grpNav},
	// commands — behind the prefix (ctrl+t)
	{kaAdd, "a", grpCmd}, {kaFind, "/", grpCmd}, {kaSubtask, "S", grpCmd},
	{kaBoardList, "b", grpCmd}, {kaBoardCfg, "c", grpCmd}, {kaLaneCfg, "g", grpCmd},
	{kaTags, "t", grpCmd}, {kaCardTags, "T", grpCmd}, {kaSettings, "s", grpCmd},
	{kaNewBoard, "+", grpCmd}, {kaCloseBoard, "ctrl+w", grpCmd},
	{kaSync, "y", grpCmd},
	{kaKeymap, "?", grpCmd}, {kaQuit, "q", grpCmd},
}

// groupOf returns the group (grpNav/grpCmd) of an action, "" if unknown.
func groupOf(id string) string {
	for _, a := range keyActions {
		if a.id == id {
			return a.grp
		}
	}
	return ""
}

// fixedAliases are keys that always fire the action, independent of config and the
// prefix (arrows/ctrl) — the ergonomic fallback that rebinding can't break.
var fixedAliases = map[string]string{
	"left": kaLeft, "right": kaRight, "down": kaDown, "up": kaUp,
	">": kaMoveRight, "<": kaMoveLeft, "ctrl+c": kaQuit,
}

// loadKeymap resolves the active keymap: defaults with the config overrides on top.
func (m *Model) loadKeymap() {
	m.keys = make(map[string]string, len(keyActions))
	for _, a := range keyActions {
		m.keys[a.id] = a.def
	}
	for id, k := range m.cfg.Keys {
		if _, ok := m.keys[id]; ok && k != "" {
			m.keys[id] = k
		}
	}
}

// actionFor returns the action bound to a key (via keyActions, deterministic order),
// or "" if none.
func (m *Model) actionFor(key string) string {
	for _, a := range keyActions {
		if m.keys[a.id] == key {
			return a.id
		}
	}
	return ""
}

func defaultKeyFor(id string) string {
	for _, a := range keyActions {
		if a.id == id {
			return a.def
		}
	}
	return ""
}

// saveKeymap persists only the overrides (what differs from the default) to config.yml.
func (m *Model) saveKeymap() {
	over := map[string]string{}
	for _, a := range keyActions {
		if m.keys[a.id] != a.def {
			over[a.id] = m.keys[a.id]
		}
	}
	if len(over) == 0 {
		over = nil // clean config.yml when everything is default
	}
	m.cfg.Keys = over
	m.store.SaveConfig(m.cfg)
}

func (m *Model) updateBoard(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.menuOpen {
		return m.updateColumnMenu(k)
	}
	// tmux-style model: ctrl+t "arms" the prefix; the next key fires ONE command
	// and disarms. Navigation (grpNav) and fixed aliases (arrows, ctrl+c) always
	// apply; commands (grpCmd) only fire with the prefix armed.
	armed := m.prefixArmed
	m.prefixArmed = false // every key disarms; we only re-arm for the prefix itself
	if !armed && k == prefixKey {
		m.prefixArmed = true
		return m, nil
	}
	if armed && (k == "esc" || k == prefixKey) {
		return m, nil // cancels the prefix without firing anything
	}
	action := m.actionFor(k)
	if alias, ok := fixedAliases[k]; ok {
		action = alias // fixed aliases ignore the prefix gate
	} else if !armed && groupOf(action) == grpCmd {
		return m, nil // command without prefix → ignore
	}
	switch action {
	case kaQuit:
		// restore the pointer before leaving (otherwise Ghostty keeps the last shape)
		return m, tea.Sequence(tea.Raw(ansi.SetPointerShape(ptrDefault)), tea.Quit)
	case kaLeft:
		if m.col > 0 {
			m.col--
			m.ensureColVisible()
		}
	case kaRight:
		if m.col < len(m.columns)-1 {
			m.col++
			m.ensureColVisible()
		}
	case kaDown:
		if m.col < len(m.cols) && m.row[m.col] < len(m.cols[m.col])-1 {
			m.row[m.col]++
		}
	case kaUp:
		if m.col < len(m.cols) && m.row[m.col] > 0 {
			m.row[m.col]--
		}
	case kaMoveRight:
		return m, m.move(1)
	case kaMoveLeft:
		return m, m.move(-1)
	case kaNextBoard:
		m.switchTab(1)
	case kaPrevBoard:
		m.switchTab(-1)
	case kaNewBoard:
		m.startNewBoard()
		return m, textinput.Blink
	case kaCloseBoard:
		m.closeTabAt(m.active)
	case kaBoardCfg:
		m.openBoardConfig()
	case kaLaneCfg:
		m.openLaneConfig(m.col) // config of the column under the cursor (partner of the gear)
	case kaBoardList:
		m.listCursor = 0
		for i, bd := range m.boards {
			if bd.ID == m.activeBoardID() {
				m.listCursor = i
			}
		}
		m.modalPlaced = false
		m.mode = modeBoardList
	case kaSettings:
		m.setCursor = 0
		m.modalPlaced = false
		m.mode = modeSettings
	case kaOpen:
		m.openDetail()
	case kaAdd:
		m.startAdd()
		return m, textinput.Blink
	case kaFind:
		m.startFilter()
		return m, textinput.Blink
	case kaSubtask:
		if t := m.current(); t != nil && !t.Mirror() { // creates a subtask of the card under the cursor (mirror is read-only)
			m.subParent = t.ID
			m.startAddSubtask()
			return m, textinput.Blink
		}
	case kaSync:
		return m, m.activateColumn(m.col) // fires the button of the column under the cursor (or opens its menu)
	case kaKeymap:
		m.startKeymap()
	case kaTags:
		m.startTags()
	case kaCardTags:
		if t := m.current(); t != nil && !t.Mirror() { // editing tags = editing content → blocked on the mirror
			m.startCardTags()
		}
	}
	return m, nil
}

// updateColumnMenu drives the open column popover. The list is drawn bottom-up, so ↑ walks
// toward the LAST configured button and ↓ back toward the first; enter fires, esc closes.
// Any other key closes too and is swallowed — no board command fires behind the popover.
func (m *Model) updateColumnMenu(k string) (tea.Model, tea.Cmd) {
	n := len(m.menuButtons())
	step := 1 // an upward menu walks the config order backwards on screen
	if !m.menuUp {
		step = -1
	}
	switch k {
	case "up", "k":
		if i := m.menuSel + step; i >= 0 && i < n {
			m.menuSel = i
		}
	case "down", "j":
		if i := m.menuSel - step; i >= 0 && i < n {
			m.menuSel = i
		}
	case "enter", " ":
		return m, m.fireMenu(m.menuSel)
	default:
		m.menuOpen = false
	}
	return m, nil
}

// startKeymap opens the shortcut config modal.
func (m *Model) startKeymap() {
	m.keymapCursor = 0
	m.keymapCapturing = false
	m.keymapConflict = ""
	m.modalPlaced = false
	m.mode = modeKeymap
}

// rebind binds action id to key key, with a guard for reserved keys and conflict
// detection (without overwriting another action). Persists when it applies.
func (m *Model) rebind(id, key string) {
	if key == "" || key == "esc" || key == "ctrl+c" {
		m.keymapConflict = "reserved"
		return
	}
	for _, a := range keyActions {
		if a.id != id && m.keys[a.id] == key {
			m.keymapConflict = a.id
			return
		}
	}
	m.keys[id] = key
	m.keymapConflict = ""
	m.saveKeymap()
}

// updateKeymap handles the shortcut modal: navigate the list, enter/r captures the
// next key as the new bind, d reverts to the default, esc closes. While capturing,
// any key (except esc) becomes the new bind of the action under the cursor.
func (m *Model) updateKeymap(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.keymapCapturing {
		if k == "esc" {
			m.keymapCapturing = false
			return m, nil
		}
		m.rebind(keyActions[m.keymapCursor].id, k)
		m.keymapCapturing = false
		return m, nil
	}
	switch k {
	case "esc", "q":
		m.mode = modeBoard
	case "j", "down":
		if m.keymapCursor < len(keyActions)-1 {
			m.keymapCursor++
		}
		m.keymapConflict = ""
	case "k", "up":
		if m.keymapCursor > 0 {
			m.keymapCursor--
		}
		m.keymapConflict = ""
	case "enter", "r":
		m.keymapCapturing = true
		m.keymapConflict = ""
	case "d":
		id := keyActions[m.keymapCursor].id
		m.rebind(id, defaultKeyFor(id))
	}
	return m, nil
}

// startTags opens the tag catalog.
func (m *Model) startTags() {
	m.tagCursor = 0
	m.modalPlaced = false
	m.mode = modeTags
}

// hueIndex finds the index of a hue in tagHues (default 0 if empty/unknown).
func hueIndex(name string) int {
	for i, h := range tagHues {
		if h == name {
			return i
		}
	}
	return 0
}

// formColor is the color chosen in the form: the selected preset, or the hex typed
// in the custom slot.
func (m *Model) formColor() string {
	if m.tagColorSel < len(tagHues) {
		return tagHues[m.tagColorSel]
	}
	return strings.TrimSpace(m.colorInput.Value())
}

// openTagForm prepares the form; edit=-1 creates, otherwise edits the tag at that
// index. An existing hex color falls into the custom slot; a named hue selects the swatch.
func (m *Model) openTagForm(edit int) {
	m.tagEditIdx = edit
	name, desc, color := "", "", tagHues[0]
	if edit >= 0 && edit < len(m.cfg.Tags) {
		td := m.cfg.Tags[edit]
		name, desc, color = td.Name, td.Desc, td.Color
	}
	m.input.SetValue(name)
	m.input.Placeholder = msg.phTagName
	m.input.SetWidth(24)
	m.descInput.SetValue(desc)
	m.descInput.Placeholder = msg.phTagDesc
	m.descInput.SetWidth(24)
	m.colorInput.Placeholder = msg.phTagColor
	m.colorInput.SetWidth(10)
	if isHexColor(color) {
		m.tagColorSel = len(tagHues) // custom slot
		m.colorInput.SetValue(color)
	} else {
		m.tagColorSel = hueIndex(color)
		m.colorInput.SetValue("")
	}
	m.tagFieldFocus(0)
	m.modalPlaced = false
	m.mode = modeTagForm
}

// tagFieldFocus moves focus between the 3 fields (only the active input blinks).
func (m *Model) tagFieldFocus(i int) {
	m.tagField = i
	focus := func(in *textinput.Model, on bool) {
		if on {
			in.Focus()
		} else {
			in.Blur()
		}
	}
	focus(&m.input, i == 0)
	focus(&m.descInput, i == 2)
	m.syncColorFocus()
}

// syncColorFocus makes the hex input blink only when the custom slot is active.
func (m *Model) syncColorFocus() {
	if m.tagField == 1 && m.tagColorSel == len(tagHues) {
		m.colorInput.Focus()
	} else {
		m.colorInput.Blur()
	}
}

// commitTagForm saves the form (creates or edits), case-insensitive dedup on
// creation, and returns to the list. Empty name cancels. Empty color becomes mauve.
func (m *Model) commitTagForm() {
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		m.mode = modeTags
		return
	}
	color := m.formColor()
	if color == "" {
		color = tagHues[0]
	}
	td := task.TagDef{Name: name, Color: color, Desc: strings.TrimSpace(m.descInput.Value())}
	if m.tagEditIdx >= 0 && m.tagEditIdx < len(m.cfg.Tags) {
		m.cfg.Tags[m.tagEditIdx] = td // editing overwrites (the catalog is a suggestion)
		m.store.SaveConfig(m.cfg)
	} else if !tagExists(name, m.cfg.Tags) {
		m.cfg.Tags = append(m.cfg.Tags, td)
		m.tagCursor = len(m.cfg.Tags) - 1
		m.store.SaveConfig(m.cfg)
	}
	m.mode = modeTags
}

// updateTags navigates the catalog: a creates, enter/e edits, d deletes.
func (m *Model) updateTags(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeBoard
	case "j", "down":
		if m.tagCursor < len(m.cfg.Tags)-1 {
			m.tagCursor++
		}
	case "k", "up":
		if m.tagCursor > 0 {
			m.tagCursor--
		}
	case "a":
		m.openTagForm(-1)
		return m, textinput.Blink
	case "enter", "e":
		if m.tagCursor < len(m.cfg.Tags) {
			m.openTagForm(m.tagCursor)
			return m, textinput.Blink
		}
	case "d": // deletes the tag under the cursor
		if i := m.tagCursor; i < len(m.cfg.Tags) {
			m.cfg.Tags = append(m.cfg.Tags[:i], m.cfg.Tags[i+1:]...)
			if m.tagCursor >= len(m.cfg.Tags) {
				m.tagCursor = max(0, len(m.cfg.Tags)-1)
			}
			m.store.SaveConfig(m.cfg)
		}
	}
	return m, nil
}

// updateTagForm handles the tag form: tab/↑↓ switch field, in the color field
// ←→/space cycles the hue, enter saves, esc cancels; name/description go to the input.
func (m *Model) updateTagForm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeTags
		return m, nil
	case "enter":
		m.commitTagForm()
		return m, nil
	case "tab", "down":
		m.tagFieldFocus((m.tagField + 1) % 3)
		return m, textinput.Blink
	case "shift+tab", "up":
		m.tagFieldFocus((m.tagField + 2) % 3)
		return m, textinput.Blink
	}
	if m.tagField == 1 { // color field: ←→ moves through the swatches (+ hex slot); typing edits the hex
		n := len(tagHues) + 1 // presets + custom slot
		switch msg.String() {
		case "left":
			m.tagColorSel = (m.tagColorSel + n - 1) % n
			m.syncColorFocus()
			return m, nil
		case "right":
			m.tagColorSel = (m.tagColorSel + 1) % n
			m.syncColorFocus()
			return m, nil
		}
		// any text edit → jumps to the hex slot and types there
		m.tagColorSel = len(tagHues)
		m.syncColorFocus()
		var cmd tea.Cmd
		m.colorInput, cmd = m.colorInput.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	if m.tagField == 0 {
		m.input, cmd = m.input.Update(msg)
	} else {
		m.descInput, cmd = m.descInput.Update(msg)
	}
	return m, cmd
}

// startCardTags opens the tag checklist for the card under the cursor. No card, no-op.
func (m *Model) startCardTags() {
	t := m.current()
	if t == nil {
		return
	}
	m.cardTagID = t.ID
	m.cardTagCursor = 0
	m.modalPlaced = false
	m.mode = modeCardTags
}

// cardTagRows are the checklist rows: the catalog tags (in order) plus the tags the
// card already has that aren't in the catalog (so they can be unchecked).
func (m *Model) cardTagRows() []string {
	var rows []string
	seen := map[string]bool{}
	for _, td := range m.cfg.Tags {
		rows = append(rows, td.Name)
		seen[strings.ToLower(td.Name)] = true
	}
	if t := m.store.Get(m.cardTagID); t != nil {
		for _, tg := range t.Tags {
			if !seen[strings.ToLower(tg)] {
				rows = append(rows, tg)
				seen[strings.ToLower(tg)] = true
			}
		}
	}
	return rows
}

// toggleCardTag checks/unchecks (case-insensitive) a tag on the card and persists.
func (m *Model) toggleCardTag(name string) {
	t := m.store.Get(m.cardTagID)
	if t == nil {
		return
	}
	for i, tg := range t.Tags {
		if strings.EqualFold(tg, name) {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			m.store.Save(t)
			m.reload()
			return
		}
	}
	t.Tags = append(t.Tags, name)
	m.store.Save(t)
	m.reload()
}

// updateCardTags navigates the checklist and toggles the tag under the cursor (space/enter).
func (m *Model) updateCardTags(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := m.cardTagRows()
	switch msg.String() {
	case "esc", "q":
		m.mode = modeBoard
	case "j", "down":
		if m.cardTagCursor < len(rows)-1 {
			m.cardTagCursor++
		}
	case "k", "up":
		if m.cardTagCursor > 0 {
			m.cardTagCursor--
		}
	case " ", "enter", "x":
		if m.cardTagCursor < len(rows) {
			m.toggleCardTag(rows[m.cardTagCursor])
		}
	}
	return m, nil
}

// move sends the selected card dir columns to the side and follows with the cursor
// to the destination column. If the transition has a configured action (F16), it
// fires the command and returns the loader cmd; otherwise it changes the status right away.
func (m *Model) move(dir int) tea.Cmd {
	t := m.current()
	if t == nil {
		return nil
	}
	dst := m.col + dir
	if dst < 0 || dst >= len(m.columns) {
		return nil
	}
	cmd, moved := m.applyMove(t, dst)
	if moved { // immediate move (no action): follow with the cursor
		m.col = dst
		m.ensureColVisible()
	}
	return cmd
}

// columnAction resolves ONE side of a transition: the on_enter (enter=true) or the
// on_exit (enter=false) of column col. On the same side, INTENTION (agent) wins over
// command (script). agent=true → text is an intention for the headless agent; false →
// the script path. ok=false if the column has no action on that side.
func (m *Model) columnAction(col string, enter bool) (text string, agent, ok bool) {
	a, has := m.store.BindingFor(m.activeBoardID(), col)
	if !has {
		return "", false, false
	}
	if enter {
		if a.OnEnter != "" {
			return a.OnEnter, true, true
		}
		if a.OnEnterCmd != "" {
			return a.OnEnterCmd, false, true
		}
		return "", false, false
	}
	if a.OnExit != "" {
		return a.OnExit, true, true
	}
	if a.OnExitCmd != "" {
		return a.OnExitCmd, false, true
	}
	return "", false, false
}

// applyMove applies (or starts) the move of card t to column dst. Returns the
// loader cmd when there's an async action, and moved=true when the status already
// changed right away (no action). It's the single move point — keyboard and drag go through here.
func (m *Model) applyMove(t *task.Task, dst int) (tea.Cmd, bool) {
	if t == nil || dst < 0 || dst >= len(m.columns) {
		return nil, false
	}
	from, to := t.Status, m.columns[dst]
	if from == to {
		return nil, false
	}
	// exit→enter chaining: run the origin's on_exit FIRST, then the destination's
	// on_enter, committing the move only when the last one succeeds. If exit fails, the
	// enter never runs and the card stays at the origin (same as any failed action).
	exitText, exitAgent, hasExit := m.columnAction(from, false)
	enterText, enterAgent, hasEnter := m.columnAction(to, true)
	if hasExit || hasEnter {
		if m.action != nil && !m.action.failed {
			return nil, false // one action at a time; ignore until the current one finishes
		}
		if hasExit {
			var next *pendingAction
			if hasEnter {
				next = &pendingAction{text: enterText, agent: enterAgent}
			}
			return m.startAction(t, from, to, exitText, exitAgent, next), false
		}
		return m.startAction(t, from, to, enterText, enterAgent, nil), false // commit comes on success
	}
	// no action: immediate local move. Applies to a local card AND to a mirror — on a
	// mirror only the POSITION is the user's (organize freely, e.g. an own QA column);
	// the content stays read-only. The next sync of the bound column may pull it back
	// if Jira still lists it (the local position is reasserted by Jira's truth).
	t.Status = to
	if err := m.store.Save(t); err != nil {
		return nil, false // ponytail: a save failure becomes a no-op in v1; error handling waits for F13
	}
	m.reload()
	return nil, true
}

func cardLines(s string) int { return strings.Count(s, "\n") + 1 }

// colAt maps a screen column x → board column index (with the horizontal scroll in
// cells added in).
func (m *Model) colAt(x int) int {
	if m.colW <= 0 || len(m.columns) == 0 {
		return 0
	}
	return max(0, min((m.hOff+x)/m.colW, len(m.columns)-1))
}

// colOverflows reports whether column col rendered more card lines than the visible area
// at the last render (i.e. it can scroll vertically).
func (m *Model) colOverflows(col int) bool {
	return col >= 0 && col < len(m.colTotal) && m.colTotal[col] > m.bodyH
}

// scrollColBy scrolls column col vertically by delta cards, moving its cursor (so the
// scroll also selects — the window follows the cursor at render). It makes col active so
// the scroll shows. Boundary rule: at the top an "up" is ignored and at the bottom a
// "down" is ignored (only the opposite direction works), so over-scroll can't accumulate.
// Returns false when the column fits on screen (nothing to scroll) → the caller pans.
func (m *Model) scrollColBy(col, delta int) bool {
	if col < 0 || col >= len(m.cols) || !m.colOverflows(col) {
		return false
	}
	m.col = col
	if r := m.row[col] + delta; r >= 0 && r < len(m.cols[col]) {
		m.row[col] = r // otherwise at a boundary in this direction → ignore
	}
	return true
}

// totalW is the total board width (all columns) in cells.
func (m *Model) totalW() int { return m.colW * len(m.columns) }

// clampHOff keeps the horizontal scroll within [0, total-view].
func (m *Model) clampHOff() {
	m.hOff = max(0, min(m.hOff, max(0, m.totalW()-m.viewW)))
}

// ensureColVisible slides the scroll the minimum for the active column to fit in the
// window (called when moving the cursor with the keyboard; the mouse on the track moves hOff directly).
func (m *Model) ensureColVisible() {
	if m.colW <= 0 || m.viewW <= 0 {
		return
	}
	if lo := m.col * m.colW; lo < m.hOff {
		m.hOff = lo
	}
	if hi := (m.col + 1) * m.colW; hi > m.hOff+m.viewW {
		m.hOff = hi - m.viewW
	}
	m.clampHOff()
}

// scrollTrackTo scrolls the board via click/drag on the track: maps x (track cell)
// straight to the offset in cells, centering the thumb on the pointer. Since hOff
// is cell-based, the drag slides smoothly instead of jumping column by column.
func (m *Model) scrollTrackTo(x int) {
	span := m.totalW() - m.viewW
	if m.trackW <= 0 || m.viewW <= 0 || span <= 0 {
		return
	}
	thumb := max(1, m.viewW*m.viewW/m.totalW()) // thumb in cells
	m.hOff = (x - thumb/2) * span / max(1, m.trackW-thumb)
	m.clampHOff()
}

// cardAt returns the card under (x,y) on the board (column by width, row by hitbox).
func (m *Model) cardAt(x, y int) (cardHit, bool) {
	c := m.colAt(x)
	for _, r := range m.cardRegions {
		if r.col == c && y >= r.y0 && y < r.y1 {
			return r, true
		}
	}
	return cardHit{}, false
}

// dropCard drops the dragged card on the target column. Goes through applyMove: fires
// the transition action (with loader) if there is one, otherwise changes the status right away.
func (m *Model) dropCard(targetCol int) tea.Cmd {
	t := m.store.Get(m.dragCardID)
	if t == nil {
		return nil
	}
	cmd, moved := m.applyMove(t, targetCol)
	if moved {
		m.col = targetCol
	}
	return cmd
}

// updateInput handles the input shared by creating a task (modeAdd), creating a board
// (modeNewBoard), naming a column (modeAddColumn) and renaming a board (modeRenameBoard).
// Enter acts per mode; add-column/rename return to the config, the rest to the board.
func (m *Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.modalPlaced = false
		switch m.mode {
		case modeAddColumn:
			m.mode = modeColumns
		case modeRenameBoard, modeRenameKey:
			m.mode = modeBoardConfig
		case modeRenameColumn:
			m.mode = modeLaneConfig
		default:
			m.mode = modeBoard
		}
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		switch m.mode {
		case modeAdd:
			if val != "" {
				// born on the active board (project = board id; "" = Inbox)
				m.store.Save(&task.Task{Title: val, Status: m.firstColumn(), Project: m.activeBoardID()})
				m.reload()
			}
			m.mode = modeBoard
		case modeAddSubtask:
			if val != "" && m.subParent != "" {
				// child: Parent links to the root (Project inherited); born in the 1st column
				m.store.Save(&task.Task{Title: val, Status: m.firstColumn(), Parent: m.subParent})
				m.reload()
			}
			m.mode = modeBoard
		case modeNewBoard:
			if slug := slugify(val); val != "" && slug != "" {
				cols := []string{} // a new board is born with no columns (config opens on its own)
				key := m.store.UniqueKey(val, slug)
				m.store.SaveBoard(&task.Board{ID: slug, Name: val, Key: key, Columns: &cols})
				m.reload()
				m.openBoard(slug) // openBoard → maybeConfigEmpty opens the config
			} else {
				m.mode = modeBoard
			}
		case modeAddColumn:
			if val != "" {
				m.columns = append(m.columns, val)
				m.cfgCursor = len(m.columns) - 1
				m.saveBoardColumns()
				m.reload()
			}
			m.modalPlaced = false
			m.mode = modeColumns
		case modeRenameBoard:
			if val != "" {
				b := m.boardCopy(m.activeBoardID())
				b.Name = val
				m.store.SaveBoard(b)
				m.reload()
			}
			m.modalPlaced = false
			m.mode = modeBoardConfig
		case modeRenameColumn:
			m.renameColumn(m.laneIdx, val)
			m.modalPlaced = false
			m.mode = modeLaneConfig
		case modeRenameKey:
			if val != "" {
				m.store.RenameBoardKey(m.activeBoardID(), val) // no-op if it collides/is empty
				m.reload()
			}
			m.modalPlaced = false
			m.mode = modeBoardConfig
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// firstColumn is the column where a new task is born (the first on the board).
func (m *Model) firstColumn() string {
	if len(m.columns) > 0 {
		return m.columns[0]
	}
	return task.StatusBacklog
}

// boardCopy returns an editable copy of board id (preserving Columns/Created). A
// discovered board (no file) becomes a new Board — saving creates the file.
func (m *Model) boardCopy(id string) *task.Board {
	for _, b := range m.boards {
		if b.ID == id {
			return &task.Board{ID: b.ID, Name: b.Name, Key: b.Key, Columns: b.Columns, Created: b.Created}
		}
	}
	return &task.Board{ID: id, Name: id}
}

// saveBoardColumns persists the column list/order of the active board (no-op on Inbox).
func (m *Model) saveBoardColumns() {
	if m.activeBoardID() == task.InboxID {
		return
	}
	b := m.boardCopy(m.activeBoardID())
	cols := append([]string(nil), m.columns...)
	b.Columns = &cols
	m.store.SaveBoard(b)
}

// --- per-column (lane) config ---

// openLaneConfig opens the column menu of column i (☰ / key g).
func (m *Model) openLaneConfig(i int) {
	if i < 0 || i >= len(m.columns) {
		return
	}
	m.laneIdx = i
	m.laneCursor = 0
	m.modalPlaced = false
	m.mode = modeLaneConfig
}

// laneMenu is the column menu (☰): "Filtros" (only when the board has a filters registry),
// then rename and delete. Order matters — updateLaneConfig indexes into it.
func (m *Model) laneMenu() []string {
	items := []string{}
	if len(m.store.BoardFilterNames(m.activeBoardID())) > 0 {
		items = append(items, msg.laneFilters)
	}
	return append(items, msg.tRenameColumn, msg.laneDelete)
}

// updateLaneConfig drives the column menu: navigate + enter selects (Filtros / rename / delete).
func (m *Model) updateLaneConfig(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.laneMenu()
	switch k.String() {
	case "esc", "q", "g":
		m.mode = modeBoard
	case "j", "down":
		if m.laneCursor < len(items)-1 {
			m.laneCursor++
		}
	case "k", "up":
		if m.laneCursor > 0 {
			m.laneCursor--
		}
	case "enter":
		if m.laneCursor >= len(items) {
			return m, nil
		}
		switch items[m.laneCursor] {
		case msg.laneFilters:
			m.openLaneFilters(m.laneIdx)
		case msg.tRenameColumn:
			m.startRenameColumn()
			return m, textinput.Blink
		case msg.laneDelete:
			m.deleteColumn(m.laneIdx)
			m.mode = modeBoard
		}
	}
	return m, nil
}

// startRenameColumn opens the input pre-filled with the focused column's name.
func (m *Model) startRenameColumn() {
	if m.laneIdx < 0 || m.laneIdx >= len(m.columns) {
		return
	}
	m.input.SetValue(m.columns[m.laneIdx])
	m.input.Placeholder = msg.phColumn
	m.input.SetWidth(40)
	m.input.Focus()
	m.modalPlaced = false
	m.mode = modeRenameColumn
}

// renameColumn renames column i and resets the status of the tasks that were in it
// (Status references the column's text). No-op if the name collides with another
// column — otherwise two columns would have the same status.
func (m *Model) renameColumn(i int, name string) {
	name = strings.TrimSpace(name)
	if i < 0 || i >= len(m.columns) || name == "" {
		return
	}
	old := m.columns[i]
	if name == old || indexOf(m.columns, name) >= 0 {
		return
	}
	for _, t := range m.store.BoardTasks(m.activeBoardID()) {
		if t.Status == old {
			t.Status = name
			m.store.Save(t)
		}
	}
	m.columns[i] = name
	m.saveBoardColumns()
	m.reload()
}

// updateBoardList navigates the board-list modal.
func (m *Model) updateBoardList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "b":
		m.mode = modeBoard
	case "j", "down":
		if m.listCursor < len(m.boards)-1 {
			m.listCursor++
		}
	case "k", "up":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "enter":
		if m.listCursor < len(m.boards) {
			m.openBoard(m.boards[m.listCursor].ID)
		}
		m.mode = modeBoard
	}
	return m, nil
}

// items of the board config main menu (index = order in the list).
const (
	miColumns = iota // opens the column sub-screen
	miRenameBoard
	miRenameKey
	miDelete
	miCount
)

// updateBoardConfig navigates the config's main MENU: Columns / Rename board /
// Identifier / Delete. Enter chooses the item under the cursor.
func (m *Model) updateBoardConfig(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "c":
		m.mode = modeBoard
	case "j", "down":
		if m.menuCursor < miCount-1 {
			m.menuCursor++
		}
	case "k", "up":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "enter", " ", "l", "right":
		switch m.menuCursor {
		case miColumns:
			m.cfgCursor, m.cfgGrab = 0, false
			m.mode = modeColumns
		case miRenameBoard:
			m.startRenameBoard()
			return m, textinput.Blink
		case miRenameKey:
			m.startRenameKey()
			return m, textinput.Blink
		case miDelete:
			m.modalPlaced, m.hoverBtn = false, 0 // "No" selected by default
			m.mode = modeConfirmDelete
		}
	}
	return m, nil
}

// updateColumns navigates the column sub-screen (the config's old behavior):
// move the cursor, "grab" (cfgGrab) to reorder with the arrows, add, delete.
// esc returns to the main menu.
func (m *Model) updateColumns(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	n := len(m.columns) // index n = the fixed "+ new column" row (last selectable)
	onCreate := m.cfgCursor == n
	switch msg.String() {
	case "esc", "q":
		if m.cfgGrab {
			m.cfgGrab = false
			return m, nil
		}
		m.mode = modeBoardConfig
	case "j", "down":
		switch {
		case m.cfgGrab && m.cfgCursor < n-1: // drags the column down (doesn't go past the last)
			m.columns[m.cfgCursor], m.columns[m.cfgCursor+1] = m.columns[m.cfgCursor+1], m.columns[m.cfgCursor]
			m.cfgCursor++
			m.saveBoardColumns()
			m.reload()
		case !m.cfgGrab && m.cfgCursor < n: // up to the create row (index n)
			m.cfgCursor++
		}
	case "k", "up":
		switch {
		case m.cfgGrab && m.cfgCursor > 0:
			m.columns[m.cfgCursor], m.columns[m.cfgCursor-1] = m.columns[m.cfgCursor-1], m.columns[m.cfgCursor]
			m.cfgCursor--
			m.saveBoardColumns()
			m.reload()
		case !m.cfgGrab && m.cfgCursor > 0:
			m.cfgCursor--
		}
	case "enter", " ":
		if onCreate { // "+ new column" row
			m.startAddColumn()
			return m, textinput.Blink
		}
		if n > 0 {
			m.cfgGrab = !m.cfgGrab
		}
	case "a":
		m.startAddColumn()
		return m, textinput.Blink
	case "d", "x":
		if !onCreate && n > 0 {
			m.deleteColumn(m.cfgCursor)
		}
	}
	return m, nil
}

// updateConfirmDelete: navigates the buttons (arrows/hjkl move, enter/space confirms
// the focused one) and accepts direct shortcuts (y/s = yes, n = no). Yes → deleteActiveBoard;
// No/esc → returns to config.
func (m *Model) updateConfirmDelete(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	cancel := func() { m.modalPlaced = false; m.mode = modeBoardConfig }
	switch msg.String() {
	case "h", "left", "k", "up":
		m.hoverBtn = 0 // No
	case "l", "right", "j", "down":
		m.hoverBtn = 1 // Yes
	case "tab":
		m.hoverBtn ^= 1 // toggle
	case "enter", " ":
		if m.hoverBtn == 1 {
			m.deleteActiveBoard()
		} else {
			cancel()
		}
	case "y", "s":
		m.deleteActiveBoard()
	case "n":
		cancel()
	case "esc", "q":
		cancel()
	}
	return m, nil
}

// openBoardConfig opens the active board's config (no-op if no board is open).
func (m *Model) openBoardConfig() {
	if m.activeBoardID() == "" {
		return
	}
	m.menuCursor = 0
	m.modalPlaced = false
	m.mode = modeBoardConfig
}

// startAddColumn opens the input to name a new column.
func (m *Model) startAddColumn() {
	m.input.SetValue("")
	m.input.Placeholder = msg.phColumn
	m.input.SetWidth(40)
	m.input.Focus()
	m.modalPlaced = false
	m.mode = modeAddColumn
}

// startRenameBoard opens the input pre-filled with the board's current name.
func (m *Model) startRenameBoard() {
	m.input.SetValue(m.store.BoardName(m.activeBoardID()))
	m.input.Placeholder = msg.phBoard
	m.input.SetWidth(40)
	m.input.Focus()
	m.modalPlaced = false
	m.mode = modeRenameBoard
}

// startRenameKey opens the input pre-filled with the board's current key.
func (m *Model) startRenameKey() {
	m.input.SetValue(m.boardCopy(m.activeBoardID()).Key)
	m.input.Placeholder = msg.phKey
	m.input.SetWidth(40)
	m.input.Focus()
	m.modalPlaced = false
	m.mode = modeRenameKey
}

// deleteColumn removes column i from the active board. Tasks that were in it end up
// with an orphan status and fall into column 0 on the next grouping (they don't disappear).
func (m *Model) deleteColumn(i int) {
	if i < 0 || i >= len(m.columns) {
		return
	}
	m.columns = append(m.columns[:i], m.columns[i+1:]...)
	if m.cfgCursor > len(m.columns) { // allows stopping on the "+ new column" row (index = len)
		m.cfgCursor = len(m.columns)
	}
	m.cfgGrab = false
	m.saveBoardColumns()
	m.reload()
}

// deleteActiveBoard archives all tasks of the active board, deletes the board and
// closes the tab. Inbox is not deletable.
func (m *Model) deleteActiveBoard() {
	id := m.activeBoardID()
	if id == "" {
		return
	}
	n := 0
	for _, t := range m.store.BoardTasks(id) {
		if err := m.store.Archive(t.ID); err == nil {
			n++
		}
	}
	m.store.DeleteBoard(id)
	// removes the board from the open tabs; with no board left, falls into the "create board" state
	open := m.open[:0]
	for _, o := range m.open {
		if o != id {
			open = append(open, o)
		}
	}
	m.open = open
	if m.active >= len(m.open) {
		m.active = max(0, len(m.open)-1)
	}
	m.resetCursor()
	m.reload()
	m.persist()
	m.showNotice(fmt.Sprintf(msg.nvBoardDeleted, n), modeBoard)
}

// --- view ---
// The styles and theme palette live in themes.go (rebuilt by applyPalette).

// View wraps the content in a tea.View with alt screen on (v2: alt screen is a
// field of View, no longer a program option).
func (m *Model) View() tea.View {
	// all-motion wherever hover drives the pointer shape: the board (card → open hand),
	// the confirmation (button hover) and the menu modals (row → clickable hand). Hover
	// has no pressed button, so cell-motion wouldn't report it. Plain input modals keep
	// cell-motion (a drag already comes with a pressed button).
	mouse := tea.MouseModeCellMotion
	_, _, menu := m.currentMenu()
	if m.mode == modeBoard || m.mode == modeConfirmDelete || m.mode == modeDetail || menu {
		mouse = tea.MouseModeAllMotion
	}
	return tea.View{Content: m.render(), AltScreen: true, MouseMode: mouse}
}

func (m *Model) render() string {
	// mode changed → re-center the modal on the next measurement (each new panel opens
	// centered; dragging preserves the position while the mode doesn't change).
	if m.mode != m.renderedMode {
		m.modalPlaced = false
		m.renderedMode = m.mode
	}
	switch m.mode {
	case modeDetail:
		help := msg.hDetail
		if t := m.detailTask(); t != nil && len(m.detailRelated(t)) > 0 {
			help = msg.hDetailSub // navigates parent/subtasks + opens
		}
		// maps the navigable rows (content) to a screen row, subtracting the scroll —
		// vp.View() starts at screen y=0. Feeds the mouse click.
		m.detailHits = m.detailHits[:0]
		for _, r := range m.detailRows {
			if y := r.line - m.vp.YOffset(); y >= 0 && y < m.vp.Height() {
				m.detailHits = append(m.detailHits, detailHit{id: r.id, y: y})
			}
		}
		if !(m.vp.AtTop() && m.vp.AtBottom()) { // only shows % if it can scroll
			help = fmt.Sprintf("%3.0f%% · %s", m.vp.ScrollPercent()*100, help)
		}
		bar := vscroll(m.vp.TotalLineCount(), m.vp.Height(), m.vp.YOffset())
		body := lipgloss.JoinHorizontal(lipgloss.Top, m.vp.View(), bar)
		return body + "\n" + helpStyle.Render("  "+help)
	case modeAdd:
		body := faint.Render(msg.boardPrefix+m.store.BoardName(m.activeBoardID())) + "\n\n" + m.input.View()
		return m.overlay(modalBox(msg.tNewTask, body, msg.hAdd))
	case modeAddSubtask:
		parent := m.store.Get(m.subParent)
		ctx := m.subParent
		if parent != nil {
			ctx += " · " + parent.Title
		}
		body := faint.Render("↳ "+ctx) + "\n\n" + m.input.View()
		return m.overlay(modalBox(msg.tNewSubtask, body, msg.hAdd))
	case modeNewBoard:
		return m.overlay(modalBox(msg.tNewBoard, m.input.View(), msg.hAdd))
	case modeRenameBoard:
		return m.overlay(modalBox(msg.tRenameBoard, m.input.View(), msg.hAdd))
	case modeAddColumn:
		return m.overlay(modalBox(msg.tAddColumn, m.input.View(), msg.hAdd))
	case modeBoardConfig:
		return m.overlay(m.boardConfigBox())
	case modeColumns:
		return m.overlay(m.columnsBox())
	case modeLaneConfig:
		return m.overlay(m.laneConfigBox())
	case modeLaneFilters:
		return m.overlay(m.laneFiltersBox())
	case modeRenameColumn:
		return m.overlay(modalBox(msg.tRenameColumn, m.input.View(), msg.hAdd))
	case modeRenameKey:
		return m.overlay(modalBox(msg.tRenameKey, m.input.View(), msg.hAdd))
	case modeFilter:
		return m.overlay(modalBox(msg.tFilter, m.filterBox(), msg.hFilter))
	case modeKeymap:
		return m.overlay(m.keymapBox())
	case modeTags:
		return m.overlay(m.tagsBox())
	case modeTagForm:
		return m.overlay(m.tagFormBox())
	case modeCardTags:
		return m.overlay(m.cardTagsBox())
	case modeBoardList:
		return m.overlay(m.boardListBox())
	case modeSettings:
		return m.overlay(m.settingsBox())
	case modePicker:
		return m.overlay(m.pickerBox())
	case modeDirBrowser:
		return m.overlay(m.dirBrowserBox())
	case modeConfirmMove:
		return m.overlay(m.confirmMoveBox())
	case modeConfirmDelete:
		return m.overlay(m.confirmDeleteBox())
	case modeNotice:
		return m.overlay(m.noticeBox())
	default:
		if m.draggingCard {
			return m.boardWithGhost()
		}
		if m.menuOpen {
			return m.boardWithMenu()
		}
		return m.boardView()
	}
}

// boardWithGhost composes the board with the dragged card (ghost + shadow) following
// the cursor, keeping the grab point under the pointer.
func (m *Model) boardWithGhost() string {
	board := m.boardView() // also repopulates colW/cardRegions
	gw, gh := lipgloss.Width(m.dragGhost), lipgloss.Height(m.dragGhost)
	gx := max(0, min(m.dragX-m.grabDX, m.w-gw))
	gy := max(0, min(m.dragY-m.grabDY, m.h-gh))
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(board),
		lipgloss.NewLayer(shadowBlock(gw, gh)).X(gx+2).Y(gy+1).Z(1), // same as the modal: right + below, no gap
		lipgloss.NewLayer(m.dragGhost).X(gx).Y(gy).Z(2),
	)
	return comp.Render()
}

// boardWithMenu composes the board with the button menu anchored to whatever opened it. A
// COLUMN menu opens to the RIGHT of the footer button and grows UP, so its last row lands on
// the button's own line; the TOP BAR menu hangs straight DOWN from its button, right-aligned
// with it. Both mirror sideways when there's no room and clamp to the screen. Not a modal:
// no title bar, no dots, no dragging — it's a popover.
func (m *Model) boardWithMenu() string {
	board := m.boardView() // also repopulates syncRegions/syncY/syncAllReg
	reg, ok := m.menuAnchor()
	if !ok {
		m.menuOpen = false // anchor scrolled out / lost its buttons
		return board
	}
	box := m.menuBox()
	w, h := lipgloss.Width(box), lipgloss.Height(box)
	var x, y int
	if m.menuUp { // column footer
		x = reg.x1 // to the right of the button
		if x+w > m.w {
			x = max(0, reg.x0-w) // no room → mirrors to the left
		}
		// the LAST row (the first button, the one `y` fires) lands on the button's own line,
		// so the eye goes straight from the footer to the default choice.
		y = max(0, min(m.syncY-h+2, m.h-h))
	} else { // top bar: hangs from the button, right edges aligned
		x = max(0, min(reg.x1-w, m.w-w))
		y = 1
	}
	m.menuX, m.menuY = x+1, y+1 // content starts after the border
	m.menuW = w - 2
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(board),
		lipgloss.NewLayer(shadowBlock(w, h)).X(x+2).Y(y+1).Z(1),
		lipgloss.NewLayer(box).X(x).Y(y).Z(2),
	)
	return comp.Render()
}

// menuAnchor is the hitbox the open menu hangs from: the column's footer button, or the top
// bar button for the board menu.
func (m *Model) menuAnchor() (clickRegion, bool) {
	if m.menuCol == boardMenuCol {
		return m.syncAllReg, m.syncAllReg.x1 > 0
	}
	if m.syncY == 0 {
		return clickRegion{}, false
	}
	return m.footerRegion(m.menuCol)
}

// menuButtons are the buttons behind the open menu (a column's, or the board's).
func (m *Model) menuButtons() []colButton {
	if m.menuCol == boardMenuCol {
		return m.boardButtons()
	}
	if m.menuCol < 0 || m.menuCol >= len(m.columns) {
		return nil
	}
	return m.columnButtons(m.columns[m.menuCol])
}

// menuRowOf maps a button index to its screen row (and back — the mapping is its own
// inverse): an upward menu is drawn in reverse, a downward one in config order. Either way
// the first button ends up next to the anchor.
func (m *Model) menuRowOf(i int) int {
	if m.menuUp {
		return m.menuRows - 1 - i
	}
	return i
}

// menuBox draws the popover: one row per button, ordered by menuRowOf.
func (m *Model) menuBox() string {
	bs := m.menuButtons()
	m.menuRows = len(bs)
	lw, iw := 0, 0
	for _, b := range bs {
		lw = max(lw, lipgloss.Width(b.Label))
		iw = max(iw, lipgloss.Width(b.Icon))
	}
	rows := make([]string, len(bs))
	for i, b := range bs {
		mark, style := "  ", lipgloss.NewStyle()
		if i == m.menuSel {
			mark, style = "› ", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
		}
		line := mark + b.Label + strings.Repeat(" ", lw-lipgloss.Width(b.Label))
		if iw > 0 {
			line += "  " + b.Icon + strings.Repeat(" ", iw-lipgloss.Width(b.Icon))
		}
		rows[m.menuRowOf(i)] = style.Render(line)
	}
	return paneBox.Render(strings.Join(rows, "\n"))
}

// footerRegion is the hitbox of column i's footer button, if it's on screen.
func (m *Model) footerRegion(i int) (clickRegion, bool) {
	for _, r := range m.syncRegions {
		if r.idx == i {
			return r, true
		}
	}
	return clickRegion{}, false
}

// menuHit maps a screen point to the BUTTON index under it in the open menu (-1 = outside).
func (m *Model) menuHit(x, y int) int {
	if !m.menuOpen || x < m.menuX || x >= m.menuX+m.menuW {
		return -1
	}
	row := y - m.menuY
	if row < 0 || row >= m.menuRows {
		return -1
	}
	return m.menuRowOf(row) // the mapping is its own inverse
}

// fireMenu closes the open menu and fires button n of whatever it belongs to.
func (m *Model) fireMenu(n int) tea.Cmd {
	col := m.menuCol
	m.menuOpen = false
	if col == boardMenuCol {
		return m.triggerBoardButton(n)
	}
	return m.triggerButton(col, n)
}

// overlay composes a modal (box) centered OVER the board, with lipgloss v2's native
// compositor (Canvas + Layer): a real modal, the board visible behind it.
func (m *Model) overlay(box string) string {
	w, h := m.w, m.h
	if w == 0 {
		w, h = 90, 30
	}
	m.modalW, m.modalH = lipgloss.Width(box), lipgloss.Height(box)
	if !m.modalPlaced { // centers on open; after that the drag rules
		m.modalX = max(0, (w-m.modalW)/2)
		m.modalY = max(0, (h-m.modalH)/2)
		m.modalPlaced = true
	}
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(m.boardView()), // background (z 0)
		lipgloss.NewLayer(shadowBlock(m.modalW, m.modalH)).X(m.modalX+2).Y(m.modalY+1).Z(1), // shadow
		lipgloss.NewLayer(box).X(m.modalX).Y(m.modalY).Z(2),                                 // modal
		lipgloss.NewLayer(" "+trafficLights()+" ").X(m.modalX+2).Y(m.modalY).Z(3),           // dots on the border, with slack
	)
	return comp.Render()
}

// shadowBlock is a WxH block of dark cells; offset beneath the modal, only the
// bottom-right border stays visible → a shadow "floating above the card".
func shadowBlock(w, h int) string {
	row := shadowStyle.Render(strings.Repeat(" ", w))
	rows := make([]string, h)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

// modalBox builds the standard frame of every modal: a title bar with the dots on
// the left (red closes, via onModalClose) and the centered title, then body + help.
func modalBox(title, body, help string) string {
	titleR := cardTitle.Render(title)
	helpR := helpStyle.Render(help)
	cw := max(lipgloss.Width(body), lipgloss.Width(helpR), lipgloss.Width(titleR))
	// The dots are drawn over the top border in overlay(); here just the centered
	// title + body + help.
	titleLine := lipgloss.PlaceHorizontal(cw, lipgloss.Center, titleR)
	return cardBox.Render(titleLine + "\n\n" + body + "\n\n" + helpR)
}

// trafficLights are the 3 macOS dots (lit red = close).
func trafficLights() string {
	return dotRed.Render("●") + " " + dotOff.Render("●") + " " + dotOff.Render("●")
}

// boardListBox is the content of the board-list modal (• marks the open ones).
func (m *Model) boardListBox() string {
	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
	var b strings.Builder
	for i, bd := range m.boards {
		mark := "○" // matches the filter modal: ◉ = open board, ○ = closed
		if m.isOpen(bd.ID) {
			mark = "◉"
		}
		line := fmt.Sprintf("%s %s", mark, bd.Name)
		if i == m.listCursor {
			line = accent.Render("› " + line)
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	return modalBox(msg.tBoards, strings.TrimRight(b.String(), "\n"), msg.hList)
}

// boardConfigBox is the board config modal: header with the name + column list
// (cursor ›; ↕ marks the column grabbed for reordering).
// boardConfigBox is the config's main MENU: Columns / Rename board / Identifier /
// Delete. The item under the cursor is highlighted (Delete in red).
func (m *Model) boardConfigBox() string {
	key := m.boardCopy(m.activeBoardID()).Key
	items := []string{
		fmt.Sprintf("%s (%d)", msg.cfgColumns, len(m.columns)),
		msg.tRenameBoard,
		fmt.Sprintf("%s (%s)", msg.cfgIdent, key),
		msg.cfgDelete,
	}
	var b strings.Builder
	b.WriteString(faint.Render(m.store.BoardName(m.activeBoardID())) + "\n\n")
	for i, it := range items {
		cursor := "  "
		if i == m.menuCursor {
			cursor = "› "
		}
		line := cursor + it
		if i == m.menuCursor {
			c := pal.accent
			if i == miDelete {
				c = pal.red
			}
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(c)).Render(line)
		}
		b.WriteString(line + "\n")
	}
	return modalBox(msg.tBoardCfg, strings.TrimRight(b.String(), "\n"), msg.hBoardMenu)
}

// keymapVisible is the max number of shortcut rows shown at once in the modal
// (excluding headers/spacing between groups).
const keymapVisible = 14

// groupLabel is the i18n label of an action group's section.
func groupLabel(grp string) string {
	if grp == grpNav {
		return msg.grpNavLabel
	}
	return msg.grpCmdLabel
}

// keymapBox is the keybind modal : key → description grouped by
// section (navigation/commands), cursor › on the active one, capture to rebind, and a
// warning line for conflict/reserved. It's also the editor — enter remaps right there.
func (m *Model) keymapBox() string {
	sel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
	// key column (left) standardized by the widest one, to align the descriptions
	kw := 0
	for _, a := range keyActions {
		if w := lipgloss.Width(m.keys[a.id]); w > kw {
			kw = w
		}
	}
	keyCol := lipgloss.NewStyle().Width(kw)

	// flattens into lines with a section header when the group changes; stores the line
	// of the selected action so the window keeps the cursor visible.
	var lines []string
	selLine, lastGrp := 0, ""
	for i, a := range keyActions {
		if a.grp != lastGrp {
			if len(lines) > 0 {
				lines = append(lines, "") // breathing room between groups
			}
			lines = append(lines, kmGroup.Render(groupLabel(a.grp)))
			lastGrp = a.grp
		}
		selected := i == m.keymapCursor
		cursor := "  "
		if selected {
			cursor = "› "
			selLine = len(lines)
		}
		keyR, label := m.keys[a.id], msg.keyLabels[a.id]
		var keyCell string
		switch {
		case m.keymapCapturing && selected:
			keyCell = faint.Render(msg.keymapPress) // full prompt, not framed in the column (avoids wrap)
		case selected:
			keyCell, label = sel.Render(keyCol.Render(keyR)), sel.Render(label)
		default:
			keyCell = keyCol.Render(keyR)
		}
		lines = append(lines, cursor+keyCell+"  "+label)
	}
	body, total, offset, _ := windowColumn(lines, keymapVisible, selLine)
	// scroll hint when there are lines outside the window (above/below)
	parts := []string{strings.TrimRight(body, "\n")}
	if offset > 0 {
		parts = append([]string{faint.Render("  ↑ …")}, parts...)
	}
	if offset+keymapVisible < total {
		parts = append(parts, faint.Render("  ↓ …"))
	}
	body = strings.Join(parts, "\n")

	sub := faint.Render(msg.kmSubtitle)
	// conflict/reserved warning
	warnLine := ""
	if m.keymapConflict == "reserved" {
		warnLine = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(pal.red)).Render(msg.keymapReserved)
	} else if m.keymapConflict != "" {
		warn := fmt.Sprintf(msg.keymapConflict, msg.keyLabels[m.keymapConflict])
		warnLine = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(pal.red)).Render(warn)
	}
	help := msg.hKeymap
	if m.keymapCapturing {
		help = msg.hKeymapCapturing
	}
	return modalBox(msg.tKeymap, sub+"\n\n"+strings.TrimRight(body, "\n")+warnLine, help)
}

// tagsVisible is the max number of tags shown at once in the catalog.
const tagsVisible = 10

// tagsBox is the tag catalog modal: colored chip + description per line, cursor ›.
// The colors are the theme's hues (they follow the theme change).
func (m *Model) tagsBox() string {
	var b strings.Builder
	n := len(m.cfg.Tags)
	if n == 0 {
		b.WriteString(faint.Render(msg.tagsEmpty) + "\n")
	}
	start := 0
	if n > tagsVisible {
		start = max(0, min(m.tagCursor-tagsVisible+1, n-tagsVisible))
		if m.tagCursor < start {
			start = m.tagCursor
		}
	}
	end := min(n, start+tagsVisible)
	if start > 0 {
		b.WriteString(faint.Render("  ↑ …") + "\n")
	}
	for i := start; i < end; i++ {
		td := m.cfg.Tags[i]
		cursor := "  "
		if i == m.tagCursor {
			cursor = "› "
		}
		chip := lipgloss.NewStyle().
			Foreground(lipgloss.Color(pal.panelBg)).Background(lipgloss.Color(hueColor(td.Color))).
			Padding(0, 1).Render(td.Name)
		line := cursor + chip
		if td.Desc != "" {
			line += "  " + faint.Render(td.Desc)
		}
		b.WriteString(line + "\n")
	}
	if end < n {
		b.WriteString(faint.Render("  ↓ …") + "\n")
	}
	return modalBox(msg.tTags, strings.TrimRight(b.String(), "\n"), msg.hTags)
}

// tagFormBox is the tag form: name + color (with chip preview) + description. The
// focused field is highlighted; the color cycles with ←→ in the middle field.
func (m *Model) tagFormBox() string {
	title := msg.tAddTag
	if m.tagEditIdx >= 0 {
		title = msg.tEditTag
	}
	labelCol := lipgloss.NewStyle().Width(maxWidth(msg.fTagName, msg.fTagColor, msg.fTagDesc) + 1)
	sel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
	label := func(s string, focused int, self int) string {
		if focused == self {
			return sel.Render(labelCol.Render(s))
		}
		return faint.Render(labelCol.Render(s))
	}
	// row of preset swatches + hex slot; the selected one gets brackets
	var sw strings.Builder
	for i, h := range tagHues {
		block := lipgloss.NewStyle().Foreground(lipgloss.Color(hueColor(h))).Render("██")
		if m.tagColorSel == i {
			sw.WriteString(sel.Render("[") + block + sel.Render("]"))
		} else {
			sw.WriteString(" " + block + " ")
		}
	}
	hexView := m.colorInput.View()
	if m.tagColorSel == len(tagHues) {
		hexView = sel.Render("[") + hexView + sel.Render("]")
	} else {
		hexView = " " + hexView + " "
	}
	// preview chip: current name (or placeholder) in the chosen color
	name := m.input.Value()
	if strings.TrimSpace(name) == "" {
		name = msg.phTagName
	}
	chip := lipgloss.NewStyle().
		Foreground(lipgloss.Color(pal.panelBg)).Background(lipgloss.Color(hueColor(m.formColor()))).
		Padding(0, 1).Render(name)
	body := label(msg.fTagName, m.tagField, 0) + m.input.View() + "\n" +
		label(msg.fTagColor, m.tagField, 1) + sw.String() + hexView + "  " + chip + "\n" +
		label(msg.fTagDesc, m.tagField, 2) + m.descInput.View()
	return modalBox(title, body, msg.hTagForm)
}

// maxWidth returns the largest display width among the strings (to align labels).
func maxWidth(ss ...string) int {
	w := 0
	for _, s := range ss {
		if x := lipgloss.Width(s); x > w {
			w = x
		}
	}
	return w
}

// tagDescOf returns the description of a tag in the catalog (case-insensitive), or "".
func tagDescOf(name string, cat []task.TagDef) string {
	for _, td := range cat {
		if strings.EqualFold(td.Name, name) {
			return td.Desc
		}
	}
	return ""
}

// cardTagsBox is the card's tag checklist: [x]/[ ] + colored chip + description.
// Checking/unchecking (space) adds/removes the tag on the card.
func (m *Model) cardTagsBox() string {
	rows := m.cardTagRows()
	t := m.store.Get(m.cardTagID)
	title := msg.tCardTags
	if t != nil {
		title += " · " + t.ID
	}
	if m.cardTagCursor >= len(rows) {
		m.cardTagCursor = max(0, len(rows)-1)
	}
	var b strings.Builder
	if len(rows) == 0 {
		return modalBox(title, faint.Render(msg.cardTagsEmpty), msg.hCardTags)
	}
	sel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
	has := func(name string) bool {
		if t == nil {
			return false
		}
		for _, tg := range t.Tags {
			if strings.EqualFold(tg, name) {
				return true
			}
		}
		return false
	}
	// window around the cursor (same pattern as the other lists)
	n := len(rows)
	start := 0
	if n > tagsVisible {
		start = max(0, min(m.cardTagCursor-tagsVisible+1, n-tagsVisible))
		if m.cardTagCursor < start {
			start = m.cardTagCursor
		}
	}
	end := min(n, start+tagsVisible)
	if start > 0 {
		b.WriteString(faint.Render("  ↑ …") + "\n")
	}
	for i := start; i < end; i++ {
		name := rows[i]
		cursor := "  "
		if i == m.cardTagCursor {
			cursor = "› "
		}
		box := "[ ]"
		if has(name) {
			box = sel.Render("[x]")
		}
		line := cursor + box + " " + chip(name, m.cfg.Tags)
		if d := tagDescOf(name, m.cfg.Tags); d != "" {
			line += "  " + faint.Render(d)
		}
		b.WriteString(line + "\n")
	}
	if end < n {
		b.WriteString(faint.Render("  ↓ …") + "\n")
	}
	return modalBox(title, strings.TrimRight(b.String(), "\n"), msg.hCardTags)
}

// colsVisible is the max number of columns shown at once; above that, the list
// scrolls (window around the cursor). Below it comes the fixed "+ new column" row.
const colsVisible = 5

// filterVisible is the max number of results shown at once in the search; above
// that the list scrolls (window around the cursor, same pattern as columnsBox).
const filterVisible = 8

// filterBox is the body of the search overlay: input on top + list of matched cards
// (id, title and the column as breadcrumb). Fumadocs command-palette style.
func (m *Model) filterBox() string {
	var b strings.Builder
	b.WriteString(m.input.View() + "\n")
	// select-style tag field: "Tag: [ <sel> ▾ ]" (tab opens); when open → list.
	if len(m.cfg.Tags) > 0 {
		selLabel := faint.Render(msg.filterAll)
		if m.filterTag != "" {
			selLabel = chip(m.filterTag, m.cfg.Tags)
		}
		b.WriteString(faint.Render(msg.filterTagLabel+"[ ") + selLabel + faint.Render(" ▾ ]") + "\n")
		if m.filterDropOpen {
			sel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
			opt := func(i int, label string) string {
				cur := "  "
				if i == m.filterDropCur {
					cur = sel.Render("▸ ")
				}
				return cur + label
			}
			var opts []string
			opts = append(opts, opt(0, faint.Render(msg.filterAll)))
			for i, td := range m.cfg.Tags {
				opts = append(opts, opt(i+1, chip(td.Name, m.cfg.Tags)))
			}
			drop := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(pal.surface1)).Padding(0, 1).
				Render(strings.Join(opts, "\n"))
			b.WriteString(drop + "\n")
		}
	}
	b.WriteString("\n")
	n := len(m.filterHits)
	if n == 0 {
		b.WriteString(faint.Render(msg.filterEmpty))
		return b.String()
	}
	// window of up to filterVisible results around the cursor
	start := 0
	if n > filterVisible {
		start = max(0, min(m.filterCursor-filterVisible+1, n-filterVisible))
		if m.filterCursor < start {
			start = m.filterCursor
		}
	}
	end := min(n, start+filterVisible)
	if start > 0 {
		b.WriteString(faint.Render("  ↑ …") + "\n")
	}
	for i := start; i < end; i++ {
		h := m.filterHits[i]
		cursor := "  "
		if i == m.filterCursor {
			cursor = "› "
		}
		head := fmt.Sprintf("%s%s  %s", cursor, h.t.ID, h.t.Title)
		if i == m.filterCursor {
			head = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent)).Render(head)
		}
		b.WriteString(head + faint.Render("  · "+m.columns[h.col]) + "\n")
	}
	if end < n {
		b.WriteString(faint.Render("  ↓ …") + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// columnsBox is the column sub-screen (the menu's "Columns" item). Up to 5 lines with
// scroll; cursor ›; ↕ marks the column grabbed for reordering; fixed row to create.
func (m *Model) columnsBox() string {
	var b strings.Builder
	b.WriteString(faint.Render(m.store.BoardName(m.activeBoardID())) + "\n\n")
	n := len(m.columns)
	if n == 0 {
		b.WriteString(faint.Render(msg.cfgNoColumns) + "\n")
	}
	// window of up to colsVisible columns around the cursor (cursor on the create
	// row → shows the end of the list)
	start := 0
	if n > colsVisible {
		c := min(m.cfgCursor, n-1)
		start = max(0, min(c-colsVisible+1, n-colsVisible))
		if c < start {
			start = c
		}
	}
	end := min(n, start+colsVisible)
	if start > 0 {
		b.WriteString(faint.Render("  ↑ …") + "\n")
	}
	for i := start; i < end; i++ {
		cursor := "  "
		if i == m.cfgCursor {
			cursor = "› "
		}
		line := fmt.Sprintf("%s%d. %s", cursor, i+1, m.columns[i])
		if m.cfgGrab && i == m.cfgCursor {
			line += "  ↕"
		}
		if i == m.cfgCursor {
			st := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
			if m.cfgGrab {
				st = st.Foreground(lipgloss.Color(pal.peach))
			}
			line = st.Render(line)
		}
		// start/end label: makes explicit where the task is born (1st) and ends
		// (last). On a 1-column board it's both things → shows both.
		var tag string
		if i == 0 {
			tag = msg.colStart
		}
		if i == n-1 {
			if tag != "" {
				tag += " · "
			}
			tag += msg.colEnd
		}
		if tag != "" {
			line += faint.Render("   ← " + tag)
		}
		b.WriteString(line + "\n")
	}
	if end < n {
		b.WriteString(faint.Render("  ↓ …") + "\n")
	}
	// fixed "+ new column" row (index n = selectable)
	create := "+ " + msg.tAddColumn
	if m.cfgCursor == n {
		create = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent)).Render(create)
	} else {
		create = faint.Render(create)
	}
	b.WriteString("\n" + create)

	help := msg.hBoardCfg
	if m.cfgGrab {
		help = msg.hBoardCfgGrab
	}
	return modalBox(msg.cfgColumns, strings.TrimRight(b.String(), "\n"), help)
}

// laneConfigBox is the column menu (opened by ☰): a cursor list of Filtros / Renomear /
// Apagar, headed by the column name.
func (m *Model) laneConfigBox() string {
	name := ""
	if m.laneIdx >= 0 && m.laneIdx < len(m.columns) {
		name = m.columns[m.laneIdx]
	}
	var b strings.Builder
	b.WriteString(faint.Render(msg.laneLabel) + " " + cardTitle.Render(name) + "\n\n")
	for i, it := range m.laneMenu() {
		line := "  " + it
		if i == m.laneCursor {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent)).Render("› " + it)
		}
		b.WriteString(line + "\n")
	}
	return modalBox(msg.tLaneCfg, strings.TrimRight(b.String(), "\n"), msg.hLaneCfg)
}

// --- settings (F13) ---

// datePresets are the date formats cyclable in the menu. Hand-editing config.yml
// accepts any Go layout; the menu only cycles the common ones.
// ponytail: cycling presets instead of free-text editing; becomes an `edit` control
// (F13 phase 2) when the config textinput arrives alongside $EDITOR.
var datePresets = []string{"2006-01-02", "02/01/2006", "01/02/2006", "02 Jan 2006"}

type settingRow struct{ label, value, desc string }

// settingRows are the menu lines (only the "live" ones of phase 1). Adding an option
// = one more line here + a case in cycleSetting.
func (m *Model) settingRows() []settingRow {
	return []settingRow{
		{msg.sPreview, onOff(m.cfg.PreviewPane), msg.dPreview},
		{msg.sTheme, m.cfg.Theme, msg.dTheme},
		// shows the format applied to today (e.g. 13/07/2026) — more readable than the raw layout
		{msg.sDate, time.Now().Format(m.cfg.DateFormat), msg.dDate},
		{msg.sLang, m.cfg.Lang, msg.dLang},
		{msg.sDataDir, truncLeft(m.store.Dir(), 30), msg.dDataDir},
		{msg.sKeys, m.keys[kaKeymap], msg.dKeys}, // opens the shortcut modal (enter)
	}
}

func onOff(b bool) string {
	if b {
		return msg.vOn
	}
	return msg.vOff
}

// cycle returns the next value of opts starting from cur (with wrap). cur outside
// opts (e.g. date_format edited by hand) starts from the first.
func cycle(opts []string, cur string, dir int) string {
	i := 0
	for j, o := range opts {
		if o == cur {
			i = j
			break
		}
	}
	return opts[(i+dir+len(opts))%len(opts)]
}

// updateSettings navigates the settings menu and changes the option under the cursor,
// persisting to config.yml on every change.
func (m *Model) updateSettings(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	rows := m.settingRows()
	switch msg.String() {
	case "esc", "q", "s":
		m.mode = modeBoard
	case "j", "down":
		if m.setCursor < len(rows)-1 {
			m.setCursor++
		}
	case "k", "up":
		if m.setCursor > 0 {
			m.setCursor--
		}
	case "l", "right", ">", "enter", " ":
		if m.setCursor == keymapRow { // shortcuts opens the rebind modal
			m.startKeymap()
			return m, nil
		}
		if m.setCursor == dirRow { // directory opens the folder browser
			m.openDirBrowser()
			return m, nil
		}
		if m.openPicker() { // theme/language open a list picker, they don't cycle
			return m, nil
		}
		m.cycleSetting(1)
	case "h", "left":
		if m.setCursor == dirRow {
			m.openDirBrowser()
			return m, nil
		}
		if m.openPicker() {
			return m, nil
		}
		m.cycleSetting(-1)
	}
	return m, nil
}

// indices of the settings rows with special behavior (picker/browser).
const (
	themeRow  = 1
	langRow   = 3
	dirRow    = 4
	keymapRow = 5
)

// cycleSetting changes the option under the cursor (dir=±1) and saves. Toggle
// ignores dir. Theme/language are handled by a picker (openPicker), not here.
func (m *Model) cycleSetting(dir int) {
	switch m.setCursor {
	case 0:
		m.cfg.PreviewPane = !m.cfg.PreviewPane
	case 2:
		m.cfg.DateFormat = cycle(datePresets, m.cfg.DateFormat, dir)
	}
	m.store.SaveConfig(m.cfg)
}

// picker is a generic list selector (used by theme and language): moving previews
// live (preview, without saving), enter chooses (commit + persist), esc restores.
// cur marks the active value (•).
type picker struct {
	title, help string
	items       []string
	cursor      int
	cur         string
	preview     func(string) // on move — visual, doesn't save
	commit      func(string) // on choose — saves
}

// openPicker opens the picker of the settings row under the cursor (theme/language).
// Returns false if the row isn't a picker (then updateSettings cycles).
func (m *Model) openPicker() bool {
	switch m.setCursor {
	case themeRow:
		m.picker = &picker{
			title: msg.tTheme, help: msg.hTheme, items: themeNames, cur: m.cfg.Theme,
			preview: m.previewTheme,
			commit:  func(n string) { m.cfg.Theme = n; m.applyTheme() },
		}
	case langRow:
		m.picker = &picker{
			title: msg.tLang, help: msg.hTheme, items: langNames, cur: m.cfg.Lang,
			preview: applyLang,
			commit:  func(n string) { m.cfg.Lang = n; applyLang(n) },
		}
	default:
		return false
	}
	for i, n := range m.picker.items {
		if n == m.picker.cur {
			m.picker.cursor = i
		}
	}
	m.modalPlaced = false
	m.mode = modePicker
	return true
}

// updatePicker navigates the active picker. esc/← return to settings restoring the
// saved state (undoing the preview); enter chooses and persists. It always reopens
// settings centered (modalPlaced=false) — the picker is bigger, so without this
// settings would reuse its position/top and appear off-center.
func (m *Model) updatePicker(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := m.picker
	backToSettings := func() {
		m.modalPlaced = false
		m.mode = modeSettings
	}
	switch msg.String() {
	case "esc", "q", "h", "left", "<":
		m.applyTheme()
		applyLang(m.cfg.Lang)
		backToSettings()
	case "j", "down":
		if p.cursor < len(p.items)-1 {
			p.cursor++
			p.preview(p.items[p.cursor])
		}
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
			p.preview(p.items[p.cursor])
		}
	case "enter", " ":
		p.commit(p.items[p.cursor])
		m.store.SaveConfig(m.cfg)
		backToSettings()
	}
	return m, nil
}

// pickerBox is the list of the active picker (◉ marks the current value, matching the
// filter modal). The live preview recolors/retranslates the frame itself according to
// the item under the cursor.
func (m *Model) pickerBox() string {
	p := m.picker
	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent))
	var b strings.Builder
	for i, n := range p.items {
		mark := "○"
		if n == p.cur {
			mark = "◉"
		}
		line := fmt.Sprintf("%s %s", mark, n)
		if i == p.cursor {
			line = accent.Render("› " + line)
		} else {
			line = "  " + line
		}
		b.WriteString(line + "\n")
	}
	return modalBox(p.title, strings.TrimRight(b.String(), "\n"), p.help)
}

// --- data directory (F13): browser + move ---

// openDirBrowser opens the folder browser in the data dir's current folder.
func (m *Model) openDirBrowser() {
	m.browseTo(m.store.Dir())
	m.modalPlaced = false
	m.mode = modeDirBrowser
}

// readSubdirs returns the subfolders of path, sorted (error/permission → empty).
func readSubdirs(path string) []string {
	var out []string
	entries, _ := os.ReadDir(path)
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// browseTo enters a folder and reloads the subfolders (cursor at the top, filter cleared).
func (m *Model) browseTo(path string) {
	m.browsePath = path
	m.browseCursor = 0
	m.browseDirs = readSubdirs(path)
	m.browseFilter = ""
	m.browseFiltering = false
}

// filteredDirs are the subfolders that pass the filter (substring, case-insensitive).
func (m *Model) filteredDirs() []string {
	if m.browseFilter == "" {
		return m.browseDirs
	}
	q := strings.ToLower(m.browseFilter)
	var out []string
	for _, d := range m.browseDirs {
		if strings.Contains(strings.ToLower(d), q) {
			out = append(out, d)
		}
	}
	return out
}

func (m *Model) updateDirBrowser(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.browseFiltering { // typing the filter: keys become text
		return m.updateBrowseFilter(msg)
	}
	dirs := m.filteredDirs()
	switch msg.String() {
	case "esc", "q":
		if m.browseFilter != "" { // esc clears the filter before closing
			m.browseFilter, m.browseCursor = "", 0
			return m, nil
		}
		m.modalPlaced = false
		m.mode = modeSettings
	case "/":
		m.browseFiltering = true
	case "j", "down":
		if m.browseCursor < len(dirs)-1 {
			m.browseCursor++
		}
	case "k", "up":
		if m.browseCursor > 0 {
			m.browseCursor--
		}
	case "l", "right", " ": // enters the highlighted subfolder (yazi style)
		if len(dirs) > 0 {
			m.browseTo(filepath.Join(m.browsePath, dirs[m.browseCursor]))
		}
	case "h", "left", "backspace": // goes up one level
		m.browseTo(filepath.Dir(m.browsePath))
	case "enter": // uses the CURRENT folder as the destination
		m.chooseDir(m.browsePath)
	}
	return m, nil
}

// updateBrowseFilter handles typing the live filter: printables enter the text,
// backspace edits, enter confirms (keeps it filtered), esc clears. The cursor
// returns to the top on every change to point at the 1st match.
func (m *Model) updateBrowseFilter(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.browseFilter, m.browseFiltering, m.browseCursor = "", false, 0
	case "enter":
		m.browseFiltering = false
		if m.browseCursor >= len(m.filteredDirs()) {
			m.browseCursor = 0
		}
	case "backspace":
		if r := []rune(m.browseFilter); len(r) > 0 {
			m.browseFilter = string(r[:len(r)-1])
		}
		m.browseCursor = 0
	default:
		if msg.Text != "" { // printable character
			m.browseFilter += msg.Text
			m.browseCursor = 0
		}
	}
	return m, nil
}

// chooseDir decides the destination: same as the current one does nothing; otherwise it asks to move.
func (m *Model) chooseDir(target string) {
	m.modalPlaced = false
	if target == m.store.Dir() {
		m.mode = modeSettings
		return
	}
	m.moveTarget = target
	m.mode = modeConfirmMove
}

// dirBrowserBox draws the yazi-style browser: Miller columns (parent · current ·
// preview), full-width selection bar in the accent, folders with "/". Breadcrumb on
// top. The preview column shows the content of the highlighted subfolder.
func (m *Model) dirBrowserBox() string {
	const rows, paneW = 14, 22

	parentPath := filepath.Dir(m.browsePath)
	parentDirs := readSubdirs(parentPath)
	selParent := indexOf(parentDirs, filepath.Base(m.browsePath))

	dirs := m.filteredDirs()
	var previewDirs []string
	if len(dirs) > 0 && m.browseCursor < len(dirs) {
		previewDirs = readSubdirs(filepath.Join(m.browsePath, dirs[m.browseCursor]))
	}

	left := renderPane(parentDirs, selParent, paneW, rows, false)
	mid := renderPane(dirs, m.browseCursor, paneW, rows, true)
	right := renderPane(previewDirs, -1, paneW, rows, false)
	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, vsep(rows), mid, vsep(rows), right)

	header := faint.Render(truncLeft(m.browsePath, paneW*3))
	if m.browseFiltering || m.browseFilter != "" {
		prompt := msg.brFilter + m.browseFilter
		if m.browseFiltering {
			prompt += "▌"
		}
		header += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(pal.accent)).Render(prompt)
	}
	return modalBox(msg.tBrowser, header+"\n\n"+panes, msg.hBrowser)
}

// folderIcon is the folder glyph (nerd font U+F07B, yazi style). A terminal without
// a nerd font shows tofu — the user runs yazi with icons, so they have the font.
const folderIcon = "\uf07b"

// renderPane draws a browser column: subfolders (with "/"), a window of h lines
// around sel. active=true uses the full-width accent bar on the cursor; on the side
// columns (active=false) the highlight is just a light emphasis.
func renderPane(entries []string, sel, w, h int, active bool) string {
	if len(entries) == 0 {
		lines := make([]string, h)
		lines[0] = lipgloss.NewStyle().Width(w).Render(faint.Render(" " + msg.brEmpty))
		for i := 1; i < h; i++ {
			lines[i] = strings.Repeat(" ", w)
		}
		return strings.Join(lines, "\n")
	}
	start := 0
	if sel >= h {
		start = sel - h + 1
	}
	end := min(len(entries), start+h)

	bar := lipgloss.NewStyle().Width(w).Background(lipgloss.Color(pal.accent)).Foreground(lipgloss.Color(pal.panelBg))
	dim := lipgloss.NewStyle().Width(w).Background(lipgloss.Color(pal.surface0))
	plain := lipgloss.NewStyle().Width(w)
	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.accent)) // icon follows the theme's accent

	lines := make([]string, 0, h)
	for i := start; i < end; i++ {
		label := truncate(entries[i], w-4)
		switch {
		case i == sel && active: // bar: the icon inherits the bar's fg (contrast)
			lines = append(lines, bar.Render(" "+folderIcon+" "+label))
		case i == sel:
			lines = append(lines, dim.Render(" "+iconStyle.Render(folderIcon)+" "+label))
		default:
			lines = append(lines, plain.Render(" "+iconStyle.Render(folderIcon)+" "+label))
		}
	}
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return strings.Join(lines, "\n")
}

// vsep is a vertical separator column of h lines.
func vsep(h int) string {
	line := faint.Render("│")
	rows := make([]string, h)
	for i := range rows {
		rows[i] = line
	}
	return strings.Join(rows, "\n")
}

// indexOf returns the index of s in list, or -1.
func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

func (m *Model) updateConfirmMove(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.modalPlaced = false
		m.mode = modeSettings
	case "enter", "s", "y":
		m.applyDir(true)
	case "n":
		m.applyDir(false)
	}
	return m, nil
}

// applyDir carries out the data dir switch: moves (or not) the files, writes the
// pointer and reopens the store in the new dir. Errors become a notice (no half-change).
func (m *Model) applyDir(move bool) {
	old := m.store.Dir()
	if move {
		if err := task.MoveData(old, m.moveTarget); err != nil {
			if errors.Is(err, task.ErrTargetHasData) {
				m.showNotice(msg.nvHasData, modeSettings)
			} else {
				m.showNotice(err.Error(), modeSettings)
			}
			return
		}
	}
	if err := task.SaveDataDir(m.moveTarget); err != nil {
		m.showNotice(err.Error(), modeSettings)
		return
	}
	if err := m.rebindStore(m.moveTarget); err != nil {
		m.showNotice(err.Error(), modeSettings)
		return
	}
	if move {
		m.showNotice(fmt.Sprintf(msg.nvMoved, m.moveTarget), modeSettings)
	} else {
		m.showNotice(fmt.Sprintf(msg.nvPointed, m.moveTarget), modeSettings)
	}
}

// rebindStore reopens the store in the new dir and rebuilds the derived state.
func (m *Model) rebindStore(dir string) error {
	st, err := task.Open(dir)
	if err != nil {
		return err
	}
	m.store = st
	m.open = nil
	m.active = 0
	m.restoreTabs()
	m.resetCursor()
	m.reload()
	m.lastSig = dirSig(dir)
	return nil
}

// btn draws a clickable button. hover highlights it (full background); danger uses
// the red hue on hover (destructive action).
func btn(label string, hover, danger bool) string {
	st := lipgloss.NewStyle().Padding(0, 2)
	if hover {
		bg := pal.accent
		if danger {
			bg = pal.red
		}
		return st.Bold(true).Background(lipgloss.Color(bg)).Foreground(lipgloss.Color(pal.panelBg)).Render(label)
	}
	return st.Background(lipgloss.Color(pal.surface1)).Foreground(lipgloss.Color(pal.text)).Render(label)
}

// confirmDeleteBox asks whether to delete the active board (with clickable No/Yes
// buttons and hover). It also registers the buttons' hitboxes relative to the modal
// top — cardBox has border(1)+padding(1,2), so the content starts at (row 2, column 3).
func (m *Model) confirmDeleteBox() string {
	n := len(m.store.BoardTasks(m.activeBoardID()))
	text := fmt.Sprintf(msg.cdBody, m.store.BoardName(m.activeBoardID()), n) // 2 lines
	noBtn := btn(msg.btnNo, m.hoverBtn == 0, false)
	yesBtn := btn(msg.btnYes, m.hoverBtn == 1, true)
	const gap = 3
	wNo, wYes := lipgloss.Width(noBtn), lipgloss.Width(yesBtn)
	btnLine := noBtn + strings.Repeat(" ", gap) + yesBtn

	title := cardTitle.Render(msg.tConfirmDelete)
	cw := max(lipgloss.Width(title), max(lipgloss.Width(text), lipgloss.Width(btnLine)))
	center := func(s string) string { return lipgloss.PlaceHorizontal(cw, lipgloss.Center, s) }
	body := center(title) + "\n\n" + centerBlock(text, cw) + "\n\n" + center(btnLine)

	// hitboxes (relative to the modal): content at (2,3); buttons on the last line
	leftPad := (cw - lipgloss.Width(btnLine)) / 2
	x0 := 3 + leftPad
	m.confirmRow = 2 + strings.Count(body, "\n") // btnLine's row (last of the body)
	m.confirmNo = clickRegion{x0: x0, x1: x0 + wNo}
	m.confirmYes = clickRegion{x0: x0 + wNo + gap, x1: x0 + wNo + gap + wYes}
	return cardBox.Render(body)
}

// centerBlock centers each line of a multi-line block to a width.
func centerBlock(block string, w int) string {
	lines := strings.Split(block, "\n")
	for i, ln := range lines {
		lines[i] = lipgloss.PlaceHorizontal(w, lipgloss.Center, ln)
	}
	return strings.Join(lines, "\n")
}

// hitConfirmBtn returns the button under (x,y): 1 yes, 0 no, -1 none. Converts the
// screen position to a coordinate relative to the modal (modalX/modalY).
func (m *Model) hitConfirmBtn(x, y int) int {
	if y != m.modalY+m.confirmRow {
		return -1
	}
	rx := x - m.modalX
	switch {
	case rx >= m.confirmNo.x0 && rx < m.confirmNo.x1:
		return 0
	case rx >= m.confirmYes.x0 && rx < m.confirmYes.x1:
		return 1
	}
	return -1
}

func (m *Model) confirmMoveBox() string {
	body := fmt.Sprintf(msg.mvBody, task.CountData(m.store.Dir()), m.moveTarget)
	return modalBox(msg.tMove, body, msg.hMove)
}

// showNotice shows a notice; when dismissed it returns to mode `back` (the settings
// flow passes modeSettings; the board flow, modeBoard).
func (m *Model) showNotice(text string, back mode) {
	m.notice = text
	m.noticeBack = back
	m.modalPlaced = false
	m.mode = modeNotice
}

func (m *Model) noticeBox() string {
	return modalBox(msg.tNotice, m.notice, "esc")
}

// updateNotice: any key dismisses the notice and returns to the origin mode.
func (m *Model) updateNotice(_ tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.modalPlaced = false
	m.mode = m.noticeBack
	return m, nil
}

// truncLeft cuts from the left keeping the end (useful for paths: shows the final
// stretch, more informative).
func truncLeft(s string, w int) string {
	r := []rune(s)
	if w <= 1 || len(r) <= w {
		return s
	}
	return "…" + string(r[len(r)-w+1:])
}

// settingsBox is the content of the settings modal (› marks the cursor). Below the
// list, a description line for the selected item (changes as you navigate).
func (m *Model) settingsBox() string {
	rows := m.settingRows()
	var b strings.Builder
	for i, r := range rows {
		cursor := "  "
		if i == m.setCursor {
			cursor = "› "
		}
		line := fmt.Sprintf("%s%-16s %s", cursor, r.label, r.value)
		if i == m.setCursor {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent)).Render(line)
		}
		b.WriteString(line + "\n")
	}
	body := strings.TrimRight(b.String(), "\n") + "\n\n" + faint.Render(rows[m.setCursor].desc)
	return modalBox(msg.tSettings, body, msg.hSettings)
}

// fixedColW is the fixed width of each column (only shrinks if 3 don't fit on the screen).
const fixedColW = 36

func (m *Model) boardView() string {
	w, h := m.w, m.h
	if w == 0 {
		w = 110
	}
	if h == 0 {
		h = 30
	}
	if len(m.open) == 0 { // no board (new app / last one deleted)
		return m.tabBar() + "\n\n  " + faint.Render(msg.noBoards) + "\n\n" + helpStyle.Render(msg.hBoard)
	}
	if len(m.columns) == 0 { // board with no columns: just the header + hint (config opens on its own)
		return m.tabBar() + "\n\n  " + faint.Render(msg.boardEmpty) + "\n\n" + helpStyle.Render(msg.hBoard)
	}
	// Columns have a fixed width; if not all fit, horizontal scroll per CELL (the whole
	// board is rendered and cropped to the visible window) — it slides smoothly, it
	// doesn't jump column by column. It only shrinks the column if even ONE doesn't fit.
	colW := fixedColW
	if colW > w {
		colW = max(18, w)
	}
	m.colW = colW
	totalW := colW * len(m.columns)
	m.viewW = min(totalW, w) // visible board width
	m.clampHOff()
	trackRows := 0 // 1 line reserved for the horizontal scroll track
	if m.viewW < totalW {
		trackRows = 1
	}
	m.cardRegions = m.cardRegions[:0]
	m.gearRegions = m.gearRegions[:0]
	m.syncRegions = m.syncRegions[:0]
	m.syncY = 0
	// Column text area = colW - border(2) - padding(2), split into cards (left) +
	// the scrollbar lane (2, right). Without subtracting the lane, colBody ends up 2
	// columns wider than the inner area and lipgloss re-wraps EVERY line → the column
	// doubles in height (only shows up when the scrollbar has a visible thumb, since
	// the blank lane is trimmed). Card: text colW-10 (+padding 2 +border 2 = frame
	// colW-6 = cardsW).
	cardsW := colW - 6
	contentW := colW - 10
	if contentW < 6 {
		contentW = 6
	}
	// preview panel in the footer (F13): reserves a fixed height when on; on a short
	// screen the board takes priority and the panel disappears.
	paneH, showPane := 0, m.cfg.PreviewPane
	if showPane {
		paneH = max(5, min(10, h/3))
	}
	// button footer (1 line) only when some column has buttons; reserved on ALL columns to
	// keep the height uniform.
	boardID := m.activeBoardID()
	footerRows := 0
	for _, c := range m.columns {
		if len(m.columnButtons(c)) > 0 {
			footerRows = 1
			break
		}
	}
	bodyH := h - 7 - paneH - trackRows - footerRows // tabs(1)+blank(1)+border(2)+title(1)+blank(1)+help(1) [+track+footer]
	if bodyH < 4 {
		paneH, showPane = 0, false
		bodyH = max(4, h-7-trackRows-footerRows)
	}
	if footerRows > 0 {
		m.syncY = boardCardsTop + bodyH // line right below the cards
	}

	m.bodyH = bodyH // for wheel scroll (overflow test)
	m.colTotal = make([]int, len(m.columns))
	rendered := make([]string, len(m.columns))
	for i, c := range m.columns {
		active := i == m.col
		var cards []string
		if len(m.cols[i]) == 0 {
			cards = []string{faint.Render("—")}
		}
		for j, t := range m.cols[i] {
			cards = append(cards, renderCard(t, active && j == m.row[i], contentW, m.cardAnnotation(t, contentW), m.cfg.Tags))
		}
		// Vertical scroll follows the cursor: the window keeps the selected card visible
		// (reveal). The wheel moves the cursor (scrollColBy), so scrolling also selects.
		reveal := -1
		if active {
			reveal = m.row[i]
		}
		body, total, offset, starts := windowColumn(cards, bodyH, reveal)
		m.colTotal[i] = total
		// hitboxes of the real visible cards (for dragging)
		for j, t := range m.cols[i] {
			y0 := boardCardsTop + starts[j] - offset
			y1 := y0 + cardLines(cards[j])
			if y1 <= boardCardsTop || y0 >= boardCardsTop+bodyH {
				continue // outside the visible window
			}
			m.cardRegions = append(m.cardRegions, cardHit{
				col: i, id: t.ID,
				y0: max(y0, boardCardsTop), y1: min(y1, boardCardsTop+bodyH),
			})
		}
		colBody := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(cardsW).Render(body), // standardizes the width → aligned bar
			vscroll(total, bodyH, offset),
		)

		ts := titleStyle
		gStyle := faint
		if active {
			ts = titleStyle.Foreground(lipgloss.Color(pal.accent))
			gStyle = titleStyle.Foreground(lipgloss.Color(pal.accent))
		}
		// title with the ☰ menu icon aligned to the right of the header (contentW = colW-4).
		// It opens the column menu (Filtros / Renomear / Apagar); it lights up in accent
		// when the column has active filters (use_filters).
		tw := colW - 4
		raw := fmt.Sprintf("%s (%d)", strings.ToUpper(statusLabel(c)), len(m.cols[i]))
		mStyle := gStyle
		if m.store.ColumnHasFilters(boardID, c) {
			mStyle = gStyle.Foreground(lipgloss.Color(pal.accent))
		}
		icon := mStyle.Render("☰")
		iw := lipgloss.Width(icon)
		label := ts.Render(ansi.Truncate(raw, max(1, tw-iw-1), "")) // reserves the icon + a gap
		titleLine := label + lipgloss.PlaceHorizontal(max(1, tw-lipgloss.Width(label)), lipgloss.Right, icon)
		// ☰ hitbox: the last few content cells of the header (generous, so the ambiguous
		// glyph width doesn't make the click land off the symbol). right = one past the
		// last content cell (border+padding = 2 on each side).
		right := (i+1)*colW - m.hOff - 2
		m.gearRegions = append(m.gearRegions, clickRegion{x0: right - 3, x1: right, kind: "gear", idx: i})
		inner := titleLine + "\n\n" + colBody
		if footerRows > 0 { // footer: the column's button, empty line on the columns without one
			footer := ""
			if label := m.footerLabel(c, m.columnButtons(c)); label != "" {
				fx := i*colW + 2 - m.hOff // start of the column's content
				// btn width is hover-independent (padding 0,2 = +4); use it to detect hover
				btnW := lipgloss.Width(label) + 4
				hover := m.syncY == m.mouseY && m.mouseX >= fx && m.mouseX < fx+btnW
				footer = m.syncFooter(c, label, colW-4, i == m.col || hover || (m.menuOpen && m.menuCol == i))
				m.syncRegions = append(m.syncRegions, clickRegion{x0: fx, x1: fx + lipgloss.Width(footer), kind: "sync", idx: i})
			}
			inner += "\n" + footer
		}

		style := colStyle
		switch {
		case m.draggingCard && i == m.dropCol: // drag target column
			style = colDropStyle
		case m.prefixArmed && active: // prefix armed → lights up the column under the cursor
			style = colDropStyle
		case active:
			style = colSelStyle
		}
		rendered[i] = style.Width(colW).Render(inner)
	}
	// full board, cropped to the window [hOff, hOff+viewW) per cell
	board := cropCols(lipgloss.JoinHorizontal(lipgloss.Top, rendered...), m.hOff, m.viewW)
	out := m.topBar() + "\n\n" + board
	m.trackY = 0       // no track by default (only valid when drawn below)
	if trackRows > 0 { // horizontal scroll track (only when there's a column outside the window)
		// the board starts on line 2 (tabs + blank); the track comes right below it
		m.trackY, m.trackW = 2+lipgloss.Height(board), m.viewW
		out += "\n" + hscroll(totalW, m.viewW, m.hOff, m.viewW)
	}
	if showPane {
		out += "\n" + m.previewPane(m.current(), m.viewW, paneH)
	}
	return out + "\n" + m.bottomBar()
}

// bottomBar is the board footer in two states: idle shows only the minimal hint
// (ctrl+t opens the commands); with the prefix armed it becomes the highlighted
// tmux-style bar — PREFIX badge + the list of available commands, cropped to the width.
func (m *Model) bottomBar() string {
	if !m.prefixArmed {
		return helpStyle.Render(msg.hIdle)
	}
	badge := prefixBadge.Render(" PREFIX ")
	hints := msg.hPrefix
	if w := m.viewW; w > 0 { // doesn't overflow the window: truncates the commands, keeps the badge
		hints = ansi.Truncate(hints, max(0, w-lipgloss.Width(badge)-1), "…")
	}
	return badge + " " + helpStyle.Render(hints)
}

// tabBar draws the tabs of the open boards + the "+" button (browser style) and,
// along the way, registers the hitboxes of each tab/button for the mouse click
// (columns relative to the start of the line, which is column 0).
func (m *Model) tabBar() string {
	parts := make([]string, 0, len(m.open)+1)
	m.tabRegions = m.tabRegions[:0]
	showClose := len(m.open) > 1 // can't close the last tab → the ✕ disappears
	x := 0
	for i, id := range m.open {
		style := tabInactive
		if i == m.active {
			style = tabActive
		}
		name := m.store.BoardName(id)
		label := " " + name + " "
		if showClose {
			label += "✕ "
		}
		parts = append(parts, style.Render(label))
		w := lipgloss.Width(label)
		if showClose {
			cc := x + 1 + lipgloss.Width(name) + 1 // column of the ✕ within the label
			m.tabRegions = append(m.tabRegions,
				clickRegion{x0: x, x1: cc, kind: "tab", idx: i},       // name → switch
				clickRegion{x0: cc, x1: x + w, kind: "close", idx: i}, // ✕ → close
			)
		} else {
			m.tabRegions = append(m.tabRegions, clickRegion{x0: x, x1: x + w, kind: "tab", idx: i})
		}
		x += w
	}
	plus := tabPlus.Render(" + ")
	m.tabRegions = append(m.tabRegions, clickRegion{x0: x, x1: x + lipgloss.Width(plus), kind: "new", idx: -1})
	parts = append(parts, plus)
	return lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
}

// renderCard draws a mini-card: id + priority on top, title (wraps), the optional
// `ann` annotation (`↳ PARENT` on a child, or the bar `▓▓░ 2/5` on a card with
// subtasks) and the tags (chips). contentW is the text area; the frame comes from it.
func renderCard(t *task.Task, sel bool, contentW int, ann string, cat []task.TagDef) string {
	var b strings.Builder
	// all lines fixed at contentW → the box (without .Width) has a uniform width and
	// doesn't re-wrap (the re-wrap trimmed the pill's padding on the 2nd line).
	b.WriteString(lipgloss.NewStyle().Width(contentW).Render(faint.Render(t.ID)+"  "+prioTag(t.Priority)) + "\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Width(contentW).Render(t.Title))
	if ann != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Width(contentW).Render(ann))
	}
	if len(t.Tags) > 0 {
		b.WriteString("\n" + chipsWrapped(t.Tags, cat, contentW))
	}
	style := miniBox
	if sel {
		style = miniBoxSel
	}
	// no .Width(boxW): all lines already have contentW, so the box stays uniform
	// without a constraint — and without a constraint lipgloss doesn't re-wrap (the
	// re-wrap trimmed the pill's left padding on the 2nd line of tags, misaligning it).
	return style.Render(b.String())
}

// doneCol is the column treated as "done" (heuristic: the last on the board).
// ponytail: becomes a column category (todo/doing/done) when roadmap item 4 arrives.
func (m *Model) doneCol() string {
	if len(m.columns) == 0 {
		return ""
	}
	return m.columns[len(m.columns)-1]
}

// childProgress counts the direct children of id and how many are in the "done" column.
func (m *Model) childProgress(id string) (done, total int) {
	done0 := m.doneCol()
	for _, c := range m.store.Children(id) {
		total++
		if c.Status == done0 {
			done++
		}
	}
	return
}

// cardAnnotation is the mini-card's extra line: `↳ PARENT` for a child; a progress
// bar for a card with subtasks; "" for a regular card.
func (m *Model) cardAnnotation(t *task.Task, w int) string {
	if m.action != nil && m.action.id == t.ID { // action in progress: loader in place of the annotation
		return m.actionBar(m.action, w)
	}
	// live % stamped by the action (e.g. agent working the card) — wins over key/parent.
	// 100 NÃO desenha: quem limpa o Progress é o move (headless.go, action.go), então um agente
	// que carimba 100 depois de mover deixa a barra cheia grudada no card pra sempre. Cheia não
	// informa nada — o card já está na coluna seguinte. Só o meio do caminho vale pixel.
	if t.Progress > 0 && t.Progress < 100 {
		style, showPct := m.loaderStyle(t.Status)
		return progressBar(t.Progress, style, showPct)
	}
	if t.Mirror() { // Jira mirror: shows the issue key (read-only, F16)
		key := t.Jira
		if key == "" {
			key = "jira"
		}
		return faint.Render("↗ " + key)
	}
	if t.Parent != "" {
		return faint.Render("↳ " + t.Parent)
	}
	if done, total := m.childProgress(t.ID); total > 0 {
		const cells = 5
		filled := done * cells / total
		bar := strings.Repeat("▓", filled) + strings.Repeat("░", cells-filled)
		return faint.Render(fmt.Sprintf("%s %d/%d", bar, done, total))
	}
	return ""
}

// progressBar draws the persisted card % (0..100) with the SAME segmented bar as the live action
// loader (segBar) — so the % stamped on the card looks identical to the loader, just read from the
// card's own field instead of the animated action. Style/number follow the card's own column.
// Rounds like actionBar.
func progressBar(pct int, style string, showPct bool) string {
	filled := clamp((pct*barCells+50)/100, 0, barCells)
	fill := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.accent))
	bar := segBar(filled, fill, style)
	if !showPct {
		return bar
	}
	return bar + faint.Render(fmt.Sprintf("  %d%%", pct))
}

// truncate cuts s to fit in w runes, with an ellipsis.
func truncate(s string, w int) string {
	r := []rune(s)
	if w <= 0 || len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

// prioTag is the compact, colored priority for the mini-card.
func prioTag(p string) string {
	switch p {
	case "high":
		return lipgloss.NewStyle().Foreground(lipgloss.Color(pal.red)).Bold(true).Render("● " + msg.prHigh)
	case "low":
		return faint.Render("○ " + msg.prLow)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(pal.blue)).Render("● " + msg.prNormal)
	}
}

// openDetail builds the selected task's card in a (scrollable) viewport and enters
// detail mode. No-op if the column is empty.
func (m *Model) openDetail() {
	if m.current() == nil {
		return
	}
	m.detailStack = []detailFrame{{id: m.current().ID}} // navigation root
	m.mode = modeDetail
	m.resizeDetail()
	m.vp.GotoTop()
}

// detailFrame is one level of the detail navigation: the displayed card + the
// related one selected on it (parent/subtask) in the drill-down.
type detailFrame struct {
	id  string
	sel int
}

// detailLink is a clickable URL in the detail: the content line and the x range the
// URL text spans, so only the text itself reacts to the click.
type detailLink struct {
	line, x0, x1 int
	url          string
}

// urlRe matches a URL as the terminal shows it. The trailing class excludes what
// usually closes a sentence or a markdown link, so "(https://x)" and "see https://x."
// don't drag ")" / "." into the URL.
var urlRe = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `)\]]+[^\s<>"'` + "`" + `)\].,;:!?]`)

// scanURLs registers every URL printed in the card's rendered lines as a clickable
// hitbox. It works on the rendered output (not the raw markdown) because that's what
// the user actually clicks: whatever Glamour chose to print is what has coordinates.
// ponytail: a URL that Glamour wrapped across two lines is clickable only on its
// first line — go through the markdown AST if that ever matters.
func (m *Model) scanURLs(lines []string) {
	for i, ln := range lines {
		plain := ansi.Strip(ln)
		for _, loc := range urlRe.FindAllStringIndex(plain, -1) {
			x0 := 3 + lipgloss.Width(plain[:loc[0]]) // 3 = box border(1) + padding(2)
			m.detailLinks = append(m.detailLinks, detailLink{
				line: i + 2, // +2 = box top border + padding (matches detailRows)
				x0:   x0, x1: x0 + lipgloss.Width(plain[loc[0]:loc[1]]),
				url: plain[loc[0]:loc[1]],
			})
		}
	}
}

// detailRow is a navigable row of the detail (parent/subtask) and its line in the
// card's content. detailHit is the same, already mapped to a screen row (for clicking).
type detailRow struct {
	id   string
	line int
}
type detailHit struct {
	id string
	y  int
}

// detailTask is the card at the top of the detail stack (nil if empty).
func (m *Model) detailTask() *task.Task {
	if len(m.detailStack) == 0 {
		return nil
	}
	return m.store.Get(m.detailStack[len(m.detailStack)-1].id)
}

// resizeDetail (re)sizes the viewport to fit the screen and reloads the card.
func (m *Model) resizeDetail() {
	t := m.detailTask()
	if t == nil {
		return
	}
	h := m.h - 2 // reserves the help line
	if h < 3 {
		h = 20
	}
	// width = exactly the card's (content + padding 4 + border 2), so the scrollbar
	// hugs the side without overflow. v2: New() with no args + setters.
	vp := viewport.New()
	vp.SetWidth(m.detailWidth() + 6)
	vp.SetHeight(h)
	vp.SetContent(m.cardContent(t, m.detailStack[len(m.detailStack)-1].sel))
	m.vp = vp
}

// refreshDetail only rewrites the content (preserves the scroll) — used when moving
// the subtask selection.
func (m *Model) refreshDetail() {
	if t := m.detailTask(); t != nil {
		m.vp.SetContent(m.cardContent(t, m.detailStack[len(m.detailStack)-1].sel))
	}
}

// updateDetail: navigates the detail. With subtasks, j/k select and enter/→ opens
// the subtask (pushes); esc/← goes back one level (or to the board). Without subtasks, j/k scroll.
func (m *Model) updateDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.detailTask()
	if t == nil {
		m.mode = modeBoard
		return m, nil
	}
	rel := m.detailRelated(t) // parent (↑) + subtasks, in the order they appear
	top := len(m.detailStack) - 1
	switch msg.String() {
	case "esc", "q", "left", "h":
		m.detailStack = m.detailStack[:top]
		if len(m.detailStack) == 0 {
			m.mode = modeBoard
		} else {
			m.resizeDetail()
		}
		return m, nil
	case "enter", "right", "l":
		if len(rel) > 0 {
			m.openDetailID(rel[m.detailStack[top].sel])
		}
		return m, nil
	case "j", "down":
		if len(rel) > 0 {
			if m.detailStack[top].sel < len(rel)-1 {
				m.detailStack[top].sel++
				m.refreshDetail()
			}
			return m, nil
		}
	case "k", "up":
		if len(rel) > 0 {
			if m.detailStack[top].sel > 0 {
				m.detailStack[top].sel--
				m.refreshDetail()
			}
			return m, nil
		}
	}
	// no related ones (or pgup/pgdn): scrolls the viewport
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

// detailRelated lists the navigable cards of t's detail, in display order: the
// parent first (↑, if it exists) and then the subtasks. The index matches `sel`.
func (m *Model) detailRelated(t *task.Task) []string {
	var ids []string
	if t.Parent != "" && m.store.Get(t.Parent) != nil {
		ids = append(ids, t.Parent)
	}
	for _, c := range m.store.Children(t.ID) {
		ids = append(ids, c.ID)
	}
	return ids
}

// openDetailID pushes a card onto the detail (parent/child drill) and goes to the top.
func (m *Model) openDetailID(id string) {
	if m.store.Get(id) == nil {
		return
	}
	m.detailStack = append(m.detailStack, detailFrame{id: id})
	m.resizeDetail()
	m.vp.GotoTop()
}

// vscroll draws the vertical bar (gap + track/thumb) with h lines, always 2 columns
// wide. The thumb is proportional to the visible fraction and its position comes
// from the offset. If it all fits (total<=h), it returns h blank lines — the lane
// stays reserved so the width doesn't vary between scrolling and non-scrolling elements.
func vscroll(total, h, offset int) string {
	scroll := h > 0 && total > h
	thumb, top := 0, 0
	if scroll {
		thumb = max(1, h*h/total)
		top = min(h-thumb, offset*(h-thumb)/(total-h))
	}
	lines := make([]string, h)
	for i := range lines {
		switch {
		case !scroll:
			lines[i] = "  "
		case i >= top && i < top+thumb:
			lines[i] = " " + sbThumb.Render("⣿") // thumb accent (consistent with the horizontal one)
		default:
			lines[i] = " " + sbTrack.Render("⣀") // muted track
		}
	}
	return strings.Join(lines, "\n")
}

// hscroll draws the horizontal track (track/thumb) with `w` columns wide, mirroring
// vscroll but in board-column units: `total` columns, visible window `shown`
// starting at `off`. The thumb is proportional to the visible fraction.
func hscroll(total, shown, off, w int) string {
	if w <= 0 {
		return ""
	}
	if !(shown > 0 && total > shown) {
		return strings.Repeat(" ", w) // it all fits: lane reserved and blank
	}
	thumb := max(1, w*shown/total)
	left := min(w-thumb, off*(w-thumb)/(total-shown))
	right := w - left - thumb
	// thumb in full braille (⣿) in the accent; track in ⣀ (background rail) muted.
	return sbTrack.Render(strings.Repeat("⣀", left)) +
		sbThumb.Render(strings.Repeat("⣿", thumb)) +
		sbTrack.Render(strings.Repeat("⣀", right))
}

// cropCols crops each line of a block to the window [off, off+w) in cells,
// preserving ANSI — it's the board's horizontal scroll at the cell level (slides
// smoothly). off==0 && block fits in w returns the block intact.
func cropCols(block string, off, w int) string {
	lines := strings.Split(block, "\n")
	for i, ln := range lines {
		lines[i] = ansi.Cut(ln, off, off+w)
	}
	return strings.Join(lines, "\n")
}

// windowColumn flattens the column's cards into lines and crops a window of bodyH
// lines that keeps the `reveal` card visible (reveal<0 = from the top). Returns the
// window (padded to bodyH), the total line count and the offset — the last two feed
// vscroll. It's the column's "scroll": cursor-driven, recomputed on every render.
// flattenCards splits each card into lines, returning the flat line list and the start
// line of each card (for hitboxes and scroll math).
func flattenCards(cards []string) (lines []string, starts []int) {
	starts = make([]int, len(cards))
	for j, c := range cards {
		starts[j] = len(lines)
		lines = append(lines, strings.Split(c, "\n")...)
	}
	return lines, starts
}

func windowColumn(cards []string, bodyH, reveal int) (body string, total, offset int, starts []int) {
	lines, starts := flattenCards(cards)
	total = len(lines)
	if reveal >= 0 && reveal < len(cards) {
		bottom := total
		if reveal+1 < len(starts) {
			bottom = starts[reveal+1]
		}
		if bottom > offset+bodyH {
			offset = bottom - bodyH
		}
		if starts[reveal] < offset { // tall card: shows from its top
			offset = starts[reveal]
		}
	}
	if maxOff := max(0, total-bodyH); offset > maxOff {
		offset = maxOff
	}
	end := min(total, offset+bodyH)
	win := append([]string{}, lines[offset:end]...)
	for len(win) < bodyH {
		win = append(win, "")
	}
	return strings.Join(win, "\n"), total, offset, starts
}

// detailWidth is the usable text width inside the card. Full screen: uses the
// terminal width minus the chrome (border 2 + padding 4 + scrollbar 2 = 8).
func (m *Model) detailWidth() int {
	w := m.w - 8
	if m.w == 0 {
		w = 72
	}
	if w < 24 {
		w = 24
	}
	return w
}

// previewPane is the preview panel in the footer (F13): the .md of the card under
// the cursor (title + badges + meta on one line + notes via Glamour), fixed height.
// No scroll — it's a "peek"; the scrollable full view is still Enter (F4). Reuses
// the same notes/badges render as the detail.
func (m *Model) previewPane(t *task.Task, width, paneH int) string {
	iw := width - 4 // Width/Height include the border; content = width - border(2) - padding(2)
	if iw < 10 {
		iw = 10
	}
	box := paneBox.Width(width).Height(paneH)
	if t == nil {
		return box.Render(faint.Render(msg.noCard))
	}

	var b strings.Builder
	b.WriteString(cardTitle.Render(truncate(t.Title, iw)) + "\n")
	b.WriteString(statusBadge(t.Status) + " " + priorityBadge(t.Priority))

	var meta []string
	if p := m.store.EffectiveProject(t.ID); p != "" {
		meta = append(meta, "@"+p)
	}
	if tags := m.store.EffectiveTags(t.ID); len(tags) > 0 {
		meta = append(meta, chips(tags, m.cfg.Tags))
	}
	if t.Due != nil {
		meta = append(meta, msg.duePrefix+t.Due.Format(m.cfg.DateFormat))
	}
	if len(meta) > 0 {
		b.WriteString("  " + faint.Render(strings.Join(meta, "  ")))
	}
	b.WriteString("\n" + paneSep.Render(strings.Repeat("─", iw)) + "\n")
	if strings.TrimSpace(t.Notes) != "" {
		b.WriteString(strings.TrimRight(m.renderNotes(t.Notes, iw), "\n"))
	} else {
		b.WriteString(faint.Render(msg.noNotes))
	}
	return box.Render(clipLines(b.String(), paneH-2))
}

// clipLines cuts the content to n lines; if there's overflow, the last one becomes a
// "…" marker pointing to the full view (Enter). n<=0 returns empty.
func clipLines(s string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	lines = lines[:n]
	lines[n-1] = faint.Render(msg.seeAll)
	return strings.Join(lines, "\n")
}

// cardContent draws the card: header + badges + metadata + notes (Glamour), all
// inside a rounded frame.
func (m *Model) cardContent(t *task.Task, sel int) string {
	iw := m.detailWidth()
	var lines []string
	add := func(s string) { lines = append(lines, s) }
	m.detailRows = m.detailRows[:0]

	add(cardTitle.Render(t.Title))
	add("")
	add(statusBadge(t.Status) + "  " + priorityBadge(t.Priority))
	add("")
	if p := m.store.EffectiveProject(t.ID); p != "" {
		add(metaKey.Render(msg.mProject) + p)
	}
	if tags := m.store.EffectiveTags(t.ID); len(tags) > 0 {
		add(metaKey.Render(msg.mTags) + chips(tags, m.cfg.Tags))
	}
	if t.Due != nil {
		add(metaKey.Render(msg.mDue) + t.Due.Format(m.cfg.DateFormat))
	}
	m.detailLinks = m.detailLinks[:0]
	idView := faint.Render(t.ID)
	if url := m.store.IssueURL(m.activeBoardID(), t.Jira); url != "" {
		// the id is a clickable link to the issue in the tracker. We don't use an OSC 8
		// hyperlink (the viewport can eat it, and Ghostty needs ⌘+click anyway): the id
		// line is registered as a hitbox and a plain click runs `open` (see updateMouse).
		linkText := t.ID + " ↗"
		idView = lipgloss.NewStyle().Foreground(lipgloss.Color(pal.accent)).Underline(true).Render(linkText)
		// x range of just the id text: box left (border 1 + padding 2 = 3) + the label width.
		x0 := 3 + lipgloss.Width(metaKey.Render(msg.mID))
		m.detailLinks = append(m.detailLinks, detailLink{
			line: len(lines) + 2, // +2 = box top border + padding (matches detailRows)
			x0:   x0, x1: x0 + lipgloss.Width(linkText), url: url,
		})
	}
	add(metaKey.Render(msg.mID) + idView)

	// navigable related ones: parent (↑) + subtasks (✓/▢). sel highlights; detailRows
	// stores each one's line (+2 = top border+padding) for the click. sel<0 = no
	// selection (e.g. the preview panel).
	idx := 0
	emit := func(mark, id, title string) {
		m.detailRows = append(m.detailRows, detailRow{id: id, line: len(lines) + 2})
		if idx == sel {
			add(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent)).
				Render(fmt.Sprintf("› %s %s %s", mark, id, title)))
		} else {
			add("  " + mark + " " + faint.Render(id) + " " + title)
		}
		idx++
	}
	if t.Parent != "" {
		if p := m.store.Get(t.Parent); p != nil {
			add("")
			emit("↑", p.ID, p.Title)
		}
	}
	if kids := m.store.Children(t.ID); len(kids) > 0 {
		done, total := m.childProgress(t.ID)
		add("")
		add(cardTitle.Render(fmt.Sprintf("%s  %d/%d", msg.subtasks, done, total)))
		for _, c := range kids {
			mark := "▢"
			if c.Status == m.doneCol() {
				mark = "✓"
			}
			emit(mark, c.ID, c.Title)
		}
	}

	add(faint.Render(strings.Repeat("─", iw)))
	if strings.TrimSpace(t.Notes) != "" {
		for _, ln := range strings.Split(strings.TrimRight(m.renderNotes(t.Notes, iw), "\n"), "\n") {
			add(ln)
		}
	} else {
		add(faint.Render(msg.noNotes))
	}
	// comments (pulled by the sync) as cards below the description; they scroll with the
	// detail viewport, and long bodies / links render via Glamour like the notes.
	if len(t.Comments) > 0 {
		add("")
		add(cardTitle.Render(fmt.Sprintf("%s · %d", msg.mComments, len(t.Comments))))
		for _, c := range t.Comments {
			add("")
			for _, ln := range strings.Split(m.commentCard(c, iw), "\n") {
				add(ln)
			}
		}
	}
	m.scanURLs(lines)
	// Width = iw + border(2) + padding(4) → content exactly iw (matches Glamour's wrap
	// and the divider). Height stretches the frame to fill the screen.
	box := cardBox.Width(iw + 6)
	if m.h > 6 {
		box = box.Height(m.h - 4)
	}
	return box.Render(strings.Join(lines, "\n"))
}

// badge is a colored label (theme text over a hue background). bg is hex/ANSI.
func badge(text, bg string) string {
	return lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color(pal.panelBg)).Background(lipgloss.Color(bg)).
		Padding(0, 1).Render(strings.ToUpper(text))
}

func statusBadge(s string) string {
	bg := pal.surface1 // backlog: neutral
	switch s {
	case task.StatusDoing:
		bg = pal.peach
	case task.StatusDone:
		bg = pal.green
	}
	return badge(statusLabel(s), bg)
}

func priorityBadge(p string) string {
	bg := pal.blue // normal priority
	switch p {
	case "high":
		bg = pal.red
	case "low":
		bg = pal.surface1
	}
	return badge(priorityLabel(p), bg)
}

// tagHues are the named colors a tag can have (order = cycling in the config). They
// are resolved in the active theme's palette → the tag color follows the theme.
var tagHues = []string{"mauve", "blue", "green", "yellow", "peach", "red", "teal"}

// isHexColor validates "#rgb" or "#rrggbb".
func isHexColor(s string) bool {
	if len(s) != 4 && len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// hueColor resolves a tag's color: hex ("#rrggbb") passes straight through; otherwise
// it's a named hue from the active palette; "" or unknown fall into mauve (historical default).
func hueColor(name string) string {
	if isHexColor(name) {
		return name
	}
	switch name {
	case "blue":
		return pal.blue
	case "green":
		return pal.green
	case "yellow":
		return pal.yellow
	case "peach":
		return pal.peach
	case "red":
		return pal.red
	case "teal":
		return pal.teal
	default:
		return pal.mauve
	}
}

// tagExists reports whether the name is already in the catalog (case-insensitive), to
// avoid a duplicate on creation.
func tagExists(name string, cat []task.TagDef) bool {
	for _, td := range cat {
		if strings.EqualFold(td.Name, name) {
			return true
		}
	}
	return false
}

// tagColor returns the hue configured for a tag (case-insensitive), or "" if it's
// not in the catalog — the catalog is a suggestion, not a lock.
func tagColor(name string, cat []task.TagDef) string {
	for _, td := range cat {
		if strings.EqualFold(td.Name, name) {
			return td.Color
		}
	}
	return ""
}

// chips renders tags as pills, colored by the catalog's hue (default mauve).
// chip renders one pill (padding 0,1 → 1 space on each side).
func chip(name string, cat []task.TagDef) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(pal.panelBg)).Background(lipgloss.Color(hueColor(tagColor(name, cat)))).
		Padding(0, 1).Render(name)
}

func chips(tags []string, cat []task.TagDef) string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = chip(t, cat)
	}
	return strings.Join(out, " ")
}

// chipsWrapped breaks the pills into lines that fit in width, each line
// left-aligned. Done by hand because lipgloss's word-wrap misaligns the background/
// padding of the colored pills when breaking (bug of tags that move to the 2nd line).
func chipsWrapped(tags []string, cat []task.TagDef, width int) string {
	var lines []string
	var cur []string
	curW := 0
	for _, t := range tags {
		w := lipgloss.Width(t) + 2 // padding 0,1
		add := w
		if len(cur) > 0 {
			add++ // separator space
		}
		if len(cur) > 0 && curW+add > width { // doesn't fit → close the line
			lines = append(lines, strings.Join(cur, " "))
			cur, curW, add = nil, 0, w
		}
		cur = append(cur, chip(t, cat))
		curW += add
	}
	if len(cur) > 0 {
		lines = append(lines, strings.Join(cur, " "))
	}
	return strings.Join(lines, "\n")
}

// commentCard renders one comment as a subtle rounded box: "author · when" header
// (author in accent) then the body via Glamour (so links/markdown work). w is the outer
// box width; the body wraps to the inner width.
func (m *Model) commentCard(c task.Comment, w int) string {
	iw := w - 4 // border(2) + padding(2)
	if iw < 6 {
		iw = 6
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(pal.accent)).Render(c.Author)
	if c.When != "" {
		header += faint.Render(" · " + c.When)
	}
	body := strings.TrimRight(m.renderNotes(c.Body, iw), "\n")
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(pal.surface1)).Padding(0, 1).Width(w)
	return box.Render(header + "\n" + body)
}

// renderNotes renders the notes' markdown with Glamour, falling back to the raw
// text if the renderer fails. The renderer is cached by width: it uses WithStandardStyle
// (dark/light of the active theme), so it NEVER queries the terminal — no hanging.
func (m *Model) renderNotes(md string, w int) string {
	if m.renderer == nil || m.rendererW != w {
		r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(m.glamStyle), glamour.WithWordWrap(w))
		if err != nil {
			return md
		}
		m.renderer, m.rendererW = r, w
	}
	out, err := m.renderer.Render(md)
	if err != nil {
		return md
	}
	return out
}
