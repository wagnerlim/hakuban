# TUI layer (Bubble Tea v2)

`internal/tui/tui.go` is the Bubble Tea `Model`. Only one view — the disk is the
source of truth (`refresh` re-reads on every change).

## Modes

A `mode` enum drives `Update` and `render`. `Update` dispatches on `m.mode`;
`render` draws according to `m.mode`.

**Adding a new modal/screen** = 4 places:
1. a new value in the `mode` enum;
2. a case in the `KeyPressMsg` dispatch inside `Update`;
3. a case in `render()`;
4. an entry in `inModal()` (if it is a modal laid over the board).

## Modals (overlay)

`overlay(box)` composes with Lip Gloss' (v2) compositor: board at the back (z0),
shadow (z1), box (z2), macOS dots (z3). Modals are **draggable** by the title bar
(`onModalHandle`) and close on the red dot (`onModalClose`).
`modalBox(title, body, help)` is the standard frame.

**Centering (the rule):** every new panel opens centered. Do not scatter
`modalPlaced=false` across each transition — it is easy to forget (it already
happened). `render()` does it centrally: when `m.mode != m.renderedMode` it clears
`modalPlaced` (so `overlay()` re-centers on the next measurement) and updates
`renderedMode`. That way any panel switch re-centers, and dragging preserves the
position as long as the mode does not change. A modal of a different size reusing the
previous one's position shows up offset — this mechanism prevents that.

## Hitboxes (mouse)

`render` **registers** the clickable regions; the mouse handlers **read** them:
`tabRegions` (tabs), `cardRegions` (cards), `gearRegions` (the lane's ☰),
`syncRegions` (per-column footer button), `syncAllReg` (the batch button in the top
bar), the column menu (`menuX/menuY/menuW/menuRows` → `menuHit`, a popover anchored to
the button, with no modal geometry), the id link in the detail view
(`detailLinkY`/`detailLinkX0..X1`, opens the issue in the browser) and the confirmation
buttons (`confirmNo`/`confirmYes`). The pattern:
- coordinates are screen coordinates;
- inside a modal, store the offset **relative** to the top-left corner and add
  `modalX/modalY` on click (this works even with the modal dragged);
- for content inside a **viewport** (the detail view): content line `i` becomes
  `i - vp.YOffset()` on screen, **and** you add the top of the wrapping box
  (border+padding = `+2`, see `detailRows`/`detailLinkLine`). Getting this offset wrong
  = a misplaced hitbox.

## Board scrolling

- **Vertical** (per column): `windowColumn` crops the cards into a window of `bodyH`
  lines; `vscroll` draws the bar.
- **Horizontal** (the board's): the offset is in **CELLS** (`m.hOff`), not in column
  index. It renders every column and crops the window `[hOff, hOff+viewW)` with
  `ansi.Cut` (which preserves ANSI) → smooth sliding. `hscroll` draws the track
  (draggable with the mouse, `scrollTrackTo`). `colAt(x)` adds `hOff` to map x→column.
  The keyboard moves the cursor and `ensureColVisible` slides the minimum.

## Mouse UX (convention) — pointer + interaction

The mouse is a first-class citizen: the keyboard is the canonical path and the mouse
**mirrors** it. When creating/changing any interactive surface, follow these patterns.

**Pointer shape** (`pointerShape`; emits OSC 22 through `ansi.SetPointerShape` only when
the shape changes — otherwise it would spam on every motion). The rule, by affordance:
- **draggable** → `ptrGrab` on hover, `ptrGrabbing` while dragging. E.g. a card, a
  scroll track, a modal title.
- **clickable** (acts on a click) → `ptrPointer`. E.g. ☰, the sync button, tabs,
  menu/config/filter/picker rows, the id link in the detail view.
- a focused **input** → `ptrText`. Everything else → `ptrDefault`. Restore `default` on
  the way out.

**Interaction (the same semantics everywhere):**
- **1 click = select** (moves the cursor/highlight to the row/card), it does not act.
- **2 clicks = activate**, equivalent to Enter. Bubble Tea v2 has no native click count →
  detection by timing (`isDoubleClick`, window `doubleClickWindow`).
- **drag** = press + motion (a card between columns, a track, a modal). A click without
  movement does not drag.
- Menus go through the **generic** path, do not duplicate the action: `currentMenu`
  (geometry: number of rows + `headerLines`) → `menuRowAt` (the row under the cursor) →
  `clickMenuRow` (selects; on a double-click it **replays Enter** in the mode's
  `updateXxx`). A new menu surface = just one more `case` in those three.
- **External link** (a card's id → opens the issue in the tracker): 1 click opens it
  through `openURL` (`open`/`xdg-open`/`rundll32`), with no modifier. Do NOT use an OSC 8
  hyperlink — the viewport can swallow it and Ghostty requires ⌘+click; handling the click
  ourselves is what makes it "I click and it goes". The URL comes from a board template
  (`issue_url`, `{key}`), tracker-agnostic.

**The hitbox is the text, not the line.** Bound the region to the target in X **and** Y
(`detailLinkX0..X1`), otherwise clicking the empty space next to it fires. x0 = the box's
border+padding (`+3`) + the label width; x1 = x0 + the text width (`lipgloss.Width`).

**Checklist for a new clickable surface:**
1. `render` registers the hitbox (screen coordinates, or an offset relative to the modal
   + `modalX/Y`);
2. the mouse handler reads it and acts;
3. `pointerShape` returns the right shape on hover;
4. make sure hover works: see Mouse mode.

## Mouse mode

`pointerShape`/hover depend on `mouseX/mouseY`, which **only** update on a motion event.
`MouseModeCellMotion` (the default) reports motion **only while a button is pressed** →
there is no hover. `MouseModeAllMotion` reports motion with no button → hover works.
`View()` turns all-motion on for the board, the delete confirmation, the detail view (the
id link) and **every menu mode** (through `currentMenu`). Need hover in a new mode? Make
sure it falls into all-motion — otherwise the pointer never changes (the classic bug:
clickable, but with no cursor signalling it). Note: the **click** works in cell-motion
(it is a press, not motion); only **hover** (the cursor) requires all-motion.

## i18n (`i18n.go`)

- **Every visible string** is a field of the `messages` struct, with one instance per
  language in `langs` (pt-BR default, en-US, zh-Hans). `msg` is the active catalog,
  swapped by `applyLang`.
- Adding a string = a field in the struct **+ an entry in all 3 languages**.
- Data keys (status/priority) are translated through `statusLabel`/`priorityLabel`, which
  return the raw key when unknown (custom columns show up as their name).

## Themes (`themes.go`)

A global active palette `pal`, applied by `applyPalette`. Styles (borders, badges,
columns) derive from `pal`. The theme is swappable at runtime (a picker) and has
light/dark variants (`isDark` aligns Glamour). New colors → a field in the palette, not a
loose hex in the view.

## Traps

- **`msg` is the global i18n catalog**, but handlers receive `msg tea.KeyPressMsg` →
  shadowing: inside the handler `msg.something` is the keyboard, not the catalog. Extract
  a method **without** that parameter (e.g. `openBoardConfig`) to reach the catalog.
- **Renaming a column migrates the `Status`** of every task in that lane (otherwise they
  become orphans). See `renameColumn`.
- When adding a line to the board (a track, an indicator…), **subtract it from `bodyH`**
  so it does not overflow the screen height.
- Scroll/cursor state is reset in `resetCursor` (on a board switch) — add any new window
  fields there.
