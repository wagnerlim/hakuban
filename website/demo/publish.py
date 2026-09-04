#!/usr/bin/env python3
"""Trims a reframed cast into the asset the hero embeds.

Two operations, both tail-only, because a frame carries just the rows that changed and
cutting one out of the middle would strip updates that never come again:

  - drop everything from the quit keystroke on (the PREFIX help bar, the teardown and the
    blank screen a viewer would otherwise loop through);
  - hold the final frame for a beat before the loop restarts.

Usage: python3 publish.py in.cast out.cast
"""
import json, io, os, sys

HOLD = 2.0


def main(src: str, dst: str) -> int:
    lines = io.open(src, encoding="utf-8").read().splitlines()
    header = json.loads(lines[0])
    for key in ("command", "env"):        # they carry local paths, and playback ignores them
        header.pop(key, None)
    events = [json.loads(l) for l in lines[1:] if json.loads(l)[1] == "o"]

    def teardown(data: str) -> bool:
        return "PREFIX" in data or "\x1b[?1049l" in data

    cut = next((i for i, e in enumerate(events) if e[0] > 5 and teardown(e[2])), len(events))
    kept = events[:cut]
    if not kept:
        print("nothing left after the cut", file=sys.stderr)
        return 1
    kept.append([kept[-1][0] + HOLD, "o", ""])

    with io.open(dst, "w", encoding="utf-8") as f:
        f.write(json.dumps(header, ensure_ascii=False) + "\n")
        for e in kept:
            f.write(json.dumps(e, ensure_ascii=False) + "\n")
    print(f"{len(events)} -> {len(kept)} frames | {kept[-1][0]:.1f}s | {os.path.getsize(dst)} bytes")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1], sys.argv[2]))
