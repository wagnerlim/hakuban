# hakuban docs site

Docusaurus 3. The homepage is a custom page (`src/pages/index.tsx` +
`index.module.css`) recreating the design handoff; the palette and the Infima mapping live
in `src/css/custom.css`.

```sh
pnpm install
./fetch-assets.sh                   # pulls the doc video (see below)
pnpm start                          # dev, default locale only
pnpm start -- --locale pt-BR        # dev, one other locale
pnpm build                          # all three locales into build/
```

Pushing to `main` deploys to GitHub Pages via `.github/workflows/deploy-docs.yml`.

## The doc video

`static/demo.mp4` is **not in git**. The repository is ~500 KB of source with a 6 MB
history; a 12 MB screen recording would double the history and never leave it again, and
every re-record would add another permanent blob. It lives on a release instead:

```sh
gh release create assets --repo wagnerlim/hakuban --title "Doc assets" --notes "" || true
gh release upload assets static/demo.mp4 --repo wagnerlim/hakuban --clobber
```

`fetch-assets.sh` copies it into `static/` before the build, so the published site serves
the file itself instead of hotlinking GitHub at runtime. A missing asset is a warning, not
an error — a docs deploy must not fail over a video, and the page falls back to its own
text. The script keeps a local copy if one is already there, so re-recording locally does
not need a release round-trip.

Re-encoded from the author's `.mov` with the system tool, no install needed:

```sh
avconvert --source Demo-Hakuban.mov --preset Preset1920x1080 --output static/demo.mp4 --replace --multiPass
```

The container matters: H.264 in QuickTime does not play in Firefox, and the source was not
fast-start, so the whole file had to download before playback could begin.

## Editing strings

Homepage strings go through `translate()` with literal ids. After adding one:

```sh
pnpm write-translations --locale pt-BR
pnpm write-translations --locale zh-Hans
```

then fill the new key in `i18n/<locale>/code.json`. Doc bodies are English only and fall
back to English in the other locales.

## Pending assets

- A hero recording driven by hand. `static/hero.cast` is a provisional take: it is
  machine-driven, so it shows cards moving on their own but **not** the hand-drag, the
  animated progress bar or the refused move. Re-record with `demo/record.sh`.
- An outlined-path SVG of the 白板 lockup, for the navbar and `static/img/favicon.svg`
  (both currently depend on a system CJK serif).
- Self-hosted IBM Plex / Noto faces instead of the Google Fonts stylesheet.
