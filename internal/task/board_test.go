package task

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A FIELD filter is evaluated locally by the core against a Task field (offline). Keep hides
// cards older than WithinDays; a tracker filter or an unknown field never hides anything.
func TestFieldFilterKeep(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	recent := &Task{Modified: now.Add(-24 * time.Hour)} // 1 day old
	old := &Task{Modified: now.Add(-72 * time.Hour)}    // 3 days old

	f := Filter{Field: "modified", WithinDays: 2}
	if !f.Keep(recent, now) {
		t.Error("card de 1 dia devia ficar")
	}
	if f.Keep(old, now) {
		t.Error("card de 3 dias devia ser escondido")
	}
	if !f.IsField() {
		t.Error("filtro com Field devia ser field filter")
	}
	// a tracker filter (no Field) never hides; an unknown field is a no-op (keeps).
	if !(Filter{Value: "x"}).Keep(old, now) {
		t.Error("filtro de tracker não devia esconder nada")
	}
	if !(Filter{Field: "bogus", WithinDays: 1}).Keep(old, now) {
		t.Error("campo desconhecido devia ser no-op")
	}
}

// Boards = boards with a file + projects in use (without the built-in board). BoardTasks
// filters by the effective project. An empty board (file only) survives a reopen.
func TestBoards(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	must(t, s.Save(&Task{Title: "relatório", Project: "trabalho"})) // derived board
	must(t, s.SaveBoard(&Board{ID: "ideias", Name: "Ideias"}))      // empty board

	names := map[string]bool{}
	for _, b := range s.Boards() {
		names[b.ID] = true
	}
	for _, want := range []string{"trabalho", "ideias"} {
		if !names[want] {
			t.Errorf("Boards() não trouxe %q; veio %v", want, names)
		}
	}
	if names[InboxID] {
		t.Errorf("Boards() não devia trazer board embutido (id vazio); veio %v", names)
	}
	if got := s.BoardTasks("trabalho"); len(got) != 1 || got[0].Title != "relatório" {
		t.Errorf("BoardTasks(trabalho): %v", got)
	}
	if got := s.BoardTasks("ideias"); len(got) != 0 {
		t.Errorf("board vazio deveria ter 0 tasks, veio %d", len(got))
	}

	s2, err := Open(dir) // reopen from disk
	if err != nil {
		t.Fatal(err)
	}
	if s2.BoardName("ideias") != "Ideias" {
		t.Errorf("board 'ideias' não persistiu: nome %q", s2.BoardName("ideias"))
	}
}

// ColumnsFor: with no config it falls back to the default; a new board is born empty; a custom order
// survives a reopen; DeleteBoard makes the board disappear.
func TestBoardColumns(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Inbox and legacy board (without `columns`) → default.
	if got := s.ColumnsFor(InboxID); len(got) != 3 || got[0] != StatusBacklog {
		t.Errorf("Inbox devia usar DefaultColumns, veio %v", got)
	}
	// A new board is born empty (Columns non-nil, len 0).
	empty := []string{}
	must(t, s.SaveBoard(&Board{ID: "novo", Name: "Novo", Columns: &empty}))
	if got := s.ColumnsFor("novo"); len(got) != 0 {
		t.Errorf("board novo devia ter 0 colunas, veio %v", got)
	}
	// Custom order persists on disk.
	custom := []string{"ideias", "fazendo", "revisão", "pronto"}
	must(t, s.SaveBoard(&Board{ID: "novo", Name: "Novo", Columns: &custom}))
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := s2.ColumnsFor("novo")
	if len(got) != 4 || got[0] != "ideias" || got[3] != "pronto" {
		t.Errorf("colunas custom não persistiram na ordem: %v", got)
	}
	// DeleteBoard removes the board with no tasks.
	must(t, s2.DeleteBoard("novo"))
	for _, b := range s2.Boards() {
		if b.ID == "novo" {
			t.Errorf("board 'novo' devia ter sido excluído, ainda aparece")
		}
	}
}

// Integration (F16): a column binding (JQL + on_enter/on_exit) and the card's
// jira/source fields survive the round-trip on disk; a column without JQL is not
// "bound"; a legacy board without `actions` stays without a binding.
func TestColumnBinding(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	cols := []string{"DRAFTS", "TO-DO"}
	must(t, s.SaveBoard(&Board{
		ID: "pessoal", Name: "Pessoal", Columns: &cols,
		Actions: map[string]ColumnAction{
			"TO-DO": {JQL: "project=ABC AND status='To Do'", OnEnter: "~/hooks/create.sh", Guide: "move it forward to refine"},
		},
	}))
	// read-only mirror of an issue
	must(t, s.Save(&Task{Title: "bug X", Project: "pessoal", Status: "TO-DO", Jira: "ABC-12", Source: SourceJira}))

	s2, err := Open(dir) // reopen from disk
	if err != nil {
		t.Fatal(err)
	}
	a, ok := s2.BindingFor("pessoal", "TO-DO")
	if !ok || a.JQL == "" || a.OnEnter != "~/hooks/create.sh" {
		t.Fatalf("binding TO-DO não persistiu: ok=%v %+v", ok, a)
	}
	// the guide is a plain struct field: it must survive SaveBoard (a loose YAML key would
	// be dropped on re-marshal — that is the whole point of parsing it into the struct).
	if a.Guide != "move it forward to refine" {
		t.Errorf("guide não persistiu: %q", a.Guide)
	}
	if !s2.ColumnBound("pessoal", "TO-DO") {
		t.Error("TO-DO com JQL devia ser bound")
	}
	if s2.ColumnBound("pessoal", "DRAFTS") {
		t.Error("DRAFTS sem action não devia ser bound")
	}
	// card fields survive + Mirror()
	var mirror *Task
	for _, tk := range s2.BoardTasks("pessoal") {
		if tk.Jira == "ABC-12" {
			mirror = tk
		}
	}
	if mirror == nil || mirror.Source != SourceJira || !mirror.Mirror() {
		t.Fatalf("espelho não persistiu jira/source: %+v", mirror)
	}
}

