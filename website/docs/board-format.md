---
sidebar_position: 3
title: Board format
---

# Board format

A board is one Markdown file with YAML frontmatter in `~/.hakuban/boards/<id>.md`. The
file name is the board id.

```yaml
---
name: My board
columns: [To-Do, Doing, Review, Done]
actions:
  Doing:
    guide: One card here at a time.
    on_enter: |
      Implement the card in the repository, run the tests,
      commit and open the PR.
  Done:
    on_enter_cmd: hooks/notify.sh
agent_tools: Read, Write, Bash
---
```

Everything below the closing `---` is free markdown — notes about the board, ignored by
the parser.

## Board fields

| field | type | what it does |
|---|---|---|
| `name` | string | board name shown on the tab |
| `key` | string | prefix for the ids of cards created on this board |
| `columns` | list | the states of the belt, in order — this is the board |
| `actions` | map | column name → its [action](#column-actions) |
| `agent_tools` | string | the only tools an instruction on this board may use, comma-separated |
| `filters` | map | named filters the TUI offers on the board |
| `sync` | string | command that materializes mirrors from an external tracker |
| `sync_on_open` | bool | run `sync` when the board is opened |
| `comments` | bool | pull the issue's comments into the mirror |
| `buttons` | list | custom actions in the card footer |

## Column actions

Each key of `actions` is a **column name**, matched exactly and case-sensitively. Its
value:

| field | what it does |
|---|---|
| `guide` | a note for the column. **Read, never executed** — it is context for whoever (or whatever) decides the next move |
| `on_enter` / `on_exit` | **instruction** — prose run by a headless agent, scoped by `agent_tools` |
| `on_enter_cmd` / `on_exit_cmd` | **script** — shell, card as JSON on stdin. See the [hook contract](./hook-contract.md) |
| `jql` | makes the column a mirror of an external query; the `sync` command materializes it |
| `status` | the status in the tracker this column represents, exported to hooks as `HAKUBAN_STATUS` |
| `percent` | show the card's `progress` as a bar in this column |
| `loader` | label for the progress indicator while the action runs |
| `use_filters` | which filters apply to this column |

An instruction **shadows** the script on the same side of the same column: `on_enter` is
checked before `on_enter_cmd`, so adding an instruction where a script already exists
silently stops the script from running. If you add one, it has to absorb what the script
did — or call it.

A column with no `jql` is local: it exists only in Hakuban, with no queryable
counterpart in the tracker.

### Instruction interpolation

An instruction is a Go template over the card's fields. Available:
`{{.id}}`, `{{.title}}`, `{{.jira}}`, `{{.status}}`, `{{.priority}}`, `{{.project}}`,
`{{.notes}}`. A broken template falls back to the raw text — the prose still makes sense.

The instruction can also be a path to a `.md` file, which is the readable option once it
grows past a couple of lines.

## The card

```yaml
---
id: PES-12
title: Ship the export command
status: pending          # pending | done
priority: high           # low | normal | high
progress: 40             # 0..100, shown on the card
project: hakuban
tags: [cli]
due: 2026-09-10
jira: PES-12             # set when a hook stamped an external key
---

Whatever context matters. This body is the card's **notes**, and it is what a
hook or an agent reads.
```

The notes are the payload. A hook that needs a parameter — a repository, a time
estimate — reads it from here, and refuses the move when it is missing. That refusal is
the feature: see [the contract](./hook-contract.md#exit-code).
