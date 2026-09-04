#!/usr/bin/env bash
# The "agent": it drives the belt with no terminal attached, using the same headless
# subcommand any hook or CI job would. The TUI re-reads the disk about once a second, so
# the card moves in front of the viewer without anyone touching the keyboard.
set -euo pipefail

sleep "${DEMO_DELAY:-14}"
"$HAKUBAN_BIN" move DEMO-2 Doing >/dev/null 2>&1 || true
