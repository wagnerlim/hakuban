# CLAUDE.md — agent guide (hakuban)

**Terminal** task manager (Kanban TUI) over `.md` files. Go 1.26, Charm ecosystem
(Bubble Tea v2, Lip Gloss v2, Bubbles, Glamour). No database, no web, no CLI —
just the TUI over the disk.

## Read first

- **Product / decisions** → [`docs/objectives.md`](../docs/objectives.md) (concept,
  principles, plain-text, hooks). Read it before changing behaviour.
- **Architecture** → [`.claude/docs/architecture.md`](docs/architecture.md)
- **TUI layer** → [`.claude/docs/tui.md`](docs/tui.md)
- **Patterns, tests, workflow** → [`.claude/docs/patterns.md`](docs/patterns.md)

## Code map

- `cmd/hakuban/main.go` — entrypoint: opens the store and starts the TUI.
- `internal/task/` — **storage. Source of truth = the disk.** (`task.go`,
  `store.go`, `board.go`, `state.go`, `config.go`, `root.go`)
- `internal/tui/` — Bubble Tea view (`tui.go`, `i18n.go`, `themes.go`).

## Golden rules (non-negotiable)

1. **The disk is the source of truth.** The `Store` is an in-memory cache; the TUI
   re-reads the disk ~1×/s (`dirSig`/`refresh`). Never treat memory as authoritative.
2. **Always write atomically** (`atomicWrite`: tmp + rename). There are two legitimate
   writers: the TUI, and a human/AI editing the `.md` by hand.
3. **Every visible string comes from i18n** (`internal/tui/i18n.go`) — nothing hardcoded
   in the view. 3 languages: pt-BR (default), en-US, zh-Hans.
4. **Ponytail.** An existing component > your own code; a native feature > reimplementing
   it; 1 line > 50. A deliberate simplification carries
   `// ponytail: <what>[, <upgrade path>]`.
5. **All code and comments in English.** Dense comments, explaining the *why*.
   Follow the file's tone.
6. **Non-trivial logic leaves 1 test behind** (`go test`, no framework).
7. **Mouse UX is part of the feature.** Every new/changed interactive surface has to work
   with the mouse **and** signal its affordance through the pointer: draggable →
   `grab`/`grabbing`, clickable → `pointer`, input → `text`. 1 click selects,
   2 clicks activate (= Enter). Rules and checklist in
   [`.claude/docs/tui.md`](docs/tui.md) → *Mouse UX*.

## Before committing

```
go build ./... && go vet ./... && gofmt -l internal/ && go test ./...
```

`gofmt -l` with no output = formatted. Verify the actual behaviour (trigger the flow /
reopen the store), not just the compiler.

## Commits

- Conventional commits in English: `feat(tui): …`, `fix(task): …`.
- Trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Personal repo, solo, straight to `main`. Identity
  `68910437+wagnerlim@users.noreply.github.com`.
- Only commit/push **when the user asks** — and their asking **is** the authorization:
  do not ask for confirmation again after a "commit this".
- **Exception: the autonomous Doing agent.** Running through `ct-doing-run.sh` (a card in
  the Doing column of the Project-Board, in a herdr pane, with nobody around to answer),
  commit, push and PR are **authorized up front** — putting the card in Doing IS the
  request. Stopping to ask there stalls the agent and fails the delivery. You recognize
  this context by the cwd `.claude/worktrees/<CARD-ID>` and by the prompt saying it is
  autonomous.
