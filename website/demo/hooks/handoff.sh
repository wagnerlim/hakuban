#!/usr/bin/env bash
# Leaving Doing: the worker is finished, so this is the handoff to review. Runs in the
# foreground on purpose — the live loader belongs to whoever starts the transition, and it
# is the only thing that ever shows a full bar: the persisted percentage is not drawn at
# 100 (a full bar says nothing once the card has already moved on).
set -euo pipefail

cat > /dev/null   # the card arrives on stdin; this hook does not need it

echo "1/3 tests green";  sleep 0.7
echo "2/3 PR opened";     sleep 0.7
echo "3/3 handing off";   sleep 0.5
