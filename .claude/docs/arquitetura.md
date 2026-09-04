# Arquitetura

Duas camadas + entrypoint. O disco é a fonte da verdade; tudo em memória é cache.

```
cmd/hakuban/main.go   → abre a Store e roda a TUI
internal/task/           → storage (lê/escreve .md, monta o grafo em memória)
internal/tui/            → view (Bubble Tea) por cima da Store
```

## Fonte da verdade = disco

- Cada task é um arquivo `<dir>/tasks/<id>.md`: **frontmatter YAML** (campos) +
  **corpo markdown** (`Notes`). O `id` vem do nome do arquivo (autoritativo), não
  do frontmatter.
- A `Store` (`store.go`) é um `map[string]*Task` + `map[string]*Board` carregado
  no `Open`. Cada comando/sessão abre, opera e sai — o map nunca é autoritativo.
- **Write atômico** (`atomicWrite`): grava `.tmp` e faz `rename` (atômico no mesmo
  fs). Leitor externo nunca vê arquivo pela metade. Inegociável: dois escritores
  (app + `$EDITOR`/IA).
- **Data dir**: `~/.hakuban` por padrão; override por `$HAKUBAN_TASK_DIR` ou pelo
  ponteiro em `<userConfigDir>/hakuban/root.yml` (`root.go`, `ResolveDir`).

## Watch de arquivos (live reload)

A TUI relê o disco sozinha: um tick de ~1s (`reloadTickMsg`) compara uma
assinatura barata do data dir (`dirSig` = nome+tamanho+mtime, sem parsear) e só
reabre a store quando algo mudou (`refresh`). Editar um `.md` por fora aparece no
board sem reabrir o app. **É feature, não bug** — não quebre isso.

## Modelo de dados

### Task (`task.go`)

- Campos: `Title`, `Status`, `Priority`, `Project`, `Tags`, `Due` (`Date` sem
  hora), `Parent`, `Created`, `Modified`, `Notes` (corpo).
- **`Status` = texto de uma coluna do board.** Status desconhecido/vazio cai na
  coluna 0 no agrupamento (`groupByStatus`) — a task nunca some da tela.
- Hierarquia por `Parent` (walk-up). `Project`/`Tags` **efetivos** são herdados da
  raiz da árvore (`EffectiveProject`/`EffectiveTags`), não guardados no filho.

### Board (`board.go`)

- Board = `<dir>/boards/<id>.md` (frontmatter: `name`, `columns`, `created`). O
  **ID do board é o `project`** das tasks que ele agrupa.
- `Boards()` = união de boards com arquivo + projects efetivos em uso (descobertos
  das tasks), ordenados por nome. **Não há board embutido.**
- **Colunas são por board**: `Board.Columns *[]string`.
  - `nil` (chave ausente) → cai em `DefaultColumns` (`backlog/doing/done`) — Inbox
    antigo, boards legados e descobertos.
  - `&[]string{}` (não-nil vazio) → board novo, sem colunas de propósito.
  - `ColumnsFor(id)` resolve isso e devolve sempre uma cópia.
- **Migração de órfãs** (`AdoptOrphans`): no boot, tasks-raiz sem `project` ganham
  um board (`geral`) com `DefaultColumns` e são movidas pra lá. Roda 1× e some
  sozinho quando não há órfã. Legado do antigo "Inbox".
- `SaveBoard`/`DeleteBoard` ignoram ID vazio (`InboxID = ""` é sentinela de "sem
  board / sem project", não um board de verdade).

## Testes da camada

`go test`, sem framework. Padrão: `t.TempDir()` → opera → **reabre a store do
disco** pra provar que persistiu (não confie no map em memória). Helper `must(t,
err)` em `store_test.go`.
