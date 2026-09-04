---
sidebar_position: 5
title: 示例
---

# 示例

这些是例子，不是功能。**没有一个随工具附带。** 作者自己的看板把任务交给一个开 PR 的智能体 —— 只是 N 种流程之一，你的没有理由长成那样。

本页所有内容都是 CC0。复制、修改、拿去用。

每个代码块都是一个完整的看板：存成 `~/.hakuban/boards/<id>.md` 就能跑。

## 通知客户

一条指令，而它获得的能力只有读取和发送邮件。

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

## 拆分卡片

看板重塑自己的工作。`Refining` 是本地列：只存在于这里。

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

## 跑检查清单

用脚本，因为"测试过了"不该是一件见解不同的事。非零退出，卡片就出不了 `Doing`。

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

## 交给智能体

作者的看板。2026-08-29，卡片 → 实现 → PR 通过评审，用时 9 分 18 秒 —— 真实发生，但依然只是一个例子。`Ready → Doing` 是那道闸门，而按下它的是一个人在拖卡片。

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

`hooks/require-pr.sh` 会在卡片没有对应 PR 时拒绝 `Validate → Review` —— 这道闸门让看板没法谎报交付了什么。

## 镜像一个外部 tracker

二进制不说 HTTP，钩子才说。仓库里的
[`examples/jira/`](https://github.com/wagnerlim/hakuban/tree/main/examples/jira) 有一套能跑的：一个 `sync` 从 JQL materialize 只读镜像，一个钩子创建 issue 并用 `jira: KEY` 把键报回来，还有一个把 issue 转到该列代表的状态。

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

集成靠组合。这是原则，不是缺口。
