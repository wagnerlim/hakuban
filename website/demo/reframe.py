#!/usr/bin/env python3
"""Rewrites a cast so every frame is a full repaint of the lines that changed.

Why: the recorded stream is a sequence of partial line updates, and replaying it outside
the terminal it was captured in leaves visible debris — a digit from the previous column
count (`DOING (21)` for two cards) and box fragments where a card used to be. The board
does not look like that in a real terminal; the artifacts belong to the playback path.

Replaying through a reference emulator and re-emitting whole lines makes the recording
independent of stale cells, and `despeckle` clears the fragments that survive that.

CAUTION: a frame here carries only the rows that CHANGED. Frames are therefore NOT
independent — dropping one in the middle loses those rows for good and the screen keeps
older content (it shows up as dashed borders). Only a tail cut is safe; see publish.py.

Usage: python3 reframe.py in.cast out.cast
"""
import json, io, sys
import pyte

RESET = "\x1b[0m"

NAMED = {"black": 0, "red": 1, "green": 2, "brown": 3, "blue": 4,
         "magenta": 5, "cyan": 6, "white": 7}


def sgr(char) -> str:
    """The escape prefix for one cell's attributes."""
    out = ["0"]
    if char.bold:
        out.append("1")
    if char.italics:
        out.append("3")
    if char.underscore:
        out.append("4")
    if char.reverse:
        out.append("7")
    for color, fg in ((char.fg, True), (char.bg, False)):
        if color == "default":
            continue
        base = "38" if fg else "48"
        if color in NAMED:
            out.append(str((30 if fg else 40) + NAMED[color]))
        elif len(color) == 6:
            r, g, b = (int(color[i:i + 2], 16) for i in (0, 2, 4))
            out.append(f"{base};2;{r};{g};{b}")
    return "\x1b[" + ";".join(out) + "m"


H, V = "─━", "│┃"
TL, TR, BL, BR = "╭┌", "╮┐", "╰└", "╯┘"
BOX = H + V + TL + TR + BL + BR + "├┤┬┴┼"


def glyph(screen, y: int, x: int, over=None) -> str:
    if not (0 <= y < screen.lines and 0 <= x < screen.columns):
        return " "
    cell = (over or {}).get((y, x)) or screen.buffer[y][x]
    return cell.data or " "


def rectangles(screen, over=None):
    """Every closed box on screen, as the set of cells its outline occupies.

    A box edge always runs corner to corner, so anything left over from a box that is
    gone fails this walk — which is exactly what has to be erased.
    """
    cells = set()
    for y in range(screen.lines):
        for x in range(screen.columns):
            if glyph(screen, y, x, over) not in TL:
                continue
            right = next((x2 for x2 in range(x + 1, screen.columns)
                          if glyph(screen, y, x2, over) in TR
                          and all(glyph(screen, y, i, over) in H for i in range(x + 1, x2))), None)
            bottom = next((y2 for y2 in range(y + 1, screen.lines)
                           if glyph(screen, y2, x, over) in BL
                           and all(glyph(screen, i, x, over) in V for i in range(y + 1, y2))), None)
            if right is None or bottom is None:
                continue
            if glyph(screen, bottom, right, over) not in BR:
                continue
            if not all(glyph(screen, bottom, i, over) in H for i in range(x + 1, right)):
                continue
            if not all(glyph(screen, i, right, over) in V for i in range(y + 1, bottom)):
                continue
            for i in range(x, right + 1):
                cells.add((y, i))
                cells.add((bottom, i))
            for i in range(y, bottom + 1):
                cells.add((i, x))
                cells.add((i, right))
    return cells


def chrome(screen, min_height: int = 10):
    """The column frames: the tall boxes of the opening frame, which never change.

    Kept as {(y, x): Char} so a later frame that erased part of a column border — the
    renderer clears from the wrong column when a card leaves — can have it put back.
    """
    out = {}
    for y in range(screen.lines):
        for x in range(screen.columns):
            if glyph(screen, y, x) not in TL:
                continue
            bottom = next((y2 for y2 in range(y + 1, screen.lines)
                           if glyph(screen, y2, x) in BL
                           and all(glyph(screen, i, x) in V for i in range(y + 1, y2))), None)
            if bottom is None or bottom - y < min_height:
                continue
            right = next((x2 for x2 in range(x + 1, screen.columns)
                          if glyph(screen, y, x2) in TR
                          and all(glyph(screen, y, i) in H for i in range(x + 1, x2))), None)
            if right is None:
                continue
            for i in range(x, right + 1):
                out[(y, i)] = screen.buffer[y][i]
                out[(bottom, i)] = screen.buffer[bottom][i]
            for i in range(y, bottom + 1):
                out[(i, x)] = screen.buffer[i][x]
                out[(i, right)] = screen.buffer[i][right]
    return out


