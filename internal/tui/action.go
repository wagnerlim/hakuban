package tui

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/wagnerlim/hakuban/internal/task"
)

// Transition actions (F16/F22): moving a card between columns can fire a
// user-configurable command (on_enter of the destination / on_exit of the origin). The
// command runs as a subprocess IN THE BACKGROUND — the TUI never blocks — and reports
// progress on stdout (one line per update). Success (exit 0) commits the
// move; failure (exit ≠0) leaves the card at the origin with the reason (last line of
// stderr). The core speaks no HTTP: whatever creates/transitions in Jira is the script.

// cardAction is the (in-memory, disposable) state of the action in flight. Its target is ONE
// card (transition, id set) OR one column (sync, col set). ponytail: one action at a
// time; queue/concurrency only if it becomes real pain.
type cardAction struct {
	id     string  // target card (transition); "" when it's a sync
	col    string  // target column (sync); "" when it's a transition
	to     string  // destination column of the transition (applied on success)
	pct    int     // 0..100 coming from the script (the TARGET)
	shown  float64 // 0..1 shown in the bar — follows the target smoothly (animates the fill)
	label  string  // label of the current step
	done   bool    // success, showing ✓ until the clear
	failed bool    // failed, showing :( until the user tries again
	mirror bool    // agent action whose payload is a list of issues → the core writes the mirrors
	board  bool    // board-wide action (top bar button): the loader takes over that button

	reason string // failure reason (last line of stderr)
	// next is the destination's on_enter to fire AFTER this on_exit succeeds — the
	// exit→enter chaining. nil = this is already the last (or only) step; on success
	// the move commits. Set only on the on_exit leg of a transition that has both sides.
	next *pendingAction
}

// pendingAction is the on_enter queued to run after the on_exit of the same move succeeds.
type pendingAction struct {
	text  string // intention (agent) or script path
	agent bool   // true = intention for the headless agent; false = script path
}

// actionEvent is what the action goroutine pushes onto the channel: progress or the end.
type actionEvent struct {
	done, ok      bool
	pct           int
	label, reason string
	key           string // Jira key reported by the script (line `jira: KEY`) → stamping
	payload       string // agent's final text; on sync it carries the issues JSON
}

// actionClearMsg clears the success ✓ after a breather.
type actionClearMsg struct{}

// actionSuccessLinger is how long the ✓ stays visible before vanishing.
const actionSuccessLinger = 800 * time.Millisecond

// onActionEvent handles an event from the action channel: progress (updates the bar and
// goes back to listening) or the end (success commits the move and shows ✓; failure leaves the
// card at the origin with the reason). Ignores orphan events (action already cleared/cancelled).
func (m *Model) onActionEvent(ev actionEvent) (tea.Model, tea.Cmd) {
	if m.action == nil {
		return m, nil
	}
	if !ev.done {
		m.action.pct, m.action.label = ev.pct, ev.label
		// stamp the % onto the card so it outlives the loader (persisted to disk) and
		// shows on the mini-card's bar while the action runs.
		if m.action.id != "" {
			if t := m.store.Get(m.action.id); t != nil && t.Progress != ev.pct {
				t.Progress = ev.pct
				_ = m.store.Save(t)
			}
		}
		// animates the bar to the new % (spring) and keeps listening on the channel
		return m, tea.Batch(waitAction(m.actionCh), m.progress.SetPercent(float64(ev.pct)/100))
	}
	if !ev.ok { // failure: card stays where it is, shows :( until a retry
		m.action.failed, m.action.reason = true, ev.reason
		m.actionCh = nil
		m.syncQueue = m.syncQueue[:0] // abort the rest of a "sync all" batch
		return m, nil
	}
	// exit→enter chaining: the on_exit of the origin finished OK → now fire the
	// destination's on_enter; the move only commits when THAT one succeeds too. A
	// created key (stamping) has no `next`, so it never collides with this branch.
	if m.action.id != "" && ev.key == "" && m.action.next != nil {
		next := m.action.next
		if t := m.store.Get(m.action.id); t != nil {
			return m, m.startAction(t, t.Status, m.action.to, next.text, next.agent, nil)
		}
	}
	// success: transition commits the move (writes the new status); sync only rereads the
	// disk (the mirrors are written by the script). Shows ✓ for a breather.
	if m.action.id != "" {
		if ev.key != "" { // script created the issue → stamp it and turn into mirror (id = key)
			_ = m.store.LinkToJira(m.action.id, ev.key, m.action.to)
		} else if t := m.store.Get(m.action.id); t != nil {
			t.Status = m.action.to
			t.Progress = 0      // action done: clear the live % (no stale bar left on the card)
			_ = m.store.Save(t) // mirror → jira/, local → tasks/ (via taskFile)
		}
	} else if m.action.mirror && ev.payload != "" {
		// sync via agent (Slice B): the payload carries the issues JSON; the CORE writes
		// the mirrors (agent only fetched). Script-sync writes on its own → empty payload.
		// Only the synthesized sync sets mirror: a custom agent button returns prose, and
		// reconciling on that would wipe the board's mirrors.
		m.materializeSync(ev.payload)
	}
	m.action.done, m.action.pct = true, 100
	m.actionCh = nil
	m.refresh()                                     // reread the disk: pick up the mirrors the script wrote
	if m.action.col != "" && len(m.syncQueue) > 0 { // "sync all" batch: chain the next column
		return m, m.nextQueuedSync()
	}
	// animate to 100% and schedule the ✓ clear after the breather
	clear := tea.Tick(actionSuccessLinger, func(time.Time) tea.Msg { return actionClearMsg{} })
	return m, tea.Batch(m.progress.SetPercent(1), clear)
}

