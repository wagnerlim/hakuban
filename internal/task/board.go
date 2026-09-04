package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Board is a board. The ID is the value of the cards' `project` field: the board groups
// the tasks whose *effective* project == ID. It exists as a boards/<id>.md file to
// allow an empty board and a display name — but boards are also discovered from
// the projects in use, so old projects show up without needing a file.
type Board struct {
	ID   string `yaml:"-"`
	Name string `yaml:"name,omitempty"`
	// Key is the prefix of the cards' keys (e.g. CASA → CASA-01). Unique across
	// boards. Empty = cards fall back to the legacy random id.
	Key string `yaml:"key,omitempty"`
	// Columns are the board's own columns (status, in order). A pointer to
	// distinguish "never configured" (nil → falls back to DefaultColumns; Inbox and
	// legacy/discovered boards) from "configured empty" (&[]{} → new board with no columns).
	Columns *[]string `yaml:"columns,omitempty"`
	// Actions is the per-column integration config (F16), keyed by the column
	// name. Optional: a column absent from the map is purely local. Nil = board with no
	// integration (omitempty → existing boards don't gain the key).
	Actions map[string]ColumnAction `yaml:"actions,omitempty"`
	// Filters is the board's registry of named, reusable filters (name → Filter). A column
	// opts into them via ColumnAction.UseFilters. The core stores/forwards, never composes.
	// Board-scoped: a filter serves this board, not all boards.
	Filters map[string]Filter `yaml:"filters,omitempty"`
	// Buttons are the TOP BAR's buttons — the board-wide counterpart of a column's. Same
	// shape (label/icon/cmd/agent), except `batch: true` here means "run the batch buttons
	// of every column, one at a time" (what the old sync-all button did). A `cmd` gets ALL
	// the board's cards as a JSON array on stdin. None declared → the sync-all button is
	// synthesized, so boards written before this keep it.
	Buttons []Button `yaml:"buttons,omitempty"`
	// ButtonLabel names the top bar button when there are several (default: the first one's
	// label). Optional.
	ButtonLabel string `yaml:"button_label,omitempty"`
	// Sync is the (user's) command that pulls the issues of the bound columns and
	// materializes the mirrors in jira/ (F16). One per board; it receives the column's
	// JQL via the environment. Empty = no sync button.
	Sync string `yaml:"sync,omitempty"`
	// AgentTools is the --allowedTools passed to the headless agent when the action is an
	// intent (prose), not a script (F16, evolution 2026-07-21). It scopes what the agent
	// may use — only the tracker's tools via MCP, never the disk. It is what keeps the core
	// tracker-agnostic: the board says "mcp__…_Atlassian_Rovo", GitHub/Linear swap the
	// string. Empty = no declared restriction (the agent decides; less safe).
	AgentTools string `yaml:"agent_tools,omitempty"`
	// SyncOnOpen triggers the sync of the bound columns when the board is opened (F16, Slice B):
	// each linked card drops into the column of its current Jira status, avoiding working
	// on top of something someone else already finished. Opt-in (costs one call to the agent).
	SyncOnOpen bool `yaml:"sync_on_open,omitempty"`
	// SyncButton places the board-wide "sync all" button on the top bar: "top-right"
	// (default when empty and the board can sync), "top-left", or "off" to hide it. The
	// button runs the same `sync` script across every bound column — nothing tracker-specific.
	SyncButton string `yaml:"sync_button,omitempty"`
	// Comments opts the board into pulling issue comments on sync. It's just a signal to
	// the sync script (via HAKUBAN_COMMENTS) — the script fetches and writes them onto each
	// card's `comments`. The detail view then shows them. Core stays tracker-agnostic.
	Comments bool `yaml:"comments,omitempty"`
	// IssueURL is a link template to open a card in the tracker; `{key}` is replaced by
	// the card's issue key (Task.Jira). E.g. "https://acme.atlassian.net/browse/{key}".
	// Tracker-agnostic: the board provides the URL shape, the core only substitutes.
	IssueURL string    `yaml:"issue_url,omitempty"`
	Created  time.Time `yaml:"created,omitempty"`
}

