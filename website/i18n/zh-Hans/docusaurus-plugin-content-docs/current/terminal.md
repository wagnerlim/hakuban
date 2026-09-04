---
sidebar_position: 2
title: 在终端里
---

# 在终端里

这里的一切都是可选的。Hakuban 完全不需要 `config.yml` 也能跑 —— 下面每个字段都有默认值，文件不存在本身就是一个合法状态。

## 两个入口

```bash
hakuban                          # TUI（需要 TTY）
hakuban move <id> "<列名>"        # 无界面：先触发来源列的 on_exit，再触发目标列的 on_enter
hakuban progress <id> <0..100>   # 无界面：卡片上显示的百分比
```

这两个子命令就是全部的无界面接口。它们故意保持通用 —— 没有 `hakuban jira ...`，没有 `hakuban deploy`。任何具体的东西都住在看板触发的钩子里。

退出码：`0` 完成，`1` 操作失败（钩子失败时卡片留在原处），`2` 用法错误。列名是精确比较且区分大小写，带空格的记得加引号。

## data dir 从哪来

启动时解析，第一个命中的生效：

1. `HAKUBAN_TASK_DIR` —— 环境变量
2. `<用户配置目录>/hakuban/root.yml` 的 `datadir` 字段 —— 你在 TUI 里换目录时写下的指针
3. `~/.hakuban`

之所以要这个指针，是因为 `config.yml` 住在 data dir *里面*，因此没法保存自己的路径。从 TUI 里换 data dir 永远不会把两份数据混在一起：目标目录里已经有 Hakuban 数据的话，这次迁移会被拒绝。

## 键盘

分两组。**导航键直接触发。** **命令键在一个 tmux 风格的前缀 `ctrl+t` 之后** —— 先按前缀，再按键。

### 导航（无前缀）

| action id | 默认键 | 作用 |
|---|---|---|
| `nav_left` / `nav_right` | `h` / `l` | 在列之间移动光标 |
| `nav_down` / `nav_up` | `j` / `k` | 在卡片之间移动光标 |
| `move_left` / `move_right` | `H` / `L` | **移动卡片**一列 —— 这会触发钩子 |
| `next_board` / `prev_board` | `tab` / `shift+tab` | 切换看板标签页 |
| `open` | `enter` | 打开选中的卡片 |

### 命令（`ctrl+t` 之后）

| action id | 默认键 | 打开什么 |
|---|---|---|
| `add` | `a` | 在当前列新建卡片 |
| `subtask` | `S` | 新建选中卡片的子卡片 |
| `find` | `/` | 卡片搜索 —— 一个输入框加一个列表，`enter` 把光标跳到结果上 |
| `board_list` | `b` | 看板列表：打开、关闭、新建 |
| `new_board` | `+` | 新建看板 |
| `close_board` | `ctrl+w` | 关闭当前标签页 |
| `board_cfg` | `c` | 看板配置：改名、key、列（重排 / 新增 / 删除） |
| `lane_cfg` | `g` | 单列配置：过滤器、改名、删除 |
| `sync` | `y` | 对当前列运行看板的 `sync` |
| `tags` | `t` | 标签目录：每个标签的颜色和描述 |
| `card_tags` | `T` | 在选中卡片上勾选/取消目录里的标签 |
| `settings` | `s` | 主题、语言、预览面板、日期格式、编辑器、data dir |
| `keymap` | `?` | 快捷键编辑器 —— 列出 action → 键，并捕获新键 |
| `quit` | `q` | 退出 |

### 重绑定改不掉的别名

不受配置和前缀影响：`←` `→` `↑` `↓` 用于导航，`<` 和 `>` 用于移动卡片，`ctrl+c` 退出。

### 改快捷键

打开编辑器（`ctrl+t` `?`），选一个 action，按下新键。只有**与默认值不同**的部分会写进 `config.yml`，所以文件很小，将来默认值变了你也还能跟上：

```yaml
keys:
  add: n
  find: f
```

这个 map 的键就是上面表格里的 action id。

## 鼠标

键盘是正统路径，鼠标是它的镜像。在你点下去之前，指针会告诉你这块区域是干什么的：

| 指针 | 含义 | 出现在 |
|---|---|---|
| `grab` / `grabbing` | 可拖动 | 卡片、滚动条轨道、弹窗标题栏 |
| `pointer` | 单击即生效 | ☰、sync 按钮、标签页、菜单/配置/过滤器行 |
| `text` | 已聚焦的输入框 | 任何输入框 |

而交互在任何地方都是同一套：

- **单击选中** —— 只移动光标，不执行
- **双击激活** —— 等同于 `enter`
- **拖动** = 按下再移动：卡片跨列、滚动条轨道、拖弹窗标题栏。按下但没移动，就不是拖动。
- **详情里的卡片 id 是链接** —— 单击，不需要修饰键，在浏览器里打开这个 issue。需要看板配置 `issue_url`（见下）。

弹窗可以拖标题栏移动，点红点关闭。

## `config.yml`

位于 `<data dir>/config.yml`。可手工编辑 —— TUI 在加载时读取，在你于 Settings 里改动时写回。所有字段都是可选的。

```yaml
theme: omni                 # 见下方清单
lang: pt-BR                 # pt-BR | en-US | zh-Hans
preview_pane: true          # 底部的 markdown 面板
date_format: 2006-01-02     # 一个 Go layout，作用于 `due`
editor: ''                  # '' = $EDITOR
confirm_delete: true        # 删除前询问
priorities: [low, normal, high]
keys:                       # 只放覆盖项
  add: n
tags:                       # 标签目录
  - name: backend
    color: blue             # 调色板里的色相名，不是 hex
    desc: touches the API
```

