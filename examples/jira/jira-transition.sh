#!/usr/bin/env bash
# jira-transition.sh — on_enter/on_exit hook of a bound column (hakuban, F16).
#
# Transitions the Jira issue to the status the destination column represents. The
# core passes that status in HAKUBAN_STATUS (the action's `status:` field on the
# board), so the script doesn't need to know the column→status map — it just finds
# the transition whose target matches HAKUBAN_STATUS and applies it.
#
# Contract:
#   - card as --json on STDIN (uses .jira = the issue key);
#   - HAKUBAN_STATUS = target status in Jira (e.g. "Concluído");
#   - HAKUBAN_FROM / HAKUBAN_TO / HAKUBAN_BOARD in the environment;
#   - progress `N/M label` on STDOUT; reason on STDERR + exit != 0 on failure.
#   - does not change the id → emits no `jira:` (the core only swaps the local status to the column).
#
# Env: JIRA_URL / JIRA_USER / JIRA_TOKEN. Requirements: bash, curl, jq.
set -euo pipefail

card=$(cat)
key=$(jq -r '.jira // empty' <<<"$card")

: "${JIRA_URL:?set JIRA_URL}"
: "${JIRA_USER:?set JIRA_USER}"
: "${JIRA_TOKEN:?set JIRA_TOKEN}"
: "${HAKUBAN_STATUS:?column has no \`status:\` on the board — nothing to transition}"
[ -n "$key" ] || { echo "local card without a Jira key (move only after linking)" >&2; exit 1; }

api="$JIRA_URL/rest/api/3/issue/$key/transitions"

echo "1/3 reading transitions for $key"
tr=$(curl -sS -u "$JIRA_USER:$JIRA_TOKEN" "$api") || { echo "failed to read transitions" >&2; exit 1; }

echo "2/3 looking for → $HAKUBAN_STATUS"
id=$(jq -r --arg s "$HAKUBAN_STATUS" 'first(.transitions[] | select(.to.name == $s) | .id) // empty' <<<"$tr")
if [ -z "$id" ]; then
  echo "no transition to \"$HAKUBAN_STATUS\" (available: $(jq -r '[.transitions[].to.name] | join(", ")' <<<"$tr"))" >&2
  exit 1
fi

echo "3/3 transitioning $key → $HAKUBAN_STATUS"
curl -sS -u "$JIRA_USER:$JIRA_TOKEN" -X POST -H "Content-Type: application/json" \
  --data "$(jq -n --arg id "$id" '{transition: {id: $id}}')" "$api" \
  || { echo "failed to apply the transition" >&2; exit 1; }