// colButton is a footer button as the TUI sees it: what the board declares, plus the
// mirror flag of the sync the core synthesizes for legacy boards (only that one turns the
// agent's payload into mirrors).
type colButton struct {
	task.Button
	mirror bool
}

// columnButtons is the column's footer buttons: what the board declares under `buttons:`
// or — when it declares none — the sync button synthesized from the legacy board-wide
// `sync:`/JQL config, so boards written before F23 keep working untouched.
func (m *Model) columnButtons(col string) []colButton {
	board := m.activeBoardID()
	if bs := m.store.ColumnButtons(board, col); len(bs) > 0 {
		out := make([]colButton, len(bs))
		for i, b := range bs {
			out[i] = colButton{Button: b}
		}
		return out
	}
	if !m.store.ColumnBound(board, col) {
		return nil
	}
	if cmd := m.store.SyncCommand(board); cmd != "" {
		return []colButton{{Button: task.Button{Label: msg.syncLabel, Icon: "⟳", Cmd: cmd, Batch: true}}}
	}
	if intention := m.boardSyncIntention(board); intention != "" {
		return []colButton{{Button: task.Button{Label: msg.syncLabel, Icon: "⟳", Agent: intention, Batch: true}, mirror: true}}
	}
	return nil
}

// boardMenuCol marks the open menu as the TOP BAR's, not a column's.
const boardMenuCol = -1

// boardButtons are the TOP BAR's buttons: what the board declares under its own `buttons:`
// or — none declared — the synthesized sync-all, which runs the columns' batch buttons. Same
// shape as a column's, so both menus share every bit of machinery.
func (m *Model) boardButtons() []colButton {
	board := m.activeBoardID()
	if bs := m.store.BoardButtons(board); len(bs) > 0 {
		out := make([]colButton, len(bs))
		for i, b := range bs {
			out[i] = colButton{Button: b}
		}
		return out
	}
	if len(m.batchQueue()) > 0 {
		return []colButton{{Button: task.Button{Label: msg.syncAllLabel, Icon: "⟳", Batch: true}}}
	}
	return nil
}

// activateBoard is what the top bar button does: one button fires, several open the menu
// below it. Mirror of activateColumn.
func (m *Model) activateBoard() tea.Cmd {
	switch bs := m.boardButtons(); len(bs) {
	case 0:
		return nil
	case 1:
		return m.triggerBoardButton(0)
	default:
		m.menuOpen, m.menuCol, m.menuSel, m.menuUp = true, boardMenuCol, 0, false
		return nil
	}
}

// triggerBoardButton fires button n of the top bar. `batch` runs the columns' batch buttons
// (each one's loader shows in its own footer); a cmd/agent runs board-wide, with the loader
// taking over the top bar button itself.
func (m *Model) triggerBoardButton(n int) tea.Cmd {
	bs := m.boardButtons()
	if n < 0 || n >= len(bs) {
		return nil
	}
	b := bs[n]
	if b.Batch {
		return m.batchAll() // has its own one-at-a-time guard
	}
	if m.action != nil && !m.action.failed {
		return nil
	}
	switch {
	case b.Cmd != "":
		return m.startBoardCmd(b.Cmd)
	case b.Agent != "":
		return m.startBoardAgent(b)
	}
	return nil
}

