# Roadmap — designed features (not yet 100% implemented)

Design decisions already discussed. The intended order: **keys → subtask → dependency
→ column-as-object (category/Jira)**. Each one makes the next more legible.

## 1. Human-readable per-board keys (`KEY-NN`) — IN PROGRESS

Replaces the random id (`4kjjy4`) with an incremental per-board key, Jira style.

- A board gains a `key` (e.g. `HOME`), **editable in the config**, defaulting to one
  derived from the name ("Holiday house" → `HOLI`).
- A card's id is `<KEY>-<NN>`, and **the id is the file name** (`tasks/HOME-01.md`) — so
  the references inside the `.md` (`parent`, and later `blocked_by`) stay readable.
  Minimum 2 digits (`HOME-01`).
- **The next number = `max(existing KEY-*) + 1`** (derived, with no saved counter;
  ponytail). Accepted trade-off: deleting the highest one can reuse its number.
- **Keys are unique across boards** (validated). The reason: `tasks/` is flat and the id
  is the file → `KEY-NN` has to be globally unique; and a dependency reference crosses
  boards, so it has to be unambiguous.
- **Renaming a key** cascades: it renames the board's files and rewrites the references
  (`parent`, and later `blocked_by`) across every board.
- **A card's number is never renamed** (a stable anchor, like Jira).
- Migrating old data: **no** — we will wipe the data dir and start from scratch.

## 2. Subtask (parent/child hierarchy) — IMPLEMENTED

The model (`Task.Parent`, `Store.Children`, inheritance from the root) + the view.

- `S` on a card creates a subtask (a child); the id comes from the parent's board key
  (`K-02`), Parent links to the root, and it is born in the first column.
- A child **shows up on the board** in the column of its own status, with the tag
  `↳ PARENT` (e.g. `↳ K-01`). The parent card shows **progress** `▓▓▓░░ 3/5` (a bar +
  a count).
- The parent's detail view lists the subtasks (`✓`/`▢`).
- **"Done" = the last column** (the `doneCol` heuristic); it becomes a column category
  once item 4 lands. Progress counts **direct children**.
- Still pending: deep nesting, a border color linking parent/children, creating a subtask
  from inside the detail view too.

## 3. Dependency (blocks / blocked by) — SEPARATE from subtask

A new relation, **not** parent/child. A new frontmatter field:

- `blocked_by: [HOME-02]` on the blocked card; the "unblocks" side is **derived** (whoever
  lists this id in their `blocked_by`), the same way `Children` derives from `Parent`.
- It can **cross boards**.
- **Behaviour: display only** (informational) — a 🔒 badge on the mini-card while a
  blocker is still pending; in the detail view, "blocked by / unblocks".
- It does **not** gate "done" for now — because there is no definition of done (a column
  is free text). Gating comes later (it depends on item 4).

## 4. A column becomes an object (category / Jira / per-lane config) — future

Today a column is a `string`. Three features want the same promotion to an object:

```yaml
columns:
  - {name: "To do", category: todo}
  - {name: "Done",  category: done}   # future: jira_status_id: 10001
```

- A **category** `todo|doing|done` (= Jira's status category) gives you the *definition of
  done* without a state machine. Optional: a board with no `done` column simply has no
  concept of completion.
- It enables: progress by "children in done", the dependency's done-gating, the Jira
  mapping, and the per-lane config (the ⚙ gear).
- **A level-2 state machine** (transition rules between columns): **dropped for now**
  (YAGNI; it fights the product's "hackable, config optional" stance). Only if there is
  concrete pain.

## Jira integration (the north star)

The keys (`KEY-NN`), the column category and the per-lane config (⚙) converge on the
mapping to Jira: key ↔ issue key, category ↔ status category, lane ↔ status. The UI
extension point is `updateLaneConfig`/`laneConfigBox`.
