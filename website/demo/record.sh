#!/usr/bin/env bash
# Records the homepage hero asset.
#
# Installs the demo board, its cards and its hooks into the data dir, opens ONLY that
# board (state.yml is backed up and restored on exit, so your own tabs come back),
# launches the "agent" in the background and hands the terminal to asciinema.
#
# Scripts only: the instruction path needs a paid Claude Code subscription, so it would
# not run on a clean machine — and the recording has to be reproducible.
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/../.." && pwd)
DIR=${HAKUBAN_TASK_DIR:-$HOME/.hakuban}
CAST=${1:-$REPO/website/static/hero.cast}
GIF=${CAST%.cast}.gif
SIZE=${SIZE:-148x34}   # wide enough for all four columns: narrower and the board scrolls

command -v asciinema >/dev/null || { echo "asciinema not installed: brew install asciinema" >&2; exit 1; }
command -v agg >/dev/null || { echo "agg not installed: brew install agg" >&2; exit 1; }
[ -d "$DIR" ] || { echo "data dir not found: $DIR" >&2; exit 1; }

echo "building the current binary…"
BIN=$(mktemp -d)/hakuban
(cd "$REPO" && go build -o "$BIN" ./cmd/hakuban)

echo "installing the demo board into $DIR"
mkdir -p "$DIR/boards" "$DIR/tasks" "$DIR/hooks"
cp "$HERE/boards/demo.md" "$DIR/boards/"
cp "$HERE/tasks/"DEMO-*.md "$DIR/tasks/"
cp "$HERE/hooks/"*.sh "$DIR/hooks/"
chmod +x "$DIR/hooks/build.sh" "$DIR/hooks/require-pr.sh" "$DIR/hooks/agent.sh"

# Open only the demo board. Your own state comes back on exit, even on ctrl+c.
STATE="$DIR/state.yml"
CFG="$DIR/config.yml"
BACKUP=$(mktemp)
CFGBACKUP=$(mktemp)
[ -f "$STATE" ] && cp "$STATE" "$BACKUP"
[ -f "$CFG" ] && cp "$CFG" "$CFGBACKUP"
cp "$HERE/config.yml" "$CFG"
restore() {
  [ -s "$BACKUP" ] && cp "$BACKUP" "$STATE"
  [ -s "$CFGBACKUP" ] && cp "$CFGBACKUP" "$CFG"
  rm -f "$BACKUP" "$CFGBACKUP"
  # The demo cards stay on disk on purpose — delete them by hand when you are done.
}
trap restore EXIT
printf 'open:\n    - demo\nactive: 0\n' > "$STATE"

export HAKUBAN_TASK_DIR="$DIR"
export HAKUBAN_BIN="$BIN"
export DEMO_DELAY=${DEMO_DELAY:-14}
"$DIR/hooks/agent.sh" &
AGENT=$!
trap 'kill $AGENT 2>/dev/null || true; restore' EXIT

cat <<TXT

  Recording to: $GIF   ($SIZE)

  The take, in order:
    1. let it sit two seconds — the board is the first thing the viewer reads
    2. drag DEMO-1 from To-Do to Doing (mouse), or press L on it
       → the hook runs and the progress bar fills
    3. at ~${DEMO_DELAY}s DEMO-2 moves to Doing ON ITS OWN — that is the agent,
       calling the same 'hakuban move' with no terminal attached. Do not touch anything.
    4. select DEMO-3 in Review and press L
       → the move is REFUSED and the reason shows up. This is the thesis.
    5. ctrl+t then q to quit, which ends the recording.

  Do NOT open the board list (ctrl+t b) — it lists every board on disk, yours included.

TXT
read -r -p "  enter to start, ctrl+c to abort " _

RAW=$(mktemp -d)/raw.cast
asciinema rec "$RAW" \
  --command "$BIN" \
  --window-size "$SIZE" \
  --idle-time-limit 2 \
  --output-format asciicast-v2 \
  --title "hakuban — a card crossing a column" \
  --overwrite

# The renderer leaves stale cells behind (a digit from the previous column count, box
# fragments where a card used to be), so the raw stream is replayed through a reference
# emulator and re-emitted as whole lines before it ships.
if python3 -c 'import pyte' 2>/dev/null; then
  python3 "$HERE/reframe.py" "$RAW" "$RAW.reframed"
  python3 "$HERE/publish.py" "$RAW.reframed" "$CAST"
else
  echo "pyte not installed (pip install pyte) — shipping the raw take, debris included" >&2
  python3 "$HERE/publish.py" "$RAW" "$CAST"
fi

# The page embeds the GIF, not the cast: asciinema-player's line height either breaks
# box-drawing continuity or collides adjacent rows. agg renders the take faithfully.
# The theme is `omni`, the project's default.
agg --font-size 14 --last-frame-duration 2 \
    --theme 191622,e1e1e6,000000,ed4556,67e480,e7de79,78d1e1,ff79c6,78d1e1,e1e1e6 \
    "$CAST" "$GIF"

echo
echo "done: $GIF  (source kept at $CAST)"
echo "the hero already points at /hero.gif — reload the page"
