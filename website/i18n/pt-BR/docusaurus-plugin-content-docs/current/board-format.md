---
sidebar_position: 3
title: Formato do quadro
---

# Formato do quadro

Um quadro é um arquivo Markdown com frontmatter YAML em `~/.hakuban/boards/<id>.md`. O
nome do arquivo é o id do quadro.

```yaml
---
name: My board
columns: [To-Do, Doing, Review, Done]
actions:
  Doing:
    guide: One card here at a time.
    on_enter: |
      Implement the card in the repository, run the tests,
      commit and open the PR.
  Done:
    on_enter_cmd: hooks/notify.sh
agent_tools: Read, Write, Bash
---
```

Tudo abaixo do `---` de fechamento é markdown livre — anotações sobre o quadro, ignoradas
pelo parser.

## Campos do quadro

| campo | tipo | o que faz |
|---|---|---|
| `name` | string | nome do quadro exibido na aba |
| `key` | string | prefixo dos ids dos cards criados neste quadro |
| `columns` | lista | os estados da esteira, em ordem — isto é o quadro |
| `actions` | mapa | nome da coluna → sua [action](#actions-de-coluna) |
| `agent_tools` | string | as únicas ferramentas que uma instrução deste quadro pode usar, separadas por vírgula |
| `filters` | mapa | filtros nomeados que a TUI oferece no quadro |
| `sync` | string | comando que materializa espelhos de um tracker externo |
| `sync_on_open` | bool | roda o `sync` ao abrir o quadro |
| `comments` | bool | traz os comentários da issue para dentro do espelho |
| `buttons` | lista | ações customizadas no rodapé do card |

## Actions de coluna

Cada chave de `actions` é um **nome de coluna**, com comparação exata e case-sensitive. O
valor dela:

| campo | o que faz |
|---|---|
| `guide` | nota da coluna. **Lida, nunca executada** — é contexto para quem (ou o que) decide o próximo movimento |
| `on_enter` / `on_exit` | **instrução** — prosa rodada por um agente headless, limitada por `agent_tools` |
| `on_enter_cmd` / `on_exit_cmd` | **script** — shell, card em JSON no stdin. Veja o [contrato do hook](./hook-contract.md) |
| `jql` | torna a coluna espelho de uma consulta externa; o comando `sync` a materializa |
| `status` | o status que a coluna representa no tracker, exportado para os hooks como `HAKUBAN_STATUS` |
| `percent` | exibe o `progress` do card como barra nesta coluna |
| `loader` | rótulo do indicador de progresso enquanto a action roda |
| `use_filters` | quais filtros se aplicam a esta coluna |

Uma instrução **sombreia** o script no mesmo lado da mesma coluna: `on_enter` é checado
antes de `on_enter_cmd`, então adicionar uma instrução onde já existe um script faz o
script parar de rodar, em silêncio. Se você adicionar uma, ela tem que absorver o que o
script fazia — ou chamá-lo.

Coluna sem `jql` é local: existe só no Hakuban, sem contrapartida consultável no tracker.

### Interpolação na instrução

Uma instrução é um template Go sobre os campos do card. Disponíveis: `{{.id}}`,
`{{.title}}`, `{{.jira}}`, `{{.status}}`, `{{.priority}}`, `{{.project}}`, `{{.notes}}`.
Template inválido cai no texto cru — a prosa continua fazendo sentido.

A instrução também pode ser um caminho para um arquivo `.md`, que é a opção legível quando
ela passa de umas poucas linhas.

## O card

```yaml
---
id: PES-12
title: Ship the export command
status: pending          # pending | done
priority: high           # low | normal | high
progress: 40             # 0..100, shown on the card
project: hakuban
tags: [cli]
due: 2026-09-10
jira: PES-12             # set when a hook stamped an external key
---

Whatever context matters. This body is the card's **notes**, and it is what a
hook or an agent reads.
```

As notes são o payload. Um hook que precisa de um parâmetro — um repositório, uma
estimativa de tempo — lê daqui, e recusa o movimento quando ele falta. Essa recusa é a
feature: veja [o contrato](./hook-contract.md#exit-code).
