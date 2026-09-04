#!/usr/bin/env bash
# Entering Doing: hands the card to a detached worker and returns AT ONCE, which is what
# lets the move commit — the card lands in Doing and the worker reports from there. Same
# shape as the author's real board, where on_enter launches a detached agent.
#
# stdout must not stay open or the core would wait on the grandchild, so the worker's
# output goes to /dev/null.
set -euo pipefail

id=$(cat | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
[ -n "$id" ] || { echo "no card id on stdin" >&2; exit 1; }

nohup "$HAKUBAN_DIR/hooks/work.sh" "$id" >/dev/null 2>&1 &
echo "1/1 dispatched $id"