// ColumnAction is the integration config of ONE column (F16): the JQL that feeds it
// (pull, read-only mirrors) and the commands fired when a card enters/leaves
// it — the seam from F22. All optional and resolved by the user's script; the
// core never speaks HTTP, it only fires the command.
type ColumnAction struct {
	JQL string `yaml:"jql,omitempty"`
	// Guide is a DECLARATIVE, read-before-acting note about the column: what it is and how
	// work advances from it (e.g. "to refine a card, move it forward from here"). Unlike the
	// on_enter/on_exit intents — which are EXECUTED reactively, as a consequence of a move —
	// the guide is never run; it is standing context an agent reads (from the board file) to
	// DECIDE what to do. The core only parses and preserves it. Optional.
	Guide string `yaml:"guide,omitempty"`
	// OnEnter/OnExit are INTENTS (Markdown prose of what the agent should do) —
	// interpolated with the card and handed to the headless agent (F16, evolution).
	OnEnter string `yaml:"on_enter,omitempty"`
	OnExit  string `yaml:"on_exit,omitempty"`
	// OnEnterCmd/OnExitCmd are the deterministic alternative (F22): a script path
	// run via sh -c, receiving the card as --json on stdin. The intent takes priority
	// over the command on the same side; the hook-script is for whoever wants exact control.
	OnEnterCmd string `yaml:"on_enter_cmd,omitempty"`
	OnExitCmd  string `yaml:"on_exit_cmd,omitempty"`
	// Status is the status in the tracker that this column represents (e.g. "Done").
	// It goes as HAKUBAN_STATUS to the action's command — it is the target of the transition,
	// without the core needing to understand Jira. Optional.
	Status string `yaml:"status,omitempty"`
	// Buttons are the column's footer buttons (F23). ONE → the footer fires it directly;
	// TWO OR MORE → the footer opens a menu listing them. This is what makes the footer
	// agnostic: the column says what its button does, sync is just one possible button.
	Buttons []Button `yaml:"buttons,omitempty"`
	// ButtonLabel names the footer button when the column has several (default: the label
	// of the first one). Optional.
	ButtonLabel string `yaml:"button_label,omitempty"`
	// Loader picks the progress style shown while THIS column's action runs — and on the %
	// stamped on its cards, so both speak the same visual language. Named style resolved by
	// the TUI (see the loaders registry); empty or unknown falls back to the default bar.
	Loader string `yaml:"loader,omitempty"`
	// Percent shows the number next to the bar. A POINTER so an absent key (the default,
	// and every board written before this) still means true — `percent: false` is what
	// leaves the bar alone.
	Percent *bool `yaml:"percent,omitempty"`
	// UseFilters is the column's selection from the board's `filters` registry: filter name
	// → selected option labels (empty list = a simple filter, just toggled on). The core
	// resolves the selections to values and hands them to the sync hook GROUPED as
	// HAKUBAN_FILTERS — it never composes them (the hook ORs a filter's options and ANDs across
	// filters). Keeps the core tracker-agnostic. Optional.
	UseFilters map[string][]string `yaml:"use_filters,omitempty"`
}

// Button is ONE action button in a column's footer. The board says what it does, the core
// only draws it and runs it — nothing tracker-specific here. Exactly one of Cmd/Agent:
//
//   - Cmd is a script run via sh -c, receiving the column's cards as a JSON array on stdin
//     (that's what lets a button act on EVERY card of the column) plus the column's
//     JQL/filters/dir in the environment, exactly like a transition hook.
//   - Agent is an intention (prose, or the path of a .md under the data dir) handed to the
//     headless agent with the board's agent_tools.
//
// Batch enrolls the button in the board-wide button on the top bar, which fires the batch
// buttons of every column, one at a time.
type Button struct {
	Label string `yaml:"label,omitempty"`
	Icon  string `yaml:"icon,omitempty"`
	Cmd   string `yaml:"cmd,omitempty"`
	Agent string `yaml:"agent,omitempty"`
	Batch bool   `yaml:"batch,omitempty"`
}

