# Roadmap — features desenhadas (ainda não 100% implementadas)

Decisões de design já conversadas. Ordem pensada: **chaves → subtask → dependência
→ coluna-objeto (categoria/Jira)**. Cada uma deixa a próxima mais legível.

## 1. Chaves legíveis por board (`KEY-NN`) — EM IMPLEMENTAÇÃO

Substitui o id aleatório (`4kjjy4`) por chave incremental por board, estilo Jira.

- Board ganha `key` (ex: `CASA`), **editável na config**, default derivado do nome
  ("Casa de férias" → `CASA`).
- Card id = `<KEY>-<NN>`, e **o id é o nome do arquivo** (`tasks/CASA-01.md`) — as
  referências no `.md` (`parent`, futuro `blocked_by`) ficam legíveis. Mín. 2
  dígitos (`CASA-01`).
- **Próximo número = `max(KEY-* existentes) + 1`** (derivado, sem contador salvo;
  ponytail). Trade-off aceito: apagar o mais alto pode reusar o número.
- **Chave única entre boards** (validada). Motivo: `tasks/` é plano e o id é o
  arquivo → `KEY-NN` tem que ser globalmente único; e referência de dependência
  cruza boards, então precisa ser inequívoca.
- **Rename da chave** = cascata: renomeia os arquivos do board + reescreve as
  referências (`parent`, e no futuro `blocked_by`) em todos os boards.
- **Número do card nunca é renomeado** (âncora estável, igual Jira).
- Migração de dados antigos: **não** — vamos zerar o data dir e começar do zero.

## 2. Subtask (hierarquia pai/filho) — IMPLEMENTADO

Modelo (`Task.Parent`, `Store.Children`, herança da raiz) + view.

- `S` num card cria uma subtask (filho); o id sai da chave do board do pai
  (`K-02`), Parent liga na raiz, nasce na 1ª coluna.
- Filho **aparece no board** na coluna do próprio status, com a tag `↳ PAI`
  (ex: `↳ K-01`). Card pai mostra **progresso** `▓▓▓░░ 3/5` (barra + contagem).
- Detalhe do pai lista as subtasks (`✓`/`▢`).
- **"Done" = última coluna** (heurística `doneCol`); vira categoria de coluna
  quando o item 4 chegar. Progresso conta **filhos diretos**.
- Pendente: aninhamento profundo, cor de borda ligando pai/filhos, criar subtask
  também de dentro do detalhe.

## 3. Dependência (bloqueia / bloqueado por) — SEPARADO de subtask

Relação nova, **não** é pai/filho. Campo novo no frontmatter:

- `blocked_by: [CASA-02]` no card bloqueado; o "desbloqueia" é **derivado** (quem
  lista este id no seu `blocked_by`), igual `Children` deriva de `Parent`.
- Pode **cruzar boards**.
- **Comportamento: só mostra** (informativo) — selo 🔒 no mini-card enquanto há
  bloqueador pendente; no detalhe "bloqueado por / desbloqueia".
- **Não trava** o "done" por ora — porque não existe definition of done (coluna é
  texto solto). Travar fica pra depois (depende do item 4).

## 4. Coluna vira objeto (categoria / Jira / config por lane) — futuro

Hoje coluna é `string`. Três features querem a mesma promoção pra objeto:

```yaml
columns:
  - {name: "A fazer", category: todo}
  - {name: "Feito",   category: done}   # futuro: jira_status_id: 10001
```

- **Categoria** `todo|doing|done` (= status category do Jira) dá o *definition of
  done* sem máquina de estados. Opcional: board sem coluna `done` simplesmente não
  tem conceito de conclusão.
- Habilita: progresso por "filhos em done", travar-done da dependência,
  mapeamento Jira, e a config por lane (engrenagem ⚙).
- **Máquina de estados nível 2** (regras de transição entre colunas): **descartada
  por ora** (YAGNI; briga com "hackeável, config opcional" do produto). Só se
  houver dor concreta.

## Integração Jira (norte)

As chaves (`KEY-NN`), a categoria de coluna e a config por lane (⚙) convergem pro
mapeamento com o Jira: chave ↔ issue key, categoria ↔ status category, lane ↔
status. O ponto de extensão na UI é `updateLaneConfig`/`laneConfigBox`.
