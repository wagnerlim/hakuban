package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wagnerlim/hakuban/internal/task"
)

// headlessStore writes a board file and returns the ready store + its data dir. The board
// id is "b" (file b.md), so cards use Project "b".
func headlessStore(t *testing.T, boardYAML string) (*task.Store, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "boards"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "boards", "b.md"), []byte(boardYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

func saveCard(t *testing.T, s *task.Store, status string) string {
	t.Helper()
	if err := s.Save(&task.Task{Title: "x", Project: "b", Status: status}); err != nil {
		t.Fatal(err)
	}
	return s.All()[0].ID
}

// Success: the destination's on_enter runs and the card commits to the target column.
func TestMoveHeadlessSuccess(t *testing.T) {
	board := "---\nname: B\nkey: B\ncolumns:\n    - FROM\n    - TO\nactions:\n    TO:\n        on_enter_cmd: \"echo '1/1 indo'\"\n---\n"
	s, _ := headlessStore(t, board)
	id := saveCard(t, s, "FROM")
	if err := MoveHeadless(s, id, "TO", &bytes.Buffer{}); err != nil {
		t.Fatalf("move devia passar: %v", err)
	}
	if got := s.Get(id).Status; got != "TO" {
		t.Fatalf("card devia commitar em TO, veio %q", got)
	}
}

// Hook failure: on_enter exits 1 → error and the card stays at the origin.
func TestMoveHeadlessHookFails(t *testing.T) {
	board := "---\nname: B\nkey: B\ncolumns:\n    - FROM\n    - TO\nactions:\n    TO:\n        on_enter_cmd: \"echo boom >&2; exit 1\"\n---\n"
	s, _ := headlessStore(t, board)
	id := saveCard(t, s, "FROM")
	if err := MoveHeadless(s, id, "TO", &bytes.Buffer{}); err == nil {
		t.Fatal("hook que falha devia retornar erro")
	}
	if got := s.Get(id).Status; got != "FROM" {
		t.Fatalf("card devia ficar em FROM quando o hook falha, veio %q", got)
	}
}

// Chaining: on_exit of the origin runs BEFORE on_enter of the destination, then commits.
func TestMoveHeadlessExitEnterChain(t *testing.T) {
	dir := t.TempDir()
	order := filepath.Join(dir, "order.txt")
	board := "---\nname: B\nkey: B\ncolumns:\n    - FROM\n    - TO\nactions:\n" +
		"    FROM:\n        on_exit_cmd: \"echo exit >> " + order + "; echo '1/1 saindo'\"\n" +
		"    TO:\n        on_enter_cmd: \"echo enter >> " + order + "; echo '1/1 entrando'\"\n---\n"
	if err := os.MkdirAll(filepath.Join(dir, "boards"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "boards", "b.md"), []byte(board), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := task.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	id := saveCard(t, s, "FROM")
	if err := MoveHeadless(s, id, "TO", &bytes.Buffer{}); err != nil {
		t.Fatalf("chain devia passar: %v", err)
	}
	if got := s.Get(id).Status; got != "TO" {
		t.Fatalf("commit só após o enter; status veio %q", got)
	}
	data, _ := os.ReadFile(order)
	if got := strings.Fields(string(data)); len(got) != 2 || got[0] != "exit" || got[1] != "enter" {
		t.Fatalf("exit devia rodar antes do enter, ordem: %v", got)
	}
}

// Relative hook path: the hook is referenced RELATIVE to the data dir; cmd.Dir must
// resolve it even though the test's CWD is the package dir (the cmd.Dir fix).
func TestMoveHeadlessRelativeHookPath(t *testing.T) {
	s, dir := headlessStore(t, "---\nname: B\nkey: B\ncolumns:\n    - FROM\n    - TO\nactions:\n    TO:\n        on_enter_cmd: \"sh hooks/mark.sh\"\n---\n")
	if err := os.MkdirAll(filepath.Join(dir, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks", "mark.sh"), []byte("echo '1/1 marcando'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	id := saveCard(t, s, "FROM")
	if err := MoveHeadless(s, id, "TO", &bytes.Buffer{}); err != nil {
		t.Fatalf("hook por caminho relativo devia resolver via cmd.Dir: %v", err)
	}
	if got := s.Get(id).Status; got != "TO" {
		t.Fatalf("card devia commitar em TO, veio %q", got)
	}
}
