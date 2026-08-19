#!/usr/bin/env bash
set -euo pipefail

# This script is used to test the example configurations found
# in the examples/ directory. If any examples fail to parse
# or are invalid, this script will fail by returning a non-zero
# exit code.
# It is used by the CI system to ensure that the examples are
# valid and up-to-date.

# ** only recurses with globstar; without it these globs stop one level deep
# and configs in nested directories go untested.
shopt -s globstar

for example in examples/**/*.cfg; do
    # examples/extensions/ configs reference probe types that only exist in a
    # custom-built prober, so the stock binary can't validate them.
    case "${example}" in
        examples/extensions/*)
            echo "Skipping ${example} (needs a custom build)"
            continue
            ;;
        *)
            ;;
    esac
    echo "Testing ${example}"
    go run ./cmd/cloudprober/. -configtest -config_file "${example}"
done
