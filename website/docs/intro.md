---
sidebar_position: 1
title: What Hakuban is
---

# What Hakuban is

A kanban in your terminal that your agents work on too — and where you declare what
happens when a card changes column.

Every card and every board is a Markdown file on disk, so an AI reads and edits all of it
the same way you do: it is just text. Columns are states, and each state is yours to
define — run a script, send an email, tag a release, hand the task off. Hakuban ships the
board; the flow is a blank page.

Reformulated for the people who want the one-sentence version: **a state machine that
lives on disk, where the transition runs an agent, with the tool scope declared in the
board file itself.**

> 看板 **kanban** — the board you are given.
> 白板 **hakuban** — the board you write.

## Watch it work

<video controls preload="metadata" style="width:100%;border:1px solid var(--border);border-radius:8px">
  <source src="/hakuban/demo.mp4" type="video/mp4" />
  Your browser cannot play this video.
</video>

Recorded by the author. No sound.

## Install

```bash
go install github.com/wagnerlim/hakuban/cmd/hakuban@latest
```

Running `hakuban` with no subcommand opens the TUI and needs a TTY. Two headless
subcommands exist so an agent can drive the belt with no terminal attached:

```bash
hakuban move <id> "<Column>"     # fires on_exit of the origin, then on_enter of the target
hakuban progress <id> <0..100>   # the percentage shown on the card
```

Column names are compared exactly and are case-sensitive.

## Honest prerequisites

The binary itself needs nothing but Go. **The agent demo does not run on a clean
machine.** It needs:

- `jq`
- `gh`, authenticated
- a paid Claude Code subscription (the instruction hook shells out to `claude -p`)
- for the author's own hooks, the [`herdr`](https://github.com/ogulcancelik/herdr) multiplexer

The pitch is *"see how it's done"*, not *"install and use"*. If you clone this and expect
the recording on the homepage to reproduce itself, you will be disappointed — and that is
on the docs, not on you, which is why it says so here.

## Where things live

```
~/.hakuban/
├── boards/<id>.md      the board: columns + what each column does
├── tasks/<ID>.md       a card: frontmatter + markdown body (the notes)
├── jira/<KEY>.md       read-only mirrors materialized by a sync hook
├── archive/            closed cards
├── config.yml          theme, language, keybindings
└── state.yml           which boards are open
```

Override the root with `HAKUBAN_TASK_DIR`.

The disk is the source of truth. The in-memory store is a cache; the TUI re-reads the disk
about once a second, so a card an agent moves appears in front of you. Writes are always
atomic (tmp + rename), because there are two legitimate writers: the TUI, and whoever
edits the file by hand — you, or an agent.

## Not in scope

These are decisions, not omissions.

- **No board marketplace.** A self-contained board is one file; GitHub already does
  hosting, search and forking.
- **No `import <url>` command.** Importing is downloading the `.md` and dropping it in the
  folder. The command would be the third-party code execution vector.
- **No GUI and no web interface.** Attempted, evaluated, cancelled.
- **Not multi-user.** No server, no auth. Local, single-player.
- **Not tracker-agnostic yet.** The board is reasonably generic; the data model is not —
  there is a `jira` field on the card struct. Do not expect Linear or GitHub Projects.

## Support

Solo project · no support · PRs may sit · MIT.

That line is expectation management, not modesty. The code is MIT; everything in
`examples/` is CC0 — copy it, change it, ship it.
