# hakuban — objetivos & estrutura

Fonte de verdade das decisões de produto e dados. Atualizar aqui antes de mudar código.

## Conceito

hakuban é um **gerenciador de tasks de terminal, open-source e hackeável**,
construído no ecossistema Charm. A identidade tem três pilares:

- **Plain-text durável.** Tasks são arquivos `.md` versionados em git. Dados são
  do usuário, legíveis e editáveis fora do app. Sem lock-in, sem banco.
- **Hiperconfigurável por composição, não por config obrigatória.** O usuário
  molda quase tudo (tema, atalhos, layout, campos/status, views), mas os
  **defaults sãos funcionam sozinhos** — configurar é opcional, nunca requisito.
- **Extensível via hooks/scripts — o coração do projeto.** Em vez de uma plugin
  API interna, o core expõe costuras: eventos disparam scripts do usuário que
  recebem a task como `--json`. Toda a extensão vive fora do binário. É o que
  torna o app "hiper" sem inchar o core.

Ecossistema Charm: Bubble Tea + Lip Gloss + Bubbles + Glamour (TUI/tema), Huh
(formulários), Log (logs). Gum é usado *pelos scripts do usuário* nos hooks (o
app só cospe `--json`). Wish / Soft Serve (servir por SSH/git) ficam pra depois.

## Princípios

- **Terminal-only.** Sem app gráfico, sem web, sem mobile.
- **Arquivos `.md` são a fonte da verdade.** Um arquivo por task: frontmatter (campos
  estruturados) + corpo markdown (notas). Sem banco de dados.
- **Sem índice/SQLite** enquanto scan de `.md` for instantâneo (< milhares de tasks).
  Reavaliar só se ficar lento — e aí como índice *derivado*, nunca fonte de verdade.
- **Dois escritores legítimos:** a própria TUI e você (ou uma IA) editando o `.md`
  na mão. A TUI **relê o disco ao vivo** — um poll de ~1s compara uma assinatura
  barata do data dir (nome+tamanho+mtime, sem parsear) e só reabre a store quando
  algo mudou (`reloadTickMsg`/`dirSig`/`refresh` no `internal/tui`). Mudança
  externa aparece no board sem reabrir o app. Isso é feature, não bug.
- **Ponytail:** componente pronto (Charm/Cobra) antes de código próprio; feature nativa
  (`$EDITOR`, git, ripgrep) antes de reimplementar.
- **Extensão = composição, não plugin API.** Configurabilidade vem de arquivos
  plain-text + shell-out (hooks, `--json`, `$EDITOR`), nunca de um motor de config
  gigante ou API de plugin em Go. Core pequeno, costuras expostas.
- **Defaults sãos > config obrigatória.** O app tem que ser útil sem nenhum ajuste.
- **Relações = grafo em memória derivado dos `.md`.** Arestas (parent, backlink,
  dependência) vivem no frontmatter/notas; o app monta um grafo no load (`map` do
  stdlib, sem graph DB nem lib de grafo). Derivado e descartável, nunca fonte de
  verdade. Herança e listas-de-filhos são walks nesse grafo.
- **Modais = janela estilo macOS (padrão único).** *Todo* modal — presente ou
  futuro — segue o mesmo molde, por um só ponto no código (`modalBox` + `overlay`
  no `internal/tui`): **traffic lights na borda de cima, com folga** (a borda não
  encosta na bolinha; vermelha acesa fecha, amarela/verde apagadas), **título
  centralizado**, **sombra** (flutua acima do board) e **arrastável pela barra de
  título**. Modal novo reusa esse molde — não se inventa outro formato.

## Estrutura de uma task (frontmatter)

```markdown
---
id: 0a6f                        # gerado
title: comprar café             # obrigatório
status: backlog                 # backlog | doing | done (colunas do board)
priority: normal                # low | normal | high
project: casa                   # opcional; = id do board a que o card pertence (F23)
tags: [compras]                 # opcional
due: 2026-07-09                 # opcional
parent: 3b21                    # opcional (vazio = task raiz)
jira: ""                        # v2 (F16); chave da issue (PROJ-123) — vazio = sem vínculo
source: ""                      # v2 (F16); "jira" = espelho read-only; vazio = card local
created: 2026-07-08T14:20:00Z   # gerado
modified: 2026-07-08T14:20:00Z  # gerado
---

## Notas
markdown livre
```

Esta estrutura cobre **todas** as features abaixo. v2 (jira) e v3 (sync) já têm o
espaço reservado — não é preciso mexer no frontmatter depois.

