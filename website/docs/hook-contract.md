---
sidebar_position: 4
title: Hook contract
---

# Hook contract

Two fields fire on the same transition, so they look interchangeable. They are not.

|  | `script` · `_cmd` | `instruction` · `on_enter` |
|---|---|---|
| behavior | deterministic — you read exactly what it does | non-deterministic — you don't know what it will do |
| **capability** | **unlimited** — there is no permission model in `bash` | **bounded** — it fits on one line: `agent_tools` |
| review scales? | no — nobody reviews 93 invocations of `jq` | yes — a capability ceiling is a short list |
| portability | ties you to a filesystem, a binary, a `$HOME` | ties you to nothing: it describes intent |

**Project doctrine:** the instruction is the format; `_cmd` is debt — an escape hatch for
exact control, at the cost of a board nobody else can run.

For sharing a board, what matters is not what the file *says* — it is what it *can do*.

## The script contract

Hakuban runs `sh -c "<the command line>"` with the data dir as the working directory, so
a relative path (`hooks/notify.sh`) resolves wherever the binary was launched from. It is
one command line, so **hooks are language-agnostic**: bash, Python, Node, Ruby, a compiled
binary — anything that honors the four channels below.

### stdin

The card and the transition, as one JSON object:

```json
{
  "id": "PES-12",
  "title": "Ship the export command",
  "status": "pending",
  "priority": "high",
  "project": "hakuban",
  "tags": ["cli"],
  "jira": "PES-12",
  "notes": "Repo: wagnerlim/hakuban\n",
  "from": "Ready",
  "to": "Doing",
  "board": "my-board"
}
```

### stdout

Two control lines, everything else ignored:

| line | effect |
|---|---|
| `N label` | `N` is 0..100 — sets the progress bar and its label |
| `N/M label` | step `N` of `M` — same bar, computed as `N*100/M` |
| `jira: KEY` | stamps `KEY` onto the card as its external key; the card becomes a mirror |

A line that does not start with a number is not a progress line, so a stray log on stdout
never turns into a bar.

### stderr

Buffered. **The last non-empty line becomes the error message** shown to whoever moved the
card. Write the reason there, not on stdout.

### exit code

`0` commits the move. Anything else **aborts it**: the card stays where it was and the
reason from stderr appears. Same whether a human dragged it or an agent called
`hakuban move`. A failing hook is usually a missing piece of data, not a transient error —
read the message before retrying the move.

### environment

On top of the process environment:

| variable | value |
|---|---|
| `HAKUBAN_FROM` | column the card left |
| `HAKUBAN_TO` | column it is entering |
| `HAKUBAN_BOARD` | board id |
| `HAKUBAN_STATUS` | the `status:` of the destination column |
| `HAKUBAN_DIR` | the data dir (`~/.hakuban` by default) |
| `HAKUBAN_TASK_BIN` | path to the running binary, for a hook that wants to call back |

## The same hook, twice

Bash:

```bash
#!/usr/bin/env bash
# Refuses the move unless the notes carry a repository.
set -euo pipefail

card=$(cat)
repo=$(jq -r '.notes' <<<"$card" | sed -n 's/^Repo: *//p' | head -1)

if [ -z "$repo" ]; then
  echo "no 'Repo: owner/name' line in the notes" >&2
  exit 1
fi

echo "1/2 cloning $repo"
gh repo clone "$repo" "$(mktemp -d)" >/dev/null 2>&1
echo "2/2 ready"
```

Python — same contract, no `jq`:

```python
#!/usr/bin/env python3
"""Refuses the move unless the notes carry a repository."""
import json, re, sys

card = json.load(sys.stdin)
match = re.search(r"^Repo: *(\S+)", card.get("notes", ""), re.M)

if not match:
    print("no 'Repo: owner/name' line in the notes", file=sys.stderr)
    sys.exit(1)

print(f"1/2 cloning {match.group(1)}", flush=True)
# ... do the work ...
print("2/2 ready", flush=True)
```

Wire either one the same way:

```yaml
actions:
  Doing:
    on_enter_cmd: hooks/require-repo.py
```

## The instruction contract

An instruction has no stdin, stdout or exit code to honor — it is prose. What it has is a
ceiling:

```yaml
actions:
  Done:
    on_enter: |
      Write a short update from the card notes and send it
      to the address in the frontmatter.
agent_tools: Read, mcp__gmail__send
```

`agent_tools` is the whole permission model, and it is one line. The agent runs headless,
with a neutral working directory so it cannot wander into whatever project you happen to
be sitting in. It fails the same way a script does: the move is aborted and the reason
shows up.

## Windows

The TUI and the board format work. Instruction hooks work. **Script hooks do not** —
`sh` does not exist there, and Hakuban does not ship a shim. If you are on Windows, the
instruction is not the preferred format, it is the only one.
