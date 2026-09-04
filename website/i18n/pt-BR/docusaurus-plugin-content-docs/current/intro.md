---
sidebar_position: 1
title: O que é o Hakuban
---

# O que é o Hakuban

Um kanban no terminal onde seus agentes também trabalham — e onde você declara o que
acontece quando um card muda de coluna.

Cada card e cada quadro é um arquivo Markdown no disco, então uma IA lê e edita tudo do
mesmo jeito que você: é só texto. Colunas são estados, e cada estado é seu para definir —
rodar um script, mandar um e-mail, marcar um release, passar a tarefa adiante. O Hakuban
entrega o quadro; o fluxo é uma página em branco.

Reformulado para quem quer a versão de uma frase: **uma máquina de estados que mora no
disco, onde a transição executa um agente, com o escopo de ferramentas declarado no
próprio arquivo do quadro.**

> 看板 **kanban** — o quadro que te dão.
> 白板 **hakuban** — o quadro que você escreve.

## Veja funcionando

<video controls preload="metadata" style="width:100%;border:1px solid var(--border);border-radius:8px">
  <source src="/hakuban/demo.mp4" type="video/mp4" />
  Seu navegador não consegue tocar este vídeo.
</video>

Gravado pelo autor. Sem som.

## Instalação

```bash
go install github.com/wagnerlim/hakuban/cmd/hakuban@latest
```

Rodar `hakuban` sem subcomando abre a TUI e exige um TTY. Existem dois subcomandos
headless para que um agente conduza a esteira sem terminal aberto:

```bash
hakuban move <id> "<Coluna>"     # dispara on_exit da origem, depois on_enter do destino
hakuban progress <id> <0..100>   # a porcentagem exibida no card
```

Nome de coluna é comparação exata e case-sensitive.

## Pré-requisitos honestos

O binário em si não precisa de nada além do Go. **A demo de agente não roda numa máquina
limpa.** Ela precisa de:

- `jq`
- `gh`, autenticado
- uma assinatura paga do Claude Code (o hook por instrução chama `claude -p`)
- para os hooks do autor, o multiplexer [`herdr`](https://github.com/ogulcancelik/herdr)

O pitch é *"veja como se faz"*, não *"instale e use"*. Se você clonar isto esperando que a
gravação da home se reproduza sozinha, vai se decepcionar — e a culpa é da documentação,
não sua, que é justamente por isso que está escrito aqui.

## Onde mora cada coisa

```
~/.hakuban/
├── boards/<id>.md      o quadro: colunas + o que cada coluna faz
├── tasks/<ID>.md       um card: frontmatter + corpo markdown (as notes)
├── jira/<KEY>.md       espelhos read-only materializados por um hook de sync
├── archive/            cards encerrados
├── config.yml          tema, idioma, atalhos
└── state.yml           quais quadros estão abertos
```

Para trocar a raiz, use `HAKUBAN_TASK_DIR`.

O disco é a fonte da verdade. A store em memória é cache; a TUI relê o disco cerca de uma
vez por segundo, então um card que um agente move aparece na sua frente. A escrita é sempre
atômica (tmp + rename), porque existem dois escritores legítimos: a TUI e quem editar o
arquivo na mão — você, ou um agente.

## Fora de escopo

Estas são decisões, não omissões.

- **Não existe marketplace de quadros.** Um quadro auto-contido é um arquivo; o GitHub já
  faz hosting, busca e fork.
- **Não existe comando `import <url>`.** Importar é baixar o `.md` e colocar na pasta. O
  comando seria o vetor de execução de código de terceiro.
- **Não existe interface gráfica nem web.** Foi tentado, avaliado e cancelado.
- **Não é multi-usuário.** Sem servidor, sem auth. Local, single-player.
- **Ainda não é agnóstico de tracker.** O quadro é razoavelmente genérico; o modelo de
  dados não — existe um campo `jira` no struct do card. Não espere Linear ou GitHub
  Projects.

## Suporte

Projeto solo · sem suporte · PRs podem esperar · MIT.

Essa linha é gestão de expectativa, não modéstia. O código é MIT; tudo em `examples/` é
CC0 — copie, mude, publique.