| 字段 | 默认 | 说明 |
|---|---|---|
| `theme` | `omni` | 可在 Settings 里运行时切换 |
| `lang` | `pt-BR` | 所有可见文字都来自 i18n 目录 |
| `preview_pane` | `false` | 底部显示卡片的 markdown |
| `date_format` | `2006-01-02` | Go layout，不是 `YYYY-MM-DD` |
| `editor` | `''` | 退回 `$EDITOR` |
| `confirm_delete` | `true` | |
| `priorities` | `[low, normal, high]` | 卡片可用的优先级 |
| `keys` | — | action → 键，只放覆盖项 |
| `tags` | — | 目录：`name`、`color`、`desc` |

它**不**保存 data dir —— 那会保存在自己里面。

## 主题

内置 19 套，可运行时切换：

`omni`（默认）· `catppuccin` · `catppuccin-latte` · `tokyo-night` · `tokyo-night-day` ·
`dracula` · `nord` · `gruvbox` · `gruvbox-light` · `one-dark` · `one-light` · `solarized` ·
`solarized-light` · `kanagawa` · `kanagawa-lotus` · `rose-pine` · `rose-pine-dawn` ·
`vesper` · `terminal`

浅色的是 `catppuccin-latte`、`gruvbox-light`、`one-light`、`solarized-light`、`tokyo-night-day`、`kanagawa-lotus` 和 `rose-pine-dawn`。`terminal` 用你终端自己的 ANSI 颜色而不是 hex，所以它继承你终端的配色。

配色分别归功于各自的上游项目（Catppuccin、Nord、Dracula、Gruvbox、Solarized、Tokyo Night、Kanagawa、Rosé Pine、One Dark）。新增一套就是一个 struct。

标签的 `color` 是**当前调色板里的一个色相名**，不是 hex：`mauve`、`blue`、`green`、`yellow`、`peach`、`red`、`teal`。所以你换主题之后标签看着依然协调。颜色留空，或者标签不在目录里，就用默认色渲染 —— 目录是建议，不是限制。

## 过滤器

过滤器在看板的 `filters` 注册表里声明一次，然后用 `use_filters` 按列启用。它有**两种性质**，把它们搞混是常见错误。

### tracker 过滤器 —— 决定 `sync` 拉什么

core 从不解释这个值；它把你的选择转交给 sync 钩子，由钩子组装成查询。只在线有效：它碰不到本地卡片。三种形态：

```yaml
filters:
  mine: assignee = currentUser()          # 简单：一个标量，开/关
  sprint:                                  # 静态选项：多选
    options:
      Current: sprint in openSprints()
      Next: sprint in futureSprints()
  epic:                                    # 动态选项：用的时候才列出来
    options_cmd: hooks/list-epics.sh
```

钩子从 `HAKUBAN_FILTERS` 收到它们，是**按过滤器分组**的 JSON。core 从不拼接这些值：约定是钩子对同一过滤器的多个选项做 OR，对不同过滤器之间做 AND。

### 字段过滤器 —— 在本地求值

一个作用于卡片自身字段的谓词。离线可用，对任何卡片都有效，本地的或镜像的：

```yaml
filters:
  recent:
    field: modified      # modified | created
    within_days: 7
```

配错的过滤器 —— 未知字段、零值时间戳 —— 会保留全部。不会有东西被悄悄藏起来。

### 在某一列启用

```yaml
actions:
  Backlog:
    jql: project = TEAM AND status = "To Do"
    use_filters:
      sprint: [Current]
      mine: []
```

## 列按钮

列底部的一个按钮，让某个操作作用于**该列的全部卡片**，而不只是选中那一张：

```yaml
actions:
  Review:
    button_label: checks
    buttons:
      - label: run CI
        icon: ▶
        cmd: hooks/ci.sh          # 脚本：该列卡片以 JSON 数组从 stdin 传入
        batch: true
      - label: summarize
        agent: |                   # 指令：自然语言，受 agent_tools 限制
          Write one paragraph per card in this column.
```

`cmd` 遵循与状态转移钩子相同的契约 —— 见[钩子契约](./hook-contract.md) —— 只是 stdin 是该列卡片的**数组**。`agent` 是一条指令，也可以是 data dir 下某个 `.md` 的路径。`batch: true` 会把这个按钮登记进顶栏的全局按钮，那个按钮会逐个触发每一列的 batch 按钮。

## 在 tracker 里打开卡片

```yaml
issue_url: https://acme.atlassian.net/browse/{key}
```

`{key}` 会被替换成卡片的外部键。配置之后，详情里的 id 就可以点击，并在浏览器中打开（`open` / `xdg-open` / `rundll32`）。不配置的话，id 只是纯文本。

## sync 契约

`sync` 和任何钩子一样是一行命令，它会收到：

| 通道 | 内容 |
|---|---|
| stdin | 该列当前的卡片，作为 JSON 数组 |
| `HAKUBAN_JQL` | 该列的 `jql` |
| `HAKUBAN_COLUMN` | 列名 |
| `HAKUBAN_BOARD` | 看板 id |
| `HAKUBAN_DIR` | data dir |
| `HAKUBAN_FILTERS` | 已选过滤器，按过滤器分组的 JSON（没有则为 `{}`） |
| `HAKUBAN_COMMENTS` | 看板设了 `comments: true` 时为 `1` |
| `HAKUBAN_TASK_BIN` | 当前运行的二进制，供需要回调的钩子使用 |

`sync_on_open: true` 会在打开看板时运行它；否则就靠 `ctrl+t` `y`、该列的 sync 按钮，或者顶栏的全局按钮。
