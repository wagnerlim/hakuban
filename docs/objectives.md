# hakuban — objectives & structure

The source of truth for product and data decisions. Update it here before changing code.

## Concept

hakuban is an **open-source, hackable terminal task manager**, built on the Charm
ecosystem. Its identity rests on three pillars:

- **Durable plain text.** Tasks are `.md` files versioned in git. The data belongs to
  the user, readable and editable outside the app. No lock-in, no database.
- **Hyper-configurable by composition, not by mandatory config.** The user shapes almost
  everything (theme, keybindings, layout, fields/statuses, views), but the **sane
  defaults work on their own** — configuring is optional, never a requirement.
- **Extensible through hooks/scripts — the heart of the project.** Instead of an internal
  plugin API, the core exposes seams: events fire the user's scripts, which receive the
  task as `--json`. All extension lives outside the binary. That is what makes the app
  "hyper" without bloating the core.

Charm ecosystem: Bubble Tea + Lip Gloss + Bubbles + Glamour (TUI/theme), Huh (forms),
Log (logging). Gum is used *by the user's scripts* in the hooks (the app only emits
`--json`). Wish / Soft Serve (serving over SSH/git) are left for later.

## Principles

- **Terminal-only.** No graphical app, no web, no mobile.
- **The `.md` files are the source of truth.** One file per task: frontmatter (structured
  fields) + markdown body (notes). No database.
- **No index/SQLite** while scanning the `.md` files is instant (< thousands of tasks).
  Revisit only if it gets slow — and then as a *derived* index, never the source of truth.
- **Two legitimate writers:** the TUI itself and you (or an AI) editing the `.md` by hand.
  The TUI **re-reads the disk live** — a ~1s poll compares a cheap signature of the data
  dir (name+size+mtime, without parsing) and only reopens the store when something changed
  (`reloadTickMsg`/`dirSig`/`refresh` in `internal/tui`). An external change shows up on
  the board without reopening the app. That is a feature, not a bug.
- **Ponytail:** an existing component (Charm/Cobra) before your own code; a native feature
  (`$EDITOR`, git, ripgrep) before reimplementing it.
- **Extension = composition, not a plugin API.** Configurability comes from plain-text
  files + shelling out (hooks, `--json`, `$EDITOR`), never from a giant config engine or a
  Go plugin API. A small core with exposed seams.
- **Sane defaults > mandatory config.** The app has to be useful with no tweaking at all.
- **Relations = an in-memory graph derived from the `.md` files.** Edges (parent, backlink,
  dependency) live in the frontmatter/notes; the app builds a graph at load time (a stdlib
  `map`, no graph DB and no graph library). Derived and disposable, never the source of
  truth. Inheritance and child-lists are walks over that graph.
- **Modals = a macOS-style window (a single pattern).** *Every* modal — present or future
  — follows the same mould, from one single place in the code (`modalBox` + `overlay` in
  `internal/tui`): **traffic lights on the top border, with breathing room** (the border
  does not touch the dot; the red one is lit and closes, yellow/green are dimmed), a
  **centered title**, a **shadow** (it floats above the board) and **draggable by the
  title bar**. A new modal reuses that mould — we do not invent another shape.

## The structure of a task (frontmatter)

```markdown
---
id: 0a6f                        # generated
title: buy coffee               # required
status: backlog                 # backlog | doing | done (the board's columns)
priority: normal                # low | normal | high
project: home                   # optional; = the id of the board the card belongs to (F23)
tags: [shopping]                # optional
due: 2026-07-09                 # optional
parent: 3b21                    # optional (empty = a root task)
jira: ""                        # v2 (F16); the issue key (PROJ-123) — empty = not linked
source: ""                      # v2 (F16); "jira" = a read-only mirror; empty = a local card
created: 2026-07-08T14:20:00Z   # generated
modified: 2026-07-08T14:20:00Z  # generated
---

## Notes
free-form markdown
```

This structure covers **every** feature below. v2 (jira) and v3 (sync) already have their
space reserved — there is no need to touch the frontmatter later.

Decision: `project` is a field separate from `tags` (filtering "everything in project X"
is common).

