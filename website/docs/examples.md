---
sidebar_position: 5
title: Examples
---

# Examples

Examples, not features. **None of these ship with the tool.** The author's own board hands
tasks to an agent that opens PRs — one flow among N, and yours has no reason to look like
it.

Everything on this page is CC0. Copy it, change it, ship it.

Each block is a whole board: save it as `~/.hakuban/boards/<id>.md` and it runs.

## Email the client

An instruction, and the only capability it gets is reading and sending mail.

```yaml
---
name: Client work
columns: [To-Do, Doing, Done]
actions:
  Done:
    on_enter: |
      Write a short update from the card notes, in the client's language,
      and send it to the address in the frontmatter. Do not invent status
      that is not in the notes.
agent_tools: Read, mcp__gmail__send
---
```

## Split the card

The board reshapes its own work. `Refining` is a local column: it exists only here.

```yaml
---
name: Intake
columns: [Inbox, Refining, Ready, Doing, Done]
actions:
  Refining:
    guide: Nothing leaves here with an open question in the notes.
    on_exit: |
      If the notes describe more than one deliverable, create one card per
      deliverable and leave {{.id}} as the parent. If they describe one,
      do nothing.
agent_tools: Read, Write
---
```

## Run the checklist

A script, because "the tests pass" should not be a matter of opinion. Non-zero exit and
the card never leaves `Doing`.

```yaml
---
name: Library
columns: [To-Do, Doing, Review, Done]
actions:
  Doing:
    on_exit_cmd: hooks/checklist.sh
---
```

```bash
#!/usr/bin/env bash
# hooks/checklist.sh — tests, linter, coverage floor.
set -euo pipefail
cd "$(jq -r '.notes' | sed -n 's/^Path: *//p' | head -1)"

echo "1/3 tests"
go test ./... >/dev/null || { echo "tests failed" >&2; exit 1; }
echo "2/3 vet"
go vet ./... || { echo "vet failed" >&2; exit 1; }
echo "3/3 coverage"
pct=$(go test -cover ./... | grep -o '[0-9.]*%' | tr -d '%' | sort -n | head -1)
awk "BEGIN{exit !($pct < 70)}" && { echo "coverage $pct% below 70%" >&2; exit 1; }
```

## Hand it to an agent

The author's board. Card → implementation → approved PR in 9m18s on 2026-08-29 — real,
and still just an example. `Ready → Doing` is the gate, and it is a human dragging a card.

```yaml
---
name: Project board
columns: [To-Do, Refining, Ready, Doing, Validate, Review, Done]
actions:
  Ready:
    guide: To start work, move the card forward. Needs 'Repo: owner/name' in the notes.
  Doing:
    loader: implementing
    percent: true
    on_enter: |
      Clone the repository named in the notes of {{.id}}, implement {{.title}},
      run the tests until they pass, and open the PR. Stamp progress with
      `hakuban progress` as you go. Move the card to Validate when the PR is up.
  Validate:
    on_exit_cmd: hooks/require-pr.sh
agent_tools: Read, Write, Edit, Bash
---
```

`hooks/require-pr.sh` refuses `Validate → Review` when no PR exists for the card — the
gate that keeps the board from lying about what shipped.

## Mirroring an external tracker

The binary speaks no HTTP. A hook does. The repository's
[`examples/jira/`](https://github.com/wagnerlim/hakuban/tree/main/examples/jira) has the
working set: a `sync` that materializes read-only mirrors from a JQL, a hook that creates
the issue and reports its key back with `jira: KEY`, and one that transitions the issue to
the status a column represents.

```yaml
---
name: Team
key: TEAM
columns: [Backlog, Doing, Done]
sync: hooks/jira-sync.sh
sync_on_open: true
actions:
  Backlog:
    jql: project = TEAM AND status = "To Do"
  Doing:
    status: In Progress
    on_enter_cmd: hooks/jira-transition.sh
  Done:
    status: Done
    jql: project = TEAM AND status = Done
    on_enter_cmd: hooks/jira-transition.sh
---
```

Integration is composition. That is a principle, not a gap.
