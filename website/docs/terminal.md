---
sidebar_position: 2
title: In the terminal
---

# In the terminal

Everything here is optional. Hakuban runs with no `config.yml` at all — every field below
has a default, and a missing file is a valid state.

## Two ways in

```bash
hakuban                          # the TUI (needs a TTY)
hakuban move <id> "<Column>"     # headless: fires on_exit of the origin, then on_enter of the target
hakuban progress <id> <0..100>   # headless: the percentage shown on the card
```

The two subcommands are the whole headless surface. They are deliberately generic — there
is no `hakuban jira ...`, no `hakuban deploy`. Anything specific lives in a hook that the
board fires.

Exit codes: `0` done, `1` the operation failed (a failed hook leaves the card where it
was), `2` bad usage. Column names are compared exactly and are case-sensitive, so quote
anything with a space.

## Where the data dir comes from

Resolved at boot, first match wins:

1. `HAKUBAN_TASK_DIR` — environment
2. `<os user config dir>/hakuban/root.yml`, field `datadir` — the pointer the TUI writes
   when you pick another folder
3. `~/.hakuban`

The pointer exists because `config.yml` lives *inside* the data dir and therefore cannot
store its own path. Moving the data dir from inside the TUI never mixes two sets: if the
target folder already holds Hakuban data, the move is refused.

## Keyboard

Two groups. **Navigation fires directly.** **Commands sit behind a tmux-style prefix,
`ctrl+t`** — press the prefix, then the key.

### Navigation (no prefix)

| action id | default | what it does |
|---|---|---|
| `nav_left` / `nav_right` | `h` / `l` | move the cursor between columns |
| `nav_down` / `nav_up` | `j` / `k` | move the cursor between cards |
| `move_left` / `move_right` | `H` / `L` | **move the card** one column — this fires the hooks |
| `next_board` / `prev_board` | `tab` / `shift+tab` | switch board tab |
| `open` | `enter` | open the selected card |

### Commands (after `ctrl+t`)

| action id | default | what it opens |
|---|---|---|
| `add` | `a` | new card in the current column |
| `subtask` | `S` | new child card of the selected one |
| `find` | `/` | card search — an input plus a list, `enter` jumps the cursor to the hit |
| `board_list` | `b` | board list: open, close, create |
| `new_board` | `+` | create a board |
| `close_board` | `ctrl+w` | close the current board tab |
| `board_cfg` | `c` | board config: rename, key prefix, columns (reorder / add / delete) |
| `lane_cfg` | `g` | config of one column: filters, rename, delete |
| `sync` | `y` | run the board's `sync` for the current column |
| `tags` | `t` | tag catalog: color and description per tag |
| `card_tags` | `T` | check/uncheck catalog tags on the selected card |
| `settings` | `s` | theme, language, preview pane, date format, editor, data dir |
| `keymap` | `?` | the shortcut editor — it lists action → key and captures a new one |
| `quit` | `q` | quit |

### Aliases that rebinding cannot break

Independent of the config and of the prefix: `←` `→` `↑` `↓` for navigation, `<` and `>`
to move the card, `ctrl+c` to quit.

### Rebinding

Open the keymap editor (`ctrl+t` `?`), pick an action, press the new key. Only what
**differs from the default** is written to `config.yml`, so the file stays small and a
future change of defaults still reaches you:

```yaml
keys:
  add: n
  find: f
```

The keys of that map are the action ids from the tables above.

## Mouse

The keyboard is the canonical path and the mouse mirrors it. The pointer tells you what a
surface does before you click:

| pointer | meaning | where |
|---|---|---|
| `grab` / `grabbing` | draggable | cards, scroll tracks, modal titlebars |
| `pointer` | acts on one click | ☰, sync button, tabs, menu/config/filter rows |
| `text` | focused input | any input |

And the interaction is the same everywhere:

- **one click selects** — it moves the cursor, it does not act
- **two clicks activate** — equivalent to `enter`
- **drag** = press plus movement: a card between columns, a scroll track, a modal by its
  titlebar. A click that does not move never drags.
- **the card id in the detail view is a link** — one click, no modifier, opens the issue in
  your browser. It needs `issue_url` on the board (see below).

Modals are draggable by the titlebar and close on the red dot.

## `config.yml`

Lives at `<data dir>/config.yml`. Hand-editable — the TUI reads it on load and writes it
back when you change something in Settings. Every field is optional.

```yaml
theme: omni                 # see the list below
lang: pt-BR                 # pt-BR | en-US | zh-Hans
preview_pane: true          # markdown pane in the footer
date_format: 2006-01-02     # a Go layout, applied to `due`
editor: ''                  # '' = $EDITOR
confirm_delete: true        # ask before deleting
priorities: [low, normal, high]
keys:                       # only the overrides
  add: n
tags:                       # the tag catalog
  - name: backend
    color: blue             # a palette hue, not a hex
    desc: touches the API
```

