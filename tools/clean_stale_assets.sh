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

usage() {
  echo "usage: clean_stale_assets.sh [-n|--dry-run] <built-site-dir>" >&2
  exit 2
}

DRY_RUN=0
DIR=""
for arg in "$@"; do
  case "$arg" in
    -n|--dry-run) DRY_RUN=1 ;;
    -h|--help) usage ;;
    -*) echo "error: unknown flag: $arg" >&2; usage ;;
    *)
      [[ -z "$DIR" ]] || { echo "error: unexpected argument: $arg" >&2; usage; }
      DIR="$arg"
      ;;
  esac
done
[[ -n "$DIR" ]] || usage
[[ -d "$DIR" ]] || { echo "error: not a directory: $DIR" >&2; exit 2; }

cd "$DIR"

# Fingerprint patterns: the asset pipeline emits <name>[.min].<sha>.js|css,
# the image pipeline <name>_hu<sha>_<params>.webp|png|jpg|avif.
FP='[A-Za-z0-9_/.-]*\.[0-9a-f]{64,}\.(js|css)|[A-Za-z0-9_/.-]*_hu[0-9a-f]{32}_[A-Za-z0-9_]*\.(webp|png|jpe?g|avif)'
CANDIDATES='\.[0-9a-f]{64,}\.(js|css)$|_hu[0-9a-f]{32}_[A-Za-z0-9_]+\.(webp|png|jpe?g|avif)$'

# Fingerprint-named files on disk. grep -c exits 1 with a zero count when
# there are none; that is a normal outcome, hence the `|| true`.
candidate_count=$(find . -path ./.git -prune -o -type f -print |
  grep -cE "$CANDIDATES" || true)

# Basenames referenced by any built file. HTML is minified with unquoted
# attributes (src=https://...), so match bare filename patterns instead of
# attribute syntax. The search bundle embeds page content, so scanning the
# JS as well covers references from pages built in this run.
#
# grep exits 1 on no matches and 2 on real errors (unreadable file, broken
# pattern); only the former is tolerable. A partial scan must never feed
# the deletion loop.
grep_rc=0
referenced=$(grep -rhoE "$FP" \
  --include='*.html' --include='*.css' --include='*.js' \
  --include='*.json' --include='*.xml' . | sed 's|.*/||' | sort -u) || grep_rc=$?
if (( grep_rc > 1 )); then
  echo "error: reference scan failed (grep exit $grep_rc); refusing to delete" >&2
  exit 1
fi
# A real built site always references its own bundles from its HTML. Zero
# references alongside candidates means the tree or the scan is broken,
# and proceeding would delete every asset.
if (( candidate_count > 0 )) && [[ -z "$referenced" ]]; then
  echo "error: $candidate_count fingerprinted assets but no references found;" \
    "the tree or the scan is broken; refusing to delete" >&2
  exit 1
fi

deleted=0
freed=0
while IFS= read -r f; do
  base=${f##*/}
  if ! grep -qxF "$base" <<<"$referenced"; then
    if [[ "$DRY_RUN" == 1 ]]; then
      echo "would delete: $f"
    else
      # wc -c is portable (GNU stat -c is not) and tolerates the file
      # disappearing between find and here.
      freed=$((freed + $(wc -c < "$f" 2>/dev/null || echo 0)))
      rm "$f"
      echo "deleted: $f"
    fi
    deleted=$((deleted + 1))
  fi
done < <(find . -path ./.git -prune -o -type f -print | grep -E "$CANDIDATES" || true)

if [[ "$DRY_RUN" == 1 ]]; then
  echo "dry run: $deleted stale assets found"
else
  echo "deleted $deleted stale assets ($((freed / 1024)) KiB)"
fi
