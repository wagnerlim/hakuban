---
sidebar_position: 1
title: Hakuban 是什么
---

# Hakuban 是什么

一个跑在终端里的看板：你的智能体同样在上面干活 —— 而卡片换列时会发生什么，由你声明。

每张卡片、每个看板都是磁盘上的一个 Markdown 文件，所以 AI 读写它们的方式和你完全一样：它只是文本。每一列是一个状态，每个状态由你定义 —— 跑脚本、发邮件、打版本标签、把任务交出去。Hakuban 提供看板；流程是一张白纸。

想要一句话版本：**一台住在磁盘上的状态机，状态转移会执行一个智能体，而它的工具范围就声明在看板文件里。**

> 看板 **kanban** —— 别人给你的板。
> 白板 **hakuban** —— 你自己写的板。

## 看它跑起来

<video controls preload="metadata" style="width:100%;border:1px solid var(--border);border-radius:8px">
  <source src="/hakuban/demo.mp4" type="video/mp4" />
  你的浏览器无法播放此视频。
</video>

由作者录制。无声。

## 安装

```bash
go install github.com/wagnerlim/hakuban/cmd/hakuban@latest
```

不带子命令运行 `hakuban` 会打开 TUI，需要 TTY。另有两个无界面子命令，让智能体在没有终端的情况下推动流水线：

```bash
hakuban move <id> "<列名>"        # 先触发来源列的 on_exit，再触发目标列的 on_enter
hakuban progress <id> <0..100>   # 卡片上显示的百分比
```

列名是精确比较，区分大小写。

## 老实说的前置条件

二进制本身只需要 Go。**但智能体演示在一台干净的机器上跑不起来**，它需要：

- `jq`
- 已登录的 `gh`
- 付费的 Claude Code 订阅（指令钩子会调用 `claude -p`）
- 作者自己的钩子还需要 [`herdr`](https://github.com/ogulcancelik/herdr) 多路复用器

这里的定位是 *"看看是怎么做的"*，而不是 *"装上就能用"*。如果你克隆下来，指望首页那段录屏自己重现，你会失望 —— 那是文档的问题，不是你的问题，所以这里先写清楚。

## 各样东西住在哪

```
~/.hakuban/
├── boards/<id>.md      看板：列，以及每列做什么
├── tasks/<ID>.md       卡片：frontmatter + markdown 正文（笔记）
├── jira/<KEY>.md       由 sync 钩子materialize 出来的只读镜像
├── archive/            已结束的卡片
├── config.yml          主题、语言、快捷键
└── state.yml           哪些看板是打开的
```

用 `HAKUBAN_TASK_DIR` 可以改根目录。

磁盘是唯一真相来源。内存里的 store 只是缓存；TUI 每秒左右重读磁盘，所以智能体移动的卡片会当着你的面出现。写入始终是原子的（tmp + rename），因为存在两个合法写入方：TUI，以及手动编辑文件的人 —— 你，或者一个智能体。

## 不在范围内

以下都是决定，不是遗漏。

- **没有看板市场。** 一个自包含的看板就是一个文件；GitHub 已经提供了托管、搜索和 fork。
- **没有 `import <url>` 命令。** 导入就是把 `.md` 下载下来放进目录。那个命令会成为第三方代码的执行入口。
- **没有图形界面，也没有网页界面。** 试过、评估过、取消了。
- **不是多用户。** 没有服务端，没有认证。本地、单人。
- **还不是 tracker 无关的。** 看板格式相当通用；数据模型不是 —— 卡片结构体里有一个 `jira` 字段。不要指望 Linear 或 GitHub Projects。

## 支持

个人项目 · 无技术支持 · PR 可能会搁置 · MIT。

这行是预期管理，不是谦虚。代码是 MIT；`examples/` 里的一切是 CC0 —— 复制、修改、拿去用。
