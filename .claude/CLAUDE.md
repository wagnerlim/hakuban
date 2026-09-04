# CLAUDE.md — guia do agente (hakuban)

Gerenciador de tasks de **terminal** (TUI Kanban) sobre arquivos `.md`. Go 1.26,
ecossistema Charm (Bubble Tea v2, Lip Gloss v2, Bubbles, Glamour). Sem banco,
sem web, sem CLI — só a TUI sobre o disco.

## Leia primeiro

- **Produto / decisões** → [`docs/objetivos.md`](../docs/objetivos.md) (conceito,
  princípios, plain-text, hooks). Leia antes de mudar comportamento.
- **Arquitetura** → [`.claude/docs/arquitetura.md`](docs/arquitetura.md)
- **Camada TUI** → [`.claude/docs/tui.md`](docs/tui.md)
- **Padrões, testes, workflow** → [`.claude/docs/padroes.md`](docs/padroes.md)

## Mapa do código

- `cmd/hakuban/main.go` — entrypoint: abre a store e sobe a TUI.
- `internal/task/` — **storage. Fonte da verdade = disco.** (`task.go`,
  `store.go`, `board.go`, `state.go`, `config.go`, `root.go`)
- `internal/tui/` — view Bubble Tea (`tui.go`, `i18n.go`, `themes.go`).

## Regras de ouro (não-negociáveis)

1. **Disco é a fonte da verdade.** A `Store` é cache em memória; a TUI relê o
   disco ~1×/s (`dirSig`/`refresh`). Nunca trate a memória como autoritativa.
2. **Write atômico sempre** (`atomicWrite`: tmp + rename). Há dois escritores
   legítimos: a TUI e humano/IA editando o `.md` na mão.
3. **Toda string visível vem do i18n** (`internal/tui/i18n.go`) — nada hardcoded
   na view. 3 idiomas: pt-BR (default), en-US, zh-Hans.
4. **Ponytail.** Componente pronto > código próprio; feature nativa >
   reimplementar; 1 linha > 50. Simplificação deliberada leva
   `// ponytail: <o quê>[, <caminho de upgrade>]`.
5. **Todo código e comentário em inglês.** Comentários densos, explicando o
   *porquê*. Siga o tom do arquivo.
6. **Lógica não-trivial deixa 1 teste** (`go test`, sem framework).
7. **UX de mouse é parte da feature.** Toda superfície interativa nova/alterada
   tem que funcionar no mouse **e** indicar o afeto pelo ponteiro: arrastável →
   `grab`/`grabbing`, clicável → `pointer`, input → `text`. 1 clique seleciona,
   2 cliques ativam (= Enter). Regras e checklist em
   [`.claude/docs/tui.md`](docs/tui.md) → *UX de mouse*.

## Antes de commitar

```
go build ./... && go vet ./... && gofmt -l internal/ && go test ./...
```

`gofmt -l` sem saída = formatado. Verifique o comportamento de verdade (dispare
o fluxo / reabra a store), não só o compilador.

## Commits

- Conventional commits em inglês: `feat(tui): …`, `fix(task): …`.
- Trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Repo pessoal, solo, direto na `main`. Identidade `68910437+wagnerlim@users.noreply.github.com`.
- Só commita/faz push **quando o usuário pedir** — e o pedido dele **é** a autorização: não
  peça confirmação de novo depois de um "commita isso".
- **Exceção: agente autônomo do Doing.** Rodando pelo `ct-doing-run.sh` (card no Doing do
  board Project-Board, pane do herdr, sem ninguém pra responder), commit, push e PR estão
  **autorizados de saída** — pôr o card em Doing É o pedido. Parar pra perguntar ali trava o
  agente e reprova a entrega. Você reconhece esse contexto pelo cwd
  `.claude/worktrees/<CARD-ID>` e pelo prompt dizendo que é autônomo.