def columns(screen, over=None):
    """The column frames as (top, left, bottom, right)."""
    out = []
    for y in range(screen.lines):
        for x in range(screen.columns):
            if glyph(screen, y, x, over) not in TL:
                continue
            bottom = next((y2 for y2 in range(y + 1, screen.lines)
                           if glyph(screen, y2, x, over) in BL
                           and all(glyph(screen, i, x, over) in V for i in range(y + 1, y2))), None)
            if bottom is None or bottom - y < 10:
                continue
            right = next((x2 for x2 in range(x + 1, screen.columns)
                          if glyph(screen, y, x2, over) in TR
                          and all(glyph(screen, y, i, over) in H for i in range(x + 1, x2))), None)
            if right is not None:
                out.append((y, x, bottom, right))
    return out


def card_boxes(keep, top, left, bottom, right):
    """The closed boxes nested inside one column, as (top, left, bottom, right)."""
    corners = sorted((y, x) for (y, x) in keep if top < y < bottom and left < x < right)
    seen, out = set(), []
    for (y, x) in corners:
        if (y, x) in seen:
            continue
        ys = [y2 for (y2, x2) in corners if x2 == x]
        xs = [x2 for (y2, x2) in corners if y2 == y]
        if len(ys) < 2 or len(xs) < 2:
            continue
        b, r = max(ys), max(xs)
        if (b, r) not in keep:
            continue
        out.append((y, x, b, r))
        for yy in range(y, b + 1):
            for xx in range(x, r + 1):
                seen.add((yy, xx))
    return out


def repair(screen, frame_chrome):
    """Per-cell overrides that hide box debris and restore erased column frames.

    Returned as an overlay instead of written into the emulator: the recorded stream is a
    diff, so every later update assumes the screen holds exactly what the app last drew.
    Editing the buffer breaks that chain — a box erased while still half-drawn never gets
    completed, because the app believes it already painted those cells.
    """
    over = {}
    for (y, x), char in frame_chrome.items():
        if glyph(screen, y, x) == " ":
            over[(y, x)] = char
    # The walk has to see the restored frames, or a column whose border was erased reads
    # as an open box and its whole outline gets classified as debris.
    keep = rectangles(screen, over)
    blank = None

    def erase(y, x):
        nonlocal blank
        if blank is None:
            blank = screen.buffer[y][x]._replace(data=" ", fg="default", bg="default")
        over[(y, x)] = blank

    for y in range(screen.lines):
        for x in range(screen.columns):
            if glyph(screen, y, x) in BOX and (y, x) not in keep:
                erase(y, x)

    # Debris is not only box edges: a card that moved away leaves its text behind too (a
    # tag badge, a title line). Inside a column, content lives only in card boxes, so
    # everything below the last card box in that column has to be empty.
    for (top, left, bottom, right) in columns(screen, over):
        cards = [r for r in card_boxes(keep, top, left, bottom, right)]
        floor = max((c[2] for c in cards), default=top + 1)
        for y in range(floor + 1, bottom):
            for x in range(left + 1, right):
                if glyph(screen, y, x, over) != " ":
                    erase(y, x)
    return over


def render_line(screen, y: int, over=None) -> str:
    """One screen row as text with attributes, trailing blanks trimmed."""
    over = over or {}
    row = screen.buffer[y]
    cells = [over.get((y, x), row[x]) for x in range(screen.columns)]
    # Trim the trailing run of plain blanks — the caller erases the rest of the line.
    end = screen.columns
    while end > 0:
        c = cells[end - 1]
        if c.data.strip() or c.bg != "default" or c.reverse:
            break
        end -= 1
    out, last = [], None
    for x in range(end):
        c = cells[x]
        code = sgr(c)
        if code != last:
            out.append(code)
            last = code
        out.append(c.data or " ")
    return "".join(out) + RESET


def main(src: str, dst: str) -> int:
    lines = io.open(src, encoding="utf-8").read().splitlines()
    header = json.loads(lines[0])
    events = [json.loads(l) for l in lines[1:] if json.loads(l)[1] == "o"]

    screen = pyte.Screen(header["width"], header["height"])
    stream = pyte.Stream(screen)

    prev = [None] * header["height"]
    out = []
    frame_chrome = {}
    for t, _, data in events:
        stream.feed(data)
        if not frame_chrome:
            # The app writes its opening frame in several chunks, so the reference is the
            # first moment the column frames actually exist on screen.
            frame_chrome = chrome(screen)
            if not frame_chrome:
                continue
        over = repair(screen, frame_chrome)
        chunks = []
        for y in range(header["height"]):
            line = render_line(screen, y, over)
            if line == prev[y]:
                continue
            prev[y] = line
            # position, erase the whole row, then paint it
            chunks.append(f"\x1b[{y + 1};1H\x1b[2K{line}")
        if chunks:
            out.append([t, "o", "".join(chunks)])

    # Hide the cursor: the take has no prompt, and a blinking block reads as a glitch.
    if out:
        out[0][2] = "\x1b[?25l\x1b[2J" + out[0][2]

    with io.open(dst, "w", encoding="utf-8") as f:
        f.write(json.dumps(header, ensure_ascii=False) + "\n")
        for e in out:
            f.write(json.dumps(e, ensure_ascii=False) + "\n")
    print(f"{len(events)} eventos -> {len(out)} frames cheios")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1], sys.argv[2]))
