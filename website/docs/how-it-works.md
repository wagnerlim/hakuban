---
sidebar_position: 1.5
title: How it works
---

# How it works

There is no database and no server. Hakuban is a folder of Markdown files and a binary
that reads it.

## The data dir

```text
~/.hakuban/
├── boards/
│   └── demo.md          # one board: columns + what each column does
├── tasks/
│   ├── DEMO-1.md        # one card per file, named by its id
│   ├── DEMO-2.md
│   └── DEMO-3.md
├── jira/                # mirrors written by a board's `sync` — read-only
├── archive/             # archived cards, same format
├── hooks/               # your scripts and instructions — by convention; paths resolve from ~/.hakuban
├── config.yml           # theme, language, tags, keys — optional
└── state.yml            # which tabs are open — disposable
```

Only `boards/` and `tasks/` matter. Delete `state.yml` and the TUI reopens every board;
delete `config.yml` and every field falls back to its default. Where the folder lives is
covered in [In the terminal](./terminal.md#where-the-data-dir-comes-from).

## A card is a file

```markdown
---
id: DEMO-2
title: Fix the timezone drift on the invoice list
status: To-Do
priority: normal
project: demo
tags:
    - FRONTEND
created: 2026-09-04T09:00:00Z
modified: 2026-09-04T18:38:41Z
---

Dates render a day early for anyone west of UTC.
```

Two fields tie it to a board:

- `project` is the board id — the file name in `boards/`, without `.md`.
- `status` is the column the card sits in. Moving a card rewrites this one line.

The id comes from the board's `key` plus a counter (`DEMO-1`, `DEMO-2`…). A card with a
`parent` is a subtask and inherits the board of its root. Everything below the closing
`---` is the card's notes — free Markdown, and the place an agent reads its task from.

## What a move does

Drag a card in the TUI, or run `hakuban move DEMO-2 Doing` — same path:

1. `on_exit` of the origin column runs, if there is one.
2. `on_enter` of the target column runs, if there is one.
3. Only if both succeed: `status` becomes the target, `progress` resets to `0`, and the
   file is written.

A hook that exits non-zero stops the chain at that step. Nothing is written, the card
stays where it was, and the hook's stderr is the reason you see. On each side,
an instruction (`on_enter`) wins over a script (`on_enter_cmd`) — see the
[hook contract](./hook-contract.md).

## Two writers, one truth

The disk is the source of truth. The TUI writes to it, and so does anything else that can
edit a file — you in `$EDITOR`, an agent, a hook calling `hakuban progress`. Every write
goes to a `.tmp` and is renamed into place, so no reader ever sees half a card. The TUI
re-reads the folder about once a second: a card an agent moves shows up in front of you
without a refresh.

That is also why the folder works with git: commit it, and the board's history is
`git log`.
