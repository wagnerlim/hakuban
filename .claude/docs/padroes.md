# Padrões, testes e workflow

## Ponytail (estilo do repo)

A solução mais preguiçosa que **funciona de verdade** — só depois de entender o
problema. A escada (pare no primeiro degrau que resolve):

1. precisa existir? (YAGNI)
2. já existe no repo? reutilize.
3. stdlib resolve?
4. feature nativa (`$EDITOR`, git, ripgrep) resolve?
5. dependência já instalada resolve?
6. dá em 1 linha?
7. só então: o mínimo que funciona.

- Sem abstração especulativa (interface com 1 impl, factory de 1 produto, config
  pra valor que nunca muda).
- Deleção > adição. Chato > esperto.
- Simplificação deliberada leva `// ponytail: <o quê>[, <upgrade se tiver teto>]`.
- **Não** simplifique: validação em fronteira, tratamento de erro que evita perda
  de dado, segurança, acessibilidade, o que foi explicitamente pedido.
- Nunca seja preguiçoso em **entender** o problema — leia o fluxo inteiro antes.

## Comentários

pt-BR, densos, explicando o **porquê** (não o quê). Combine com a densidade e o
tom do arquivo em volta. Um comentário bom sobrevive a refactor.

## Testes

- `go test`, **sem framework**, sem fixtures elaboradas.
- Helper `must(t, err)` (`internal/task/store_test.go`).
- `t.TempDir()` pra isolar; **reabra a store do disco** pra provar persistência
  (não confie no map em memória).
- Toda lógica não-trivial (branch, loop, parser, migração, dinheiro/segurança)
  deixa **1 teste** que quebra se a lógica quebrar. One-liner trivial não precisa.
- Na TUI, dá pra dirigir por eventos: `m.Update(tea.KeyPressMsg{…})`,
  `tea.MouseClickMsg{…}`, e checar estado / `m.boardView()`.

## Antes de commitar

```
go build ./...          # compila
go vet ./...            # estático
gofmt -l internal/      # sem saída = formatado (use -w pra formatar)
go test ./...           # tudo verde
```

Além do compilador: **verifique o comportamento** disparando o fluxo real
(reabrir a store, simular o input). Se um passo foi pulado, diga.

## Commits

- Conventional commits em **pt-BR**: `feat(tui): …`, `fix(task): …`,
  `docs: …`, `refactor(tui): …`.
- Trailer obrigatório:
  `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- Repo **pessoal, solo, direto na `main`**. Identidade `wagnerlima0910@gmail.com`
  (remote `github.com/wagnerlim/hakuban`).
- **Só commita/faz push quando o usuário pedir.** Split honesto: se as mudanças de
  N features vivem no mesmo arquivo, prefira 1 commit atômico bem descrito a uma
  divisão falsa (não há `git add -p` interativo neste ambiente).

## Roadmap / costuras

- **Config por lane** (`updateLaneConfig`/`laneConfigBox` na `tui.go`) é o ponto
  de extensão pra **integração com Jira**. Quando entrar, o passo natural é
  promover a coluna de `string` → struct (ex: `{Nome, JiraStatusID, …}`) e migrar
  o YAML dos boards (cuidando de `ColumnsFor`, `groupByStatus` e o match do
  `Status` das tasks).
- Extensão do **produto** = hooks/scripts + `--json` + `$EDITOR`, nunca plugin API
  interna. Ver [`docs/objetivos.md`](../../docs/objetivos.md).
