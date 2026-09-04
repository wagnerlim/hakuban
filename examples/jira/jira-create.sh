#!/usr/bin/env bash
# jira-create.sh — on_enter hook of a bound column (hakuban, F16).
#
# Creates a Jira issue from the card that entered the column and returns the key
# for hakuban to stamp (the card becomes a read-only mirror with id = the key).
#
# Contract with the core:
#   - receives the card as JSON on STDIN (fields: id, title, notes, project, ...);
#   - HAKUBAN_FROM / HAKUBAN_TO / HAKUBAN_BOARD in the environment;
#   - emits progress on STDOUT: `N/M label` per line;
#   - emits `jira: KEY` on STDOUT when it creates (→ stamping);
#   - error = write the reason to STDERR and exit != 0 (the card does NOT move).
#
# Config (env, outside versioned plain-text — e.g. in your ~/.zshrc or a .env):
#   JIRA_URL=https://yourcompany.atlassian.net
#   JIRA_USER=you@company.com
#   JIRA_TOKEN=<api token>          # id.atlassian.com/manage-profile/security/api-tokens
#   JIRA_PROJECT=ABC                # Jira project key for this board
#   JIRA_ISSUETYPE=Task             # optional (default: Task)
#
# Requirements: bash, curl, jq. Make it executable: chmod +x jira-create.sh
set -euo pipefail

card=$(cat)                          # the card as JSON on stdin
title=$(jq -r '.title' <<<"$card")
notes=$(jq -r '.notes // ""' <<<"$card")

: "${JIRA_URL:?set JIRA_URL}"
: "${JIRA_USER:?set JIRA_USER}"
: "${JIRA_TOKEN:?set JIRA_TOKEN}"
: "${JIRA_PROJECT:?set JIRA_PROJECT}"
issuetype=${JIRA_ISSUETYPE:-Task}

echo "1/3 building payload"
# API v2 accepts a plain-text description (v3 requires ADF/JSON — more verbose).
body=$(jq -n \
  --arg p "$JIRA_PROJECT" --arg s "$title" --arg d "$notes" --arg t "$issuetype" \
  '{fields: {project: {key: $p}, summary: $s, description: $d, issuetype: {name: $t}}}')

echo "2/3 creating in Jira"
resp=$(curl -sS -u "$JIRA_USER:$JIRA_TOKEN" \
  -X POST -H "Content-Type: application/json" --data "$body" \
  "$JIRA_URL/rest/api/2/issue") || { echo "failed to call Jira" >&2; exit 1; }

key=$(jq -r '.key // empty' <<<"$resp")
if [ -z "$key" ]; then
  # Jira returns errors in .errorMessages / .errors
  echo "Jira rejected: $(jq -rc '.errorMessages + (.errors // {} | to_entries | map(.value)) | join("; ")' <<<"$resp")" >&2
  exit 1
fi

echo "3/3 linking $key"
echo "jira: $key"                    # <- the core stamps the card and it becomes a mirror
