#!/usr/bin/env bash
set -euo pipefail

# This script is used to test the example configurations found
# in the examples/ directory. If any examples fail to parse
# or are invalid, this script will fail by returning a non-zero
# exit code.
# It is used by the CI system to ensure that the examples are
# valid and up-to-date.

for example in examples/**/*.cfg; do
    echo "Testing ${example}"
    go run cmd/cloudprober.go -configtest -config_file "${example}"
done
