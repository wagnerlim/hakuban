package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/wagnerlim/hakuban/internal/task"
)

// boardWithAction assembles a personal board with DRAFTS (local) + TO-DO (on_enter_cmd =
// script) and a local card in DRAFTS. Uses the DETERMINISTIC path (F22): the intention
// (on_enter) calls the real `claude` agent, which doesn't run in a test. Returns the ready
// model and the card id.
func boardWithAction(t *testing.T, onEnter string) (*Model, string) {
	t.Helper()
	dir := t.TempDir()
	board := "---\nname: Pessoal\nkey: PES\ncolumns:\n    - DRAFTS\n    - TO-DO\nactions:\n    TO-DO:\n        jql: \"x\"\n        status: Done\n        on_enter_cmd: " + onEnter + "\n---\n"
	if err := os.MkdirAll(filepath.Join(dir, "boards"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "boards", "pessoal.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&task.Task{Title: "rascunho", Project: "pessoal", Status: "DRAFTS"}); err != nil {
		t.Fatal(err)
	}
	m := New(s)
	for i, id := range m.open {
		if id == "pessoal" {
			m.active = i
		}
	}
	m.w, m.h = 100, 24
	m.reload()
	m.col = 0 // DRAFTS
	id := m.cols[0][0].ID
	return m, id
}

// pump runs the action's event loop until it finishes (done or failed). Reads the
// channel directly (ignoring the spring animation cmds, which the runtime would play) and
// feeds each actionEvent into Update — which exercises the commit/failure logic.
func pump(t *testing.T, m *Model, _ tea.Cmd) {
	t.Helper()
	ch := m.actionCh
	for i := 0; i < 50 && ch != nil && m.action != nil && !m.action.done && !m.action.failed; i++ {
		m.Update(<-ch)
	}
}

// Success: on_enter emits progress and exits 0 → the move commits (card goes to TO-DO)
// and the loader stays at ✓.
func TestActionSuccess(t *testing.T) {
	m, id := boardWithAction(t, `"printf '1/2 validando\n2/2 criando\n'"`)
	cmd := m.move(1) // DRAFTS → TO-DO fires on_enter
	if cmd == nil || m.action == nil {
		t.Fatal("a move into a bound column should start the action")
	}
	if got := m.store.Get(id).Status; got != "DRAFTS" {
		t.Fatalf("the status should not change before success, got %q", got)
	}
	pump(t, m, cmd)
	if !m.action.done || m.action.failed {
		t.Fatalf("the action should end in success: %+v", m.action)
	}
	if got := m.store.Get(id).Status; got != "TO-DO" {
		t.Fatalf("success should commit the move to TO-DO, got %q", got)
	}
}

// Progress: a progress line stamps the % onto the card while the action runs, and a
// successful commit clears it (no stale bar lingering on the mini-card).
func TestActionProgressStamped(t *testing.T) {
	m, id := boardWithAction(t, `"echo '40 indo'"`)
	_ = m.move(1)          // DRAFTS → TO-DO fires on_enter
	m.Update(<-m.actionCh) // first event: the 40% progress line
	if got := m.store.Get(id).Progress; got != 40 {
		t.Fatalf("progress devia ser carimbado no card (40), veio %d", got)
	}
	for m.action != nil && !m.action.done && !m.action.failed { // drain to completion
		m.Update(<-m.actionCh)
	}
	if got := m.store.Get(id).Status; got != "TO-DO" {
		t.Fatalf("success should commit to TO-DO, got %q", got)
	}
	if got := m.store.Get(id).Progress; got != 0 {
		t.Fatalf("commit devia zerar o progress, veio %d", got)
	}
}

// chainModel builds a board FROM (on_exit_cmd) → TO (on_enter_cmd), with each side
// appending its name to an order file, and a card sitting in FROM. exitOK=false makes the
// exit side fail (exit 1). Returns the model, the card id, and the order file path.
func chainModel(t *testing.T, exitOK bool) (*Model, string, string) {
	t.Helper()
	dir := t.TempDir()
	order := filepath.Join(dir, "order.txt")
	exitCmd := `"echo exit >> ` + order + `; echo '1/1 saindo'"`
	if !exitOK {
		exitCmd = `"echo exit >> ` + order + `; exit 1"`
	}
	enterCmd := `"echo enter >> ` + order + `; echo '1/1 entrando'"`
	board := "---\nname: Chain\nkey: CH\ncolumns:\n    - FROM\n    - TO\nactions:\n" +
		"    FROM:\n        on_exit_cmd: " + exitCmd + "\n" +
		"    TO:\n        on_enter_cmd: " + enterCmd + "\n---\n"
	if err := os.MkdirAll(filepath.Join(dir, "boards"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "boards", "chain.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&task.Task{Title: "c", Project: "chain", Status: "FROM"}); err != nil {
		t.Fatal(err)
	}
	m := New(s)
	for i, id := range m.open {
		if id == "chain" {
			m.active = i
		}
	}
	m.w, m.h = 100, 24
	m.reload()
	m.col = 0 // FROM
	return m, m.cols[0][0].ID, order
}

// pumpChain drains an action to completion, re-reading m.actionCh each iteration so it
// follows the second leg of the exit→enter chain (which swaps the channel mid-flight).
func pumpChain(t *testing.T, m *Model) {
	t.Helper()
	for i := 0; i < 100 && m.actionCh != nil && m.action != nil && !m.action.done && !m.action.failed; i++ {
		m.Update(<-m.actionCh)
	}
}

// Chaining: moving FROM→TO runs the origin's on_exit FIRST, then the destination's
// on_enter, and only then commits the move.
func TestActionExitEnterChain(t *testing.T) {
	m, id, order := chainModel(t, true)
	m.move(1) // FROM → TO
	pumpChain(t, m)
	if !m.action.done || m.action.failed {
		t.Fatalf("chain devia terminar em sucesso: %+v", m.action)
	}
	if got := m.store.Get(id).Status; got != "TO" {
		t.Fatalf("commit devia rolar só após o enter, status veio %q", got)
	}
	data, _ := os.ReadFile(order)
	if got := strings.Fields(string(data)); len(got) != 2 || got[0] != "exit" || got[1] != "enter" {
		t.Fatalf("exit devia rodar antes do enter, ordem veio %v", got)
	}
}

// Chaining, exit fails: the destination's on_enter never runs and the card stays at the origin.
func TestActionExitFailsSkipsEnter(t *testing.T) {
	m, id, order := chainModel(t, false)
	m.move(1) // FROM → TO
	pumpChain(t, m)
	if !m.action.failed {
		t.Fatalf("a failed exit should leave the action in failed: %+v", m.action)
	}
	if got := m.store.Get(id).Status; got != "FROM" {
		t.Fatalf("card devia ficar na origem quando o exit falha, veio %q", got)
	}
	data, _ := os.ReadFile(order)
	if got := strings.Fields(string(data)); len(got) != 1 || got[0] != "exit" {
		t.Fatalf("enter should not run after a failed exit, the order was %v", got)
	}
}

// Stamping: on_enter emits `jira: KEY` and exits 0 → the local card turns into a mirror with
// id = key (the old one disappears), so the sync matches by id and doesn't duplicate.
func TestActionStamping(t *testing.T) {
	m, id := boardWithAction(t, `"printf '1/2 criando\njira: ABC-500\n2/2 vinculando\n'"`)
	cmd := m.move(1) // DRAFTS → TO-DO
	pump(t, m, cmd)
	if !m.action.done || m.action.failed {
		t.Fatalf("the action should end in success: %+v", m.action)
	}
	if m.store.Get(id) != nil {
		t.Errorf("card local %q devia ter sumido após virar espelho", id)
	}
	mir := m.store.Get("ABC-500")
	if mir == nil || !mir.Mirror() || mir.Jira != "ABC-500" || mir.Status != "TO-DO" {
		t.Fatalf("the card did not become the ABC-500 mirror: %+v", mir)
	}
}

// parseAgentLine: tool_use becomes a loader step (ladder up to 90%), result is the end,
// and a non-JSON line is ignored (a stray log from the agent doesn't break the parser).
func TestParseAgentLine(t *testing.T) {
	step := 0
	toolUse := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__x__transitionJiraIssue"}]}}`
	ev, isResult, _ := parseAgentLine([]byte(toolUse), &step)
	if isResult || ev == nil || ev.label != "transitionJiraIssue" || ev.pct != 25 {
		t.Fatalf("tool_use devia virar passo (25%%, label curto), veio %+v", ev)
	}
	if _, _, _ = parseAgentLine([]byte("not json"), &step); step != 1 {
		t.Fatalf("a non-JSON line should not advance the step, step=%d", step)
	}
	_, isResult, res := parseAgentLine([]byte(`{"type":"result","subtype":"success","result":"feito"}`), &step)
	if !isResult || res != "feito" {
		t.Fatalf("result should signal the end along with the text, isResult=%v res=%q", isResult, res)
	}
}

// Real integration (F16, model 1): exercises runAgent against the real Jira.
// Skipped by default — needs `claude` logged into claude.ai (without ANTHROPIC_API_KEY)
// and the Atlassian connector. Run with: HAKUBAN_JIRA_IT=1 go test -run AgentReal -v ./internal/tui
func TestAgentRealTransition(t *testing.T) {
	if os.Getenv("HAKUBAN_JIRA_IT") == "" {
		t.Skip("real integration: set HAKUBAN_JIRA_IT=1")
	}
	ch := make(chan actionEvent, 64)
	card := cardJSON(&task.Task{ID: "DEMO-1", Jira: "DEMO-1", Title: "placeholder"}, "DOING", "DONE", "demo")
	go runAgent(`Move issue DEMO-1 to "Done" in Jira, site acme.atlassian.net. Use the Atlassian (Rovo) connector over MCP.`,
		card, "mcp__claude_ai_Atlassian_Rovo", ch)
	var last actionEvent
	for ev := range ch {
		t.Logf("ev: pct=%d label=%q done=%v ok=%v reason=%q", ev.pct, ev.label, ev.done, ev.ok, ev.reason)
		if ev.done {
			last = ev
			break
		}
	}
	if !last.ok {
		t.Fatalf("the real transition failed: %q", last.reason)
	}
}

// parseSyncIssues extracts the array even with prose and ```json fences around it; text
// without an array becomes nil (reconcile then empties the mirrors, the expected behavior).
func TestParseSyncIssues(t *testing.T) {
	s := "Here they are:\n```json\n[{\"key\":\"A-1\",\"summary\":\"x\",\"status\":\"Done\",\"priority\":\"High\"}]\n```\n"
	got := parseSyncIssues(s)
	if len(got) != 1 || got[0].Key != "A-1" || got[0].Status != "Done" || got[0].Priority != "High" {
		t.Fatalf("extraction failed: %+v", got)
	}
	if parseSyncIssues("nenhum array aqui") != nil {
		t.Fatal("text with no array should give nil")
	}
}

// Real integration (F16, Slice B): the agent runs a JQL and returns parseable JSON.
// Skipped by default (see TestAgentRealTransition). Proves the PULL side end to end.
func TestAgentRealSync(t *testing.T) {
	if os.Getenv("HAKUBAN_JIRA_IT") == "" {
		t.Skip("real integration: set HAKUBAN_JIRA_IT=1")
	}
	ch := make(chan actionEvent, 64)
	intention := `List the Jira issues (Atlassian/Rovo connector over MCP) matching: project = DEMO.
Reply with ONLY a JSON array, one object per issue: {key, summary, status, priority}.`
	go runAgent(intention, nil, "mcp__claude_ai_Atlassian_Rovo", ch)
	var last actionEvent
	for ev := range ch {
		if ev.done {
			last = ev
			break
		}
	}
	if !last.ok {
		t.Fatalf("sync real falhou: %q", last.reason)
	}
	issues := parseSyncIssues(last.payload)
	t.Logf("payload=%q → %d issues", last.payload, len(issues))
	if len(issues) == 0 || issues[0].Key == "" {
		t.Fatalf("the agent did not return parseable issues: %+v", issues)
	}
}

// resolveIntention: path to an existing .md → reads the file; inline prose → passes through.
func TestResolveIntention(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks", "fin.md"), []byte("conteúdo do arquivo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveIntention("hooks/fin.md", dir); got != "conteúdo do arquivo" {
		t.Fatalf("caminho relativo devia ler o arquivo, veio %q", got)
	}
	inline := "Move issue {{.jira}} to Done"
	if got := resolveIntention(inline, dir); got != inline {
		t.Fatalf("prosa inline devia passar direto, veio %q", got)
	}
}

// interpolate resolves the card's fields in the intention; a missing field becomes empty, not an error.
func TestInterpolate(t *testing.T) {
	got := interpolate("Move {{.jira}} to Done ({{.title}})", &task.Task{Jira: "ABC-1", Title: "x"})
	if got != "Move ABC-1 to Done (x)" {
		t.Fatalf("wrong interpolation: %q", got)
	}
}

// HAKUBAN_STATUS: the core passes the target status (the `status:` field of the destination column)
// to the command. The on_enter here only exits 0 if it receives "Done".
func TestActionStatusEnv(t *testing.T) {
	m, _ := boardWithAction(t, `"test \"$HAKUBAN_STATUS\" = Done && printf '1/1 ok\n' || { echo sem-status >&2; exit 1; }"`)
	pump(t, m, m.move(1))
	if m.action == nil || !m.action.done || m.action.failed {
		t.Fatalf("HAKUBAN_STATUS did not arrive as \"Done\": %+v", m.action)
	}
}

// Failure: on_enter exits ≠0 → card stays at the origin and the reason (last line of stderr)
// appears in the loader.
func TestActionFailure(t *testing.T) {
	m, id := boardWithAction(t, `"echo 1/1 indo; echo 'jira fora do ar' >&2; exit 1"`)
	cmd := m.move(1)
	if cmd == nil {
		t.Fatal("the move should start the action")
	}
	pump(t, m, cmd)
	if !m.action.failed {
		t.Fatalf("the action should fail: %+v", m.action)
	}
	if got := m.store.Get(id).Status; got != "DRAFTS" {
		t.Fatalf("a failure should not move the card, got %q", got)
	}
	if !strings.Contains(m.action.reason, "jira fora do ar") {
		t.Fatalf("motivo devia vir do stderr, veio %q", m.action.reason)
	}
}

// syncColumns queues every bound column when the board has a sync SCRIPT (one call each);
// canSync gates the board-wide button on there being a binding + a sync path.
func TestSyncColumnsScript(t *testing.T) {
	dir := t.TempDir()
	board := `---
name: PGM
key: PGM
columns:
    - A
    - B
    - C
sync: hooks/sync.sh
actions:
    A:
        jql: 'project = X'
    B:
        jql: 'project = Y'
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
	m.reload()

	if !m.canBatch() {
		t.Fatal("a board with a binding + sync should be able to sync")
	}
	got := m.batchQueue() // A=0, B=1 bound; C=2 unbound
	if len(got) != 2 || got[0].col != 0 || got[1].col != 1 {
		t.Fatalf("script sync devia enfileirar todas as colunas bound: %v", got)
	}
}

// A column that declares `buttons:` drives the footer on its own: several buttons open the
// menu instead of firing, only the ones marked `batch` join the board-wide button, and a
// script button gets the column's cards on stdin.
func TestColumnButtons(t *testing.T) {
	dir := t.TempDir()
	board := `---
name: PGM
key: PGM
columns:
    - A
    - B
actions:
    A:
        buttons:
            - label: sincronizar
              cmd: hooks/sync.sh
              batch: true
            - label: limpar
              cmd: hooks/clear.sh
    B:
        buttons:
            - label: refinar
              agent: hooks/refine.md
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
	m.reload()

	if got := len(m.columnButtons("A")); got != 2 {
		t.Fatalf("column A should have 2 buttons, got %d", got)
	}
	if got := m.footerLabel("A", m.columnButtons("A")); got != "sincronizar ▾" {
		t.Fatalf("with several buttons the footer opens a menu: %q", got)
	}
	if got := m.footerLabel("B", m.columnButtons("B")); got != "refinar" {
		t.Fatalf("a single button goes straight through: %q", got)
	}
	// A (2 buttons) opens the menu instead of firing; B (single) fires right away.
	if cmd := m.activateColumn(0); cmd != nil || !m.menuOpen || m.menuCol != 0 {
		t.Fatal("a column with several buttons should open the menu, not fire")
	}
	m.menuOpen = false
	// only `batch: true` enters the board-wide queue
	q := m.batchQueue()
	if len(q) != 1 || q[0].col != 0 || q[0].btn != 0 {
		t.Fatalf("only the batch button enters the board queue: %v", q)
	}
	// the script's stdin carries the column's cards
	cards := columnJSON([]*task.Task{{ID: "x1", Title: "um"}}, "A", "acme")
	if !strings.Contains(string(cards), `"id":"x1"`) || !strings.HasPrefix(string(cards), "[") {
		t.Fatalf("the button's stdin should be the column's card array: %s", cards)
	}
}

// Each column picks its own loader glyphs and whether the number shows; an unknown name
// falls back to the default bar instead of rendering nothing.
func TestColumnLoader(t *testing.T) {
	dir := t.TempDir()
	board := `---
name: PGM
key: PGM
columns:
    - A
    - B
    - C
    - D
actions:
    A:
        loader: blocks
    B:
        loader: dots
        percent: false
    C:
        loader: nao-existe
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
	m.reload()

	for _, c := range []struct {
		col, glyph string
		pct        bool
	}{
		{"A", "█", true},  // named style
		{"B", "●", false}, // named style + percent off
		{"C", "▰", true},  // unknown name → default
		{"D", "▰", true},  // no action at all → default
	} {
		style, showPct := m.loaderStyle(c.col)
		got := progressBar(50, style, showPct)
		if !strings.Contains(got, c.glyph) {
			t.Fatalf("coluna %s devia usar %q: %q", c.col, c.glyph, got)
		}
		if strings.Contains(got, "50%") != c.pct {
			t.Fatalf("coluna %s: percent=%v, veio %q", c.col, c.pct, got)
		}
	}
}

// The top bar is the same structure as a column's footer: the board declares its buttons,
// several open a menu that hangs DOWN, and a board with none still gets the synthesized
// sync-all as long as some column has a batch button.
func TestBoardButtons(t *testing.T) {
	write := func(body string) *Model {
		t.Helper()
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "boards"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "boards", "acme.md"), []byte(body), 0o644); err != nil {
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
		m.reload()
		return m
	}

	// declared: three buttons → the top bar opens a menu growing down
	m := write(`---
name: PGM
key: PGM
columns:
    - A
button_label: board
buttons:
    - label: Sincronizar tudo
      icon: ⟳
      batch: true
    - label: relatório
      cmd: hooks/report.sh
---
`)
	bs := m.boardButtons()
	if len(bs) != 2 {
		t.Fatalf("the board should have 2 buttons, got %d", len(bs))
	}
	if got := m.boardButtonLabel(bs); got != "board ▾" {
		t.Fatalf("with several buttons the top bar opens a menu: %q", got)
	}
	if cmd := m.activateBoard(); cmd != nil || !m.menuOpen || m.menuCol != boardMenuCol || m.menuUp {
		t.Fatalf("it should open the board menu downwards: open=%v col=%d up=%v", m.menuOpen, m.menuCol, m.menuUp)
	}
	if got := m.menuRowOf(0); got != 0 { // the first button hugging the anchor = row 1
		t.Fatalf("a downward menu draws in config order, row of the first: %d", got)
	}

	// legacy: nothing declared, but a column has a batch button → sync-all synthesized
	m = write(`---
name: PGM
key: PGM
columns:
    - A
sync: hooks/sync.sh
actions:
    A:
        jql: 'project = X'
---
`)
	bs = m.boardButtons()
	if len(bs) != 1 || !bs[0].Batch {
		t.Fatalf("board legado devia sintetizar o sync-all: %+v", bs)
	}
	if got := m.boardButtonLabel(bs); got != msg.syncAllLabel+" ⟳" {
		t.Fatalf("rótulo do sync-all sintetizado: %q", got)
	}
}