// startBoardCmd runs a top-bar button's script with ALL the board's cards as a JSON array on
// stdin — the board-wide counterpart of startButton (no column, so no JQL/filters).
func (m *Model) startBoardCmd(command string) tea.Cmd {
	ch := make(chan actionEvent, 16)
	m.action = &cardAction{board: true}
	m.actionCh = ch
	m.progress = newProgress()
	board, dir := m.activeBoardID(), m.store.Dir()
	cards := m.boardCardsJSON()
	go func() {
		env := []string{
			"HAKUBAN_BOARD=" + board, "HAKUBAN_DIR=" + dir,
			"HAKUBAN_TASK_BIN=" + selfBin(),
		}
		runCommand(command, cards, env, dir, ch)
	}()
	return waitAction(ch)
}

// startBoardAgent hands a top-bar button's intention to the headless agent, scoped by the
// board's agent_tools. Never sets mirror: only the synthesized sync reconciles the mirrors.
func (m *Model) startBoardAgent(b colButton) tea.Cmd {
	ch := make(chan actionEvent, 64)
	m.action = &cardAction{board: true}
	m.actionCh = ch
	m.progress = newProgress()
	go runAgent(resolveIntention(b.Agent, m.store.Dir()), nil, m.store.AgentTools(m.activeBoardID()), ch)
	return waitAction(ch)
}

// boardCardsJSON is every card of the active board as a JSON array, in column order — the
// stdin of a top-bar button's script. Each card carries its own column in from/to.
func (m *Model) boardCardsJSON() []byte {
	board := m.activeBoardID()
	out := make([]json.RawMessage, 0, 32)
	for _, col := range m.cols {
		for _, t := range col {
			out = append(out, cardJSON(t, t.Status, t.Status, board))
		}
	}
	b, _ := json.Marshal(out)
	return b
}

// btnRef points at one button of one column — what a batch queues.
type btnRef struct{ col, btn int }

// canBatch reports whether the active board has any button enrolled in the batch — gates
// the board-wide button on the top bar.
func (m *Model) canBatch() bool { return len(m.batchQueue()) > 0 }

// batchAll fires the board-wide batch: every `batch: true` button of every column, chained
// one at a time (see onActionEvent). No-op while an action runs.
func (m *Model) batchAll() tea.Cmd {
	if m.action != nil && !m.action.failed {
		return nil // one action at a time
	}
	m.syncQueue = m.batchQueue()
	return m.nextQueuedSync()
}

// batchQueue is what the board-wide button covers: the batch buttons of every column, in
// column order. A synthesized agent-sync appears once — it pulls the whole board in a
// single call, so running it per column would just repeat the same query.
func (m *Model) batchQueue() []btnRef {
	var q []btnRef
	agentSync := false
	for i, c := range m.columns {
		for n, b := range m.columnButtons(c) {
			if !b.Batch {
				continue
			}
			if b.mirror {
				if agentSync {
					continue
				}
				agentSync = true
			}
			q = append(q, btnRef{col: i, btn: n})
		}
	}
	return q
}

// nextQueuedSync pops the next button of a batch and fires it. Clears m.action first so
// triggerButton's one-at-a-time guard lets the chained call through.
func (m *Model) nextQueuedSync() tea.Cmd {
	if len(m.syncQueue) == 0 {
		return nil
	}
	r := m.syncQueue[0]
	m.syncQueue = m.syncQueue[1:]
	m.action = nil
	return m.triggerButton(r.col, r.btn)
}

// boardHasBinding says whether the active board has any bound column (F16) — decides
// whether the sync footer is reserved.
func (m *Model) boardHasBinding() bool {
	b := m.activeBoardID()
	for _, c := range m.columns {
		if m.store.ColumnBound(b, c) {
			return true
		}
	}
	return false
}

// syncFooter draws the footer of a column that has buttons: the loader while its action
// runs, otherwise the button itself (same mold as the modal buttons, via btn); it lights
// up when the column is active (the one the `y` key activates).
func (m *Model) syncFooter(col, label string, w int, active bool) string {
	if m.action != nil && m.action.col == col {
		return m.actionBar(m.action, w)
	}
	return btn(label, active, false)
}

