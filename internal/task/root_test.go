package task

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDirEnv(t *testing.T) {
	t.Setenv("HAKUBAN_TASK_DIR", "/tmp/xyz-charm")
	if got := ResolveDir(); got != "/tmp/xyz-charm" {
		t.Fatalf("env devia mandar: %q", got)
	}
}

// MoveData moves everything to the empty target and aborts if the target already has data.
func TestMoveData(t *testing.T) {
	from, to := t.TempDir(), filepath.Join(t.TempDir(), "novo")

	s, err := Open(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(&Task{Title: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveConfig(DefaultConfig()); err != nil { // ensures config.yml
		t.Fatal(err)
	}
	if n := CountData(from); n < 2 {
		t.Fatalf("CountData baixo: %d", n)
	}

	if err := MoveData(from, to); err != nil {
		t.Fatalf("move falhou: %v", err)
	}
	if HasData(from) {
		t.Fatal("origem ainda tem dados após mover")
	}
	if !HasData(to) {
		t.Fatal("the target did not receive the data")
	}

	// target now has data → a new source can't move there
	other := t.TempDir()
	if err := MoveData(other, to); !errors.Is(err, ErrTargetHasData) {
		t.Fatalf("a collision should abort with ErrTargetHasData, got: %v", err)
	}
}

func TestHasData(t *testing.T) {
	dir := t.TempDir()
	if HasData(dir) {
		t.Fatal("an empty dir has no data")
	}
	if err := os.MkdirAll(filepath.Join(dir, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks", "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !HasData(dir) {
		t.Fatal("a dir with tasks/*.md has data")
	}
}