Decisão: `project` é campo separado de `tags` (filtrar "tudo do projeto X" é comum).

## Features

Cada feature é um bloco fechado. `[v1]` = núcleo usável no dia a dia (offline, uma
máquina). `[v2]` = integração e o que é mais pesado.

**Status de implementação:** ✅ feito · 🟡 parcial · ⬜ (ou sem marcador) = não começou.
Onde há progresso, a linha **Feito/Falta** resume o estado real do código.

### Mapa de progresso

| | Feature | Estado |
|---|---|---|
| ✅ | **F6b** Board Kanban (TUI principal) | colunas fixas, mini-cards, mover por teclado **e arrastando** (fantasma + coluna-alvo), scroll por coluna |
| 🟡 | **F23** Boards & abas | abas (troca/`+`/`✕`/fechar), filtro por board, modais macOS flutuantes+arrastáveis, mouse completo, persistência — falta deletar/renomear board |
| 🟡 | **F1** Criar task | `a` cria só título → falta parser da linha e demais campos |
| 🟡 | **F2** Árvore & links | storage completo (parent/herança) → falta UI |
| 🟡 | **F4** Detalhe | view tela cheia + Glamour + scroll → falta ações e herdado-vs-próprio |
| 🟡 | **F5** Concluir & arquivar | concluir = mover pra DONE → arquivar sem tecla |
| 🟡 | **F9** Prioridade | exibida com cor → falta setar/customizar |
| 🟡 | **F18** Estados extras | só `doing` → faltam bloqueada/esperando/cancelada |
| 🟡 | **F20** Visões | kanban feito → faltam agenda e stats |
| 🟡 | **F13** Configurações | `config.yml` + menu `modeSettings` (tecla `s`): 19 temas (paleta semântica, afeta a UI toda), formato de data, painel de preview no rodapé → falta fase 2 (confirm delete, $EDITOR) e submenus (prioridade, rebind, overrides de cor) |
| 🟡 | **F21** i18n | catálogo `messages` + `applyLang` (padrão da paleta); pt-BR, en-US, zh-Hans; trocável no menu → falta só mais idiomas |
| ⬜ | **Não começadas** | F3 editar · F6 listagem · F7 filtros · F8 labels · F10 massa · F11 git no app · F12 CLI `--json` · F14–F17 · F19 · F22 hooks |

### F1 — Criar task `[v1]` 🟡
Quick-add numa linha: `comprar café +compras @casa !high due:amanhã`. Campos:
título, status, prioridade, projeto, tags, due, notas (markdown), parent.
Subtask = criar task com `parent` preenchido (reusa esta feature).

**Feito:** `a` cria task só com título, vai pro backlog. **Falta:** parser da
linha (`+tag @proj !prio due:`), demais campos e criar subtask pela UI.

### F2 — Árvore & links entre tasks `[v1]` 🟡
N níveis. Criar filho a partir do pai; re-parentar (mover na árvore).

**Feito (storage):** `parent`, filhos derivados (`Children`), herança dinâmica de
`project`/`tags` (walk-up), tolera parent dangling e ciclo. **Falta (UI):** criar
filho, re-parentar, deletar com cascata/promover, backlinks `[[id]]` e `blocked-by`.

**Mecanismo de vínculo:**
- Linka por **`id` estável** (nunca por título/arquivo — título muda, id não).
- **Arestas tipadas no frontmatter:** `parent: <id>` (hierarquia), `blocked-by:
  [<id>...]` (dependência, F17). Livre e opcional: `[[id]]` na nota (backlink
  "veja também", sem tipo).
- **Guarda a aresta num lado só, deriva o inverso** no load: filho guarda
  `parent` → lista de filhos derivada; `blocked-by` → "o que bloqueia" derivado;
  `[[id]]` → "quem me referencia" derivado varrendo os `.md`. Um escritor por
  aresta, zero sync bidirecional.

**Herança (dinâmica):** valores descem a aresta `parent` ao vivo (derivados pelo
grafo, não copiados) — coerente com labels configuráveis (F8).
- `project`: herdado da **raiz** da árvore (walk-up). Valor único; filho não guarda.
- `tags`: **união aditiva** — efetivas = próprias ∪ de todos os ancestrais. Filho
  soma as suas; herdadas não somem (remover herdada fica pra depois do v1).
- Re-parentar re-deriva tudo automaticamente.
- Custo aceito: `.md` isolado não mostra o herdado; F4/F12 expõem o *efetivo*.

