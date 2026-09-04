# Recording fixture for the hero

`record.sh` produces `website/static/hero.gif` — what the homepage hero embeds — and keeps
`hero.cast` next to it as the source. It installs `boards/demo.md`, the `DEMO-*` cards and the three hooks into your data
dir, opens only that board, and restores your `state.yml` afterwards.

Needs `asciinema`, `agg` and `pyte` (`brew install asciinema agg && pip install pyte`).

```sh
./record.sh                      # writes website/static/hero.gif (+ hero.cast)
DEMO_DELAY=20 ./record.sh        # give yourself more time before the agent moves a card
SIZE=100x28 ./record.sh          # a different terminal size
```

Scripts only, on purpose: the instruction path needs a paid Claude Code subscription, and a
hero asset that cannot be re-recorded on a clean machine is a liability.

## Why the take is post-processed

`record.sh` pipes the raw recording through `reframe.py` and `publish.py`. The TUI renders
differentially and leaves stale cells behind — the column header keeps a digit from the
previous count (`DOING (21)` for two cards), and box fragments survive where a card used
to be. Both reproduce in `agg`/avt and in asciinema-player, so they are not emulator
quirks. `reframe.py` replays the stream through a reference emulator, re-emits whole
lines, puts back column borders the renderer cleared and erases anything left below the
last card of a column; `publish.py` trims the quit keystroke and holds the final frame.

The page then embeds the **GIF**, not the cast. asciinema-player's own line height either
breaks box-drawing continuity (dashed column borders) or collides adjacent rows, and neither
is acceptable for the one asset a first-time visitor looks at. `agg` renders the same
recording faithfully, so the cast stays as the source and the GIF is what ships.

A frame in a reframed cast carries only the rows that changed, so **frames are not
independent** — cutting one out of the middle strips updates that never come again and the
result shows dashed borders. Only tail cuts are safe.

The demo cards are left on disk after the take — delete `DEMO-*.md` from `tasks/` when you
are done with them.
