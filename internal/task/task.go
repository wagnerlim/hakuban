// Package task is the storage layer of hakuban: it reads/writes tasks as
// .md files (YAML frontmatter + markdown body) and builds the relationship graph
// in memory. It is the app's source of truth — CLI, TUI and git sit on top of it.
package task

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Task is a task. Fields with a yaml tag go into the frontmatter; Notes is the
// markdown body (outside the frontmatter). ID is derived from the file name (<id>.md) —
// the field in the frontmatter exists only for external readability.
type Task struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Status   string   `yaml:"status"`             // pending | done
	Priority string   `yaml:"priority,omitempty"` // low | normal | high
	Progress int      `yaml:"progress,omitempty"` // 0..100 shown on the card; stamped live by the action loader
	Project  string   `yaml:"project,omitempty"`  // empty on a child = inherited from the root
	Tags     []string `yaml:"tags,omitempty"`     // own; effective = ∪ ancestors
	Due      *Date    `yaml:"due,omitempty"`
	Parent   string   `yaml:"parent,omitempty"` // parent id ("" = root)
	// Integration (F16): Jira is the key of the linked issue (PROJ-123); Source
	// marks the origin — "jira" = read-only mirror materialized by the sync, empty
	// = editable local card. Both omitempty: a plain local card writes nothing.
	Jira     string    `yaml:"jira,omitempty"`
	Source   string    `yaml:"source,omitempty"`
	Created  time.Time `yaml:"created"`
	Modified time.Time `yaml:"modified"`
	// Comments are the issue's comments pulled by the sync when the board opts in
	// (board `comments: true`). The core never fetches them — the sync script writes
	// this list; author/when/body are universal, keeping the core tracker-agnostic.
	Comments []Comment `yaml:"comments,omitempty"`
	Notes    string    `yaml:"-"` // markdown body, not written to the frontmatter
}

// Comment is one comment on a card (from the tracker, written by the sync script).
// Body is markdown (may carry links). When is a free-form display string the script
// provides — the core doesn't parse it.
type Comment struct {
	Author string `yaml:"author"`
	When   string `yaml:"when,omitempty"`
	Body   string `yaml:"body"`
}

// SourceJira marks a task as a read-only mirror of a Jira issue (F16).
const SourceJira = "jira"

// Mirror reports whether the task is a read-only mirror of an external tracker (content
// comes from outside; only its column position is actionable — see F16).
func (t *Task) Mirror() bool { return t.Source != "" }

// Status are the columns of the Kanban board, in order backlog → doing → done.
// Legacy `pending` (old frontmatter) is treated as backlog on load.
const (
	StatusBacklog = "backlog"
	StatusDoing   = "doing"
	StatusDone    = "done"
)

// DefaultColumns are the columns of a board with no config of its own (Inbox and
// legacy/discovered boards). A board created by the TUI is born with its own (empty) list.
var DefaultColumns = []string{StatusBacklog, StatusDoing, StatusDone}

// Date is a date without time. It exists to keep the frontmatter clean and
// hand-editable (`due: 2026-07-09`) instead of the RFC3339 that time.Time renders.
// ponytail: only the due date needs this; created/modified are normal timestamps.
type Date struct{ time.Time }

const dateLayout = "2006-01-02"

func (d Date) MarshalYAML() (any, error) { return d.Format(dateLayout), nil }

func (d *Date) UnmarshalYAML(n *yaml.Node) error {
	if strings.TrimSpace(n.Value) == "" {
		return nil
	}
	t, err := time.Parse(dateLayout, strings.TrimSpace(n.Value))
	if err != nil {
		return fmt.Errorf("due inválido %q: %w", n.Value, err)
	}
	d.Time = t
	return nil
}

const delim = "---"

// splitFrontmatter separates the YAML frontmatter from the body. If the content does not
// start with `---` (or the closing delimiter is missing), it treats everything as body — so
// a hand-written .md with no frontmatter is still a valid task (id comes from the name).
// ponytail: manual split instead of a dep; tolerates CRLF via TrimRight.
func splitFrontmatter(content string) (frontmatter, body string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != delim {
		return "", content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == delim {
			fm := strings.Join(lines[1:i], "\n")
			bd := strings.Join(lines[i+1:], "\n")
			return fm, strings.TrimPrefix(bd, "\n")
		}
	}
	return "", content // frontmatter with no closing delimiter
}

// unmarshal parses the content of a .md into a Task. The id is always the one from the
// file name (authoritative), not the one from the frontmatter.
func unmarshal(id, content string) (*Task, error) {
	fm, body := splitFrontmatter(content)
	var t Task
	if strings.TrimSpace(fm) != "" {
		if err := yaml.Unmarshal([]byte(fm), &t); err != nil {
			return nil, fmt.Errorf("frontmatter de %s: %w", id, err)
		}
	}
	t.ID = id
	t.Notes = strings.TrimRight(body, "\n")
	if t.Status == "" || t.Status == "pending" {
		t.Status = StatusBacklog
	}
	return &t, nil
}

// marshal serializes the Task into the on-disk format: frontmatter + body.
func (t *Task) marshal() ([]byte, error) {
	fm, err := yaml.Marshal(t)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString(delim + "\n")
	b.Write(fm)
	b.WriteString(delim + "\n")
	if t.Notes != "" {
		b.WriteString("\n" + strings.TrimRight(t.Notes, "\n") + "\n")
	}
	return []byte(b.String()), nil
}