**Deletar pai com filhos → pergunta (nunca cascata automática):** a subárvore pode
ter vida própria.
- **Cascata:** apaga o pai + toda a subárvore.
- **Promover** (default seguro): apaga só o pai; filhos sobem um nível — reconectam
  ao avô se existir, senão viram raízes. Ao virar raiz, o app **materializa** no
  novo root o `project` que era derivado do pai (senão sumiria).

Task sem filhos deleta direto (sujeito ao "confirmar antes de deletar" da F13).

### F3 — Editar task `[v1]`
Editar campos; editar notas no `$EDITOR` (shell-out, sem editor próprio).

### F4 — Detalhe da task `[v1]` 🟡
Abrir (Enter) e ver todos os campos + notas markdown renderizadas (Glamour).
Mostra o que é herdado (projeto/tags do pai) vs próprio. De lá: editar, marcar
done, abrir `$EDITOR`, ver filhos.

**Feito:** Enter abre o detalhe em tela cheia com campos + `project`/`tags`
efetivos + notas via Glamour + scrollbar. **Falta:** distinguir herdado vs próprio
e as ações a partir dele (editar, done, `$EDITOR`, ver filhos).

### F5 — Concluir & arquivar `[v1]` 🟡
Marcar done. Arquivar concluídas (`.md` → `archive/`) pra manter a lista leve.

**Feito:** concluir = mover o card pra coluna DONE (`H/L`). `Store.Archive` existe.
**Falta:** tecla na UI pra arquivar.

### F6b — Board Kanban `[v1]` — TUI principal ✅
Board de 3 colunas (`backlog / doing / done`) sobre os mesmos `.md`. Coluna =
campo `status`; mover card = trocar status + salvar. Navegação vim (`hjkl`,
`H/L` troca coluna), `enter` abre o detalhe (F4), `a` cria. É a primeira e
principal TUI do app — puxa o kanban da F20 e os estados `doing` da F18 pro v1.
Ordem dentro da coluna = criação (reorder manual fica pra depois; precisaria de
campo `order` no frontmatter).

**Feito:** colunas de largura fixa, mini-cards (id + prioridade + tags + título),
mover entre colunas via teclado (`H/L`) **ou arrastando com o mouse** (o card vira
um fantasma que segue o cursor + a coluna-alvo acende verde; solta = troca status +
`Save`), scroll por coluna com barra,
altura uniforme, navegação vim. **Falta:** reorder manual dentro da coluna
(precisa do campo `order` no frontmatter).

### F23 — Boards & abas `[v1]` 🟡
Vários boards, interface de **abas estilo navegador**. Cada card pertence a um
board via o campo `project` (**board = project**): o board agrupa as tasks cujo
project *efetivo* == id do board. Subtask herda o board do pai (herança do
`project`, F2).

- **Board = entidade** com arquivo `boards/<id>.md` (nome + created) — permite
  board vazio e nome de exibição. Boards também são **descobertos** dos projects
  em uso, então projects antigos viram abas sem precisar de arquivo.
- **Inbox** é o board embutido pras tasks sem project (id vazio); sempre existe.
- **Abas:** trocar aba, `+` cria board novo, fechar aba (some da barra, não deleta
  o board), e um **modal** lista todos os boards pra abrir.

**Feito:** barra de abas, filtro por board ativo, `+` cria board (arquivo) e abre
a aba, `^w` fecha, `tab`/`shift+tab` troca, `b` abre o modal de lista, criar task
cai no board ativo. Modais **flutuam sobre o board** (compositor do Lip Gloss v2).
Modais têm **barra de título estilo macOS** (traffic lights: vermelha acesa fecha,
amarela/verde apagadas) e **sombra** (flutuam acima do board) — padrão comum a
todos via `modalBox`. Abas abertas **persistem** entre sessões (`state.yml`).

**Mouse:** clicar na aba troca, `✕` na aba (ou botão do meio) fecha — o `✕` some
quando só há 1 aba; clicar no `+` cria; no modal, a bolinha vermelha fecha e a
barra de título arrasta; roda do mouse rola o detalhe. **Falta:** deletar/renomear
board, colunas customizadas por board.

### F6 — Listagem `[v1]`
Paginada, com drill-down na árvore (entra no nível do pai). Linha: título,
prioridade, due, projeto, tags (chip colorido), progresso dos filhos (2/5).
Ordenação (due, prioridade, criação, alfabética) e agrupamento (projeto,
prioridade, prazo).

