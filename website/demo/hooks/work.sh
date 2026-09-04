#!/usr/bin/env bash
# The detached worker: stamps the card's percentage while it works. The TUI re-reads the
# disk about once a second, so the segmented bar fills ON THE CARD, in Doing — no terminal
# attached, nobody watching this process.
#
# A real worker would clone, build and open a PR here. This one only sleeps: the recording
# has to be reproducible on a machine with no gh, no network and no Claude subscription.
set -euo pipefail

id=$1
bin=${HAKUBAN_TASK_BIN:-hakuban}

for pct in 20 40 55 70 85; do
  sleep 1.0
  "$bin" progress "$id" "$pct" >/dev/null 2>&1 || true
done
