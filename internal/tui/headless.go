package tui

import (
	"fmt"
	"io"

	"github.com/wagnerlim/hakuban/internal/task"
)

// MoveHeadless moves a card between columns OUTSIDE the TUI, firing the same
// on_exit→on_enter hooks synchronously and committing only when they succeed. It is the
// headless entry (LLM-driven board): an external agent runs `hakuban move <id> <col>`
// to drive the board and its integrations without a human at the TUI. Progress lines go
// to out; a hook failure returns an error and leaves the card at the origin.
func MoveHeadless(s *task.Store, id, toCol string, out io.Writer) error {
	t := s.Get(id)
	if t == nil {
		return fmt.Errorf("card %q não existe", id)
	}
	board := s.EffectiveProject(id)
	from := t.Status
	if from == toCol {
		return nil // já está na coluna: no-op
	}
	found := false
	for _, c := range s.ColumnsFor(board) {
		if c == toCol {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("coluna %q não existe no board %q", toCol, board)
	}

	// exit→enter chaining: run the origin's on_exit first, then the destination's
	// on_enter; commit only if the last one succeeds (same rule as the TUI's applyMove).
	exitText, exitAgent, hasExit := sideAction(s, board, from, false)
	enterText, enterAgent, hasEnter := sideAction(s, board, toCol, true)
	key := ""
	if hasExit {
		k, err := runHookSync(s, t, from, toCol, exitText, exitAgent, out)
		if err != nil {
			return fmt.Errorf("on_exit de %q: %w", from, err)
		}
		key = firstNonEmpty(k, key)
	}
	if hasEnter {
		k, err := runHookSync(s, t, from, toCol, enterText, enterAgent, out)
		if err != nil {
			return fmt.Errorf("on_enter de %q: %w", toCol, err)
		}
		key = firstNonEmpty(k, key)
	}

	if key != "" { // a hook created a Jira issue → stamp the card as a mirror (id = key)
		return s.LinkToJira(id, key, toCol)
	}
	t.Status = toCol
	t.Progress = 0
	return s.Save(t)
}

// sideAction resolves one side of a transition from the store (no Model): on_enter
// (enter=true) or on_exit of col. On the same side, INTENTION (agent) wins over command
// (script). Store-based mirror of the TUI's columnAction, for the headless path.
func sideAction(s *task.Store, board, col string, enter bool) (text string, agent, ok bool) {
	a, has := s.BindingFor(board, col)
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

// runHookSync runs ONE hook (script or agent) to completion, streaming progress lines to
// out, and returns the Jira key it reported (`jira: KEY`, if any). Blocks until the hook's
// terminal event; a non-zero exit becomes an error carrying the hook's reason.
func runHookSync(s *task.Store, t *task.Task, from, to, text string, agent bool, out io.Writer) (string, error) {
	ch := make(chan actionEvent, 64)
	board := s.EffectiveProject(t.ID)
	card := cardJSON(t, from, to, board)
	go func() {
		if agent {
			runAgent(interpolate(resolveIntention(text, s.Dir()), t), card, s.AgentTools(board), ch)
			return
		}
		target, _ := s.BindingFor(board, to)
		env := []string{
			"HAKUBAN_FROM=" + from, "HAKUBAN_TO=" + to, "HAKUBAN_BOARD=" + board,
			"HAKUBAN_STATUS=" + target.Status, "HAKUBAN_DIR=" + s.Dir(),
			"HAKUBAN_TASK_BIN=" + selfBin(),
		}
		runCommand(text, card, env, s.Dir(), ch)
	}()
	for {
		ev := <-ch
		if !ev.done {
			if ev.label != "" {
				fmt.Fprintf(out, "  %d%% %s\n", ev.pct, ev.label)
			}
			continue
		}
		if !ev.ok {
			return "", fmt.Errorf("%s", ev.reason)
		}
		return ev.key, nil
	}
}
