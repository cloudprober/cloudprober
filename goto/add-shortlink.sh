#!/bin/bash
# Add a new cloudprober.org/goto/<keyword> shortlink.
# Usage: ./add-shortlink.sh <keyword> <destination-url>
#
# Example:
#   ./add-shortlink.sh docs https://cloudprober.org/docs/
#   => creates cloudprober.org/goto/docs -> https://cloudprober.org/docs/

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ $# -ne 2 ]; then
  echo "Usage: $0 <keyword> <destination-url>" >&2
  exit 1
fi

KEYWORD="$1"
DEST="$2"
FILE="${SCRIPT_DIR}/${KEYWORD}.md"

if [ -f "$FILE" ]; then
  echo "Error: shortlink '${KEYWORD}' already exists at ${FILE}" >&2
  exit 1
fi

cat > "$FILE" <<EOF
---
title: "${KEYWORD}"
dest: "${DEST}"
sitemap_exclude: true
---
EOF

echo "Created shortlink: cloudprober.org/goto/${KEYWORD} -> ${DEST}"
