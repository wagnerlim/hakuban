---
sidebar_position: 5
title: Exemplos
---

# Exemplos

Exemplos, não features. **Nenhum destes vem com a ferramenta.** O quadro do autor entrega
tarefas a um agente que abre PRs — um fluxo entre N, e o seu não tem motivo para se
parecer com ele.

Tudo nesta página é CC0. Copie, mude, publique.

Cada bloco é um quadro inteiro: salve como `~/.hakuban/boards/<id>.md` e ele roda.

## Avisar o cliente

Uma instrução, e a única capacidade que ela recebe é ler e enviar e-mail.

```yaml
---
name: Client work
columns: [To-Do, Doing, Done]
actions:
  Done:
    on_enter: |
      Write a short update from the card notes, in the client's language,
      and send it to the address in the frontmatter. Do not invent status
      that is not in the notes.
agent_tools: Read, mcp__gmail__send
---
```

## Quebrar o card

O quadro remodela o próprio trabalho. `Refining` é uma coluna local: existe só aqui.

```yaml
---
name: Intake
columns: [Inbox, Refining, Ready, Doing, Done]
actions:
  Refining:
    guide: Nothing leaves here with an open question in the notes.
    on_exit: |
      If the notes describe more than one deliverable, create one card per
      deliverable and leave {{.id}} as the parent. If they describe one,
      do nothing.
agent_tools: Read, Write
---
```

## Rodar o checklist

Um script, porque "os testes passam" não deveria ser questão de opinião. Saída não-zero e
o card não sai de `Doing`.

```yaml
---
name: Library
columns: [To-Do, Doing, Review, Done]
actions:
  Doing:
    on_exit_cmd: hooks/checklist.sh
---
```

```bash
#!/usr/bin/env bash
# hooks/checklist.sh — tests, linter, coverage floor.
set -euo pipefail
cd "$(jq -r '.notes' | sed -n 's/^Path: *//p' | head -1)"

echo "1/3 tests"
go test ./... >/dev/null || { echo "tests failed" >&2; exit 1; }
echo "2/3 vet"
go vet ./... || { echo "vet failed" >&2; exit 1; }
echo "3/3 coverage"
pct=$(go test -cover ./... | grep -o '[0-9.]*%' | tr -d '%' | sort -n | head -1)
awk "BEGIN{exit !($pct < 70)}" && { echo "coverage $pct% below 70%" >&2; exit 1; }
```

## Passar para um agente

O quadro do autor. Card → implementação → PR aprovado em 9min18s em 29/08/2026 — real, e
ainda assim só um exemplo. `Ready → Doing` é o portão, e é um humano arrastando um card.

```yaml
---
name: Project board
columns: [To-Do, Refining, Ready, Doing, Validate, Review, Done]
actions:
  Ready:
    guide: To start work, move the card forward. Needs 'Repo: owner/name' in the notes.
  Doing:
    loader: implementing
    percent: true
    on_enter: |
      Clone the repository named in the notes of {{.id}}, implement {{.title}},
      run the tests until they pass, and open the PR. Stamp progress with
      `hakuban progress` as you go. Move the card to Validate when the PR is up.
  Validate:
    on_exit_cmd: hooks/require-pr.sh
agent_tools: Read, Write, Edit, Bash
---
```

O `hooks/require-pr.sh` recusa `Validate → Review` quando não existe PR para o card — o
portão que impede o quadro de mentir sobre o que foi entregue.

## Espelhando um tracker externo

O binário não fala HTTP. Um hook fala. O
[`examples/jira/`](https://github.com/wagnerlim/hakuban/tree/main/examples/jira) do
repositório tem o conjunto funcionando: um `sync` que materializa espelhos read-only a
partir de um JQL, um hook que cria a issue e reporta a chave de volta com `jira: KEY`, e um
que transiciona a issue para o status que a coluna representa.

```yaml
---
name: Team
key: TEAM
columns: [Backlog, Doing, Done]
sync: hooks/jira-sync.sh
sync_on_open: true
actions:
  Backlog:
    jql: project = TEAM AND status = "To Do"
  Doing:
    status: In Progress
    on_enter_cmd: hooks/jira-transition.sh
  Done:
    status: Done
    jql: project = TEAM AND status = Done
    on_enter_cmd: hooks/jira-transition.sh
---
```

Integração é composição. Isso é princípio, não limitação.
