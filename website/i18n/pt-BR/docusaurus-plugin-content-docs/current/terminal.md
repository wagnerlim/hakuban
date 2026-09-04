---
sidebar_position: 2
title: No terminal
---

# No terminal

Tudo aqui é opcional. O Hakuban roda sem `config.yml` nenhum — todo campo abaixo tem
default, e arquivo ausente é um estado válido.

## Duas portas de entrada

```bash
hakuban                          # a TUI (exige um TTY)
hakuban move <id> "<Coluna>"     # headless: dispara on_exit da origem, depois on_enter do destino
hakuban progress <id> <0..100>   # headless: a porcentagem exibida no card
```

Esses dois subcomandos são toda a superfície headless. São genéricos de propósito — não
existe `hakuban jira ...`, não existe `hakuban deploy`. Qualquer coisa específica mora num
hook que o quadro dispara.

Códigos de saída: `0` feito, `1` a operação falhou (hook que falha deixa o card onde
estava), `2` uso incorreto. Nome de coluna é comparação exata e case-sensitive, então use
aspas em qualquer um que tenha espaço.

## De onde vem o data dir

Resolvido no boot, o primeiro que casar ganha:

1. `HAKUBAN_TASK_DIR` — ambiente
2. `<config dir do usuário>/hakuban/root.yml`, campo `datadir` — o ponteiro que a TUI
   escreve quando você escolhe outra pasta
3. `~/.hakuban`

O ponteiro existe porque o `config.yml` mora *dentro* do data dir e por isso não pode
guardar o próprio caminho. Mudar o data dir pela TUI nunca mistura dois conjuntos: se a
pasta de destino já tem dados do Hakuban, a mudança é recusada.

## Teclado

Dois grupos. **Navegação dispara direto.** **Comandos ficam atrás de um prefixo estilo
tmux, `ctrl+t`** — aperta o prefixo, depois a tecla.

### Navegação (sem prefixo)

| id da ação | default | o que faz |
|---|---|---|
| `nav_left` / `nav_right` | `h` / `l` | move o cursor entre colunas |
| `nav_down` / `nav_up` | `j` / `k` | move o cursor entre cards |
| `move_left` / `move_right` | `H` / `L` | **move o card** uma coluna — isto dispara os hooks |
| `next_board` / `prev_board` | `tab` / `shift+tab` | troca a aba de quadro |
| `open` | `enter` | abre o card selecionado |

### Comandos (depois do `ctrl+t`)

| id da ação | default | o que abre |
|---|---|---|
| `add` | `a` | card novo na coluna atual |
| `subtask` | `S` | card filho do selecionado |
| `find` | `/` | busca de card — um input e uma lista, `enter` leva o cursor ao resultado |
| `board_list` | `b` | lista de quadros: abrir, fechar, criar |
| `new_board` | `+` | cria um quadro |
| `close_board` | `ctrl+w` | fecha a aba atual |
| `board_cfg` | `c` | config do quadro: renomear, key, colunas (reordenar / adicionar / apagar) |
| `lane_cfg` | `g` | config de uma coluna: filtros, renomear, apagar |
| `sync` | `y` | roda o `sync` do quadro para a coluna atual |
| `tags` | `t` | catálogo de tags: cor e descrição por tag |
| `card_tags` | `T` | marca/desmarca tags do catálogo no card selecionado |
| `settings` | `s` | tema, idioma, painel de preview, formato de data, editor, data dir |
| `keymap` | `?` | o editor de atalhos — lista ação → tecla e captura uma nova |
| `quit` | `q` | sai |

### Aliases que o rebind não quebra

Independentes do config e do prefixo: `←` `→` `↑` `↓` para navegar, `<` e `>` para mover o
card, `ctrl+c` para sair.

### Trocando atalhos

Abra o editor (`ctrl+t` `?`), escolha a ação, aperte a tecla nova. Só o que **difere do
default** vai pro `config.yml`, então o arquivo fica pequeno e uma mudança futura de
defaults ainda te alcança:

```yaml
keys:
  add: n
  find: f
```

As chaves desse mapa são os ids de ação das tabelas acima.

## Mouse

O teclado é o caminho canônico e o mouse espelha. O ponteiro te diz o que a superfície faz
antes de você clicar:

| ponteiro | significa | onde |
|---|---|---|
| `grab` / `grabbing` | arrastável | cards, trilhas de scroll, títulos de modal |
| `pointer` | age em um clique | ☰, botão de sync, abas, linhas de menu/config/filtro |
| `text` | input focado | qualquer input |

E a interação é a mesma em todo lugar:

- **1 clique seleciona** — move o cursor, não age
- **2 cliques ativam** — equivalente ao `enter`
- **arrastar** = press mais movimento: um card entre colunas, uma trilha de scroll, um
  modal pelo título. Clique que não move nunca arrasta.
- **o id do card no detalhe é um link** — 1 clique, sem modificador, abre a issue no seu
  browser. Precisa do `issue_url` no quadro (veja abaixo).

Modais são arrastáveis pelo título e fecham na bolinha vermelha.

## `config.yml`

Fica em `<data dir>/config.yml`. Editável à mão — a TUI lê no load e escreve de volta
quando você muda algo em Settings. Todo campo é opcional.

```yaml
theme: omni                 # veja a lista abaixo
lang: pt-BR                 # pt-BR | en-US | zh-Hans
preview_pane: true          # painel markdown no rodapé
date_format: 2006-01-02     # um layout Go, aplicado ao `due`
editor: ''                  # '' = $EDITOR
confirm_delete: true        # pergunta antes de apagar
priorities: [low, normal, high]
keys:                       # só os overrides
  add: n
tags:                       # o catálogo de tags
  - name: backend
    color: blue             # uma hue da paleta, não um hex
    desc: mexe na API
```