// footerLabel is what the column's footer button reads: the single button's label (+ its
// icon), or — with several — the column's button_label (default: the first one's label)
// plus the ▾ that says "this opens a menu".
func (m *Model) footerLabel(col string, bs []colButton) string {
	if len(bs) == 0 {
		return ""
	}
	if len(bs) == 1 {
		return strings.TrimSpace(bs[0].Label + " " + bs[0].Icon)
	}
	label := m.store.ColumnButtonLabel(m.activeBoardID(), col)
	if label == "" {
		label = bs[0].Label
	}
	return label + " ▾"
}

// activateColumn is what pressing the footer button (or `y`) does: a single button fires
// right away; several open the column menu. No buttons → nothing.
func (m *Model) activateColumn(i int) tea.Cmd {
	if i < 0 || i >= len(m.columns) {
		return nil
	}
	switch bs := m.columnButtons(m.columns[i]); len(bs) {
	case 0:
		return nil
	case 1:
		return m.triggerButton(i, 0)
	default:
		m.menuOpen, m.menuCol, m.menuSel, m.menuUp = true, i, 0, true
		return nil
	}
}

// triggerButton fires button n of column i: a script gets the column's cards on stdin, an
// intention goes to the headless agent. No-op while another action runs.
func (m *Model) triggerButton(i, n int) tea.Cmd {
	if i < 0 || i >= len(m.columns) {
		return nil
	}
	col := m.columns[i]
	bs := m.columnButtons(col)
	if n < 0 || n >= len(bs) {
		return nil
	}
	if m.action != nil && !m.action.failed {
		return nil // one action at a time
	}
	b := bs[n]
	switch {
	case b.Cmd != "":
		return m.startButton(i, col, b.Cmd)
	case b.Agent != "":
		return m.startButtonAgent(col, b)
	}
	return nil
}

// syncOnOpen fires the board's first batch button when it opens, if the board asked for it
// (sync_on_open). Anchors the loader on that column. Nothing → Init proceeds with just the tick.
func (m *Model) syncOnOpen() tea.Cmd {
	if !m.store.SyncOnOpen(m.activeBoardID()) {
		return nil
	}
	if q := m.batchQueue(); len(q) > 0 {
		return m.triggerButton(q[0].col, q[0].btn)
	}
	return nil
}

// actionBar draws the action loader: a bar of 20 FIXED segments (each ▱/▰ = 5%),
// filling as the pct progresses (▰ = done, ▱ = to do). Determined and driven by the
// pct the agent reports. Palette styles (lipgloss): filled in accent (green
// on success), empty muted; ✗ + reason in red on failure. 20 blocks + " 100%" = 25
// runes, fits the slot (~26 with colW=36); on a very narrow terminal it can get tight.
const barCells = 20 // segments in a progress bar (each = 5%)

// loaders are the progress styles a column can pick (`loader: blocks` in its action). A style
// is just the pair of glyphs the bar is made of, so every one of them keeps the SAME geometry
// (barCells wide) and fits the footer and the mini-card alike. Unknown name → loaderDefault,
// never a broken render: a typo in the board file must not cost you the bar.
// Glyph pairs whose metrics don't match in the terminal font make the filled run look like it
// spills past the empty one's outline (many fonts draw ▰ as a slab taller than ▱) — cosmetic,
// and the price of picking the pair by shape.
var loaders = map[string][2]string{
	"segments": {"▰", "▱"},
	"blocks":   {"█", "░"},
	"line":     {"━", "─"},
	"dots":     {"●", "○"},
	"ascii":    {"#", "-"},
}

const loaderDefault = "segments"

// segBar renders a SEGMENTED bar in the named style: `filled` accent cells + the rest faint.
// The cells are NOT space-separated: at barCells=20 that would be 39 cells wide and blow past
// both the footer (colW-4) and the card (colW-10) — the color change already tells filled from
// empty. fill = accent (or okStyle on success). Shared by the live action loader and the
// persisted card % (progressBar) so the two look identical.
func segBar(filled int, fill lipgloss.Style, style string) string {
	g, ok := loaders[style]
	if !ok {
		g = loaders[loaderDefault]
	}
	filled = clamp(filled, 0, barCells)
	return fill.Render(strings.Repeat(g[0], filled)) + faint.Render(strings.Repeat(g[1], barCells-filled))
}

