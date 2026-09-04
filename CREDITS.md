# Credits

hakuban is MIT (see [LICENSE](LICENSE)). The board examples under
[`examples/`](examples/) are dedicated to the public domain under
[CC0 1.0](examples/LICENSE) — copy a board file, change the columns, ship it.

## Color themes

The token vocabulary (`base`, `surface0/1`, `overlay0/1`, `subtext0` and the
named hues `mauve`, `peach`, `teal`…) is Catppuccin's. Each theme carries the
hex values published by its own upstream, mapped onto those tokens:

| Theme | Upstream | License |
|---|---|---|
| `catppuccin`, `catppuccin-latte` | [catppuccin/catppuccin](https://github.com/catppuccin/catppuccin) | MIT |
| `tokyo-night`, `tokyo-night-day` | [folke/tokyonight.nvim](https://github.com/folke/tokyonight.nvim) | Apache-2.0 |
| `dracula` | [dracula/dracula-theme](https://github.com/dracula/dracula-theme) | MIT |
| `nord` | [nordtheme/nord](https://github.com/nordtheme/nord) | MIT |
| `gruvbox`, `gruvbox-light` | [morhetz/gruvbox](https://github.com/morhetz/gruvbox) | MIT/X11 |
| `one-dark`, `one-light` | [atom/one-dark-syntax](https://github.com/atom/one-dark-syntax) | MIT |
| `solarized`, `solarized-light` | [altercation/solarized](https://github.com/altercation/solarized) | MIT |
| `kanagawa`, `kanagawa-lotus` | [rebelot/kanagawa.nvim](https://github.com/rebelot/kanagawa.nvim) | MIT |
| `rose-pine` and variants | [rose-pine/rose-pine-theme](https://github.com/rose-pine/rose-pine-theme) | MIT |
| `vesper` | [raunofreiberg/vesper](https://github.com/raunofreiberg/vesper) | MIT |

`omni` is the author's own theme. `terminal` inherits the emulator's 16 colors
and owes nothing to anyone.

The semantic mapping of these tokens onto a Bubble Tea UI was first prototyped
in [herdr](https://github.com/ogulcancelik/herdr) (Apache-2.0), which hakuban's
author uses as a multiplexer.

## Dependencies

[Bubble Tea, Lip Gloss and Glamour](https://github.com/charmbracelet) (MIT),
[goldmark](https://github.com/yuin/goldmark) (MIT),
[bluemonday](https://github.com/microcosm-cc/bluemonday) (BSD-3-Clause),
[yaml.v3](https://github.com/go-yaml/yaml) (MIT + Apache-2.0).