### F7 — Filtros & views `[v1]`
Filtrar por campo: status, prioridade, projeto, tag, due, parent. Filtros
**combináveis** (ex: projeto=casa E prioridade=alta). Views prontas: hoje,
semana, sem prazo, alta prioridade, sem projeto. Busca por texto em título/notas.

### F8 — Labels `[v1]`
Tag = label de primeira classe: nome + cor. CRUD: criar, renomear, recolorir,
mesclar, remover (e o efeito nas tasks que a usam).

### F9 — Prioridade `[v1]` 🟡
Níveis com cor. (Fixa low/normal/high vs customizável → Decisões em aberto.)

**Feito:** exibida com cor (● high/normal, ○ low) no card e badge no detalhe.
**Falta:** setar/alterar pela UI e níveis customizáveis (F13).

### F10 — Ações em massa `[v1]`
Seleção múltipla: marcar done, mover, taggear várias de uma vez.

### F11 — Storage git-backed `[v1]`
`.md` versionados → histórico por task, undo = `git revert`. Base pro sync (F14).

### F12 — CLI + `--json` `[v1]`
`hakuban add/list --json`. Scriptável e legível por ferramentas externas.
É o que habilita a integração Claude Code (F15).

### F13 — Configurações `[v1]` 🟡
Menu de settings (e arquivo de config no data dir, editável na mão). Opções pro
usuário ajustar o comportamento — várias "decisões em aberto" viram config com
default em vez de escolha travada.

**`config.yml`** vive no data dir ao lado do `state.yml`, mesmo padrão do
`state.go` (struct + yaml + `atomicWrite`). Defaults sãos: o arquivo pode não
existir e tudo funciona.

```yaml
theme: omni              # nome do tema (omni pessoal + 19 dos upstreams) — vivo (paleta semântica)
lang: pt-BR              # pt-BR | en-US | zh-Hans        — vivo (catálogo i18n)
preview_pane: false      # painel do .md no rodapé, segue o cursor — vivo (previewPane)
date_format: "2006-01-02"# layout Go                     — vivo hoje (render do due)
editor: ""               # "" = $EDITOR do ambiente      — F3 (shell-out)
confirm_delete: true     #                               — F2 (delete na UI)
priorities: [low, normal, high]  # níveis                — F9
# colors / priority_colors: submenus de paleta           — fase 3
```

**Menu (`modeSettings`, tecla `s`):** modal reusando `modalBox` + navegação
estilo `boardListBox`/`updateBoardList`. Cada linha é `Label: valor`. `j/k`
move, `←→`/`space` cicla toggle/enum, `enter` entra em submenu ou edição inline
(reusa `textinput`). Persiste no `config.yml` a cada mudança. Três tipos de
controle cobrem quase tudo — `toggle` (bool), `cycle` (enum), `edit` (texto);
`submenu` (cores, níveis, rebind) é o 4º, mais pesado.

- **Data dir selecionável, via ponteiro fora do config.** O `config.yml` mora
  dentro do data dir, então não pode guardar o próprio caminho (circular) — o
  caminho fica num ponteiro `<os.UserConfigDir>/hakuban/root.yml`
  (`internal/task/root.go`). Boot resolve `env HAKUBAN_TASK_DIR > root.yml >
  ~/.hakuban` (`ResolveDir`). No menu, a linha "Diretório de dados" abre um
  **navegador de pastas estilo yazi** (`modeDirBrowser` — Miller columns
  pai·atual·preview, barra de seleção no accent, ícones de pasta nerd-font
  coloridos pelo tema, filtro live com `/`); ao escolher um destino diferente,
  pergunta se quer **mover** os arquivos (`modeConfirmMove`): sim move
  (`MoveData`, via `os.Rename`, **aborta se o destino já tiver dados** — nunca
  mistura; cross-filesystem ainda não suportado), não só reaponta. Depois
  regrava o ponteiro e reabre a store no dir novo.
- **Sem seção de listagem** (densidade/ordenação/agrupamento/filtro padrão):
  pressupunham a F6/F7 que não existem — cortadas até fazerem sentido.
- **Tema = paleta semântica, não detecção.** `internal/tui/themes.go` tem uma
  struct `palette` (tokens accent/superfícies/overlays/hues, vocabulário do Catppuccin) e
  os temas: **`omni`** (pessoal do usuário — Rocketseat, do
  seu próprio config; default) + 19 dos upstreams (Catppuccin, Tokyo
  Night, Dracula, Nord, Gruvbox, One, Solarized, Kanagawa, Rosé Pine, Vesper +
  variantes light, e um fallback `terminal` 16-cores que herda o fundo). `applyPalette` reconstrói
  todos os estilos do TUI a partir do tema ativo — o tema afeta a UI inteira,
  não só o Glamour. **Sem `auto`/detecção:** o tema é escolha explícita;
  `isDark` do tema decide o estilo do Glamour. TUI não pinta o fundo do
  terminal — dark/light = valores de cor com contraste, não uma tela pintada.