| campo | default | notas |
|---|---|---|
| `theme` | `omni` | trocável em runtime pelo Settings |
| `lang` | `pt-BR` | toda string visível vem do catálogo i18n |
| `preview_pane` | `false` | o markdown do card no rodapé |
| `date_format` | `2006-01-02` | layout Go, não `YYYY-MM-DD` |
| `editor` | `''` | cai no `$EDITOR` |
| `confirm_delete` | `true` | |
| `priorities` | `[low, normal, high]` | os níveis que um card pode carregar |
| `keys` | — | ação → tecla, só overrides |
| `tags` | — | catálogo: `name`, `color`, `desc` |

Ele **não** guarda o data dir — isso moraria dentro de si mesmo.

## Temas

19 na caixa, trocáveis em runtime:

`omni` (default) · `catppuccin` · `catppuccin-latte` · `tokyo-night` · `tokyo-night-day` ·
`dracula` · `nord` · `gruvbox` · `gruvbox-light` · `one-dark` · `one-light` · `solarized` ·
`solarized-light` · `kanagawa` · `kanagawa-lotus` · `rose-pine` · `rose-pine-dawn` ·
`vesper` · `terminal`

Os claros são `catppuccin-latte`, `gruvbox-light`, `one-light`, `solarized-light`,
`tokyo-night-day`, `kanagawa-lotus` e `rose-pine-dawn`. O `terminal` usa as cores ANSI do
seu próprio emulador em vez de hex, então herda o que o seu terminal estiver usando.

As paletas são creditadas aos projetos originais (Catppuccin, Nord, Dracula, Gruvbox,
Solarized, Tokyo Night, Kanagawa, Rosé Pine, One Dark). Adicionar uma é uma struct.

A `color` de uma tag é uma **hue nomeada da paleta ativa**, não um hex: `mauve`, `blue`,
`green`, `yellow`, `peach`, `red`, `teal`. É por isso que uma tag continua fazendo sentido
quando você troca o tema. Cor vazia, ou tag fora do catálogo, renderiza na cor default — o
catálogo é sugestão, não trava.

## Filtros

Filtros são declarados uma vez no registro `filters` do quadro e depois habilitados por
coluna com `use_filters`. Existem **duas naturezas**, e confundi-las é o erro comum.

### Filtro de tracker — molda o que o `sync` traz

O core nunca interpreta o valor; ele repassa a sua seleção para o hook de sync, que compõe
a consulta. Só online: não consegue tocar num card local. Três formas:

```yaml
filters:
  mine: assignee = currentUser()          # simples: escalar, liga/desliga
  sprint:                                  # opções estáticas: multi-seleção
    options:
      Atual: sprint in openSprints()
      Próxima: sprint in futureSprints()
  epic:                                    # opções dinâmicas: listadas na hora do uso
    options_cmd: hooks/list-epics.sh
```

O hook recebe tudo em `HAKUBAN_FILTERS`, como JSON **agrupado por filtro**. O core nunca
junta os valores: a convenção é que o hook faça OR entre as opções de um filtro e AND entre
filtros.

### Filtro de campo — avaliado localmente

Um predicado sobre os campos do próprio card. Funciona offline, em qualquer card, local ou
espelho:

```yaml
filters:
  recentes:
    field: modified      # modified | created
    within_days: 7
```

Filtro malconfigurado — campo desconhecido, timestamp zerado — mantém tudo. Nada é
escondido em silêncio.

### Habilitando numa coluna

```yaml
actions:
  Backlog:
    jql: project = TEAM AND status = "To Do"
    use_filters:
      sprint: [Atual]
      mine: []
```

## Botões de coluna

Um botão no rodapé da coluna, para que uma ação rode sobre **todos os cards da coluna** e
não só sobre o selecionado:

```yaml
actions:
  Review:
    button_label: checks
    buttons:
      - label: rodar CI
        icon: ▶
        cmd: hooks/ci.sh          # script: os cards da coluna como array JSON no stdin
        batch: true
      - label: resumir
        agent: |                   # instrução: prosa, limitada por agent_tools
          Write one paragraph per card in this column.
```

O `cmd` segue o mesmo contrato de um hook de transição — veja
[o contrato do hook](./hook-contract.md) — exceto que o stdin é um **array** dos cards da
coluna. O `agent` é uma instrução, e pode ser o caminho de um `.md` dentro do data dir.
`batch: true` inscreve o botão no botão geral da barra de topo, que dispara os botões batch
de todas as colunas, um por vez.

## Abrindo um card no tracker

```yaml
issue_url: https://acme.atlassian.net/browse/{key}
```

O `{key}` é substituído pela chave externa do card. Com isso configurado, o id no detalhe
fica clicável e abre no seu browser (`open` / `xdg-open` / `rundll32`). Sem isso, o id é
texto puro.

## O contrato do sync

O `sync` é uma linha de comando, como qualquer hook, e ele recebe:

| canal | conteúdo |
|---|---|
| stdin | os cards atuais da coluna, como array JSON |
| `HAKUBAN_JQL` | o `jql` da coluna |
| `HAKUBAN_COLUMN` | o nome da coluna |
| `HAKUBAN_BOARD` | o id do quadro |
| `HAKUBAN_DIR` | o data dir |
| `HAKUBAN_FILTERS` | os filtros selecionados, JSON agrupado por filtro (`{}` quando nenhum) |
| `HAKUBAN_COMMENTS` | `1` quando o quadro tem `comments: true` |
| `HAKUBAN_TASK_BIN` | o binário em execução, para um hook que chama de volta |

`sync_on_open: true` roda ao abrir o quadro; fora disso é `ctrl+t` `y`, o botão de sync da
coluna, ou o botão geral.
