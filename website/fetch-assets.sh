#!/usr/bin/env bash
# Pulls the large doc assets into website/static/ so the site can serve them itself.
#
# They are NOT in git: the repository is ~500 KB of source with a 6 MB history, and a
# screen recording is 12 MB that would never leave it again — every re-record adding
# another permanent blob. They live on a release instead, and the published site copies
# them in at build time, so nothing hotlinks to GitHub at runtime.
#
# Missing assets are a warning, not an error: a docs deploy must not fail over a video.
# The page falls back to its own text.
set -uo pipefail

TAG=${ASSETS_TAG:-assets}
# Derived, never hardcoded: this tree becomes a new repository under a different name, and
# a pinned owner/name would send the workflow looking in the wrong place after the move.
# CI passes github.repository; locally it comes from the git remote.
REPO=${ASSETS_REPO:-$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null)}
[ -n "$REPO" ] || { echo "could not tell which repository to fetch assets from" >&2; exit 0; }
DEST=$(cd "$(dirname "$0")" && pwd)/static
ASSETS=(demo.mp4)

command -v gh >/dev/null || { echo "gh not installed — skipping doc assets" >&2; exit 0; }

for name in "${ASSETS[@]}"; do
  if [ -f "$DEST/$name" ]; then
    echo "$name already present, keeping it"
    continue
  fi
  if gh release download "$TAG" --repo "$REPO" --pattern "$name" --dir "$DEST" 2>/dev/null; then
    echo "$name downloaded from $REPO@$TAG"
  else
    # GitHub Actions surfaces this in the run summary.
    echo "::warning::$name missing from $REPO release '$TAG' — the docs will show the text fallback"
    echo "could not fetch $name from $REPO@$TAG" >&2
  fi
done