func (m *Model) actionBar(a *cardAction, w int) string {
	if a.failed {
		return failStyle.Render(truncate("✗ "+a.reason, w))
	}
	filled := clamp(int(a.shown*barCells+0.5), 0, barCells) // from the ANIMATED value
	fill := lipgloss.NewStyle().Foreground(lipgloss.Color(pal.accent))
	pctText := strconv.Itoa(clamp(int(a.shown*100+0.5), 0, 100)) + "%" // number climbs together with the bar
	if a.done {
		filled, fill, pctText = barCells, okStyle, "100%"
	}
	col := a.col
	if col == "" { // transition: the loader sits on the card, so its column drives the look
		if t := m.store.Get(a.id); t != nil {
			col = t.Status
		}
	}
	style, showPct := m.loaderStyle(col)
	bar := segBar(filled, fill, style)
	if !showPct {
		return bar
	}
	return bar + "  " + faint.Render(pctText)
}

// loaderStyle is a column's progress style + whether its bar shows the number. Both come
// from the column's own action (`loader:`/`percent:`), so a board can give each stage its
// own look; an empty column name (or one with no action) gets the defaults.
func (m *Model) loaderStyle(col string) (string, bool) {
	board := m.activeBoardID()
	return m.store.ColumnLoader(board, col), m.store.ColumnPercent(board, col)
}

// newProgress creates the themed progress bar (green→accent gradient of the active
// palette), with no percentage of its own. Recreated per theme and on each action start
// (so the bar starts from zero, not from where the previous one stopped).
func newProgress() progress.Model {
	return progress.New(
		progress.WithoutPercentage(),
		progress.WithColors(lipgloss.Color(pal.green), lipgloss.Color(pal.accent)),
	)
}

// startAction fires a card's transition action and returns the cmd that listens on the
// channel. agent=true → text is an INTENTION (prose) handed to the headless agent
// (model 1, F16); false → text is a script path (F22), which receives the card as
// --json on stdin + from/to/board in the env.
func (m *Model) startAction(t *task.Task, from, to, text string, agent bool, next *pendingAction) tea.Cmd {
	ch := make(chan actionEvent, 16)
	m.action = &cardAction{id: t.ID, to: to, next: next}
	m.actionCh = ch
	m.progress = newProgress() // bar from zero
	board := m.activeBoardID()
	card := cardJSON(t, from, to, board)
	if agent {
		// intention: if `text` points to a file (e.g. hooks/finalizar.md), read its
		// content — the hyper-personalized integration logic lives in dedicated .md files,
		// outside the board. Then interpolate {{.jira}}/… with the card and hand it to the agent.
		go runAgent(interpolate(resolveIntention(text, m.store.Dir()), t), card, m.store.AgentTools(board), ch)
		return waitAction(ch)
	}
	// HAKUBAN_STATUS = the status in the tracker that the destination column represents (target of the
	// transition). Empty if the destination is a local column with no mapping.
	target, _ := m.store.BindingFor(board, to)
	env := []string{
		"HAKUBAN_FROM=" + from, "HAKUBAN_TO=" + to, "HAKUBAN_BOARD=" + board,
		"HAKUBAN_STATUS=" + target.Status, "HAKUBAN_DIR=" + m.store.Dir(),
		"HAKUBAN_TASK_BIN=" + selfBin(),
	}
	go runCommand(text, card, env, m.store.Dir(), ch)
	return waitAction(ch)
}

// startButton fires a column button's script: it gets the column's cards as a JSON array
// on stdin and the column's JQL/filters/dir in the environment. Whatever it writes (the
// sync's mirrors, or anything else) is its own business — on success the core only rereads
// the disk.
func (m *Model) startButton(i int, col, command string) tea.Cmd {
	ch := make(chan actionEvent, 16)
	m.action = &cardAction{col: col}
	m.actionCh = ch
	m.progress = newProgress() // bar from zero
	board, dir, store := m.activeBoardID(), m.store.Dir(), m.store
	a, _ := store.BindingFor(board, col)
	jql := a.JQL
	cards := columnJSON(m.cols[i], col, board)
	comments := "0"
	if m.store.CommentsEnabled(board) { // signal the script to also pull comments
		comments = "1"
	}
	// resolve the filters (may run a dynamic options_cmd) inside the goroutine so it never
	// blocks the UI; then run the sync command with HAKUBAN_FILTERS grouped by filter.
	go func() {
		env := []string{
			"HAKUBAN_JQL=" + jql, "HAKUBAN_COLUMN=" + col,
			"HAKUBAN_BOARD=" + board, "HAKUBAN_DIR=" + dir,
			"HAKUBAN_FILTERS=" + filtersJSON(resolveSyncFilters(store, board, col)),
			"HAKUBAN_COMMENTS=" + comments,
			"HAKUBAN_TASK_BIN=" + selfBin(), // a button acting on the column's cards needs it
		}
		runCommand(command, cards, env, dir, ch)
	}()
	return waitAction(ch)
}

