---
sidebar_position: 1.5
title: Como funciona
---

# Como funciona

Não existe banco de dados nem servidor. O Hakuban é uma pasta de arquivos Markdown e um
binário que lê essa pasta.

## O data dir

```text
~/.hakuban/
├── boards/
│   └── demo.md          # um quadro: as colunas e o que cada uma faz
├── tasks/
│   ├── DEMO-1.md        # um card por arquivo, nomeado pelo id
│   ├── DEMO-2.md
│   └── DEMO-3.md
├── jira/                # espelhos escritos pelo `sync` do quadro — read-only
├── archive/             # cards arquivados, mesmo formato
├── hooks/               # seus scripts e instruções — por convenção; os caminhos resolvem a partir de ~/.hakuban
├── config.yml           # tema, idioma, tags, teclas — opcional
└── state.yml            # quais abas estão abertas — descartável
```

Só `boards/` e `tasks/` importam. Apague o `state.yml` e a TUI reabre todos os quadros;
apague o `config.yml` e cada campo cai no seu default. De onde vem essa pasta está em
[No terminal](./terminal.md#de-onde-vem-o-data-dir).

## Um card é um arquivo

```markdown
---
id: DEMO-2
title: Corrigir o deslocamento de fuso na lista de faturas
status: To-Do
priority: normal
project: demo
tags:
    - FRONTEND
created: 2026-09-04T09:00:00Z
modified: 2026-09-04T18:38:41Z
---

As datas aparecem um dia adiantadas para quem está a oeste do UTC.
```

Dois campos amarram o card a um quadro:

- `project` é o id do quadro — o nome do arquivo em `boards/`, sem o `.md`.
- `status` é a coluna onde o card está. Mover um card reescreve essa única linha.

O id vem da `key` do quadro mais um contador (`DEMO-1`, `DEMO-2`…). Um card com `parent` é
uma subtarefa e herda o quadro da sua raiz. Tudo abaixo do `---` de fechamento são as notes
do card — Markdown livre, e o lugar de onde um agente lê a tarefa dele.

## O que um move faz

Arraste um card na TUI, ou rode `hakuban move DEMO-2 Doing` — é o mesmo caminho:

1. Roda o `on_exit` da coluna de origem, se houver.
2. Roda o `on_enter` da coluna de destino, se houver.
3. Só se os dois derem certo: o `status` vira o destino, o `progress` volta para `0` e o
   arquivo é escrito.

Um hook que sai com código diferente de zero interrompe a cadeia naquele passo. Nada é
escrito, o card fica onde estava, e o stderr do hook é o motivo que você vê. De cada lado,
uma instrução (`on_enter`) ganha de um script (`on_enter_cmd`) — veja o
[contrato do hook](./hook-contract.md).

## Dois escritores, uma verdade

O disco é a fonte da verdade. A TUI escreve nele, e qualquer outra coisa capaz de editar um
arquivo também — você no `$EDITOR`, um agente, um hook chamando `hakuban progress`. Toda
escrita vai para um `.tmp` e é renomeada para o lugar, então nenhum leitor vê meio card. A
TUI relê a pasta cerca de uma vez por segundo: um card que um agente move aparece na sua
frente sem refresh.

É também por isso que a pasta funciona com git: commite ela, e o histórico do quadro é o
`git log`.
