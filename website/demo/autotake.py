#!/usr/bin/env python3
"""Machine-driven take of the hero recording.

asciinema records whatever this script writes to stdout. So it opens a PTY of its own,
runs the TUI inside it, relays the bytes both ways, and types the take on a schedule —
which means the moves are initiated BY THE TUI, the way a person would. That is what makes
the animated segmented bar and the refused-move message appear at all: both belong to
whoever starts the transition, and a headless `hakuban move` shows neither.

Usage: HAKUBAN_TASK_DIR=... HAKUBAN_BIN=... python3 autotake.py
"""
import fcntl, os, pty, select, signal, struct, subprocess, sys, termios, time

BIN = os.environ["HAKUBAN_BIN"]
DIR = os.environ["HAKUBAN_TASK_DIR"]
COLS, ROWS = (int(v) for v in os.environ.get("SIZE", "148x34").split("x"))

CTRL_T = b"\x14"

# A real terminal answers capability queries. Nothing does in a headless capture, and the
# TUI then falls back to computing widths itself — which is how the recording ends up with
# cells one column off and box fragments the renderer never reclaims. Answering the way a
# modern terminal does keeps it on its normal path.
QUERIES = [
    (b"\x1b[?2026$p", b"\x1b[?2026;2$y"),   # synchronized output: supported, currently off
    (b"\x1b[?2027$p", b"\x1b[?2027;2$y"),   # grapheme clustering: supported, currently off
    (b"\x1b[?u", b"\x1b[?0u"),              # kitty keyboard: no flags set
]


def answer(fd: int, data: bytes) -> None:
    """Replies to any capability query in the TUI's output."""
    for query, reply in QUERIES:
        for _ in range(data.count(query)):
            os.write(fd, reply)

# (delay in seconds since the previous step, what to do)
SCRIPT = [
    # Both cards enter Doing headlessly. A TUI-initiated move would flash a full bar in
    # To-Do first — every successful action ends at 100% — and the point of the take is
    # the bar filling in Doing.
    (2.0, ("move", "DEMO-1", "Doing")),
    (7.0, ("move", "DEMO-2", "Doing")),
    (4.5, b"l"),          # cursor: To-Do -> Doing, onto DEMO-1
    (1.2, b"L"),          # DEMO-1 -> Review. Doing's on_exit runs in the foreground, so the
                          # live loader climbs to 100% ON THE CARD and only then it moves.
    (4.5, b"l"),          # cursor: Doing -> Review
    (1.2, b"L"),          # refused: nothing leaves Review without a PR in the notes.
    (4.0, "redraw"),
    (1.5, CTRL_T),        # quit
    (0.4, b"q"),
]


def resize(fd, cols, rows):
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", rows, cols, 0, 0))


def main() -> int:
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", ROWS, COLS, 0, 0))

    env = dict(os.environ, HAKUBAN_TASK_DIR=DIR, TERM=os.environ.get("TERM", "xterm-256color"))
    proc = subprocess.Popen([BIN], stdin=slave, stdout=slave, stderr=slave,
                            env=env, start_new_session=True)
    os.close(slave)

    out = sys.stdout.buffer
    step, deadline = 0, time.monotonic() + SCRIPT[0][0]

    while proc.poll() is None:
        ready, _, _ = select.select([master, sys.stdin.buffer], [], [], 0.05)

        if master in ready:
            try:
                data = os.read(master, 65536)
            except OSError:
                break
            if not data:
                break
            answer(master, data)
            out.write(data)
            out.flush()

        # Relay the outer terminal's own replies (size and capability queries) inward.
        if sys.stdin.buffer in ready:
            try:
                data = os.read(sys.stdin.buffer.fileno(), 4096)
                if data:
                    os.write(master, data)
            except OSError:
                pass

        if step < len(SCRIPT) and time.monotonic() >= deadline:
            _, what = SCRIPT[step]
            if isinstance(what, tuple) and what[0] == "move":
                # Fire and forget: the dispatch hook returns at once, but waiting on any
                # subprocess would stall the relay loop and freeze the board.
                subprocess.Popen([BIN, "move", what[1], what[2]], env=env,
                                 stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            elif what == "redraw":
                # Differential rendering leaves stale cells behind, and emulators disagree
                # on ECH. A size change makes the TUI repaint the whole screen, which is
                # the only cheap way to guarantee a pristine final frame.
                resize(master, COLS - 1, ROWS)
                time.sleep(0.12)
                resize(master, COLS, ROWS)
            else:
                os.write(master, what)
            step += 1
            if step < len(SCRIPT):
                deadline = time.monotonic() + SCRIPT[step][0]

    # Drain whatever the TUI wrote on its way out.
    time.sleep(0.3)
    while True:
        ready, _, _ = select.select([master], [], [], 0.1)
        if not ready:
            break
        try:
            data = os.read(master, 65536)
        except OSError:
            break
        if not data:
            break
        out.write(data)
        out.flush()

    os.close(master)
    try:
        proc.wait(timeout=3)
    except subprocess.TimeoutExpired:
        proc.send_signal(signal.SIGINT)
    return 0


if __name__ == "__main__":
    sys.exit(main())