// Mirror hand-written in jira/ (the sync path): even without the `source` field in the
// frontmatter, a file in that dir is always loaded as a read-only mirror.
func TestMirrorScan(t *testing.T) {
	dir := t.TempDir()
	must(t, os.MkdirAll(jiraDir(dir), 0o755))
	// no `source:` on purpose — the dir wins
	must(t, os.WriteFile(filepath.Join(jiraDir(dir), "ABC-7.md"),
		[]byte("---\nid: ABC-7\ntitle: issue\nstatus: TO-DO\nproject: p\njira: ABC-7\n---\n"), 0o644))
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	tk := s.Get("ABC-7")
	if tk == nil || !tk.Mirror() {
		t.Fatalf("arquivo em jira/ devia carregar como espelho: %+v", tk)
	}
	// Deleting a mirror has to target jira/, not tasks/
	must(t, s.Delete("ABC-7"))
	if _, err := os.Stat(filepath.Join(jiraDir(dir), "ABC-7.md")); !os.IsNotExist(err) {
		t.Error("Delete não removeu o espelho de jira/")
	}
}

// ReconcileMirrors (Slice B): the 1st sync writes the mirrors; the 2nd sync pulls the status
// back from Jira's truth, drops what left the set (stale) and doesn't touch another board.
func TestReconcileMirrors(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	must(t, err)
	// mirror of ANOTHER board, to ensure the reconcile doesn't remove it
	other := &Task{ID: "X-9", Title: "outro", Status: "TODO", Project: "outro", Jira: "X-9", Source: SourceJira}
	must(t, s.Save(other))

	if _, err := s.ReconcileMirrors("proj", []MirrorSpec{
		{Key: "P-1", Title: "a", Status: "TODO"}, {Key: "P-2", Title: "b", Status: "DOING"},
	}); err != nil {
		t.Fatal(err)
	}
	if p := s.Get("P-1"); p == nil || !p.Mirror() || p.Status != "TODO" || p.Priority != "normal" {
		t.Fatalf("P-1 não virou espelho com default de prioridade: %+v", p)
	}

	// 2nd sync: P-1 left (completed elsewhere), P-2 changed column, P-3 is new.
	if _, err := s.ReconcileMirrors("proj", []MirrorSpec{
		{Key: "P-2", Title: "b", Status: "DONE"}, {Key: "P-3", Title: "c", Status: "TODO"},
	}); err != nil {
		t.Fatal(err)
	}
	if s.Get("P-1") != nil {
		t.Error("P-1 devia ter sido removida (stale)")
	}
	if p := s.Get("P-2"); p == nil || p.Status != "DONE" {
		t.Errorf("P-2 devia repuxar pra DONE: %+v", p)
	}
	if s.Get("P-3") == nil {
		t.Error("P-3 (nova) devia existir")
	}
	if s.Get("X-9") == nil {
		t.Error("espelho de outro board não devia ser removido")
	}

	// an empty sync (agent gave up / [] ) must NOT delete the existing mirrors —
	// this was the "card disappeared" bug.
	before := len(s.tasks)
	if _, err := s.ReconcileMirrors("proj", nil); err != nil {
		t.Fatal(err)
	}
	if len(s.tasks) != before || s.Get("P-2") == nil || s.Get("P-3") == nil {
		t.Errorf("sync vazio não devia apagar espelhos (antes=%d, depois=%d)", before, len(s.tasks))
	}
}

// IssueURL substitutes {key} with the card's key; returns "" without a template or key.
func TestIssueURL(t *testing.T) {
	dir := t.TempDir()
	board := `---
name: PGM
key: PGM
issue_url: 'https://acme.atlassian.net/browse/{key}'
---
`
	if err := os.MkdirAll(filepath.Join(dir, "boards"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "boards", "acme.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.IssueURL("acme", "ACME-17"); got != "https://acme.atlassian.net/browse/ACME-17" {
		t.Fatalf("URL resolvida errada: %q", got)
	}
	if got := s.IssueURL("acme", ""); got != "" {
		t.Fatalf("sem key deveria ser vazio: %q", got)
	}
	if got := s.IssueURL("inexistente", "ACME-17"); got != "" {
		t.Fatalf("board sem template deveria ser vazio: %q", got)
	}
}