## Features

Each feature is a self-contained block. `[v1]` = the core, usable day to day (offline, one
machine). `[v2]` = integration and the heavier work.

**Implementation status:** ✅ done · 🟡 partial · ⬜ (or no marker) = not started.
Where there is progress, the **Done/Missing** line summarizes the real state of the code.

### Progress map

| | Feature | State |
|---|---|---|
| ✅ | **F6b** Kanban board (the main TUI) | fixed columns, mini-cards, moving by keyboard **and by dragging** (a ghost + the target column), per-column scrolling |
| 🟡 | **F23** Boards & tabs | tabs (switch/`+`/`✕`/close), per-board filter, floating+draggable macOS modals, full mouse support, persistence — still missing delete/rename board |
| 🟡 | **F1** Create task | `a` only creates a title → still missing the line parser and the other fields |
| 🟡 | **F2** Tree & links | storage complete (parent/inheritance) → still missing the UI |
| 🟡 | **F4** Detail view | full-screen view + Glamour + scrolling → still missing actions and inherited-vs-own |
| 🟡 | **F5** Complete & archive | completing = moving to DONE → archiving has no key yet |
| 🟡 | **F9** Priority | displayed with a color → still missing setting/customizing it |
| 🟡 | **F18** Extra states | only `doing` → still missing blocked/waiting/cancelled |
| 🟡 | **F20** Views | kanban done → still missing the agenda and stats |
| 🟡 | **F13** Settings | `config.yml` + the `modeSettings` menu (key `s`): 19 themes (a semantic palette, it affects the whole UI), date format, a preview panel in the footer → still missing phase 2 (confirm delete, $EDITOR) and the submenus (priority, rebinding, color overrides) |
| 🟡 | **F21** i18n | the `messages` catalog + `applyLang` (the palette's pattern); pt-BR, en-US, zh-Hans; switchable in the menu → only more languages missing |
| ⬜ | **Not started** | F3 edit · F6 listing · F7 filters · F8 labels · F10 bulk · F11 git in the app · F12 CLI `--json` · F14–F17 · F19 · F22 hooks |

### F1 — Create task `[v1]` 🟡
One-line quick-add: `buy coffee +shopping @home !high due:tomorrow`. Fields: title,
status, priority, project, tags, due, notes (markdown), parent.
A subtask = creating a task with `parent` filled in (it reuses this feature).

**Done:** `a` creates a task with only a title, landing in the backlog. **Missing:** the
line parser (`+tag @proj !prio due:`), the other fields, and creating a subtask from the UI.

### F2 — Tree & links between tasks `[v1]` 🟡
N levels. Creating a child from the parent; re-parenting (moving within the tree).

**Done (storage):** `parent`, derived children (`Children`), dynamic inheritance of
`project`/`tags` (walk-up), tolerating a dangling parent and cycles. **Missing (UI):**
creating a child, re-parenting, deleting with cascade/promote, `[[id]]` backlinks and
`blocked-by`.

**The linking mechanism:**
- Links by **stable `id`** (never by title/file — the title changes, the id does not).
- **Typed edges in the frontmatter:** `parent: <id>` (hierarchy), `blocked-by: [<id>...]`
  (dependency, F17). Free and optional: `[[id]]` in a note (a "see also" backlink, untyped).
- **Store the edge on one side only and derive the inverse** at load time: the child stores
  `parent` → the list of children is derived; `blocked-by` → "what blocks it" is derived;
  `[[id]]` → "who references me" is derived by sweeping the `.md` files. One writer per
  edge, zero bidirectional sync.

**Inheritance (dynamic):** values flow down the `parent` edge live (derived from the graph,
not copied) — consistent with configurable labels (F8).
- `project`: inherited from the **root** of the tree (walk-up). A single value; the child
  does not store it.
- `tags`: an **additive union** — the effective set = its own ∪ every ancestor's. A child
  adds its own; inherited ones do not disappear (removing an inherited one is left for
  after v1).
- Re-parenting re-derives everything automatically.
- Accepted cost: an isolated `.md` does not show what is inherited; F4/F12 expose the
  *effective* values.

**Deleting a parent with children → it asks (never an automatic cascade):** the subtree may
have a life of its own.
- **Cascade:** deletes the parent + the whole subtree.
- **Promote** (the safe default): deletes only the parent; the children move up one level —
  reconnecting to the grandparent if there is one, otherwise becoming roots. On becoming a
  root, the app **materializes** on the new root the `project` that was derived from the
  parent (otherwise it would vanish).

A task with no children is deleted straight away (subject to F13's "confirm before
deleting").

### F3 — Edit task `[v1]`
Edit the fields; edit the notes in `$EDITOR` (shelling out, no editor of our own).

### F4 — Task detail `[v1]` 🟡
Open it (Enter) and see every field + the markdown notes rendered (Glamour). It shows what
is inherited (the parent's project/tags) vs what is its own. From there: edit, mark done,
open `$EDITOR`, see the children.

**Done:** Enter opens the detail full-screen with the fields + the effective
`project`/`tags` + the notes through Glamour + a scrollbar. **Missing:** distinguishing
inherited from own, and the actions launched from it (edit, done, `$EDITOR`, see children).

### F5 — Complete & archive `[v1]` 🟡
Marking done. Archiving completed ones (`.md` → `archive/`) to keep the list light.

**Done:** completing = moving the card to the DONE column (`H/L`). `Store.Archive` exists.
**Missing:** a key in the UI to archive.

### F6b — Kanban board `[v1]` — the main TUI ✅
A 3-column board (`backlog / doing / done`) over the same `.md` files. A column = the
`status` field; moving a card = changing the status + saving. Vim navigation (`hjkl`,
`H/L` switches column), `enter` opens the detail (F4), `a` creates. It is the app's first
and main TUI — it pulls F20's kanban and F18's `doing` state into v1.
The order inside a column = creation order (manual reordering is left for later; it would
need an `order` field in the frontmatter).

**Done:** fixed-width columns, mini-cards (id + priority + tags + title), moving between
columns by keyboard (`H/L`) **or by dragging with the mouse** (the card becomes a ghost
following the cursor + the target column lights up green; dropping it = change the status +
`Save`), per-column scrolling with a bar, uniform height, vim navigation. **Missing:**
manual reordering inside a column (it needs the `order` field in the frontmatter).

### F23 — Boards & tabs `[v1]` 🟡
Multiple boards, with a **browser-style tab** interface. Each card belongs to a board
through the `project` field (**board = project**): the board groups the tasks whose
*effective* project == the board's id. A subtask inherits the parent's board (the `project`
inheritance, F2).

- **A board is an entity** with a file `boards/<id>.md` (name + created) — which allows an
  empty board and a display name. Boards are also **discovered** from the projects in use,
  so old projects become tabs without needing a file.
- **Inbox** is the built-in board for tasks with no project (an empty id); it always exists.
- **Tabs:** switching tabs, `+` creates a new board, closing a tab (it disappears from the
  bar, it does not delete the board), and a **modal** lists every board to open.

**Done:** the tab bar, the active-board filter, `+` creates a board (a file) and opens the
tab, `^w` closes, `tab`/`shift+tab` switches, `b` opens the list modal, creating a task
lands in the active board. Modals **float over the board** (the Lip Gloss v2 compositor).
Modals have a **macOS-style title bar** (traffic lights: the red one is lit and closes,
yellow/green are dimmed) and a **shadow** (they float above the board) — a pattern shared
by all of them through `modalBox`. Open tabs **persist** between sessions (`state.yml`).

**Mouse:** clicking a tab switches to it, the tab's `✕` (or the middle button) closes it —
the `✕` disappears when there is only 1 tab; clicking `+` creates one; in a modal, the red
dot closes it and the title bar drags it; the scroll wheel scrolls the detail view.
**Missing:** deleting/renaming a board, custom per-board columns.

### F6 — Listing `[v1]`
Paginated, with drill-down into the tree (entering the parent's level). A row: title,
priority, due, project, tags (a colored chip), children progress (2/5). Sorting (due,
priority, creation, alphabetical) and grouping (project, priority, deadline).

### F7 — Filters & views `[v1]`
Filtering by field: status, priority, project, tag, due, parent. **Combinable** filters
(e.g. project=home AND priority=high). Ready-made views: today, this week, no deadline,
high priority, no project. Text search over title/notes.

### F8 — Labels `[v1]`
A tag is a first-class label: a name + a color. CRUD: create, rename, recolor, merge,
remove (and what that does to the tasks using it).

### F9 — Priority `[v1]` 🟡
Levels with a color. (Fixed low/normal/high vs customizable → see Open decisions.)

**Done:** displayed with a color (● high/normal, ○ low) on the card and as a badge in the
detail view. **Missing:** setting/changing it from the UI and customizable levels (F13).

### F10 — Bulk actions `[v1]`
Multiple selection: marking done, moving, tagging several at once.

### F11 — Git-backed storage `[v1]`
Versioned `.md` files → per-task history, undo = `git revert`. The foundation for sync (F14).

### F12 — CLI + `--json` `[v1]`
`hakuban add/list --json`. Scriptable and readable by external tools.
It is what enables the Claude Code integration (F15).

### F13 — Settings `[v1]` 🟡
A settings menu (and a config file in the data dir, editable by hand). Options for the user
to adjust the behaviour — several "open decisions" become config with a default instead of
a locked-in choice.

**`config.yml`** lives in the data dir next to `state.yml`, following the same pattern as
`state.go` (a struct + yaml + `atomicWrite`). Sane defaults: the file may not exist and
everything still works.

```yaml
theme: omni              # the theme name (the personal omni + 19 from upstreams) — live (semantic palette)
lang: pt-BR              # pt-BR | en-US | zh-Hans        — live (the i18n catalog)
preview_pane: false      # the .md panel in the footer, following the cursor — live (previewPane)
date_format: "2006-01-02"# a Go layout                    — live today (rendering the due date)
editor: ""               # "" = the environment's $EDITOR — F3 (shelling out)
confirm_delete: true     #                                — F2 (deleting in the UI)
priorities: [low, normal, high]  # the levels            — F9
# colors / priority_colors: palette submenus             — phase 3
```

**The menu (`modeSettings`, key `s`):** a modal reusing `modalBox` + navigation in the
style of `boardListBox`/`updateBoardList`. Each row is `Label: value`. `j/k` moves,
`←→`/`space` cycles a toggle/enum, `enter` enters a submenu or inline editing (reusing
`textinput`). It persists to `config.yml` on every change. Three kinds of control cover
almost everything — `toggle` (bool), `cycle` (enum), `edit` (text); `submenu` (colors,
levels, rebinding) is the fourth, and the heaviest.

- **A selectable data dir, through a pointer outside the config.** `config.yml` lives
  inside the data dir, so it cannot store its own path (that would be circular) — the path
  lives in a pointer at `<os.UserConfigDir>/hakuban/root.yml`
  (`internal/task/root.go`). Boot resolves `env HAKUBAN_TASK_DIR > root.yml > ~/.hakuban`
  (`ResolveDir`). In the menu, the "Data directory" row opens a **yazi-style folder
  browser** (`modeDirBrowser` — Miller columns parent·current·preview, a selection bar in
  the accent color, nerd-font folder icons colored by the theme, live filtering with `/`);
  when you pick a different destination it asks whether to **move** the files
  (`modeConfirmMove`): yes moves them (`MoveData`, through `os.Rename`, **aborting if the
  destination already has data** — it never mixes; cross-filesystem is not supported yet),
  no just repoints. Afterwards it rewrites the pointer and reopens the store in the new dir.
- **No listing section** (density/sorting/grouping/default filter): those presupposed
  F6/F7, which do not exist — cut until they make sense.
- **A theme is a semantic palette, not detection.** `internal/tui/themes.go` has a
  `palette` struct (accent/surfaces/overlays/hues tokens, Catppuccin's vocabulary) and the
  themes: **`omni`** (the author's personal one — Rocketseat, from his own config; the
  default) + 19 from upstreams (Catppuccin, Tokyo Night, Dracula, Nord, Gruvbox, One,
  Solarized, Kanagawa, Rosé Pine, Vesper + light variants, and a 16-color `terminal`
  fallback that inherits the background). `applyPalette` rebuilds every TUI style from the
  active theme — the theme affects the whole UI, not just Glamour. **No `auto`/detection:**
  the theme is an explicit choice; the theme's `isDark` decides Glamour's style. The TUI
  does not paint the terminal's background — dark/light = color values with contrast, not a
  painted canvas.

**Slicing:**
1. The `config.yml` plumbing + a navigable menu + the live ones (theme, `date_format`,
   `preview_pane` — the `.md` panel in the footer following the card under the cursor,
   reusing the detail view's notes/badges rendering; a peek without scrolling, the full
   view on Enter).
2. Inert but persistable toggles/edits (`confirm_delete`, `editor`) — they become effective
   when F2/F3 arrive.
3. The heavy submenus (priority levels, key rebinding, and color *overrides* on top of the
   theme) — each one is a miniature feature; they come last. The base palette/themes are
   already done (slice 1).

Ponytail: the config is a simple file in the data dir; the menu only edits that file.

### F14 — Sync between machines `[v2]`
Formalizing the git-backed data dir (F11) — `push`/`pull` as sync.

### F15 — Claude Code integration `[v2]`
Reading/creating tasks through `--json` (F12) / MCP.

### F16 — Integration with Jira (and friends) `[v2]`
Integration with external trackers (Jira first; GitHub/Linear/… reuse the same seam) **by
composition, not by an HTTP client in the core**. The binary never speaks over the network:
it exposes hooks and fires **the user's configurable commands** (F22), which do the REST
outside (bash + `gum`, or Claude Code). The golden rule is preserved: **the core stays
offline; the disk stays the source of truth.**

**A bound column (pull, read-only).** A board's column can be *local* (the default — your
`.md` files, editable, a new card is born here) or *bound* to a **JQL**. A configurable
**sync** command runs the JQL and **materializes mirror `.md` files** in a derived cache (a
gitignored dir, `source: jira`, `jira: PROJ-123`) with `status` = the column. The TUI
re-reads the disk every ~1s and shows them as normal cards.

**A column button (the column says what it does).** In the **column footer** lives a
configurable control (the counterpart of the ⚙ at the top): the column declares `buttons:`
— each one with a `label`, an `icon` and **one** `cmd` (a script, receiving the column's
cards as JSON on stdin) or `agent` (an intention). One button → the click (or the key, with
the cursor on the column) fires it directly; **several** → the footer becomes `label ▾` and
opens a **popover** anchored to the button, growing upwards and to the right, with the
first option on the button's own row. `batch: true` enlists the button in the board-wide
button in the top bar, which runs the marked ones one at a time.

**A top-bar button (the same structure, board scope).** The board declares its own in
`buttons:` at the top of the file: `cmd:` receives **every** card on the board on stdin,
`agent:` is an intention, and `batch: true` means "run the columns' `batch` buttons". One
button fires directly; several open the same popover, which here **drops down** from the
button (the invariant: the first button is always hugging the anchor). A `cmd`/`agent`
loader takes the button's own place — `batch` shows up in each column's footer, one at a
time. A board with no `buttons:` synthesizes "Sync everything" from the columns' `batch`
buttons; `sync_button:` still controls the position (`top-left`/`top-right`/`off`).
While it runs, the button becomes the **progress loader**; when it ends it shows ✓ or `:(`.
The **loader style is per column**: `loader:` picks the glyphs (`segments` is the default,
plus `blocks`, `line`, `dots`, `ascii` — an unknown name falls back to the default, it never
breaks the render) and `percent: false` hides the number. It also applies to the `%`
persisted on that column's cards, so the card and the button speak the same language. Sync
is just **one use case** of this: a bound column with no `buttons:` synthesizes the sync
button from the board's `sync:` (for compatibility). A column with no buttons at all has an
empty footer. Besides the click, the first batch button optionally runs when the board
opens (`sync_on_open`). The mirrors are a **disposable cache** (like `state.yml`/the
graph), not the user's data — which is why "the disk is the truth" still holds: the truth
for that card is *Jira*, the `.md` is only the local projection. The content
(title/notes/fields) is **read-only**; the **position in the column is not** — moving is an
*action*, not an edit (see below).

**Column actions (enter/exit).** Each column optionally carries an `on_enter` and an
`on_exit` action — each one either "nothing" (the default) or a **user command**. Moving a
card **A → B** fires A's exit action and then B's enter action (usually one of the sides is
"nothing", so they do not conflict). This is where writing happens — always in the *script*,
never in the core:
- A local **DRAFTS → TO-DO** (bound): TO-DO's `on_enter` = a script creates the issue in
  Jira, stamps `jira: PROJ-456` into the `.md`, and the card becomes a mirror.
- **TO-DO → DOING** (both bound): DOING's enter = a script transitions the issue in Jira.
  (Or hang it off the exit — the user picks the side.)

**A real progress loader.** The action runs as a **background subprocess** (the TUI never
freezes). The card shows a **progress bar** driven by the script itself through stdout (the
protocol is in F22). **Success** (exit 0) → a full bar + ✓ and the move **commits** (writes
the `.md`). **Failure** (exit ≠0) → `██░░░░ :( reason` (the last line of stderr), the card
**stays at the origin** and **nothing** is written. The loader's state is **in memory and
disposable**: closing the app midway = the card goes back to the origin, nothing committed.
When rendering, the loader takes the mini-card's annotation slot (`↳ PARENT`/the subtask
bar) — no new layout.

**Config (per board).** What is bound lives in `boards/<id>.md` (each board = one Jira
project): per column, a `jql` + `on_enter`/`on_exit`. Auth lives **outside the versioned
plain text**: a token in the environment (`JIRA_TOKEN`) or in a gitignored file.

```yaml
# boards/personal.md
columns: [DRAFTS, TO-DO, DOING, DONE]
actions:
  TO-DO: { jql: "project=ABC AND status='To Do'",     on_enter: "~/.hakuban/hooks/jira-create.sh" }
  DOING: { jql: "project=ABC AND status='In Progress'", on_enter: "~/.hakuban/hooks/jira-move.sh" }
```

**Publishing (push) is a particular case of this:** a bound column's `on_enter` publishes
the `.md` to Jira (title→summary, notes→description, priority→priority) — zero REST code in
the app, exactly as the original note predicted.

#### Evolution (decision of 2026-07-21): an action = an **agent intention**, not a script

**Motivation.** The script approach (curl+jq+basic-auth) proved brittle in a real test:
`JIRA_URL` vs the token's account, `.io` vs `.ai`, a discontinued search endpoint, a 404
from permissions/identity. Meanwhile, the transitions performed **by an agent over MCP
worked first try** — because the agent uses an already authenticated connection (no token,
no endpoint, no account). Decision: `on_enter`/`on_exit` become a **Markdown description of
what the agent should do**, not a path to a script. The core stays offline: it only
**hands over the intention + the card (`--json`) + `from`/`to`/board** to an agent and shows
the progress/result. The one who knows about Jira is the agent, through its own tools.

```yaml
# boards/<id>.md — an action = an intention (prose); on_enter_cmd = a deterministic script
agent_tools: mcp__claude_ai_Atlassian_Rovo   # the agent's --allowedTools (MCP only, never the disk)
actions:
  Done:
    jql: 'project = DEMO AND status = "Done"'
    on_enter: |                               # an intention → a headless agent
      Move issue {{.jira}} to the "Done" status in Jira.
    # on_enter_cmd: ~/.hakuban/hooks/x.sh  # the deterministic alternative (F22)
```

Fields interpolated into the intention: `{{.jira}}`, `{{.title}}`, `{{.id}}`,
`{{.status}}`, `{{.priority}}`, `{{.project}}`, `{{.notes}}`. The whole card also goes into
the prompt as JSON (a `<card>` block), so the agent has all the context.

**How the agent is invoked — 2 models (the core does not run an LLM):**

1. **Headless per move (real time, with a loader).** The move fires
   `claude -p "<intention + card>"`. **Blocker found (2026-07-21):** the CLI shows
   *"claude.ai connectors are disabled because ANTHROPIC_API_KEY … takes precedence"* — that
   is, `claude -p` **cannot see the Atlassian MCP** with the current setup. To make it
   viable: `unset ANTHROPIC_API_KEY` + log into claude.ai (then `claude mcp list` should
   list Atlassian), **or** configure a local Atlassian MCP for headless use (the credential
   at the MCP layer, once). Validate before building: run a `claude -p` that transitions
   DEMO-1.

2. **A queue + an agent in the session (works today, token-free).** The move **executes
   nothing at the time** — the core writes `outbox/<ts>_<card>.md` (the MD intention + the
   card as json + from/to) and marks the card as **⏳ pending**. In a Claude Code session,
   the user asks to "process the queue" and the agent executes it over MCP (the live
   connection), stamps the result and clears the queue.

**Resumption plan (in this order):**
- [x] Test model **1** (headless) — **validated 2026-07-21**: without `ANTHROPIC_API_KEY`
  (the key turns off the connectors), `claude -p` transitions DEMO-1 over MCP on a claude.ai
  subscription. That is the way (real time). Model **2** (the `outbox` queue) is archived as
  a fallback, not implemented.
- [x] **Slice A — the write side (transition).** In the core, `on_enter`/`on_exit` are an
  **intention** (MD prose, interpolated with `{{.jira}}` etc.) handed to the headless agent;
  `on_enter_cmd`/`on_exit_cmd` remain a deterministic script (F22). The recipe is locked:
  a neutral cwd (the agent does not explore the project), `env -u ANTHROPIC_API_KEY`,
  `--allowedTools <board.agent_tools>` (MCP only, never the disk), `--output-format
  stream-json` → each `tool_use` becomes a loader step. It only commits on exit 0.
  (`runAgent`/`interpolate` in `internal/tui/action.go`; `agent_tools` on the board.)
- [x] **Slice B — the pull side (sync) + sync-on-open.** Sync through an agent,
  **board-wide in a single call**: the core synthesizes the intention (the OR of the bound
  columns' JQLs), the agent **fetches over MCP and returns the issues as JSON**, and the
  **core writes the mirrors** (`ReconcileMirrors`: maps the Jira status → a column through
  `ColumnAction.Status`, reconciles, and clears stale ones). The disk stays in the core; the
  agent only touches MCP. The board's `sync_on_open` flag fires it in `Init()` → a linked
  card lands in the column of its current Jira status (you do not work on top of a card
  someone else already finished). The old script (`sync:`) remains as a deterministic escape
  hatch. Verified: a real pull of DEMO returned parseable JSON (even with ```json fences).

**Current state (branch `feat/integracao-jira`):** slices A and B are done and verified
against the real Jira (a transition + a pull of DEMO). A column action = an **agent
intention** (`on_enter`/`on_exit` prose; `*_cmd` = the deterministic F22 script); sync = a
**board-wide agent** with the core writing the mirrors (`sync:` = the legacy script). Board
config: `agent_tools`, `sync_on_open`, `jql`+`status` per column. A real loader through
stream-json (each tool_use becomes a step). Still to mature: stamping an issue CREATED
through the agent (today only transition/pull), and other trackers reusing the same seam.

### F17 — Dependencies between tasks `[v2]`
"Blocked by X" — a relation different from parent/child (ordering, not decomposition).
It enables "what can I do right now".

### F18 — Extra states `[v2]` 🟡
Beyond pending/done: in-progress, blocked, waiting, cancelled.

**Done:** `doing` (in-progress) already exists as a board column. **Missing:** blocked,
waiting, cancelled.

### F19 — Start/defer date + snooze `[v2]`
A task only shows up from a given day; postponing a deadline with one key.

### F20 — Views `[v2]` 🟡
An agenda/calendar by date, a kanban by status, stats (finished this week, overdue).

**Done:** the kanban by status (that is F6b). **Missing:** the agenda/calendar and the stats.

### F21 — Language support (i18n) `[v1]` 🟡
Every UI string comes from a message catalog (not hardcoded). The language is chosen in the
config (F13). Cross-cutting — it is born in v1 so the whole UI does not have to be rewritten
later.

**Done:** `internal/tui/i18n.go` has a `messages` struct (every UI string) + a catalog per
language; a global active instance (`msg`) swapped by `applyLang`, the same pattern as the
palette. Languages: **pt-BR** (the default), **en-US** and **zh-Hans** (Simplified Chinese).
Switchable in the settings menu (the "Language" row, which cycles). `statusLabel`/
`priorityLabel` translate the data keys (backlog/high/…) without touching what is saved in
the `.md`. Columns, badges, priority, help text, modal titles, placeholders and metadata all
come from the catalog. **Missing:** only adding more languages (= a new entry in `langs` +
`langNames`).

### F22 — Hooks & extensibility `[v1]` — the heart of the project
Events fire the user's scripts, which receive the task as `--json` on stdin (e.g.
`on-done`, `on-create`, `on-edit`). Infinite extension with no plugin API: the user wires
up notifications, sync, publishing, integration — whatever they want — outside the binary,
using `gum` and other tools in their own scripts. v1 ships the basic mechanism (events +
`--json`); a broad catalog of events and custom commands mapped to keys mature later.

**Transition actions (the seam F16 uses).** The richest event is `on-move`, modelled as
**per-column actions** (`on_enter`/`on_exit`, see F16): the core fires the configured
command passing the card as `--json` on stdin + the context (`from`/`to`/board through
flags/env). The command runs as a **background subprocess** — the TUI never blocks (the
Bubble Tea pattern: listen on a channel → it becomes a `msg` → listen again). It is generic:
Jira is only the first consumer; the same seam serves notifications, git, a webhook,
whatever.

**Progress protocol (stdout streaming).** To give a *real* progress bar without the core
knowing anything about the task, the script streams the progress — one update per line on
**stdout**:
- `N` → a percentage 0–100, **or** `N/M` → step N of M (the core computes the %);
- optional text after it becomes the label next to the bar (`echo "2/3 creating issue"`).

The core reads each line → `actionProgressMsg{id, pct, label}` → redraws the card. **exit 0**
= success (a full bar + ✓, it commits the move); **exit ≠0** = failure (`:(` + the last line
of **stderr** as the reason, with no commit). **stderr** = logs. The loader's state is in
memory and disposable. It is the heaviest part of the extensibility (a subprocess + a stream
parser), but it is built **once** and reused by every integration.

## Non-goals (traps — do not build early)

- A notification/reminder daemon (use `hakuban due` + the OS' cron).
- A text editor of our own (use `$EDITOR`).
- A full-text search engine (use ripgrep/fzf over the `.md` files).
- Recurring tasks (a pit of edge cases; only if it becomes real pain).
- Time tracking, multi-user, accounts.

## Data layout (proposal)

```
~/.hakuban/     # the data dir (the default; selectable — see F13)
  tasks/        # <id>.md — active tasks
  boards/       # <id>.md — named/empty boards (F23)
  config.yml    # preferences (theme, language, etc. — F13)
  state.yml     # open tabs + the active one (UI, derived/disposable)
  archive/      # archived done tasks
  jira/         # <id>.md — read-only Jira mirrors (F16); a derived cache, gitignored
  hooks/        # the user's scripts (on_enter/on_exit/sync — F16/F22)
  .git/         # history + sync (optional, opt-in)

<os.UserConfigDir>/hakuban/root.yml   # the pointer to the data dir (outside it; F13)
```

## Open decisions

- Config *defaults* (F13): the list's default filter, fixed vs customizable priority, a
  preview in the listing — to be decided while implementing.
- **F16 — integration (assumed, revisit while implementing):** the binding/actions live
  **per board** (`boards/<id>.md`), not globally in `config.yml`. When the sync fires:
  manually (a key, behind the prefix) + optionally when opening the board. If one day a
  global config makes more sense (one Jira project shared by several boards), migrate while
  keeping the board as an override.