| field | default | notes |
|---|---|---|
| `theme` | `omni` | swappable at runtime in Settings |
| `lang` | `pt-BR` | every visible string comes from the i18n catalog |
| `preview_pane` | `false` | the card's markdown in the footer |
| `date_format` | `2006-01-02` | Go layout, not `YYYY-MM-DD` |
| `editor` | `''` | falls back to `$EDITOR` |
| `confirm_delete` | `true` | |
| `priorities` | `[low, normal, high]` | the levels a card can carry |
| `keys` | — | action → key, overrides only |
| `tags` | — | catalog: `name`, `color`, `desc` |

It does **not** hold the data dir — that would live inside itself.

## Themes

19 in the box, swappable at runtime:

`omni` (default) · `catppuccin` · `catppuccin-latte` · `tokyo-night` · `tokyo-night-day` ·
`dracula` · `nord` · `gruvbox` · `gruvbox-light` · `one-dark` · `one-light` · `solarized` ·
`solarized-light` · `kanagawa` · `kanagawa-lotus` · `rose-pine` · `rose-pine-dawn` ·
`vesper` · `terminal`

The light ones are `catppuccin-latte`, `gruvbox-light`, `one-light`, `solarized-light`,
`tokyo-night-day`, `kanagawa-lotus` and `rose-pine-dawn`. `terminal` uses your emulator's
own ANSI colors instead of hex, so it inherits whatever your terminal is set to.

Palettes are credited to their upstreams (Catppuccin, Nord, Dracula, Gruvbox, Solarized,
Tokyo Night, Kanagawa, Rosé Pine, One Dark). Adding one is a struct.

A tag's `color` is a **named hue from the active palette**, not a hex: `mauve`, `blue`,
`green`, `yellow`, `peach`, `red`, `teal`. That is why a tag keeps looking right when you
change the theme. An empty color, or a tag outside the catalog, renders in the default
color — the catalog is a suggestion, not a lock.

## Filters

Filters are declared once in the board's `filters` registry and then enabled per column
with `use_filters`. There are **two natures**, and confusing them is the usual mistake.

### Tracker filter — shapes what `sync` pulls

The core never interprets the value; it forwards your selection to the sync hook, which
composes it into the query. Online only: it cannot touch a local card. Three forms:

```yaml
filters:
  mine: assignee = currentUser()          # simple: a scalar, toggled on/off
  sprint:                                  # static options: multi-select
    options:
      Current: sprint in openSprints()
      Next: sprint in futureSprints()
  epic:                                    # dynamic options: listed at use time
    options_cmd: hooks/list-epics.sh
```

The hook receives them in `HAKUBAN_FILTERS`, as JSON **grouped by filter**. The core never
joins them: the convention is that the hook ORs the options of one filter and ANDs across
filters.

### Field filter — evaluated locally

A predicate over the card's own fields. Works offline, on any card, local or mirror:

```yaml
filters:
  recent:
    field: modified      # modified | created
    within_days: 7
```

A misconfigured filter — unknown field, zero timestamp — keeps everything. Nothing is
hidden silently.

### Enabling them on a column

```yaml
actions:
  Backlog:
    jql: project = TEAM AND status = "To Do"
    use_filters:
      sprint: [Current]
      mine: []
```

## Column buttons

A button in a column's footer, so an action can run over **every card of the column** and
not just the selected one:

```yaml
actions:
  Review:
    button_label: checks
    buttons:
      - label: run CI
        icon: ▶
        cmd: hooks/ci.sh          # script: the column's cards as a JSON array on stdin
        batch: true
      - label: summarize
        agent: |                   # instruction: prose, scoped by agent_tools
          Write one paragraph per card in this column.
```

`cmd` follows the same contract as a transition hook — see
[the hook contract](./hook-contract.md) — except stdin is an **array** of the column's
cards. `agent` is an instruction, and it may be the path of a `.md` under the data dir.
`batch: true` enrolls the button in the board-wide button on the top bar, which fires the
batch buttons of every column, one at a time.

## Opening a card in the tracker

```yaml
issue_url: https://acme.atlassian.net/browse/{key}
```

`{key}` is replaced by the card's external key. With this set, the id in the detail view
becomes clickable and opens in your browser (`open` / `xdg-open` / `rundll32`). Without it,
the id is plain text.

## The sync contract

`sync` is one command line, like any hook, and it receives:

| channel | content |
|---|---|
| stdin | the column's current cards, as a JSON array |
| `HAKUBAN_JQL` | the column's `jql` |
| `HAKUBAN_COLUMN` | the column name |
| `HAKUBAN_BOARD` | the board id |
| `HAKUBAN_DIR` | the data dir |
| `HAKUBAN_FILTERS` | the selected filters, JSON grouped by filter (`{}` when none) |
| `HAKUBAN_COMMENTS` | `1` when the board sets `comments: true` |
| `HAKUBAN_TASK_BIN` | the running binary, for a hook that calls back |

`sync_on_open: true` runs it when the board is opened; otherwise it is `ctrl+t` `y`, the
column's sync button, or the board-wide button.
