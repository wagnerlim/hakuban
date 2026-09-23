# Jira integration (example) — hakuban F16

**Example** scripts that connect a hakuban board to a Jira project **by
composition**: the binary speaks no HTTP, it just fires these scripts (see
[`docs/objectives.md`](../../docs/objectives.md) → F16/F22). Adapt at will.

- **`jira-sync.sh`** — *pull*. Runs a bound column's JQL and materializes
  read-only mirrors in `~/.hakuban/jira/<KEY>.md`.
- **`jira-create.sh`** — *on_enter*. When you move a local card to a Jira column,
  creates the issue and returns the key; hakuban stamps the card (it becomes a mirror).
- **`jira-transition.sh`** — *on_enter/on_exit*. When you drag a mirror to a
  column, transitions the issue to the status the column represents (`HAKUBAN_STATUS`).

## Requirements

`bash`, `curl`, `jq`. Make the scripts executable:

```sh
chmod +x jira-create.sh jira-sync.sh
```

## Credentials (outside versioned plain-text)

In your shell (or a `.env` you `source`) — **never** in a `.md`:

```sh
export JIRA_URL=https://yourcompany.atlassian.net
export JIRA_USER=you@company.com
export JIRA_TOKEN=…            # id.atlassian.com/manage-profile/security/api-tokens
export JIRA_PROJECT=ABC        # Jira project key for this board
```

## Wiring it to the board

Edit `~/.hakuban/boards/<id>.md` (or the paths where you stored the scripts):

```yaml
---
name: Pessoal
key: PES
columns: [DRAFTS, TO-DO, DOING, DONE]
sync: "~/.hakuban/hooks/jira-sync.sh"
actions:
  TO-DO:
    jql: "project = ABC AND status = 'To Do'"
    on_enter: "~/.hakuban/hooks/jira-create.sh"
  DOING:
    jql: "project = ABC AND status = 'In Progress'"
    status: "In Progress"                       # transition target (HAKUBAN_STATUS)
    on_enter: "~/.hakuban/hooks/jira-transition.sh"
  DONE:
    jql: "project = ABC AND status = 'Done'"
    status: "Done"
    on_enter: "~/.hakuban/hooks/jira-transition.sh"
---
```

- `DRAFTS` stays local (your `.md`). Moving a card DRAFTS → TO-DO creates it in Jira.
- Dragging a mirror to DOING/DONE transitions the issue in Jira (`status:` = the target).
- The **⟳ sync** button in the footer (or the `y` key) pulls the column's JQL.

## Contract (to write your own)

**Progress** — each line on `stdout`: `N` (0–100) or `N/M`, with an optional label
(`echo "2/3 creating issue"`). The last `stderr` line = the reason, shown in the `:(`.
Exit `0` = success, `≠0` = failure.

**Stamping** — `on_enter` emits `jira: KEY` on `stdout` when creating; the core
recreates the card with id = the key (it becomes a mirror), so `sync` matches by id
and doesn't duplicate.

**Environment** — `on_enter`/`on_exit`: card as `--json` on `stdin` +
`HAKUBAN_FROM`/`HAKUBAN_TO`/`HAKUBAN_BOARD` + `HAKUBAN_STATUS` (the destination column's
`status:` — the transition target). `sync`:
`HAKUBAN_JQL`/`HAKUBAN_COLUMN`/`HAKUBAN_BOARD`/`HAKUBAN_DIR` + `HAKUBAN_COMMENTS`.

**Comments** — set `comments: true` on the board to pull each issue's comments. The
core just passes `HAKUBAN_COMMENTS=1` to `sync`; the script fetches them and writes a
`comments:` list (`author`/`when`/`body`) onto the mirror. The card detail then shows
them as cards below the description (they scroll; links render).

**Open in Jira** — set `issue_url: 'https://<you>.atlassian.net/browse/{key}'` on the
board. In the card detail the id itself (e.g. `ACME-1631 ↗`) is clickable — a plain click
opens the issue in the browser (via `open`/`xdg-open`); `{key}` is replaced by the card's
`jira:` key. No script change needed.

**Column guide** — set `guide:` on a column's action with a declarative note of what the
column is and how work advances from it (e.g. `guide: "to refine a card, move it forward from
here"`). It is never executed — unlike the `on_enter`/`on_exit` intents, which fire as a
consequence of a move — it is standing context an agent reads from the board file to *decide*
what to do (a card sitting in To-do carries no hint that "refine" means "move it on"). The core
only parses and preserves it; agnostic, no script needed.

> The mirrors in `jira/` are derived cache — ignore them in your data dir's git
> (`echo 'jira/' >> ~/.hakuban/.gitignore`).

> Search uses the current `POST /rest/api/3/search/jql` endpoint (the old
> `GET /rest/api/2/search` was deprecated by Jira Cloud). Create uses
> `POST /rest/api/2/issue` (plain-text description; v3 requires ADF/JSON).