// Filter is a named, reusable filter in a board's registry (Board.Filters). Two NATURES:
//
//   - TRACKER filter (Value/Options/OptionsCmd): an opaque value the core never interprets —
//     it just forwards the selection to the sync hook, which composes it into the query (JQL).
//     Online only: it shapes what `sync` pulls; it can't touch a local card.
//   - FIELD filter (Field/WithinDays): a predicate over the Task's OWN fields, which the core
//     evaluates LOCALLY (see Keep). Works offline, on ANY card — local or mirror. This is NOT
//     tracker coupling: the core reads its own model (modified/created), never tracker semantics.
//
// Tracker forms: simple (single Value, toggled on/off), static options (Options label→value,
// multi-select), or dynamic options (OptionsCmd lists them at use time, e.g. Jira sprints).
// YAML accepts a scalar shorthand for the simple tracker form.
type Filter struct {
	Value      string            `yaml:"value,omitempty"`
	Options    map[string]string `yaml:"options,omitempty"`
	OptionsCmd string            `yaml:"options_cmd,omitempty"`
	// Field/WithinDays make this a FIELD filter (evaluated locally). Field names a core Task
	// field ("modified" | "created"); WithinDays keeps cards whose Field is within N days of now.
	Field      string `yaml:"field,omitempty"`
	WithinDays int    `yaml:"within_days,omitempty"`
}

// IsField reports whether this filter is evaluated locally against a Task field (offline),
// as opposed to a tracker filter composed into the query by the sync hook.
func (f Filter) IsField() bool { return f.Field != "" }

// Keep evaluates a FIELD filter against a task: true = the card stays visible. `now` is passed
// in (no hidden clock — testable). A tracker filter, an unknown field, or a zero timestamp
// keeps everything, so a misconfigured filter never hides cards silently.
func (f Filter) Keep(t *Task, now time.Time) bool {
	if !f.IsField() {
		return true
	}
	var ts time.Time
	switch f.Field {
	case "modified":
		ts = t.Modified
	case "created":
		ts = t.Created
	default:
		return true // unknown field → no-op
	}
	if f.WithinDays > 0 {
		if ts.IsZero() {
			return true
		}
		return now.Sub(ts) <= time.Duration(f.WithinDays)*24*time.Hour
	}
	return true
}

// UnmarshalYAML lets a filter be written as a scalar (`contas: '<jql>'` → Value) or a
// mapping (`sprint: {options_cmd: hooks/jira-sprints.sh}`). ponytail: hand-friendly shorthand.
func (f *Filter) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		f.Value = n.Value
		return nil
	}
	type raw Filter // break the recursion into this method
	var r raw
	if err := n.Decode(&r); err != nil {
		return err
	}
	*f = Filter(r)
	return nil
}

// InboxID is the empty id: a sentinel for "task with no project / no active board". It
// used to be a built-in board ("Inbox"); today there is no built-in board — every task is born
// in a real board and AdoptOrphans migrates the legacy ones with no project.
const InboxID = ""

func boardsDir(dir string) string { return filepath.Join(dir, "boards") }

// loadBoards reads boards/*.md into memory. A missing directory is not an error.
func (s *Store) loadBoards() error {
	entries, err := os.ReadDir(boardsDir(s.dir))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		content, err := os.ReadFile(filepath.Join(boardsDir(s.dir), e.Name()))
		if err != nil {
			return err
		}
		fm, _ := splitFrontmatter(string(content))
		var b Board
		if err := yaml.Unmarshal([]byte(fm), &b); err != nil {
			return err
		}
		b.ID = id
		s.boards[id] = &b
	}
	return nil
}

