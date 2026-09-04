package tui

// messages is the UI string catalog (F21). Every visible string comes from here
// — nothing hardcoded in the view. One catalog per language; adding a language =
// new instance + entry in langs. Same pattern as the palette: a global active
// instance (msg) swapped by applyLang.
type messages struct {
	// status (columns + badge) — natural case; the column/badge do ToUpper
	stBacklog, stDoing, stDone string
	// priority
	prLow, prNormal, prHigh string
	// input placeholders
	phTask, phBoard, phColumn, phKey, phFilter, phTagName, phTagDesc, phTagColor string
	// tag catalog (modeTags) + form (modeTagForm)
	tTags, hTags, tagsEmpty, tAddTag, tEditTag string
	fTagName, fTagColor, fTagDesc, hTagForm    string
	// card tags checklist (modeCardTags)
	tCardTags, hCardTags, cardTagsEmpty string
	// card search (command-palette-style overlay)
	tFilter, hFilter, filterEmpty, filterTagLabel, filterAll string
	// shortcut config (modeKeymap): labels per action + titles/help/warnings
	keyLabels                                                       map[string]string
	tKeymap, hKeymap, hKeymapCapturing, keymapPress, keymapReserved string
	keymapConflict, sKeys, dKeys                                    string
	// keybind/prefix modal: modal subtitle, sections and footer
	kmSubtitle, grpNavLabel, grpCmdLabel, hIdle, hPrefix string
	// integration (F16): sync button label in the bound column's footer
	syncLabel    string
	syncAllLabel string
	// modal titles (tTheme/tLang are picker titles — a list, in the plural)
	tNewTask, tNewBoard, tNewSubtask, tBoards, tSettings, tTheme, tLang string
	// subtasks (parent/child hierarchy)
	subtasks string
	// board config: main menu (hBoardMenu) + columns sub-screen (hBoardCfg).
	tBoardCfg, tRenameBoard, tAddColumn, tRenameKey, cfgColumns, cfgNoColumns, boardEmpty string
	cfgIdent, cfgDelete                                                                   string // menu items
	colStart, colEnd                                                                      string // start/end labels in the columns list
	hBoardMenu, hBoardCfg, hBoardCfgGrab                                                  string
	nvBoardDeleted                                                                        string
	// per-column config (lane)
	tLaneCfg, hLaneCfg, tRenameColumn, laneLabel                              string
	laneFilters, laneDelete, tLaneFilters, hLaneFilters, hLaneList, noFilters string
	// delete-board confirmation (clickable buttons)
	tConfirmDelete, cdBody, btnNo, btnYes string
	// default board for orphan tasks (migration) + no-board-at-all state
	defaultBoard, noBoards string
	// settings menu labels + values
	sPreview, sTheme, sDate, sLang, sDataDir, vOn, vOff string
	// descriptions (shown below the list, per the selected item)
	dPreview, dTheme, dDate, dLang, dDataDir string
	// directory browser (yazi-style)
	tBrowser, hBrowser, brEmpty, brFilter string
	// move confirmation + warnings
	tMove, mvBody, hMove, tNotice, nvMoved, nvPointed, nvHasData string
	// help lines
	hBoard, hAdd, hList, hSettings, hTheme, hDetail, hDetailSub string
	// metadata keys in the detail/preview
	mProject, mTags, mDue, mID, mComments string
	// misc
	boardPrefix, noNotes, noCard, seeAll, duePrefix string
}

// langNames is the language cycling order in the menu.
var langNames = []string{"pt-BR", "en-US", "zh-Hans"}

