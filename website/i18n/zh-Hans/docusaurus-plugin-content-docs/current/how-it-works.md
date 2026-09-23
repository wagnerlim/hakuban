---
sidebar_position: 1.5
title: 它是怎么运作的
---

# 它是怎么运作的

没有数据库，也没有服务器。Hakuban 就是一个装满 Markdown 文件的文件夹，外加一个读它的二进制程序。

## data dir

```text
~/.hakuban/
├── boards/
│   └── demo.md          # 一个看板：有哪些列，每一列做什么
├── tasks/
│   ├── DEMO-1.md        # 一张卡片一个文件，以 id 命名
│   ├── DEMO-2.md
│   └── DEMO-3.md
├── jira/                # 看板的 `sync` 写出的镜像 —— 只读
├── archive/             # 已归档的卡片，格式相同
├── hooks/               # 你的脚本和指令 —— 约定俗成；路径以 ~/.hakuban 为起点解析
├── config.yml           # 主题、语言、标签、按键 —— 可选
└── state.yml            # 哪些标签页开着 —— 可丢弃
```

真正重要的只有 `boards/` 和 `tasks/`。删掉 `state.yml`，TUI 会重新打开全部看板；删掉
`config.yml`，每个字段都回到默认值。这个文件夹从哪来，见
[在终端里](./terminal.md#data-dir-从哪来)。

## 一张卡片就是一个文件

```markdown
---
id: DEMO-2
title: 修复发票列表里的时区偏移
status: To-Do
priority: normal
project: demo
tags:
    - FRONTEND
created: 2026-09-04T09:00:00Z
modified: 2026-09-04T18:38:41Z
---

对 UTC 以西的人来说，日期早显示了一天。
```

把卡片绑到看板上的是两个字段：

- `project` 是看板的 id —— `boards/` 里的文件名，去掉 `.md`。
- `status` 是卡片所在的列。移动卡片改写的就是这一行。

id 由看板的 `key` 加一个计数器构成（`DEMO-1`、`DEMO-2`……）。带 `parent` 的卡片是子任务，
继承其根卡片的看板。结尾 `---` 以下的全部内容是卡片的 notes —— 自由的 Markdown，也是智能体
读取自己任务的地方。

## 一次 move 会做什么

在 TUI 里拖动卡片，或者运行 `hakuban move DEMO-2 Doing` —— 走的是同一条路：

1. 先跑源列的 `on_exit`，如果有的话。
2. 再跑目标列的 `on_enter`，如果有的话。
3. 只有两者都成功：`status` 变成目标列，`progress` 归 `0`，文件才被写入。

任何一个钩子以非零码退出，链条就在那一步停住。什么都不会写，卡片留在原地，而你看到的原因
就是该钩子的 stderr。在每一侧，指令（`on_enter`）都胜过脚本（`on_enter_cmd`）—— 见
[钩子契约](./hook-contract.md)。

## 两个写入者，一份真相

磁盘就是事实来源。TUI 往里写，任何能编辑文件的东西也一样 —— 你在 `$EDITOR` 里、一个智能体、
一个调用 `hakuban progress` 的钩子。每次写入都先落到 `.tmp` 再重命名到位，所以任何读取者都
不会看到半张卡片。TUI 大约每秒重读一次这个文件夹：智能体移动的卡片会直接出现在你眼前，不用
刷新。

这也正是这个文件夹和 git 合得来的原因：把它提交进去，看板的历史就是 `git log`。