// startButtonAgent fires a column button's intention on the headless agent, scoped by the
// board's agent_tools. The intention may be a .md path under the data dir (resolveIntention).
// On the synthesized sync (mirror) the agent returns the issues as JSON and the CORE writes
// the mirrors (materializeSync) — the agent never touches the disk.
func (m *Model) startButtonAgent(col string, b colButton) tea.Cmd {
	ch := make(chan actionEvent, 64)
	m.action = &cardAction{col: col, mirror: b.mirror}
	m.actionCh = ch
	m.progress = newProgress()
	go runAgent(resolveIntention(b.Agent, m.store.Dir()), nil, m.store.AgentTools(m.activeBoardID()), ch)
	return waitAction(ch)
}

// columnJSON is the column's cards as a JSON array — what a button's script gets on stdin,
// so it can act on every card of the column. Same per-card shape as a transition hook
// (from/to are the column itself: nothing is moving).
func columnJSON(cards []*task.Task, col, board string) []byte {
	out := make([]json.RawMessage, 0, len(cards))
	for _, t := range cards {
		out = append(out, cardJSON(t, col, col, board))
	}
	b, _ := json.Marshal(out)
	return b
}

// boardSyncIntention assembles the board-wide sync intention: the OR of the JQLs of all
// bound columns + the request to return ONLY a JSON. "" if no column is bound.
func (m *Model) boardSyncIntention(board string) string {
	var jqls []string
	for _, c := range m.columns {
		if a, ok := m.store.BindingFor(board, c); ok && a.JQL != "" {
			jqls = append(jqls, "("+a.JQL+")")
		}
	}
	if len(jqls) == 0 {
		return ""
	}
	return "Liste no Jira (connector Atlassian/Rovo via MCP) as issues que casam com esta JQL:\n\n    " +
		strings.Join(jqls, " OR ") +
		"\n\nResponda SOMENTE com um array JSON (sem prosa, sem cercas ```), um objeto por " +
		"issue com os campos: key (string), summary (string), status (string, o NOME do " +
		"status atual), priority (string, o nome da prioridade ou \"\").\n" +
		`Exemplo: [{"key":"ABC-1","summary":"t","status":"Concluído","priority":"High"}]`
}

// materializeSync converts the JSON the agent returned into mirrors: maps the Jira
// status → column (via ColumnAction.Status) and reconciles. An issue in a status that maps
// to no column is ignored. The core writes; the agent never touched disk.
func (m *Model) materializeSync(payload string) {
	board := m.activeBoardID()
	stat2col := map[string]string{} // tracker status → board column
	for _, c := range m.columns {
		if a, ok := m.store.BindingFor(board, c); ok && a.Status != "" {
			stat2col[a.Status] = c
		}
	}
	var specs []task.MirrorSpec
	for _, it := range parseSyncIssues(payload) {
		if col := stat2col[it.Status]; col != "" {
			specs = append(specs, task.MirrorSpec{
				Key: it.Key, Title: it.Summary, Status: col,
				Priority: strings.ToLower(it.Priority),
			})
		}
	}
	_, _ = m.store.ReconcileMirrors(board, specs)
}

