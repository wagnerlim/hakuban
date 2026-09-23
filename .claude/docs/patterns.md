# Patterns, tests and workflow

## Ponytail (the repo's style)

The laziest solution that **actually works** — and only after understanding the
problem. The ladder (stop at the first rung that holds):

1. does it need to exist? (YAGNI)
2. does it already exist in the repo? reuse it.
3. does the stdlib solve it?
4. does a native feature (`$EDITOR`, git, ripgrep) solve it?
5. does an already-installed dependency solve it?
6. can it be done in 1 line?
7. only then: the minimum that works.

- No speculative abstraction (an interface with 1 impl, a factory for 1 product,
  config for a value that never changes).
- Deletion > addition. Boring > clever.
- A deliberate simplification carries `// ponytail: <what>[, <upgrade path if it has a ceiling>]`.
- Do **not** simplify away: validation at boundaries, error handling that prevents
  data loss, security, accessibility, anything explicitly requested.
- Never be lazy about **understanding** the problem — read the whole flow first.

## Comments

English, dense, explaining the **why** (not the what). Match the density and tone
of the surrounding file. A good comment survives a refactor.

## Tests

- `go test`, **no framework**, no elaborate fixtures.
- Helper `must(t, err)` (`internal/task/store_test.go`).
- `t.TempDir()` to isolate; **reopen the store from disk** to prove persistence
  (do not trust the in-memory map).
- Every non-trivial piece of logic (branch, loop, parser, migration, money/security)
  leaves **1 test** that breaks if the logic breaks. A trivial one-liner does not need one.
- In the TUI you can drive it through events: `m.Update(tea.KeyPressMsg{…})`,
  `tea.MouseClickMsg{…}`, and assert on state / `m.boardView()`.

## Before committing

```
go build ./...          # compiles
go vet ./...            # static analysis
gofmt -l internal/      # no output = formatted (use -w to format)
go test ./...           # all green
```

Beyond the compiler: **verify the behaviour** by triggering the real flow
(reopen the store, simulate the input). If a step was skipped, say so.

## Commits

- Conventional commits in **English**: `feat(tui): …`, `fix(task): …`,
  `docs: …`, `refactor(tui): …`.
- Required trailer:
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Personal repo, solo, straight to `main`.** Identity
  `68910437+wagnerlim@users.noreply.github.com` (remote `github.com/wagnerlim/hakuban`).
- **Only commit/push when the user asks.** Honest splitting: if the changes for
  N features live in the same file, prefer 1 well-described atomic commit over a
  fake split (there is no interactive `git add -p` in this environment).

## Roadmap / seams

- **Per-lane config** (`updateLaneConfig`/`laneConfigBox` in `tui.go`) is the extension
  point for **Jira integration**. When it lands, the natural step is promoting the column
  from `string` → struct (e.g. `{Name, JiraStatusID, …}`) and migrating the boards' YAML
  (minding `ColumnsFor`, `groupByStatus` and the match against the tasks' `Status`).
- Extending the **product** = hooks/scripts + `--json` + `$EDITOR`, never an internal
  plugin API. See [`docs/objectives.md`](../../docs/objectives.md).