var langs = map[string]messages{
	"pt-BR": {
		stBacklog: "a fazer", stDoing: "fazendo", stDone: "feito",
		prLow: "baixa", prNormal: "normal", prHigh: "alta",
		phTask: "título da task…", phBoard: "nome do board…", phColumn: "nome da coluna…", phKey: "chave (ex: CASA)…",
		phFilter: "buscar por título ou id…", tFilter: "Buscar cards", hFilter: "tab escolhe tag · ↑↓ navega · enter vai · esc fecha", filterEmpty: "nenhum card encontrado", filterTagLabel: "Tag: ", filterAll: "todas",
		phTagName: "nome da tag…", phTagDesc: "descrição…", phTagColor: "#hex", tTags: "Tags", tAddTag: "Nova tag", tEditTag: "Editar tag",
		tagsEmpty: "sem tags — aperte a pra criar", hTags: "↑↓/jk move · a nova · enter/e edita · d apaga · esc volta",
		fTagName: "Nome", fTagColor: "Cor", fTagDesc: "Descrição", hTagForm: "tab campo · ←→ escolhe a cor · digite p/ #hex · enter salva · esc cancela",
		tCardTags: "Tags do card", hCardTags: "↑↓/jk move · espaço marca · esc volta", cardTagsEmpty: "catálogo vazio — aperte t pra criar tags",
		tKeymap: "Atalhos", hKeymap: "↑↓/jk move · enter/r remapeia · d padrão · esc fecha", hKeymapCapturing: "aperte a nova tecla · esc cancela",
		keymapPress: "aperte a tecla…", keymapReserved: "tecla reservada (esc/ctrl+c)", keymapConflict: "já usada por: %s",
		sKeys: "Atalhos", dKeys: "Remapeie as teclas do board",
		kmSubtitle: "comandos disponíveis e atalhos configurados", grpNavLabel: "navegação", grpCmdLabel: "comandos",
		hIdle:        "ctrl+t comandos · hjkl navega · enter abre",
		hPrefix:      "esc cancela · a nova · / busca · S sub · b boards · c board · g lane · t tags · T card · s config · + novo · y sync · ? atalhos · q sair",
		syncLabel:    "Sincronizar",
		syncAllLabel: "Sincronizar tudo",
		keyLabels: map[string]string{
			"add": "Nova task", "find": "Buscar cards", "open": "Abrir card", "subtask": "Nova subtask",
			"move_left": "Mover card ←", "move_right": "Mover card →",
			"nav_left": "Ir ←", "nav_right": "Ir →", "nav_down": "Ir ↓", "nav_up": "Ir ↑",
			"next_board": "Próximo board", "prev_board": "Board anterior",
			"new_board": "Novo board", "close_board": "Fechar board",
			"board_list": "Lista de boards", "board_cfg": "Config do board", "lane_cfg": "Config da coluna",
			"settings": "Configurações", "keymap": "Atalhos", "tags": "Config de tags", "card_tags": "Tags do card", "sync": "Sincronizar coluna", "quit": "Sair",
		},
		tNewTask: "Nova task", tNewBoard: "Novo board", tNewSubtask: "Nova subtask", tBoards: "Boards", tSettings: "Configurações", tTheme: "Temas", tLang: "Idiomas", subtasks: "Subtasks",
		tBoardCfg: "Config do board", tRenameBoard: "Renomear board", tAddColumn: "Nova coluna", tRenameKey: "Chave do board",
		cfgColumns: "Colunas", cfgNoColumns: "sem colunas — aperte a pra adicionar", boardEmpty: "board sem colunas — aperte c pra configurar",
		cfgIdent: "Identificador", cfgDelete: "Excluir board",
		colStart: "início · tasks nascem aqui", colEnd: "final",
		hBoardMenu:     "↑↓/jk mover · enter escolher · esc fecha",
		hBoardCfg:      "↑↓/jk mover · a coluna · enter reordena · d apaga · esc volta",
		hBoardCfgGrab:  "↑↓/jk reposiciona · enter/esc solta",
		nvBoardDeleted: "Board excluído — %d task(s) arquivada(s)",
		defaultBoard:   "Geral", noBoards: "nenhum board — aperte + pra criar o primeiro",
		tLaneCfg: "Config da coluna", hLaneCfg: "↑↓/jk mover · enter · esc fechar", tRenameColumn: "Renomear coluna", laneLabel: "Coluna:",
		laneFilters: "Filtros", laneDelete: "Apagar coluna", tLaneFilters: "Filtros da coluna", hLaneList: "↑↓/jk mover · → abrir · espaço marca · esc salva", hLaneFilters: "↑↓/jk mover · espaço marca · ←/esc volta", noFilters: "nenhum filtro cadastrado no board",
		tConfirmDelete: "Excluir board?", cdBody: "Excluir o board \"%s\"?\n%d task(s) serão arquivadas.", btnNo: "Não", btnYes: "Sim",
		sPreview: "Painel de preview", sTheme: "Tema", sDate: "Formato de data", sLang: "Idioma", sDataDir: "Diretório de dados", vOn: "ligado", vOff: "desligado",
		dPreview: "Painel no rodapé com o conteúdo do card selecionado.", dTheme: "Paleta de cores de toda a interface.", dDate: "Como o prazo das tasks é exibido.", dLang: "Idioma dos textos da interface.", dDataDir: "Onde os arquivos .md ficam salvos.",
		tBrowser: "Escolher diretório", hBrowser: "l/→ entrar · h/← subir · / filtrar · enter usar esta · esc cancelar", brEmpty: "(vazio)", brFilter: "filtro: ",
		tMove: "Mover dados?", mvBody: "Mover %d arquivo(s) para\n%s?", hMove: "enter/s mover · n só apontar · esc cancelar",
		tNotice: "Aviso", nvMoved: "Dados movidos para %s", nvPointed: "Agora usando %s", nvHasData: "A pasta de destino já tem dados do hakuban. Nada foi movido.",
		hBoard:     "hjkl/HL cards · tab troca board · + novo · b lista · a nova · / busca · S sub · enter abre · c board · g lane · t tags · T tag no card · s config · ? atalhos · q sair",
		hAdd:       "enter cria · esc cancela",
		hList:      "↑↓/jk mover · enter abrir · esc fechar",
		hSettings:  "↑↓/jk mover · ←→/space muda · enter lista (tema/idioma) · esc fecha",
		hTheme:     "↑↓/jk mover · enter escolher · esc voltar",
		hDetail:    "↑↓/jk rolar · esc voltar",
		hDetailSub: "↑↓/jk pai/subtask · enter abre · esc volta · pgup/pgdn rola",
		mProject:   "projeto", mTags: "tags", mDue: "prazo", mID: "id", mComments: "Comentários",
		boardPrefix: "board: ", noNotes: "(sem notas)", noCard: "(nenhum card selecionado)", seeAll: "… (enter pra ver tudo)", duePrefix: "prazo ",
	},
	"en-US": {
		stBacklog: "backlog", stDoing: "doing", stDone: "done",
		prLow: "low", prNormal: "normal", prHigh: "high",
		phTask: "task title…", phBoard: "board name…", phColumn: "column name…", phKey: "key (e.g. CASA)…",
		phFilter: "search by title or id…", tFilter: "Find cards", hFilter: "tab pick tag · ↑↓ move · enter go · esc close", filterEmpty: "no cards found", filterTagLabel: "Tag: ", filterAll: "all",
		phTagName: "tag name…", phTagDesc: "description…", phTagColor: "#hex", tTags: "Tags", tAddTag: "New tag", tEditTag: "Edit tag",
		tagsEmpty: "no tags — press a to add", hTags: "↑↓/jk move · a add · enter/e edit · d delete · esc back",
		fTagName: "Name", fTagColor: "Color", fTagDesc: "Description", hTagForm: "tab field · ←→ pick color · type for #hex · enter save · esc cancel",
		tCardTags: "Card tags", hCardTags: "↑↓/jk move · space toggles · esc back", cardTagsEmpty: "empty catalog — press t to create tags",
		tKeymap: "Shortcuts", hKeymap: "↑↓/jk move · enter/r rebind · d default · esc close", hKeymapCapturing: "press the new key · esc cancels",
		keymapPress: "press a key…", keymapReserved: "reserved key (esc/ctrl+c)", keymapConflict: "already used by: %s",
		sKeys: "Shortcuts", dKeys: "Remap the board keys",
		kmSubtitle: "available commands and configured shortcuts", grpNavLabel: "navigation", grpCmdLabel: "commands",
		hIdle:        "ctrl+t commands · hjkl navigate · enter open",
		hPrefix:      "esc cancel · a add · / find · S sub · b boards · c board · g lane · t tags · T card · s config · + new · y sync · ? keybinds · q quit",
		syncLabel:    "Sync",
		syncAllLabel: "Sync all",
		keyLabels: map[string]string{
			"add": "New task", "find": "Find cards", "open": "Open card", "subtask": "New subtask",
			"move_left": "Move card ←", "move_right": "Move card →",
			"nav_left": "Go ←", "nav_right": "Go →", "nav_down": "Go ↓", "nav_up": "Go ↑",
			"next_board": "Next board", "prev_board": "Prev board",
			"new_board": "New board", "close_board": "Close board",
			"board_list": "Board list", "board_cfg": "Board config", "lane_cfg": "Column config",
			"settings": "Settings", "keymap": "Shortcuts", "tags": "Tag config", "card_tags": "Card tags", "sync": "Sync column", "quit": "Quit",
		},
		tNewTask: "New task", tNewBoard: "New board", tNewSubtask: "New subtask", tBoards: "Boards", tSettings: "Settings", tTheme: "Themes", tLang: "Languages", subtasks: "Subtasks",
		tBoardCfg: "Board config", tRenameBoard: "Rename board", tAddColumn: "New column", tRenameKey: "Board key",
		cfgColumns: "Columns", cfgNoColumns: "no columns — press a to add one", boardEmpty: "board has no columns — press c to configure",
		cfgIdent: "Identifier", cfgDelete: "Delete board",
		colStart: "start · new tasks land here", colEnd: "end",
		hBoardMenu:     "↑↓/jk move · enter select · esc close",
		hBoardCfg:      "↑↓/jk move · a column · enter reorder · d delete · esc back",
		hBoardCfgGrab:  "↑↓/jk reposition · enter/esc drop",
		nvBoardDeleted: "Board deleted — %d task(s) archived",
		defaultBoard:   "General", noBoards: "no boards — press + to create the first one",
		tLaneCfg: "Column config", hLaneCfg: "↑↓/jk move · enter · esc close", tRenameColumn: "Rename column", laneLabel: "Column:",
		laneFilters: "Filters", laneDelete: "Delete column", tLaneFilters: "Column filters", hLaneList: "↑↓/jk move · → open · space toggle · esc save", hLaneFilters: "↑↓/jk move · space toggle · ←/esc back", noFilters: "no filters registered on the board",
		tConfirmDelete: "Delete board?", cdBody: "Delete board \"%s\"?\n%d task(s) will be archived.", btnNo: "No", btnYes: "Yes",
		sPreview: "Preview pane", sTheme: "Theme", sDate: "Date format", sLang: "Language", sDataDir: "Data directory", vOn: "on", vOff: "off",
		dPreview: "Bottom pane showing the selected card's content.", dTheme: "Color palette for the whole interface.", dDate: "How task due dates are shown.", dLang: "Language of the interface text.", dDataDir: "Where the .md files are stored.",
		tBrowser: "Choose directory", hBrowser: "l/→ open · h/← up · / filter · enter use this · esc cancel", brEmpty: "(empty)", brFilter: "filter: ",
		tMove: "Move data?", mvBody: "Move %d file(s) to\n%s?", hMove: "enter/s move · n just point · esc cancel",
		tNotice: "Notice", nvMoved: "Data moved to %s", nvPointed: "Now using %s", nvHasData: "Target folder already has hakuban data. Nothing was moved.",
		hBoard:     "hjkl/HL cards · tab switch board · + new · b list · a add · / find · S sub · enter open · c board · g lane · t tags · T tag card · s config · ? keys · q quit",
		hAdd:       "enter create · esc cancel",
		hList:      "↑↓/jk move · enter open · esc close",
		hSettings:  "↑↓/jk move · ←→/space change · enter list (theme/lang) · esc close",
		hTheme:     "↑↓/jk move · enter select · esc back",
		hDetail:    "↑↓/jk scroll · esc back",
		hDetailSub: "↑↓/jk parent/subtask · enter open · esc back · pgup/pgdn scroll",
		mProject:   "project", mTags: "tags", mDue: "due", mID: "id", mComments: "Comments",
		boardPrefix: "board: ", noNotes: "(no notes)", noCard: "(no card selected)", seeAll: "… (enter to see all)", duePrefix: "due ",
	},
	"zh-Hans": {
		stBacklog: "待办", stDoing: "进行中", stDone: "已完成",
		prLow: "低", prNormal: "普通", prHigh: "高",
		phTask: "任务标题…", phBoard: "看板名称…", phColumn: "列名称…", phKey: "标识（如 CASA）…",
		phFilter: "按标题或 id 搜索…", tFilter: "查找卡片", hFilter: "tab 选标签 · ↑↓ 导航 · enter 跳转 · esc 关闭", filterEmpty: "未找到卡片", filterTagLabel: "标签: ", filterAll: "全部",
		phTagName: "标签名称…", phTagDesc: "描述…", phTagColor: "#hex", tTags: "标签", tAddTag: "新建标签", tEditTag: "编辑标签",
		tagsEmpty: "暂无标签 — 按 a 添加", hTags: "↑↓/jk 移动 · a 添加 · enter/e 编辑 · d 删除 · esc 返回",
		fTagName: "名称", fTagColor: "颜色", fTagDesc: "描述", hTagForm: "tab 字段 · ←→ 选颜色 · 输入 #hex · enter 保存 · esc 取消",
		tCardTags: "卡片标签", hCardTags: "↑↓/jk 移动 · 空格切换 · esc 返回", cardTagsEmpty: "目录为空 — 按 t 创建标签",
		tKeymap: "快捷键", hKeymap: "↑↓/jk 移动 · enter/r 重映射 · d 默认 · esc 关闭", hKeymapCapturing: "按新键 · esc 取消",
		keymapPress: "按下按键…", keymapReserved: "保留键 (esc/ctrl+c)", keymapConflict: "已被占用：%s",
		sKeys: "快捷键", dKeys: "重新映射看板按键",
		kmSubtitle: "可用命令与已配置快捷键", grpNavLabel: "导航", grpCmdLabel: "命令",
		hIdle:        "ctrl+t 命令 · hjkl 导航 · enter 打开",
		hPrefix:      "esc 取消 · a 新增 · / 搜索 · S 子 · b 看板 · c 看板设置 · g 列 · t 标签 · T 卡片标签 · s 设置 · + 新建 · y 同步 · ? 快捷键 · q 退出",
		syncLabel:    "同步",
		syncAllLabel: "全部同步",
		keyLabels: map[string]string{
			"add": "新建任务", "find": "查找卡片", "open": "打开卡片", "subtask": "新建子任务",
			"move_left": "左移卡片", "move_right": "右移卡片",
			"nav_left": "左", "nav_right": "右", "nav_down": "下", "nav_up": "上",
			"next_board": "下一个看板", "prev_board": "上一个看板",
			"new_board": "新建看板", "close_board": "关闭看板",
			"board_list": "看板列表", "board_cfg": "看板配置", "lane_cfg": "列配置",
			"settings": "设置", "keymap": "快捷键", "tags": "标签配置", "card_tags": "卡片标签", "sync": "同步列", "quit": "退出",
		},
		tNewTask: "新建任务", tNewBoard: "新建看板", tNewSubtask: "新建子任务", tBoards: "看板", tSettings: "设置", tTheme: "主题", tLang: "语言", subtasks: "子任务",
		tBoardCfg: "看板配置", tRenameBoard: "重命名看板", tAddColumn: "新建列", tRenameKey: "看板标识",
		cfgColumns: "列", cfgNoColumns: "暂无列 — 按 a 添加", boardEmpty: "看板暂无列 — 按 c 配置",
		cfgIdent: "标识", cfgDelete: "删除看板",
		colStart: "起点 · 新任务在此", colEnd: "终点",
		hBoardMenu:     "↑↓/jk 移动 · enter 选择 · esc 关闭",
		hBoardCfg:      "↑↓/jk 移动 · a 新列 · enter 重排 · d 删除 · esc 返回",
		hBoardCfgGrab:  "↑↓/jk 重新排列 · enter/esc 放下",
		nvBoardDeleted: "看板已删除 — 已归档 %d 个任务",
		defaultBoard:   "通用", noBoards: "暂无看板 — 按 + 创建第一个",
		tLaneCfg: "列配置", hLaneCfg: "↑↓/jk 移动 · enter · esc 关闭", tRenameColumn: "重命名列", laneLabel: "列：",
		laneFilters: "筛选", laneDelete: "删除列", tLaneFilters: "列筛选", hLaneList: "↑↓/jk 移动 · → 展开 · 空格 切换 · esc 保存", hLaneFilters: "↑↓/jk 移动 · 空格 切换 · ←/esc 返回", noFilters: "看板未注册筛选",
		tConfirmDelete: "删除看板？", cdBody: "删除看板 \"%s\"？\n将归档 %d 个任务。", btnNo: "取消", btnYes: "确认",
		sPreview: "预览面板", sTheme: "主题", sDate: "日期格式", sLang: "语言", sDataDir: "数据目录", vOn: "开", vOff: "关",
		dPreview: "底部面板显示所选卡片的内容。", dTheme: "整个界面的配色方案。", dDate: "任务截止日期的显示方式。", dLang: "界面文字的语言。", dDataDir: "存放 .md 文件的位置。",
		tBrowser: "选择目录", hBrowser: "l/→ 进入 · h/← 上级 · / 过滤 · enter 使用此处 · esc 取消", brEmpty: "(空)", brFilter: "过滤：",
		tMove: "移动数据？", mvBody: "将 %d 个文件移动到\n%s？", hMove: "enter/s 移动 · n 仅指向 · esc 取消",
		tNotice: "提示", nvMoved: "数据已移动到 %s", nvPointed: "现在使用 %s", nvHasData: "目标文件夹已有 hakuban 数据。未移动任何文件。",
		hBoard:     "hjkl/HL 卡片 · tab 切换看板 · + 新建 · b 列表 · a 新增 · / 搜索 · S 子 · enter 打开 · c 看板 · g 列 · t 标签 · T 卡片标签 · s 设置 · ? 快捷键 · q 退出",
		hAdd:       "enter 创建 · esc 取消",
		hList:      "↑↓/jk 移动 · enter 打开 · esc 关闭",
		hSettings:  "↑↓/jk 移动 · ←→/space 修改 · enter 列表（主题/语言）· esc 关闭",
		hTheme:     "↑↓/jk 移动 · enter 选择 · esc 返回",
		hDetail:    "↑↓/jk 滚动 · esc 返回",
		hDetailSub: "↑↓/jk 父级/子任务 · enter 打开 · esc 返回 · pgup/pgdn 滚动",
		mProject:   "项目", mTags: "标签", mDue: "截止", mID: "ID", mComments: "评论",
		boardPrefix: "看板：", noNotes: "（无备注）", noCard: "（未选择卡片）", seeAll: "…（enter 查看全部）", duePrefix: "截止 ",
	},
}

// msg is the active catalog.
var msg messages

func init() { applyLang(langNames[0]) }

// applyLang makes the named language the active one; unknown falls back to the first (pt-BR).
func applyLang(name string) {
	if m, ok := langs[name]; ok {
		msg = m
		return
	}
	msg = langs[langNames[0]]
}

// statusLabel / priorityLabel translate the data keys (backlog/high/…) into the
// active language. Unknown key comes back as-is.
func statusLabel(status string) string {
	switch status {
	case "doing":
		return msg.stDoing
	case "done":
		return msg.stDone
	case "backlog":
		return msg.stBacklog
	}
	return status
}

func priorityLabel(p string) string {
	switch p {
	case "high":
		return msg.prHigh
	case "low":
		return msg.prLow
	case "normal", "":
		return msg.prNormal
	}
	return p
}