// syncIssue is an issue in the JSON the agent returns on sync.
type syncIssue struct {
	Key      string `json:"key"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

// parseSyncIssues extracts the JSON array of issues from the agent's final text — which may
// come with prose or ```json fences around it. Takes from the 1st '[' to the last ']'. Empty
// list if it doesn't find/parse it (reconcile then empties the board's mirrors).
func parseSyncIssues(s string) []syncIssue {
	i, j := strings.IndexByte(s, '['), strings.LastIndexByte(s, ']')
	if i < 0 || j <= i {
		return nil
	}
	var out []syncIssue
	if json.Unmarshal([]byte(s[i:j+1]), &out) != nil {
		return nil
	}
	return out
}

// waitAction blocks on one item from the channel and returns it as a msg (the Bubble Tea
// "listen on a channel" pattern); Update re-emits this cmd until the final event arrives.
func waitAction(ch <-chan actionEvent) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

// runCommand runs `sh -c command` with the given stdin/extraEnv, streaming each
// progress line from stdout through the channel. stderr is buffered; the last line
// becomes the reason on failure. Serves both the transition action and sync. dir is the
// working directory (the data dir) so a hook referenced by a RELATIVE path
// (`hooks/jira-transition.sh`) resolves regardless of where the binary was launched.
func runCommand(command string, stdin []byte, extraEnv []string, dir string, ch chan<- actionEvent) {
	cmd := exec.Command("sh", "-c", command)
	if dir != "" {
		cmd.Dir = dir
	}
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	cmd.Env = append(os.Environ(), extraEnv...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch <- actionEvent{done: true, ok: false, reason: err.Error()}
		return
	}
	if err := cmd.Start(); err != nil {
		ch <- actionEvent{done: true, ok: false, reason: err.Error()}
		return
	}
	sc := bufio.NewScanner(stdout)
	var key string
	for sc.Scan() {
		line := sc.Text()
		if k, ok := parseKey(line); ok { // line `jira: KEY` → key created
			if k != "" {
				key = k
			}
			continue
		}
		if pct, label, ok := parseProgress(line); ok {
			ch <- actionEvent{pct: pct, label: label}
		}
	}
	err = cmd.Wait()
	if err == nil {
		ch <- actionEvent{done: true, ok: true, pct: 100, key: key}
		return
	}
	reason := lastLine(stderr.String())
	if reason == "" {
		reason = err.Error()
	}
	ch <- actionEvent{done: true, ok: false, reason: reason}
}

// runAgent runs the headless agent (`claude -p`) with the interpolated intention + the card,
// scoped to the tracker's tools via --allowedTools (MCP only; never disk). Findings
// from the real test (2026-07-21): NEUTRAL cwd (otherwise the agent explores the user's
// project and inherits its permissions) and ANTHROPIC_API_KEY REMOVED (the key turns off the
// claude.ai connectors — without it the agent uses the subscription, which can see the MCP).
// Progress comes from the stream-json: each tool_use of the agent becomes a loader step.
// ponytail: no --disallowedTools — in testing it made the agent give up on the MCP.
func runAgent(intention string, card []byte, tools string, ch chan<- actionEvent) {
	prompt := intention
	if len(card) > 0 { // transition sends the card; sync (pull) has no card
		prompt += "\n\n<card>\n" + string(card) + "\n</card>"
	}
	args := []string{"-p", prompt, "--output-format", "stream-json", "--verbose"}
	if tools != "" {
		args = append(args, "--allowedTools", tools)
	}
	cmd := exec.Command("claude", args...)
	cmd.Dir = agentCwd()
	cmd.Env = envWithout("ANTHROPIC_API_KEY")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch <- actionEvent{done: true, reason: err.Error()}
		return
	}
	if err := cmd.Start(); err != nil {
		ch <- actionEvent{done: true, reason: err.Error()} // e.g. claude not installed / outside PATH
		return
	}
	ch <- actionEvent{pct: 5} // move the bar off the start line, before the 1st tool_use
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024) // stream-json lines are large
	step := 0
	var lastResult string
	for sc.Scan() {
		ev, isResult, res := parseAgentLine(sc.Bytes(), &step)
		if res != "" {
			lastResult = res
		}
		if isResult {
			continue // the final verdict comes from the exit code in Wait()
		}
		if ev != nil {
			ch <- *ev
		}
	}
	if err := cmd.Wait(); err != nil {
		reason := firstNonEmpty(lastResult, lastLine(stderr.String()), err.Error())
		ch <- actionEvent{done: true, reason: truncate(reason, 200)}
		return
	}
	// payload = agent's final text; on sync (pull) it carries the issues JSON.
	ch <- actionEvent{done: true, ok: true, pct: 100, payload: lastResult}
}

