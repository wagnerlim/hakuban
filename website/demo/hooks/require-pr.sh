#!/usr/bin/env bash
# Leaving Review: refuses the move when the notes carry no PR. This is the gate that keeps
# the board from lying about what shipped — and the abort is the point of the recording.
set -euo pipefail

card=$(cat)
case "$card" in
  *'PR: '*) ;;
  *) echo "no 'PR: <url>' line in the notes" >&2; exit 1 ;;
esac
echo "1/1 PR linked"
