// Command hakuban: opens the Kanban board over the default data dir.
//
// Headless subcommands (LLM-driven board) — generic, tracker-agnostic primitives that
// the board's hooks compose; all the automation lives in the hooks, not here:
//
//	hakuban move <id> <coluna>     move a card firing the column hooks (on_exit→on_enter)
//	hakuban progress <id> <pct>    set the card's displayed % (0..100)
//
// With no subcommand it opens the TUI.
package main

import (
	"fmt"
	"os"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/wagnerlim/hakuban/internal/task"
	"github.com/wagnerlim/hakuban/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "move" {
		os.Exit(runMove(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "progress" {
		os.Exit(runProgress(os.Args[2:]))
	}
	store, err := task.Open(task.ResolveDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, "hakuban: erro abrindo store:", err)
		os.Exit(1)
	}
	// v2: alt screen is a field of the View (tui.Model.View), no longer a program option.
	if _, err := tea.NewProgram(tui.New(store)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "hakuban:", err)
		os.Exit(1)
	}
}

// runMove is the `hakuban move <id> <coluna>` subcommand: it drives one card move
// headlessly (same hooks as the TUI), so an LLM can advance the board. Returns the process
// exit code — non-zero on a bad usage or a failed hook (the card then stays put).
func runMove(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: hakuban move <id> <coluna>")
		return 2
	}
	store, err := task.Open(task.ResolveDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, "hakuban: erro abrindo store:", err)
		return 1
	}
	if err := tui.MoveHeadless(store, args[0], args[1], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "hakuban: move falhou:", err)
		return 1
	}
	fmt.Printf("movido %s → %s\n", args[0], args[1])
	return 0
}

// runProgress is the `hakuban progress <id> <pct>` subcommand: a generic primitive to
// stamp a card's displayed % (0..100). A board hook (e.g. the detached Doing agent) calls
// it to report progress; the TUI reflects it on the next disk reread. Tracker-agnostic.
func runProgress(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: hakuban progress <id> <pct>")
		return 2
	}
	pct, err := strconv.Atoi(args[1])
	if err != nil || pct < 0 || pct > 100 {
		fmt.Fprintln(os.Stderr, "hakuban: pct deve ser um inteiro 0..100")
		return 2
	}
	store, err := task.Open(task.ResolveDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, "hakuban: erro abrindo store:", err)
		return 1
	}
	t := store.Get(args[0])
	if t == nil {
		fmt.Fprintf(os.Stderr, "hakuban: card %q does not exist\n", args[0])
		return 1
	}
	t.Progress = pct
	if err := store.Save(t); err != nil {
		fmt.Fprintln(os.Stderr, "hakuban: erro salvando:", err)
		return 1
	}
	return 0
}