**Fatiamento:**
1. Plumbing do `config.yml` + menu navegável + os vivos (tema, `date_format`,
   `preview_pane` — painel do `.md` no rodapé que segue o card sob o cursor,
   reusando o render de notas/badges do detalhe; peek sem scroll, full view no
   Enter).
2. Toggles/edits inertes mas persistíveis (`confirm_delete`, `editor`) —
   viram efetivos quando F2/F3 chegarem.
3. Submenus pesados (níveis de prioridade, rebind de teclas, e *overrides* de
   cor por cima do tema) — cada um é uma miniatura de feature; entram por
   último. A paleta/temas base já estão feitos (fatiamento 1).

Ponytail: config é arquivo simples no data dir; o menu só edita esse arquivo.

### F14 — Sync entre máquinas `[v2]`
Formalizar o data dir git-backed (F11) — `push`/`pull` como sync.

### F15 — Integração Claude Code `[v2]`
Ler/criar tasks via `--json` (F12) / MCP.

### F16 — Integração com Jira (e afins) `[v2]`
Integração com trackers externos (Jira primeiro; GitHub/Linear/… reusam a mesma
costura) **por composição, não por cliente HTTP no core**. O binário nunca fala
rede: ele expõe ganchos e dispara **comandos configuráveis do usuário** (a F22),
que fazem o REST por fora (bash + `gum`, ou o Claude Code). Regra de ouro
preservada: **o core continua offline; o disco continua a fonte da verdade.**

**Coluna vinculada (pull, read-only).** Uma coluna do board pode ser *local*
(default — seus `.md`, editáveis, card novo nasce aqui) ou *bound* a uma **JQL**.
Um comando de **sync** (configurável) roda a JQL e **materializa `.md` espelhos**
num cache derivado (dir gitignored, `source: jira`, `jira: PROJ-123`) com o
`status` = a coluna. A TUI relê o disco ~1s e mostra como card normal.

**Botão de coluna (a coluna diz o que ele faz).** No **rodapé da coluna** mora um
controle configurável (par do ⚙ do topo): a coluna declara `buttons:` — cada um com
`label`, `icon` e **um** `cmd` (script, recebe os cards da coluna em JSON no stdin) ou
`agent` (intenção). Um botão → o clique (ou a tecla, com o cursor na coluna) dispara
direto; **vários** → o rodapé vira `label ▾` e abre um **popover** ancorado no botão,
crescendo pra cima e pra direita, com a 1ª opção na linha do próprio botão. `batch: true`
alista o botão no botão board-wide da barra de topo, que roda os marcados um de cada vez.

**Botão da barra de topo (mesma estrutura, escopo board).** O board declara os seus em
`buttons:` no topo do arquivo: `cmd:` recebe **todos** os cards do board no stdin, `agent:`
é intenção, e `batch: true` significa "rode os botões `batch` das colunas". Um botão dispara
direto; vários abrem o mesmo popover, que aqui **desce** do botão (invariante: o primeiro
botão sempre encostado no anchor). Loader de `cmd`/`agent` toma o lugar do próprio botão —
`batch` mostra no rodapé de cada coluna, uma por vez. Board sem `buttons:` sintetiza o
"Sincronizar tudo" a partir dos botões `batch` das colunas; `sync_button:` segue mandando na
posição (`top-left`/`top-right`/`off`).
Enquanto roda, o botão vira o **loader de progresso**; ao terminar mostra ✓ ou `:(`. O
**estilo do loader é por coluna**: `loader:` escolhe os glifos (`segments` default, `blocks`,
`line`, `dots`, `ascii` — nome desconhecido cai no default, nunca quebra o render) e
`percent: false` esconde o número. Vale também pro `%` persistido nos cards daquela coluna,
pra card e botão falarem a mesma língua. Sync é só **um caso de uso** disso: coluna bound sem `buttons:` sintetiza
o botão de sync a partir do `sync:` do board (compatibilidade). Coluna sem botão nenhum tem
o rodapé vazio. Além do clique, o primeiro botão de batch roda opcionalmente ao abrir o
board (`sync_on_open`). Os espelhos
são **cache descartável** (como `state.yml`/o grafo), não dados do usuário — por
isso "disco é a verdade" continua valendo: a verdade daquele card é o *Jira*, o
`.md` é só a projeção local. Conteúdo (título/notas/campos) é **read-only**; a
**posição na coluna, não** — mover é uma *ação*, não uma edição (ver abaixo).

