---
sidebar_position: 4
title: Contrato do hook
---

# Contrato do hook

Dois campos disparam na mesma transição, então parecem intercambiáveis. Não são.

|  | `script` · `_cmd` | `instrução` · `on_enter` |
|---|---|---|
| comportamento | determinístico — você lê exatamente o que faz | não-determinístico — você não sabe o que vai fazer |
| **capacidade** | **ilimitada** — não existe modelo de permissão em `bash` | **limitada** — cabe numa linha: `agent_tools` |
| revisão escala? | não — ninguém revisa 93 invocações de `jq` | sim — um teto de capacidade é uma lista curta |
| portabilidade | amarra a um filesystem, um binário, um `$HOME` | não amarra a nada: descreve intenção |

**Doutrina do projeto:** a instrução é o formato; `_cmd` é dívida — um escape hatch para
controle exato, ao custo de um quadro que mais ninguém consegue rodar.

Para compartilhar um quadro, o que importa não é o que o arquivo *diz* — é o que ele *pode
fazer*.

## O contrato do script

O Hakuban roda `sh -c "<a linha de comando>"` com o data dir como diretório de trabalho,
então um caminho relativo (`hooks/notify.sh`) resolve de onde quer que o binário tenha sido
lançado. É uma linha de comando, então **hooks são agnósticos de linguagem**: bash, Python,
Node, Ruby, um binário compilado — qualquer coisa que honre os quatro canais abaixo.

### stdin

O card e a transição, como um objeto JSON:

```json
{
  "id": "PES-12",
  "title": "Ship the export command",
  "status": "pending",
  "priority": "high",
  "project": "hakuban",
  "tags": ["cli"],
  "jira": "PES-12",
  "notes": "Repo: wagnerlim/hakuban\n",
  "from": "Ready",
  "to": "Doing",
  "board": "my-board"
}
```

### stdout

Duas linhas de controle, todo o resto é ignorado:

| linha | efeito |
|---|---|
| `N label` | `N` é 0..100 — define a barra de progresso e seu rótulo |
| `N/M label` | passo `N` de `M` — mesma barra, calculada como `N*100/M` |
| `jira: KEY` | carimba `KEY` no card como chave externa; o card vira um espelho |

Linha que não começa com número não é linha de progresso, então um log solto no stdout
nunca vira barra.

### stderr

Bufferizado. **A última linha não vazia se torna a mensagem de erro** exibida a quem moveu
o card. Escreva o motivo ali, não no stdout.

### código de saída {#exit-code}

`0` commita o movimento. Qualquer outro valor **aborta**: o card fica onde estava e o
motivo do stderr aparece. Igual se um humano arrastou ou se um agente chamou
`hakuban move`. Hook que falha é em geral dado faltando, não erro transitório — leia a
mensagem antes de tentar o movimento de novo.

### ambiente

Em cima do ambiente do processo:

| variável | valor |
|---|---|
| `HAKUBAN_FROM` | coluna que o card deixou |
| `HAKUBAN_TO` | coluna em que está entrando |
| `HAKUBAN_BOARD` | id do quadro |
| `HAKUBAN_STATUS` | o `status:` da coluna de destino |
| `HAKUBAN_DIR` | o data dir (`~/.hakuban` por padrão) |
| `HAKUBAN_TASK_BIN` | caminho do binário em execução, para um hook que queira chamar de volta |

## O mesmo hook, duas vezes

Bash:

```bash
#!/usr/bin/env bash
# Refuses the move unless the notes carry a repository.
set -euo pipefail

card=$(cat)
repo=$(jq -r '.notes' <<<"$card" | sed -n 's/^Repo: *//p' | head -1)

if [ -z "$repo" ]; then
  echo "no 'Repo: owner/name' line in the notes" >&2
  exit 1
fi

echo "1/2 cloning $repo"
gh repo clone "$repo" "$(mktemp -d)" >/dev/null 2>&1
echo "2/2 ready"
```

Python — mesmo contrato, sem `jq`:

```python
#!/usr/bin/env python3
"""Refuses the move unless the notes carry a repository."""
import json, re, sys

card = json.load(sys.stdin)
match = re.search(r"^Repo: *(\S+)", card.get("notes", ""), re.M)

if not match:
    print("no 'Repo: owner/name' line in the notes", file=sys.stderr)
    sys.exit(1)

print(f"1/2 cloning {match.group(1)}", flush=True)
# ... do the work ...
print("2/2 ready", flush=True)
```

Ligue qualquer um dos dois do mesmo jeito:

```yaml
actions:
  Doing:
    on_enter_cmd: hooks/require-repo.py
```

## O contrato da instrução

Uma instrução não tem stdin, stdout nem código de saída para honrar — ela é prosa. O que
ela tem é um teto:

```yaml
actions:
  Done:
    on_enter: |
      Write a short update from the card notes and send it
      to the address in the frontmatter.
agent_tools: Read, mcp__gmail__send
```

`agent_tools` é o modelo de permissão inteiro, e é uma linha. O agente roda headless, com
um diretório de trabalho neutro para que não saia explorando o projeto em que você por
acaso esteja sentado. Ela falha do mesmo jeito que um script: o movimento é abortado e o
motivo aparece.

## Windows

A TUI e o formato do quadro funcionam. Hook por instrução funciona. **Hook por script
não** — `sh` não existe lá, e o Hakuban não embarca um shim. Se você está no Windows, a
instrução não é o formato preferido, é o único.
