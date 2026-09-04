# hakuban

Gerenciador de tasks no terminal, construído no ecossistema [Charm](https://charm.sh)
(Bubble Tea + Lip Gloss + Bubbles + Glamour).

## Objetivos

- TUI própria em Go (aprendizado + controle total).
- Notas/contexto rico por task via Markdown (renderizado com Glamour).
- Storage em arquivos, amigável a git e legível por ferramentas externas.

## Status

Bootstrap. Arquitetura em discussão.

## Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — framework TUI (arquitetura Elm).
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — estilo/layout.
- [Bubbles](https://github.com/charmbracelet/bubbles) — componentes (list, textinput, viewport).
- [Glamour](https://github.com/charmbracelet/glamour) — renderização de Markdown.