**Ações de coluna (entrada/saída).** Cada coluna carrega, opcionalmente, uma ação
`on_enter` e uma `on_exit` — cada uma "nada" (default) ou um **comando do usuário**.
Mover um card **A → B** dispara a ação de saída de A e depois a de entrada de B
(normalmente um dos lados é "nada", então não conflita). É aqui que a escrita
acontece — sempre no *script*, nunca no core:
- Local **DRAFTS → TO-DO** (bound): `on_enter` de TO-DO = script cria a issue no
  Jira, carimba `jira: PROJ-456` no `.md`, e o card vira espelho.
- **TO-DO → DOING** (ambas bound): entrada de DOING = script transiciona a issue
  no Jira. (Ou pendura na saída — o usuário escolhe o lado.)

**Loader de progresso real.** A ação roda como **subprocesso em background** (TUI
nunca trava). O card mostra uma **barra de progresso** movida pelo próprio script
via stdout (protocolo na F22). **Sucesso** (exit 0) → barra cheia + ✓ e o move
**commita** (escreve o `.md`). **Falha** (exit ≠0) → `██░░░░ :( motivo` (última
linha do stderr), o card **fica na origem** e **nada** é escrito. O estado do
loader é **em memória, descartável**: fechar o app no meio = card volta pra
origem, nada commitado. No render, o loader ocupa o slot de anotação do mini-card
(`↳ PAI`/barra de subtasks) — sem layout novo.

**Config (por board).** O que está vinculado mora em `boards/<id>.md` (cada board
= um projeto Jira): por coluna, `jql` + `on_enter`/`on_exit`. Auth **fora do
plain-text versionado**: token em env (`JIRA_TOKEN`) ou arquivo gitignored.

```yaml
# boards/pessoal.md
columns: [DRAFTS, TO-DO, DOING, DONE]
actions:
  TO-DO: { jql: "project=ABC AND status='To Do'",     on_enter: "~/.hakuban/hooks/jira-create.sh" }
  DOING: { jql: "project=ABC AND status='In Progress'", on_enter: "~/.hakuban/hooks/jira-move.sh" }
```

**Publicar (push) é um caso particular disto:** `on_enter` de uma coluna bound
publica o `.md` no Jira (title→summary, notas→description, priority→priority) —
zero código de REST no app, exatamente como a nota original previa.

#### Evolução (decisão 2026-07-21): ação = **intenção de agente**, não script

**Motivação.** A abordagem script (curl+jq+basic-auth) provou-se frágil no teste
real: `JIRA_URL` vs conta do token, `.io` vs `.ai`, endpoint de busca
descontinuado, 404 por permissão/identidade. Enquanto isso, as transições feitas
**por um agente via MCP saíram de primeira** — porque o agente usa uma conexão
já autenticada (zero token, endpoint, conta). Decisão: `on_enter`/`on_exit`
passam a ser uma **descrição em Markdown do que o agente deve fazer**, não um
caminho de script. O core continua offline: ele só **entrega a intenção + o card
(`--json`) + `from`/`to`/board** a um agente e mostra progresso/resultado. Quem
sabe de Jira é o agente, via as ferramentas dele.

```yaml
# boards/<id>.md — ação = intenção (prosa); on_enter_cmd = script determinístico
agent_tools: mcp__claude_ai_Atlassian_Rovo   # --allowedTools do agente (só MCP, nunca disco)
actions:
  Finalizado:
    jql: 'project = DEMO AND status = "Concluído"'
    on_enter: |                               # intenção → agente headless
      Mova a issue {{.jira}} para o status "Concluído" no Jira.
    # on_enter_cmd: ~/.hakuban/hooks/x.sh  # alternativa determinística (F22)
```

Campos interpolados na intenção: `{{.jira}}`, `{{.title}}`, `{{.id}}`, `{{.status}}`,
`{{.priority}}`, `{{.project}}`, `{{.notes}}`. O card inteiro também vai em JSON no
prompt (bloco `<card>`), então o agente tem todo o contexto.

**Como o agente é chamado — 2 modelos (o core não roda LLM):**