// Boards returns all boards sorted by name. It is the union of the boards with
// a file + the effective projects in use by the tasks. There is no longer a built-in board.
func (s *Store) Boards() []*Board {
	set := map[string]*Board{}
	for id, b := range s.boards {
		set[id] = b
	}
	for _, t := range s.tasks {
		if p := s.EffectiveProject(t.ID); p != "" && set[p] == nil {
			set[p] = &Board{ID: p, Name: p}
		}
	}
	out := make([]*Board, 0, len(set))
	for _, b := range set {
		if b.Name == "" {
			b.Name = b.ID
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// BoardName is a board's display name (the id itself if there is no file).
func (s *Store) BoardName(id string) string {
	if b := s.boards[id]; b != nil && b.Name != "" {
		return b.Name
	}
	return id
}

// AdoptOrphans gives a home to root tasks with no project (legacy from the old Inbox): it creates
// the board boardID (if it does not exist yet) with the given columns and moves the orphans into
// it. Returns how many it adopted. Runs at boot; goes away on its own when there are no orphans.
func (s *Store) AdoptOrphans(boardID, name string, cols []string) int {
	var orphans []*Task
	for _, t := range s.tasks {
		if t.Parent == "" && t.Project == "" {
			orphans = append(orphans, t)
		}
	}
	if len(orphans) == 0 {
		return 0
	}
	if _, ok := s.boards[boardID]; !ok {
		c := append([]string(nil), cols...)
		key := s.UniqueKey(name, boardID)
		if err := s.SaveBoard(&Board{ID: boardID, Name: name, Key: key, Columns: &c}); err != nil {
			return 0
		}
	}
	for _, t := range orphans {
		t.Project = boardID
		s.Save(t)
	}
	return len(orphans)
}

// ColumnsFor returns the columns (status, in order) of a board. A board with its own
// config uses its own; Inbox and boards with no `columns` fall back to DefaultColumns. It always
// returns a copy — the caller can reorder freely.
func (s *Store) ColumnsFor(boardID string) []string {
	if b := s.boards[boardID]; b != nil && b.Columns != nil {
		return append([]string(nil), *b.Columns...)
	}
	return append([]string(nil), DefaultColumns...)
}

// BindingFor returns the integration config of a column and whether it exists (F16).
func (s *Store) BindingFor(boardID, col string) (ColumnAction, bool) {
	if b := s.boards[boardID]; b != nil {
		a, ok := b.Actions[col]
		return a, ok
	}
	return ColumnAction{}, false
}

// ColumnBound reports whether the column is bound to an integration (a JQL and/or
// use_filters) — it controls the sync button and the read-only render of the mirrors. An
// action with no JQL nor filters (just on_enter/exit on a local board) does not count.
func (s *Store) ColumnBound(boardID, col string) bool {
	a, ok := s.BindingFor(boardID, col)
	return ok && (a.JQL != "" || len(a.UseFilters) > 0)
}

// ColumnButtons returns the column's configured footer buttons (nil = none configured;
// the TUI may still synthesize the legacy sync button from the board's `sync`).
func (s *Store) ColumnButtons(boardID, col string) []Button {
	a, _ := s.BindingFor(boardID, col)
	return a.Buttons
}

// BoardButtons returns the board's top-bar buttons as configured (nil = none; the TUI may
// still synthesize the sync-all button).
func (s *Store) BoardButtons(boardID string) []Button {
	if b := s.boards[boardID]; b != nil {
		return b.Buttons
	}
	return nil
}

// BoardButtonLabel is the label of the top-bar button when the board has several
// ("" = the TUI falls back to the first button's label).
func (s *Store) BoardButtonLabel(boardID string) string {
	if b := s.boards[boardID]; b != nil {
		return b.ButtonLabel
	}
	return ""
}

// ColumnLoader is the name of the column's progress style ("" = the TUI's default).
func (s *Store) ColumnLoader(boardID, col string) string {
	a, _ := s.BindingFor(boardID, col)
	return a.Loader
}

// ColumnPercent reports whether the column's bar shows the number. Default (absent key): yes.
func (s *Store) ColumnPercent(boardID, col string) bool {
	a, _ := s.BindingFor(boardID, col)
	return a.Percent == nil || *a.Percent
}

// ColumnButtonLabel is the label of the footer button that opens the column's menu
// ("" = the TUI falls back to the first button's label).
func (s *Store) ColumnButtonLabel(boardID, col string) string {
	a, _ := s.BindingFor(boardID, col)
	return a.ButtonLabel
}

// FilterDef returns a board's filter definition by name (for the resolver in the tui layer,
// which also runs OptionsCmd for the dynamic ones).
func (s *Store) FilterDef(boardID, name string) (Filter, bool) {
	if b := s.boards[boardID]; b != nil {
		f, ok := b.Filters[name]
		return f, ok
	}
	return Filter{}, false
}

// ColumnUseFilters returns the column's selection (filter name → selected option labels).
func (s *Store) ColumnUseFilters(boardID, col string) map[string][]string {
	if a, ok := s.BindingFor(boardID, col); ok {
		return a.UseFilters
	}
	return nil
}

// ColumnHasFilters reports whether the column declares any use_filters (for the funnel
// icon in the header). Checks the declaration, not whether the names resolve.
func (s *Store) ColumnHasFilters(boardID, col string) bool {
	a, ok := s.BindingFor(boardID, col)
	return ok && len(a.UseFilters) > 0
}

// BoardFilterNames returns the board's registered filter names, sorted (stable UI order).
func (s *Store) BoardFilterNames(boardID string) []string {
	b := s.boards[boardID]
	if b == nil || len(b.Filters) == 0 {
		return nil
	}
	out := make([]string, 0, len(b.Filters))
	for name := range b.Filters {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// SetColumnUseFilters persists a column's filter selection (use_filters) and saves the
// board. An empty selection is dropped so the column stops being "bound" by filters.
func (s *Store) SetColumnUseFilters(boardID, col string, use map[string][]string) error {
	b := s.boards[boardID]
	if b == nil {
		return nil
	}
	if b.Actions == nil {
		b.Actions = map[string]ColumnAction{}
	}
	a := b.Actions[col]
	if len(use) == 0 {
		use = nil
	}
	a.UseFilters = use
	b.Actions[col] = a
	return s.SaveBoard(b)
}

// SyncCommand is the board's sync command ("" = none).
func (s *Store) SyncCommand(boardID string) string {
	if b := s.boards[boardID]; b != nil {
		return b.Sync
	}
	return ""
}

// SyncButtonPos is where the board-wide sync button sits: "top-left", "top-right" or
// "off". Empty defaults to "top-right".
func (s *Store) SyncButtonPos(boardID string) string {
	if b := s.boards[boardID]; b != nil && b.SyncButton != "" {
		return b.SyncButton
	}
	return "top-right"
}

// CommentsEnabled reports whether the board pulls issue comments on sync.
func (s *Store) CommentsEnabled(boardID string) bool {
	b := s.boards[boardID]
	return b != nil && b.Comments
}

// IssueURL resolves the board's link template for a card's issue key, or "" when there's
// no template or no key. `{key}` in the template is replaced by the key.
func (s *Store) IssueURL(boardID, key string) string {
	b := s.boards[boardID]
	if b == nil || b.IssueURL == "" || key == "" {
		return ""
	}
	return strings.ReplaceAll(b.IssueURL, "{key}", key)
}

// AgentTools is this board's headless agent --allowedTools ("" = no restriction).
func (s *Store) AgentTools(boardID string) string {
	if b := s.boards[boardID]; b != nil {
		return b.AgentTools
	}
	return ""
}

// SyncOnOpen reports whether the board should sync the bound columns on open (F16, Slice B).
func (s *Store) SyncOnOpen(boardID string) bool {
	b := s.boards[boardID]
	return b != nil && b.SyncOnOpen
}

// DeleteBoard removes the boards/<id>.md file (and the board from memory). It does NOT touch
// the tasks — the front-end archives/moves them first, otherwise the board is reborn discovered
// from the projects in use. Inbox is built-in and never goes away.
func (s *Store) DeleteBoard(id string) error {
	if id == InboxID {
		return nil
	}
	if err := os.Remove(filepath.Join(boardsDir(s.dir), id+".md")); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(s.boards, id)
	return nil
}

// normalizeKey derives a key prefix from a text: first word,
// only [A-Z0-9], uppercase, up to 8 chars. "Casa de férias" → "CASA"; "hakuban" →
// "CHARM". Empty if nothing usable is left.
func normalizeKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(s)) {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			if b.Len() > 0 { // cut at the 1st word
				return trunc8(b.String())
			}
		}
	}
	return trunc8(b.String())
}

func trunc8(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// keyTaken reports whether some board (≠ exceptID) already uses this key (case-insensitive).
func (s *Store) keyTaken(key, exceptID string) bool {
	for id, b := range s.boards {
		if id != exceptID && b.Key != "" && strings.EqualFold(b.Key, key) {
			return true
		}
	}
	return false
}

// UniqueKey returns a free key derived from base (derives + resolves collisions with
// a numeric suffix). exceptID is the board being saved (does not count as a collision).
func (s *Store) UniqueKey(base, exceptID string) string {
	k := normalizeKey(base)
	if k == "" {
		k = "TASK"
	}
	out := k
	for i := 2; s.keyTaken(out, exceptID); i++ {
		out = fmt.Sprintf("%s%d", k, i)
	}
	return out
}

// RenameBoardKey changes a board's key: it renames the cards' files
// (KEY-NN → NEWKEY-NN) and rewrites the references (`parent`) in all tasks.
// No-op if the new key collides with another board. (When `blocked_by` exists,
// rewrite it here too.)
func (s *Store) RenameBoardKey(boardID, newKey string) error {
	b := s.boards[boardID]
	if b == nil {
		return nil
	}
	newKey = normalizeKey(newKey)
	if newKey == "" || newKey == b.Key || s.keyTaken(newKey, boardID) {
		return nil
	}
	old := b.Key
	remap := map[string]string{} // old id → new
	if old != "" {
		for id := range s.tasks {
			if strings.HasPrefix(id, old+"-") {
				remap[id] = newKey + id[len(old):]
			}
		}
	}
	// tasks (from any board) that point to a renamed card need their
	// parent rewritten; if they themselves are not renamed, re-save them in place
	var repoint []*Task
	for _, t := range s.tasks {
		if nid, ok := remap[t.Parent]; ok {
			t.Parent = nid
			if _, renamed := remap[t.ID]; !renamed {
				repoint = append(repoint, t)
			}
		}
	}
	// renames the board's cards: writes under the new id + removes the old file
	for oldID, newID := range remap {
		t := s.tasks[oldID]
		delete(s.tasks, oldID)
		os.Remove(filepath.Join(tasksDir(s.dir), oldID+".md"))
		t.ID = newID
		if err := s.Save(t); err != nil {
			return err
		}
	}
	for _, t := range repoint { // persists the rewritten parent of the non-renamed ones
		if err := s.Save(t); err != nil {
			return err
		}
	}
	b.Key = newKey
	return s.SaveBoard(b)
}

// BoardTasks returns the active tasks whose effective project == boardID, sorted.
func (s *Store) BoardTasks(boardID string) []*Task {
	var out []*Task
	for _, t := range s.tasks {
		if s.EffectiveProject(t.ID) == boardID {
			out = append(out, t)
		}
	}
	sortTasks(out)
	return out
}

// SaveBoard persists the board (boards/<id>.md). An empty ID (Inbox) is built-in and does not
// become a file.
func (s *Store) SaveBoard(b *Board) error {
	if b.ID == InboxID {
		return nil
	}
	if b.Created.IsZero() {
		b.Created = now()
	}
	if err := os.MkdirAll(boardsDir(s.dir), 0o755); err != nil {
		return err
	}
	fm, err := yaml.Marshal(b)
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(delim + "\n")
	sb.Write(fm)
	sb.WriteString(delim + "\n")
	if err := atomicWrite(filepath.Join(boardsDir(s.dir), b.ID+".md"), []byte(sb.String())); err != nil {
		return err
	}
	s.boards[b.ID] = b
	return nil
}
