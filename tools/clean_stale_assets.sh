#!/bin/bash

# Delete fingerprinted assets that no page in the built site references.
#
# Hugo rewrites its output tree in place and never deletes files it no
# longer emits, so every time an asset's content (and therefore its
# fingerprint) changes -- e.g. the search bundle, which embeds the site
# content -- a new index.min.<sha>.js / main.<sha>.css / *_hu<sha>_....webp
# lands in the tree while the old one is kept. Over time that accumulates
# hundreds of stale bundles in the gh-pages branch.
#
# Only fingerprint-named files are ever considered, and only when
# unreferenced, so files that live on in the tree without being rebuilt by
# this run -- in particular the versioned config docs, which reference the
# bundles they were built with -- keep working.

set -euo pipefail

DRY_RUN=0
if [[ "${1:-}" == "-n" ]]; then
  DRY_RUN=1
  shift
fi
DIR="${1:?usage: clean_stale_assets.sh [-n] <built-site-dir>}"

cd "$DIR"

# Fingerprint patterns: the asset pipeline emits <name>[.min].<sha>.js|css,
# the image pipeline <name>_hu<sha>_<params>.webp|png|jpg|avif.
FP='[A-Za-z0-9_/.-]*\.[0-9a-f]{64,}\.(js|css)|[A-Za-z0-9_/.-]*_hu[0-9a-f]{32}_[A-Za-z0-9_]*\.(webp|png|jpe?g|avif)'

# Basenames referenced by any built file. HTML is minified with unquoted
# attributes (src=https://...), so match bare filename patterns instead of
# attribute syntax. The search bundle embeds page content, so scanning the
# JS as well covers references from pages built in this run.
referenced=$(grep -rhoE "$FP" \
  --include='*.html' --include='*.css' --include='*.js' \
  --include='*.json' --include='*.xml' . | sed 's|.*/||' | sort -u || true)

deleted=0
freed=0
while IFS= read -r f; do
  base=${f##*/}
  if ! grep -qxF "$base" <<<"$referenced"; then
    if [[ "$DRY_RUN" == 1 ]]; then
      echo "would delete: $f"
    else
      freed=$((freed + $(stat -c%s "$f")))
      rm "$f"
      echo "deleted: $f"
    fi
    deleted=$((deleted + 1))
  fi
done < <(find . -path ./.git -prune -o -type f -print |
  grep -E '\.[0-9a-f]{64,}\.(js|css)$|_hu[0-9a-f]{32}_[A-Za-z0-9_]+\.(webp|png|jpe?g|avif)$' || true)

if [[ "$DRY_RUN" == 1 ]]; then
  echo "dry run: $deleted stale assets found"
else
  echo "deleted $deleted stale assets ($((freed / 1024)) KiB)"
fi
