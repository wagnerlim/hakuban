# hakuban

Terminal task manager, built on the [Charm](https://charm.sh) ecosystem
(Bubble Tea + Lip Gloss + Bubbles + Glamour).

## Install

```bash
brew install wagnerlim/tap/hakuban          # macOS and Linux

scoop bucket add wagnerlim https://github.com/wagnerlim/scoop-bucket
scoop install hakuban                       # Windows

go install github.com/wagnerlim/hakuban/cmd/hakuban@latest
```

Docs: https://wagnerlim.github.io/hakuban/

## Goals

- Own TUI in Go (learning + full control).
- Rich per-task notes/context through Markdown (rendered with Glamour).
- File-based storage, git-friendly and readable by external tools.

## Status

Bootstrap. Architecture under discussion.

## Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework (Elm architecture).
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — styling/layout.
- [Bubbles](https://github.com/charmbracelet/bubbles) — components (list, textinput, viewport).
- [Glamour](https://github.com/charmbracelet/glamour) — Markdown rendering.