1. **Headless por move (tempo-real, com loader).** O move dispara
   `claude -p "<intenção + card>"`. **Bloqueio encontrado (2026-07-21):** o CLI
   mostra *"claude.ai connectors are disabled because ANTHROPIC_API_KEY … takes
   precedence"* — ou seja, o `claude -p` **não enxerga o MCP do Atlassian** com o
   setup atual. Pra viabilizar: `unset ANTHROPIC_API_KEY` + login claude.ai (aí
   `claude mcp list` deve listar o Atlassian), **ou** configurar um MCP de
   Atlassian local pro headless (credencial na camada do MCP, uma vez). Validar
   antes de construir: rodar um `claude -p` que transiciona o DEMO-1.

2. **Fila + agente na sessão (funciona hoje, token-free).** O move **não executa
   nada na hora** — o core grava `outbox/<ts>_<card>.md` (a intenção MD + o card
   em json + from/to) e marca o card como **⏳ pendente**. Numa sessão do Claude
   Code, o usuário pede "processa a fila" e o agente executa via MCP (a conexão
   viva), carimba o resultado e limpa a fila.

**Plano de retomada (nesta ordem):**
- [x] Testar o modelo **1** (headless) — **validado 2026-07-21**: sem
  `ANTHROPIC_API_KEY` (a key desliga os connectors), o `claude -p` transiciona o
  DEMO-1 via MCP na assinatura claude.ai. É o caminho (tempo-real). Modelo **2**
  (fila/`outbox`) fica arquivado como fallback, não implementado.
- [x] **Fatia A — lado da escrita (transição).** No core, `on_enter`/`on_exit` são
  **intenção** (prosa MD, interpolada com `{{.jira}}` etc.) entregue ao agente
  headless; `on_enter_cmd`/`on_exit_cmd` seguem como script determinístico (F22).
  Recipe travado: cwd neutro (o agente não explora o projeto), `env -u
  ANTHROPIC_API_KEY`, `--allowedTools <board.agent_tools>` (só MCP, nunca disco),
  `--output-format stream-json` → cada `tool_use` vira passo do loader. Commit só no
  exit 0. (`runAgent`/`interpolate` em `internal/tui/action.go`; `agent_tools` no board.)
