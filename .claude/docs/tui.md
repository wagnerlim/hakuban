# Camada TUI (Bubble Tea v2)

`internal/tui/tui.go` é o `Model` do Bubble Tea. Só uma view — o disco é a fonte
da verdade (`refresh` relê a cada mudança).

## Modes

Um enum `mode` dirige `Update` e `render`. O `Update` despacha por `m.mode`; o
`render` desenha conforme o `m.mode`.

**Adicionar um modal/tela nova** = 4 pontos:
1. novo valor no enum `mode`;
2. case no dispatch de `KeyPressMsg` dentro de `Update`;
3. case no `render()`;
4. entrada em `inModal()` (se for modal sobreposto ao board).

## Modais (overlay)

`overlay(box)` compõe com o compositor do Lip Gloss (v2): board ao fundo (z0),
sombra (z1), box (z2), bolinhas macOS (z3). Modais são **arrastáveis** pela barra
de título (`onModalHandle`) e fecham na bolinha vermelha (`onModalClose`).
`modalBox(title, body, help)` é a moldura padrão.

**Centralização (regra):** todo painel novo abre centralizado. Não espalhe
`modalPlaced=false` em cada transição — é fácil esquecer (já aconteceu). O
`render()` faz isso central: quando `m.mode != m.renderedMode`, zera
`modalPlaced` (o `overlay()` re-centraliza na próxima medição) e atualiza
`renderedMode`. Assim qualquer troca de painel re-centraliza, e arrastar preserva
a posição enquanto o modo não muda. Um modal de tamanho diferente reusando a
posição do anterior aparece deslocado — este mecanismo evita isso.

## Hitboxes (mouse)

O `render` **registra** as regiões clicáveis; os handlers de mouse **leem**:
`tabRegions` (abas), `cardRegions` (cards), `gearRegions` (☰ da lane),
`syncRegions` (botão de rodapé por coluna), `syncAllReg` (botão do batch na barra
de topo), o menu de coluna (`menuX/menuY/menuW/menuRows` → `menuHit`, popover
ancorado no botão, sem geometria de modal), o link do id no detalhe (`detailLinkY`/`detailLinkX0..X1`, abre o issue no
browser) e os botões da confirmação (`confirmNo`/`confirmYes`). Padrão:
- coordenadas em tela;
- dentro de modal, guarde offset **relativo** ao topo-esquerdo e some
  `modalX/modalY` no clique (funciona mesmo com o modal arrastado);
- conteúdo dentro de um **viewport** (detalhe): a linha de conteúdo `i` vira
  `i - vp.YOffset()` na tela, **e** some o topo do box que embrulha (borda+padding
  = `+2`, ver `detailRows`/`detailLinkLine`). Errar esse offset = hitbox deslocado.

## Scroll do board

- **Vertical** (por coluna): `windowColumn` recorta os cards numa janela de
  `bodyH` linhas; `vscroll` desenha a barra.
- **Horizontal** (do board): offset em **CÉLULAS** (`m.hOff`), não em índice de
  coluna. Renderiza todas as colunas e recorta a janela `[hOff, hOff+viewW)` com
  `ansi.Cut` (preserva ANSI) → desliza suave. `hscroll` desenha a trilha
  (arrastável pelo mouse, `scrollTrackTo`). `colAt(x)` soma `hOff` pra mapear
  x→coluna. Teclado move o cursor e `ensureColVisible` desliza o mínimo.

## UX de mouse (convenção) — ponteiro + interação

O mouse é cidadão de primeira classe: o teclado é o caminho canônico e o mouse
**espelha**. Ao criar/alterar qualquer superfície interativa, siga estes padrões.

**Forma do ponteiro** (`pointerShape`; emite OSC 22 via `ansi.SetPointerShape` só
quando a forma muda — senão spammaria a cada motion). Regra por afeto:
- **arrastável** → `ptrGrab` no hover, `ptrGrabbing` enquanto arrasta. Ex.: card,
  trilha de scroll, título de modal.
- **clicável** (age num clique) → `ptrPointer`. Ex.: ☰, botão sync, abas, linhas
  de menu/config/filtro/picker, link do id no detalhe.
