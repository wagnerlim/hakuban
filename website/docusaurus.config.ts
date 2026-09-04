import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';
import {themes as prismThemes} from 'prism-react-renderer';

// Header/footer kanji lockup: 白 solid, 板 outlined. Rendered as an HTML navbar item
// because Docusaurus' navbar logo only takes an image, and an <img>-referenced SVG
// cannot load Noto Serif JP.
// ponytail: swap for an inline SVG lockup once the glyphs are outlined (handoff, Assets).
const lockup = (size: number) =>
  `<span class="lockup" style="font-size:${size}px"><span class="lockup-solid">白</span><span class="lockup-outline">板</span></span>`;

const config: Config = {
  title: 'Hakuban',
  tagline: 'A kanban in your terminal that your agents work on too.',
  favicon: 'img/favicon.svg',

  future: {v4: true, faster: true},

  url: 'https://wagnerlim.github.io',
  baseUrl: '/hakuban/',
  organizationName: 'wagnerlim',
  projectName: 'hakuban',

  onBrokenLinks: 'throw',
  // 'detect' parses .md as CommonMark (only .mdx gets MDX). The docs are plain
  // markdown, and this is what makes `{#explicit-id}` headings and literal braces
  // like {{.id}} safe in prose — under MDX a brace opens a JS expression.
  markdown: {format: 'detect', hooks: {onBrokenMarkdownLinks: 'warn'}},

  // The product ships pt-BR, en-US and zh-Hans. English is the root locale because the
  // entry point has to land in English (briefing 9/10); UI chrome and the homepage are
  // fully translated, doc bodies fall back to English.
  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'pt-BR', 'zh-Hans'],
    localeConfigs: {
      en: {label: 'English', htmlLang: 'en-US'},
      'pt-BR': {label: 'Português', htmlLang: 'pt-BR'},
      'zh-Hans': {label: '中文', htmlLang: 'zh-Hans'},
    },
  },

  stylesheets: [
    'https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500;600&family=IBM+Plex+Sans:wght@400;500;600;700&family=Noto+Serif+JP:wght@600&family=Noto+Sans+SC:wght@400;500;600&display=swap',
  ],

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/wagnerlim/hakuban/tree/main/website/',
        },
        blog: false,
        theme: {customCss: './src/css/custom.css'},
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    // Binary toggle, dark by default — the design has exactly two palettes and the
    // button shows the current one by name. Flip to true for a 3-state cycle that
    // seeds from the OS but shows a "system" label the design does not define.
    colorMode: {defaultMode: 'dark', respectPrefersColorScheme: false},
    navbar: {
      items: [
        {
          type: 'html',
          position: 'left',
          value: `<a class="navbar-lockup" href="/hakuban/">${lockup(26)}<span class="navbar-wordmark">Hakuban</span></a>`,
        },
        {to: '/docs/intro', label: 'Docs', position: 'right'},
        {href: 'https://github.com/wagnerlim/hakuban', label: 'GitHub', position: 'right'},
        {type: 'localeDropdown', position: 'right'},
      ],
    },
    footer: {
      // The palette comes from --shell in custom.css, not from Infima's dark/light.
      style: 'light',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'In the terminal', to: '/docs/terminal'},
            {label: 'Board format', to: '/docs/board-format'},
            {label: 'Hook contract', to: '/docs/hook-contract'},
            {label: 'Examples', to: '/docs/examples'},
          ],
        },
        {
          title: 'Not in scope',
          items: [
            {label: 'No marketplace', to: '/docs/intro'},
            {label: 'No web UI', to: '/docs/intro'},
            {label: 'Single-player, local', to: '/docs/intro'},
          ],
        },
      ],
      copyright:
        `<div class="footer-brand">${lockup(22)}<span class="footer-wordmark">Hakuban</span></div>` +
        `<div class="footer-legal">MIT. Theme palettes credited to Catppuccin, Nord, Dracula, Gruvbox, Solarized, Tokyo Night, Kanagawa, Rosé Pine and One Dark. 19 themes in the box; adding one is a struct.</div>`,
    },
    prism: {
      theme: prismThemes.vsLight,
      darkTheme: prismThemes.vsDark,
      additionalLanguages: ['bash', 'yaml', 'python', 'go'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