- [x] **Fatia B — lado do pull (sync) + sync-on-open.** Sync via agente, **board-wide
  numa chamada só**: o core sintetiza a intenção (OR das JQLs das colunas bound), o
  agente **busca via MCP e devolve as issues em JSON**, e o **core escreve os
  espelhos** (`ReconcileMirrors`: mapeia status-do-Jira → coluna via `ColumnAction.
  Status`, reconcilia, some com stale). Disco fica no core; agente só toca MCP. Flag
  `sync_on_open` no board dispara no `Init()` → card vinculado cai na coluna do status
  atual do Jira (não se trabalha em cima de card já concluído por outro). Script antigo
  (`sync:`) segue como escape hatch determinístico. Verificado: pull real do DEMO
  devolveu JSON parseável (mesmo com cercas ```json).

**Estado atual (branch `feat/integracao-jira`):** Fatias A e B prontas e verificadas
contra o Jira real (transição + pull do DEMO). Ação de coluna = **intenção de agente**
(`on_enter`/`on_exit` prosa; `*_cmd` = script determinístico F22); sync = **agente
board-wide** com o core escrevendo os espelhos (`sync:` = script legado). Config do
board: `agent_tools`, `sync_on_open`, `jql`+`status` por coluna. Loader real via
stream-json (cada tool_use vira passo). Falta amadurecer: stamping de issue CRIADA via
agente (hoje só transição/pull), e outros trackers reusando a mesma costura.

### F17 — Dependências entre tasks `[v2]`
"Bloqueada por X" — relação diferente de pai/filho (ordem, não decomposição).
Habilita "o que posso fazer agora".

### F18 — Estados extras `[v2]` 🟡
Além de pending/done: em-progresso, bloqueada, esperando, cancelada.

**Feito:** `doing` (em-progresso) já existe como coluna do board. **Falta:**
bloqueada, esperando, cancelada.

### F19 — Start/defer date + snooze `[v2]`
Task só aparece a partir de tal dia; adiar prazo com uma tecla.

### F20 — Visões `[v2]` 🟡
Agenda/calendário por data, kanban por status, stats (feitas na semana, atrasadas).

**Feito:** kanban por status (é a F6b). **Falta:** agenda/calendário e stats.

### F21 — Suporte a idiomas (i18n) `[v1]` 🟡
Toda string da UI vem de um catálogo de mensagens (não hardcoded). Idioma
escolhido na config (F13). Transversal — nasce no v1 pra não ter que reescrever
toda a UI depois.

**Feito:** `internal/tui/i18n.go` tem uma struct `messages` (todas as strings da
UI) + catálogo por idioma; instância ativa global (`msg`) trocada por
`applyLang`, mesmo padrão da paleta. Idiomas: **pt-BR** (default), **en-US** e
**zh-Hans** (chinês simplificado). Trocável no menu de settings (linha "Idioma",
cicla). `statusLabel`/`priorityLabel` traduzem as chaves de dados
(backlog/high/…) sem mexer no que fica salvo no `.md`. Colunas, badges,
prioridade, ajudas, títulos de modal, placeholders e metadados vêm do catálogo.
**Falta:** só adicionar mais idiomas (= nova entrada em `langs` + `langNames`).

### F22 — Hooks & extensibilidade `[v1]` — coração do projeto
Eventos disparam scripts do usuário, que recebem a task como `--json` no stdin
(ex: `on-done`, `on-create`, `on-edit`). Extensão infinita sem plugin API: o
usuário liga notificação, sync, publicação, integração — o que quiser — por fora
do binário, usando `gum` e outras ferramentas nos próprios scripts. v1 entrega o
mecanismo básico (eventos + `--json`); catálogo amplo de eventos e comandos
custom mapeados a teclas amadurecem depois.

**Ações de transição (a costura que a F16 usa).** O evento mais rico é o
`on-move`, modelado como **ações por coluna** (`on_enter`/`on_exit`, ver F16): o
core dispara o comando configurado passando o card em `--json` no stdin +
contexto (`from`/`to`/board via flags/env). O comando roda como **subprocesso em
background** — a TUI nunca bloqueia (padrão Bubble Tea: escuta um canal → vira
`msg` → re-escuta). É genérico: Jira é só o primeiro consumidor; a mesma costura
serve notificação, git, webhook, o que for.

**Protocolo de progresso (stdout streaming).** Pra dar barra de progresso *real*
sem o core saber nada da tarefa, o script transmite o progresso — uma atualização
por linha no **stdout**:
- `N` → porcentagem 0–100, **ou** `N/M` → passo N de M (core calcula a %);
- texto opcional depois vira o rótulo ao lado da barra (`echo "2/3 criando issue"`).

O core lê cada linha → `actionProgressMsg{id, pct, label}` → redesenha o card. **exit
0** = sucesso (barra cheia + ✓, commita o move); **exit ≠0** = falha (`:(` + última
linha do **stderr** como motivo, sem commit). **stderr** = logs. Estado do loader é
em memória e descartável. É a parte mais pesada da extensibilidade (subprocesso +
parser de stream), mas construída **uma vez** e reusada por toda integração.

## Não-metas (armadilhas — não construir cedo)

- Daemon de notificações/lembretes (usar `hakuban due` + cron do SO).
- Editor de texto próprio (usar `$EDITOR`).
- Engine de busca full-text (usar ripgrep/fzf sobre os `.md`).
- Recorrência de tasks (poço de casos-limite; só se virar dor real).
- Time tracking, multiusuário, contas.

## Layout de dados (proposta)

```
~/.hakuban/          # data dir (padrão; selecionável — ver F13)
  tasks/        # <id>.md — tasks ativas
  boards/       # <id>.md — boards nomeados/vazios (F23)
  config.yml    # preferências (tema, idioma, etc. — F13)
  state.yml     # abas abertas + ativa (UI, derivado/descartável)
  archive/      # done arquivadas
  jira/         # <id>.md — espelhos read-only do Jira (F16); cache derivado, gitignored
  hooks/        # scripts do usuário (on_enter/on_exit/sync — F16/F22)
  .git/         # histórico + sync (opcional, ativável)

<os.UserConfigDir>/hakuban/root.yml   # ponteiro pro data dir (fora dele; F13)
```

## Decisões em aberto

- *Defaults* de config (F13): filtro padrão da lista, prioridade fixa vs
  customizável, preview na listagem — decidir ao implementar.
- **F16 — integração (assumido, revisitar ao implementar):** binding/ações ficam
  **por board** (`boards/<id>.md`), não globais no `config.yml`. Quando o sync
  dispara: manual (tecla, atrás do prefixo) + opcional ao abrir o board. Se um dia
  a config global fizer mais sentido (um projeto Jira compartilhado por vários
  boards), migrar mantendo o board como override.
