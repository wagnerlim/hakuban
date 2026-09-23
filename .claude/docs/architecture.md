# Architecture

Two layers + an entrypoint. The disk is the source of truth; everything in memory
is a cache.

```
cmd/hakuban/main.go   → opens the Store and runs the TUI
internal/task/           → storage (reads/writes .md, builds the in-memory graph)
internal/tui/            → view (Bubble Tea) on top of the Store
```

## Source of truth = the disk

- Each task is a file `<dir>/tasks/<id>.md`: **YAML frontmatter** (fields) +
  **markdown body** (`Notes`). The `id` comes from the file name (authoritative),
  not from the frontmatter.
- The `Store` (`store.go`) is a `map[string]*Task` + `map[string]*Board` loaded in
  `Open`. Every command/session opens, operates and exits — the map is never
  authoritative.
- **Atomic write** (`atomicWrite`): writes `.tmp` and `rename`s it (atomic on the same
  fs). An external reader never sees a half-written file. Non-negotiable: there are two
  writers (the app + `$EDITOR`/AI).
- **Data dir**: `~/.hakuban` by default; overridden by `$HAKUBAN_TASK_DIR` or by the
  pointer in `<userConfigDir>/hakuban/root.yml` (`root.go`, `ResolveDir`).

## File watching (live reload)

The TUI re-reads the disk on its own: a ~1s tick (`reloadTickMsg`) compares a cheap
signature of the data dir (`dirSig` = name+size+mtime, no parsing) and only reopens the
store when something changed (`refresh`). Editing a `.md` from outside shows up on the
board without reopening the app. **That is a feature, not a bug** — do not break it.

## Data model

### Task (`task.go`)

- Fields: `Title`, `Status`, `Priority`, `Project`, `Tags`, `Due` (`Date`, no time),
  `Parent`, `Created`, `Modified`, `Notes` (the body).
- **`Status` = the text of a board column.** An unknown/empty status falls into column 0
  when grouping (`groupByStatus`) — a task never disappears from the screen.
- Hierarchy through `Parent` (walk-up). The **effective** `Project`/`Tags` are inherited
  from the root of the tree (`EffectiveProject`/`EffectiveTags`), not stored on the child.

### Board (`board.go`)

- A board is `<dir>/boards/<id>.md` (frontmatter: `name`, `columns`, `created`). The
  **board ID is the `project`** of the tasks it groups.
- `Boards()` = the union of boards that have a file + the effective projects actually in
  use (discovered from the tasks), ordered by name. **There is no built-in board.**
- **Columns are per board**: `Board.Columns *[]string`.
  - `nil` (key absent) → falls back to `DefaultColumns` (`backlog/doing/done`) — the old
    Inbox, legacy boards and discovered ones.
  - `&[]string{}` (non-nil, empty) → a new board, deliberately without columns.
  - `ColumnsFor(id)` resolves this and always returns a copy.
- **Orphan migration** (`AdoptOrphans`): at boot, root tasks with no `project` get a board
  (`geral`) with `DefaultColumns` and are moved into it. It runs once and disappears on its
  own when there are no orphans left. A legacy of the old "Inbox".
- `SaveBoard`/`DeleteBoard` ignore an empty ID (`InboxID = ""` is a sentinel for "no board /
  no project", not an actual board).

## Tests for this layer

`go test`, no framework. The pattern: `t.TempDir()` → operate → **reopen the store from
disk** to prove it persisted (do not trust the in-memory map). Helper `must(t, err)` in
`store_test.go`.
