package task

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Round-trip: save, reopen from disk and check that the fields (including
// notes and a date-only due) survive. Fails if parse/serialize breaks.
func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	due := &Date{time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC)}
	orig := &Task{
		Title:    "comprar café",
		Priority: "high",
		Project:  "casa",
		Tags:     []string{"compras"},
		Due:      due,
		Notes:    "## Notes\nground, not whole bean.",
	}
	if err := s.Save(orig); err != nil {
		t.Fatal(err)
	}
	if orig.ID == "" {
		t.Fatal("Save did not generate an id")
	}
	// file is <id>.md
	if _, err := os.Stat(filepath.Join(dir, "tasks", orig.ID+".md")); err != nil {
		t.Fatalf("esperava tasks/%s.md: %v", orig.ID, err)
	}

	s2, err := Open(dir) // reopen from scratch, no in-memory state
	if err != nil {
		t.Fatal(err)
	}
	got := s2.Get(orig.ID)
	if got == nil {
		t.Fatal("task sumiu após reabrir")
	}
	if got.Title != "comprar café" || got.Priority != "high" || got.Project != "casa" {
		t.Errorf("campos errados: %+v", got)
	}
	if !reflect.DeepEqual(got.Tags, []string{"compras"}) {
		t.Errorf("tags: %v", got.Tags)
	}
	if got.Due == nil || !got.Due.Equal(due.Time) {
		t.Errorf("due: %v", got.Due)
	}
	if got.Notes != "## Notes\nground, not whole bean." {
		t.Errorf("notes: %q", got.Notes)
	}
	if got.Status != StatusBacklog {
		t.Errorf("status default deveria ser backlog, veio %q", got.Status)
	}
}

// Dynamic inheritance: project comes from the root, tags are the additive union of
// the ancestors. Fails if the graph walk-up breaks.
func TestInheritance(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := &Task{Title: "reforma", Project: "casa", Tags: []string{"grande"}}
	must(t, s.Save(root))
	child := &Task{Title: "pintar", Parent: root.ID, Tags: []string{"urgente"}}
	must(t, s.Save(child))
	grand := &Task{Title: "comprar tinta", Parent: child.ID}
	must(t, s.Save(grand))

	if got := s.EffectiveProject(grand.ID); got != "casa" {
		t.Errorf("project herdado da raiz: quero casa, veio %q", got)
	}
	if got := s.EffectiveTags(grand.ID); !reflect.DeepEqual(got, []string{"grande", "urgente"}) {
		t.Errorf("tags efetivas: quero [grande urgente], veio %v", got)
	}
	// child stores only its own on disk (inheritance is derived, not copied)
	if !reflect.DeepEqual(s.Get(grand.ID).Tags, []string(nil)) {
		t.Errorf("the grandchild should not store its own tags, it has %v", s.Get(grand.ID).Tags)
	}
	if kids := s.Children(root.ID); len(kids) != 1 || kids[0].ID != child.ID {
		t.Errorf("Children(root) errado: %v", kids)
	}

	// dynamic inheritance: changing the root reflects on the grandchild without rewriting the grandchild
	root.Project = "apê"
	must(t, s.Save(root))
	if got := s.EffectiveProject(grand.ID); got != "apê" {
		t.Errorf("inheritance is not dynamic: wanted flat, got %q", got)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// Comments (pulled by the sync) survive a save/reopen roundtrip: author, when and the
// markdown body (which may carry links).
func TestCommentsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	orig := &Task{
		Title: "bug do checkout",
		Comments: []Comment{
			{Author: "Alice Souza", When: "20/07", Body: "Subi o fix, veja [PR](http://x/pr/1)."},
			{Author: "Bruno Lima", When: "21/07", Body: "Revisado, pode mergear."},
		},
	}
	if err := s.Save(orig); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := s2.Get(orig.ID)
	if got == nil {
		t.Fatal("task sumiu")
	}
	if !reflect.DeepEqual(got.Comments, orig.Comments) {
		t.Fatalf("comments did not survive:\n got  %+v\n want %+v", got.Comments, orig.Comments)
	}
}