- **input** focado → `ptrText`. Resto → `ptrDefault`. Restaure `default` ao sair.

**Interação (mesma semântica em todo lugar):**
- **1 clique = seleciona** (move cursor/realce pra linha/card), não age.
- **2 cliques = ativa**, equivale a Enter. Sem contagem nativa no Bubble Tea v2 →
  detecção por tempo (`isDoubleClick`, janela `doubleClickWindow`).
- **arrastar** = press + motion (card entre colunas, trilha, modal). Clique sem
  mover não arrasta.
- Menus passam pelo caminho **genérico**, não duplique a ação: `currentMenu`
  (geometria: nº de linhas + `headerLines`) → `menuRowAt` (linha sob o cursor) →
  `clickMenuRow` (seleciona; no double-click **replaya Enter** no `updateXxx` do
  modo). Superfície de menu nova = só mais um `case` nesses três.
- **Link externo** (id do card → abre o issue no tracker): 1 clique abre via
  `openURL` (`open`/`xdg-open`/`rundll32`), sem modificador. NÃO use hyperlink OSC 8
  — o viewport pode engolir e o Ghostty exige ⌘+clique; o clique-próprio é o "clico
  e vai". A URL vem de um template do board (`issue_url`, `{key}`), agnóstico.

**Hitbox = o texto, não a linha.** Limite a região ao alvo em X **e** Y
(`detailLinkX0..X1`), senão clicar no vazio ao lado dispara. x0 = borda+padding do
box (`+3`) + largura do label; x1 = x0 + largura do texto (`lipgloss.Width`).

**Checklist de superfície clicável nova:**
1. `render` registra a hitbox (tela, ou offset relativo ao modal + `modalX/Y`);
2. handler de mouse lê e age;
3. `pointerShape` devolve a forma certa no hover;
4. garanta hover: veja Mouse mode.

## Mouse mode

`pointerShape`/hover dependem de `mouseX/mouseY`, que **só** atualizam com evento
de motion. `MouseModeCellMotion` (padrão) reporta motion **apenas com botão
pressionado** → não há hover. `MouseModeAllMotion` reporta motion sem botão →
hover funciona. `View()` liga all-motion no board, na confirmação de excluir, no
detalhe (link do id) e em **todo modo de menu** (via `currentMenu`). Precisa de hover
num mode novo? Garanta que ele caia no all-motion — senão o ponteiro nunca muda (bug
clássico: clicável, mas sem cursor indicando). Nota: o **clique** funciona em
cell-motion (é press, não motion); só o **hover** (cursor) exige all-motion.

## i18n (`i18n.go`)

- **Toda string visível** é campo do struct `messages`, com uma instância por
  idioma em `langs` (pt-BR default, en-US, zh-Hans). `msg` é o catálogo ativo,
  trocado por `applyLang`.
- Adicionar string = campo no struct **+ entrada nos 3 idiomas**.
- Chaves de dados (status/prioridade) traduzem por `statusLabel`/`priorityLabel`,
  que devolvem a chave crua se desconhecida (colunas custom aparecem como o nome).

## Temas (`themes.go`)

Paleta ativa global `pal`, aplicada por `applyPalette`. Estilos (bordas, badges,
colunas) derivam de `pal`. Tema é trocável em runtime (picker) e tem variantes
claro/escuro (`isDark` alinha o Glamour). Cores novas → campo na paleta, não hex
solto na view.

## Armadilhas

- **`msg` é o catálogo i18n global**, mas handlers recebem `msg tea.KeyPressMsg`
  → sombreamento: dentro do handler `msg.algo` vira o teclado, não o catálogo.
  Extraia um método **sem** esse parâmetro (ex: `openBoardConfig`) pra acessar o
  catálogo.
- **Renomear coluna migra o `Status`** de todas as tasks da lane (senão viram
  órfãs). Ver `renameColumn`.
- Ao adicionar linha ao board (trilha, indicador…), **desconte de `bodyH`** pra
  não estourar a altura da tela.
- Estado de scroll/cursor reseta no `resetCursor` (troca de board) — inclua
  campos novos de janela ali.
