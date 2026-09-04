#!/usr/bin/env bash
# jira-sync.sh — the board's `sync` command (hakuban, F16).
#
# Runs the bound column's JQL and materializes read-only mirrors in
# $HAKUBAN_DIR/jira/<KEY>.md. hakuban re-reads the disk and shows the cards. Since
# the .md id is the issue key, running again OVERWRITES (no duplicates), and matches
# the cards already linked by on_enter (stamping).
#
# Contract with the core (all via environment):
#   HAKUBAN_JQL       the column's JQL
#   HAKUBAN_COLUMN    the column name (becomes the mirror's `status`)
#   HAKUBAN_BOARD     the board id (becomes the `project`)
#   HAKUBAN_DIR       the data dir (write into $HAKUBAN_DIR/jira)
#   HAKUBAN_COMMENTS  "1" when the board opts in (board `comments: true`) → also pull the
#                   issue comments into the mirror's `comments:` list (author/when/body)
#   + JIRA_URL / JIRA_USER / JIRA_TOKEN (see jira-create.sh)
# Emits progress `N/M label` on STDOUT; error on STDERR + exit != 0.
#
# Requirements: bash, curl, jq. Make it executable: chmod +x jira-sync.sh
set -euo pipefail

: "${JIRA_URL:?set JIRA_URL}"
: "${JIRA_USER:?set JIRA_USER}"
: "${JIRA_TOKEN:?set JIRA_TOKEN}"
: "${HAKUBAN_DIR:?}"; : "${HAKUBAN_JQL:?}"; : "${HAKUBAN_COLUMN:?}"; : "${HAKUBAN_BOARD:?}"

dest="$HAKUBAN_DIR/jira"
mkdir -p "$dest"

echo "1/2 querying Jira"
# current search endpoint (the old GET /rest/api/2/search was deprecated):
# POST /rest/api/3/search/jql with a JSON body; returns {issues, nextPageToken, isLast}.
fields='["summary","priority"]'
[ "${HAKUBAN_COMMENTS:-0}" = "1" ] && fields='["summary","priority","comment"]'
body=$(jq -n --arg jql "$HAKUBAN_JQL" --argjson f "$fields" '{jql: $jql, fields: $f, maxResults: 200}')
resp=$(curl -sS -u "$JIRA_USER:$JIRA_TOKEN" \
  -X POST -H "Content-Type: application/json" --data "$body" \
  "$JIRA_URL/rest/api/3/search/jql") || { echo "JQL search failed" >&2; exit 1; }

if ! n=$(jq -e '.issues | length' <<<"$resp" 2>/dev/null); then
  echo "unexpected response from Jira: $(jq -rc '.errorMessages // .' <<<"$resp")" >&2
  exit 1
fi

# write one mirror per issue; summary comes out as a quoted YAML scalar (jq without -r),
# which already escapes colons and quotes in the title.
i=0
while IFS= read -r issue; do
  i=$((i + 1))
  key=$(jq -r '.key' <<<"$issue")
  summary=$(jq '.fields.summary' <<<"$issue")   # quoted → YAML-safe
  prio=$(jq -r '.fields.priority.name // "normal" | ascii_downcase' <<<"$issue")
  fm="---
id: $key
title: $summary
status: $HAKUBAN_COLUMN
priority: $prio
project: $HAKUBAN_BOARD
jira: $key
source: jira"
  # comments (opt-in): map each Jira comment to author/when/body. Bodies come as ADF
  # (a JSON tree); we flatten its text nodes best-effort. tojson quotes every value →
  # valid YAML (YAML is a superset of JSON flow scalars), safe with colons/quotes/newlines.
  if [ "${HAKUBAN_COMMENTS:-0}" = "1" ]; then
    cy=$(jq -r '
      (.fields.comment.comments // [])
      | map("  - author: " + (.author.displayName // "?" | tojson)
            + "\n    when: " + (.created[0:10] // "" | tojson)
            + "\n    body: " + ([.body | .. | .text? // empty] | join(" ") | tojson))
      | if length > 0 then "comments:\n" + join("\n") else "" end' <<<"$issue")
    [ -n "$cy" ] && fm="$fm
$cy"
  fi
  printf '%s\n---\n' "$fm" > "$dest/$key.md"
  echo "2/2 $key ($i/$n)"
done < <(jq -c '.issues[]' <<<"$resp")

echo "2/2 $n issues synced"