// agentLine is the minimum of the `claude -p` stream-json that the loader needs.
type agentLine struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"content"`
	} `json:"message"`
}

// parseAgentLine turns one stream-json line into a progress event. Each
// tool_use of the agent advances a step (we don't know the total → ladder up to 90%) and becomes the
// label. Returns isResult=true on the final line (`type:result`), with the verdict
// text in res (used as the reason if the exit is ≠0).
func parseAgentLine(line []byte, step *int) (ev *actionEvent, isResult bool, res string) {
	var a agentLine
	if json.Unmarshal(line, &a) != nil {
		return nil, false, ""
	}
	switch a.Type {
	case "assistant":
		for _, c := range a.Message.Content {
			if c.Type == "tool_use" {
				*step++
				pct := min(90, 10+*step*15)
				return &actionEvent{pct: pct, label: shortTool(c.Name)}, false, ""
			}
		}
	case "result":
		return nil, true, a.Result
	}
	return nil, false, ""
}

// shortTool shortens the tool name for the loader label: mcp__…__transitionJiraIssue
// becomes "transitionJiraIssue"; a name without "__" stays as is (e.g. "ToolSearch").
func shortTool(name string) string {
	if i := strings.LastIndex(name, "__"); i >= 0 {
		return name[i+2:]
	}
	return name
}

// selfBin is the path of the running hakuban binary, exported to hooks as
// HAKUBAN_TASK_BIN so a hook (jira-doing.sh) can re-invoke it (e.g. `hakuban doing`)
// without depending on PATH. Falls back to os.Args[0] if os.Executable fails.
func selfBin() string {
	if p, err := os.Executable(); err == nil {
		return p
	}
	return os.Args[0]
}

// agentCwd is a neutral, empty directory where the agent runs — with no project to
// explore nor .claude/settings to inherit. Reused across calls.
func agentCwd() string {
	d := filepath.Join(os.TempDir(), "hakuban-agent")
	_ = os.MkdirAll(d, 0o755)
	return d
}

// envWithout returns os.Environ() without the given variable (used to strip the
// ANTHROPIC_API_KEY, which turns off the claude.ai connectors).
func envWithout(key string) []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, key+"=") {
			continue
		}
		out = append(out, e)
	}
	return out
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// resolveIntention returns the agent's intention. If `v` points to an existing
// file — absolute, ~/…, or relative to the data dir (e.g. hooks/finalizar.md) — it reads
// the content: this way the hyper-personalized intention lives in a dedicated .md in the
// config folder, and the board keeps only the structure (points to the file). Otherwise, `v` is already the
// inline intention (prose with spaces doesn't match as a path → falls through here).
func resolveIntention(v, baseDir string) string {
	p := strings.TrimSpace(v)
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(h, p[2:])
		}
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, p)
	}
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		if b, err := os.ReadFile(p); err == nil {
			return string(b)
		}
	}
	return v
}

// interpolate resolves {{.jira}}/{{.title}}/… in the intention with the card's fields.
// Invalid template or error → returns the raw text (the intention still makes sense).
func interpolate(tmpl string, t *task.Task) string {
	tp, err := template.New("i").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return tmpl
	}
	data := map[string]string{
		"jira": t.Jira, "title": t.Title, "id": t.ID, "status": t.Status,
		"priority": t.Priority, "project": t.Project, "notes": t.Notes,
	}
	var b strings.Builder
	if tp.Execute(&b, data) != nil {
		return tmpl
	}
	return b.String()
}

// parseProgress reads a progress line: `N` (0..100) or `N/M` (step N of M),
// with an optional label after. ok=false for a line that doesn't start with a number —
// this way a stray log from the script on stdout is ignored, doesn't become a bar.
func parseProgress(line string) (pct int, label string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, "", false
	}
	num, rest, _ := strings.Cut(line, " ")
	label = strings.TrimSpace(rest)
	if n, den, isStep := strings.Cut(num, "/"); isStep {
		a, e1 := strconv.Atoi(n)
		b, e2 := strconv.Atoi(den)
		if e1 != nil || e2 != nil || b <= 0 {
			return 0, "", false
		}
		pct = a * 100 / b
	} else {
		p, e := strconv.Atoi(num)
		if e != nil {
			return 0, "", false
		}
		pct = p
	}
	return clamp(pct, 0, 100), label, true
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }

// parseKey recognizes the `jira: KEY` control line in the script's stdout — how the
// core stamps the created issue's key onto the card (stamping, F16).
func parseKey(line string) (string, bool) {
	if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "jira:"); ok {
		return strings.TrimSpace(rest), true
	}
	return "", false
}

// lastLine returns the last non-empty line of s (the failure reason).
func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); t != "" {
			return t
		}
	}
	return ""
}

// cardJSON serializes the card + transition context for the script's stdin.
func cardJSON(t *task.Task, from, to, board string) []byte {
	payload := struct {
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		Status   string   `json:"status"`
		Priority string   `json:"priority,omitempty"`
		Project  string   `json:"project,omitempty"`
		Tags     []string `json:"tags,omitempty"`
		Jira     string   `json:"jira,omitempty"`
		Notes    string   `json:"notes,omitempty"`
		From     string   `json:"from"`
		To       string   `json:"to"`
		Board    string   `json:"board"`
	}{t.ID, t.Title, t.Status, t.Priority, t.Project, t.Tags, t.Jira, t.Notes, from, to, board}
	b, _ := json.Marshal(payload)
	return b
}
