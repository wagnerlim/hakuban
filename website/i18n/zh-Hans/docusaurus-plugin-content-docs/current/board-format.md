---
sidebar_position: 3
title: 看板格式
---

# 看板格式

一个看板就是 `~/.hakuban/boards/<id>.md` 里一个带 YAML frontmatter 的 Markdown 文件。文件名就是看板 id。

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

结尾 `---` 之下是自由 markdown —— 关于这个看板的笔记，解析器不读。

## 看板字段

| 字段 | 类型 | 作用 |
|---|---|---|
| `name` | string | 标签页上显示的看板名 |
| `key` | string | 在此看板创建的卡片 id 前缀 |
| `columns` | list | 流水线的状态，按顺序 —— 这就是看板本体 |
| `actions` | map | 列名 → 它的 [action](#列的-action) |
| `agent_tools` | string | 本看板的指令唯一可用的工具清单，逗号分隔 |
| `filters` | map | TUI 在此看板提供的命名过滤器 |
| `sync` | string | 从外部 tracker materialize 镜像的命令 |
| `sync_on_open` | bool | 打开看板时运行 `sync` |
| `comments` | bool | 把 issue 的评论一起拉进镜像 |
| `buttons` | list | 卡片底部的自定义操作 |

## 列的 action

`actions` 的每个键是一个**列名**，精确比较且区分大小写。它的值：

| 字段 | 作用 |
|---|---|
| `guide` | 该列的说明。**只读，永不执行** —— 它是给决定下一步的人（或程序）看的上下文 |
| `on_enter` / `on_exit` | **指令** —— 由无界面智能体执行的自然语言，受 `agent_tools` 限制 |
| `on_enter_cmd` / `on_exit_cmd` | **脚本** —— shell，卡片以 JSON 从 stdin 传入。见[钩子契约](./hook-contract.md) |
| `jql` | 让该列成为某个外部查询的镜像；由 `sync` 命令 materialize |
| `status` | 该列在 tracker 中代表的状态，以 `HAKUBAN_STATUS` 传给钩子 |
| `percent` | 在该列把卡片的 `progress` 显示成进度条 |
| `loader` | action 运行期间进度指示器的文字 |
| `use_filters` | 哪些过滤器作用于该列 |

同一列同一侧的指令会**遮蔽**脚本：`on_enter` 先于 `on_enter_cmd` 被检查，所以在已有脚本的地方加一条指令，会让脚本静默地不再运行。如果你要加，这条指令必须吸收脚本原来做的事 —— 或者去调用它。

没有 `jql` 的列是本地列：只存在于 Hakuban，在 tracker 里没有可查询的对应物。

### 指令里的插值

指令是一个作用于卡片字段的 Go template。可用：`{{.id}}`、`{{.title}}`、`{{.jira}}`、`{{.status}}`、`{{.priority}}`、`{{.project}}`、`{{.notes}}`。template 有错就退回原文 —— 那段自然语言本身依然成立。

指令也可以是一个 `.md` 文件路径，超过几行之后这才是可读的写法。

## 卡片

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

笔记就是载荷。需要参数的钩子 —— 一个仓库、一个工时估算 —— 从这里读，缺了就拒绝移动。这个拒绝就是功能本身：见[契约](./hook-contract.md#exit-code)。
