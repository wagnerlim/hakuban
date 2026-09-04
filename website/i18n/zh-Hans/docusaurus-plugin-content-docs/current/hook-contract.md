---
sidebar_position: 4
title: 钩子契约
---

# 钩子契约

两个字段在同一次状态转移时触发，所以看起来可以互换。并不能。

|  | `脚本` · `_cmd` | `指令` · `on_enter` |
|---|---|---|
| 行为 | 确定性 —— 你读到的就是它做的 | 非确定性 —— 你不知道它会做什么 |
| **能力上限** | **无限** —— `bash` 里不存在权限模型 | **有界** —— 一行就写完了：`agent_tools` |
| 审阅可扩展吗？ | 不 —— 没人会去审 93 次 `jq` 调用 | 可以 —— 能力上限就是一份短清单 |
| 可移植性 | 把你绑在某个文件系统、某个二进制、某个 `$HOME` 上 | 什么都不绑：它描述的是意图 |

**项目主张：** 指令才是格式；`_cmd` 是债 —— 为精确控制留的逃生口，代价是别人跑不起来的看板。

要分享一个看板，重要的不是文件*说了什么* —— 而是它*能做什么*。

## 脚本契约

Hakuban 以 data dir 为工作目录执行 `sh -c "<那行命令>"`，所以相对路径（`hooks/notify.sh`）无论从哪里启动二进制都能解析。它就是一行命令，因此**钩子与语言无关**：bash、Python、Node、Ruby、编译出来的二进制 —— 只要遵守下面四个通道。

### stdin

卡片与本次转移，作为一个 JSON 对象：

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

两种控制行，其余一概忽略：

| 行 | 效果 |
|---|---|
| `N label` | `N` 取 0..100 —— 设置进度条及其文字 |
| `N/M label` | 第 `N` 步 / 共 `M` 步 —— 同一进度条，按 `N*100/M` 计算 |
| `jira: KEY` | 把 `KEY` 作为外部键盖在卡片上；卡片变成镜像 |

不以数字开头的行不是进度行，所以 stdout 上一条零散日志永远不会变成进度条。

### stderr

会被缓冲。**最后一条非空行成为错误信息**，显示给移动卡片的人。把原因写在这里，不要写在 stdout。

### 退出码 {#exit-code}

`0` 提交这次移动。任何其他值都会**中止**它：卡片留在原处，并显示 stderr 里的原因。无论是人拖动的，还是智能体调用了 `hakuban move`。钩子失败通常是缺了某项数据，而不是偶发错误 —— 先读信息，再重试移动。

### 环境变量

在进程环境之上追加：

| 变量 | 值 |
|---|---|
| `HAKUBAN_FROM` | 卡片离开的列 |
| `HAKUBAN_TO` | 卡片正在进入的列 |
| `HAKUBAN_BOARD` | 看板 id |
| `HAKUBAN_STATUS` | 目标列的 `status:` |
| `HAKUBAN_DIR` | data dir（默认 `~/.hakuban`） |
| `HAKUBAN_TASK_BIN` | 当前运行的二进制路径，供需要回调的钩子使用 |

## 同一个钩子，两种写法

Bash：

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

Python —— 同一份契约，不需要 `jq`：

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

两者接进看板的方式完全一样：

```yaml
actions:
  Doing:
    on_enter_cmd: hooks/require-repo.py
```

## 指令契约

指令没有 stdin、stdout 或退出码要遵守 —— 它就是自然语言。它有的是一个上限：

```yaml
actions:
  Done:
    on_enter: |
      Write a short update from the card notes and send it
      to the address in the frontmatter.
agent_tools: Read, mcp__gmail__send
```

`agent_tools` 就是全部的权限模型，而它只有一行。智能体以无界面方式运行，工作目录是中性的，所以它不会跑去翻你当下正好在做的那个项目。它失败的方式和脚本一样：移动被中止，原因会显示出来。

## Windows

TUI 和看板格式可用。指令钩子可用。**脚本钩子不可用** —— 那里没有 `sh`，Hakuban 也不附带 shim。如果你在 Windows 上，指令不是首选格式，而是唯一格式。
