#!/bin/bash

# To generate config docs for a specific version, copy this script to a
# different location and checkout cloudprober repo at the desired version.
# Then run the script with the version as an argument. For example:
#   cp tools/gen_config_docs.sh /tmp
#   git checkout v0.13.7
#   /tmp/gen_config_docs.sh v0.13.7
DOCS_VERSION=$1

if [ -z "${DOCS_VERSION}" ]; then
    DOCS_VERSION=$(git describe --exact-match --exclude tip --tags HEAD 2>/dev/null || /bin/true)

    if [ -z "${DOCS_VERSION}" ]; then
        DOCS_VERSION=$(git rev-parse --abbrev-ref HEAD)
    fi
fi

go install github.com/manugarg/protodoc/cmd/protodoc@latest
protodoc --proto_root_dir=. --package_prefix=github.com/cloudprober/cloudprober \
    --format=yaml --out_dir=docs/_config_docs/${DOCS_VERSION}/yaml --extra_msgs=cloudprober.rds.file.FileResources
protodoc --proto_root_dir=. --package_prefix=github.com/cloudprober/cloudprober \
    --format=textpb --out_dir=docs/_config_docs/${DOCS_VERSION}/textpb  --extra_msgs=cloudprober.rds.file.FileResources
mkdir -p docs/content/docs/config/${DOCS_VERSION}

for dir in docs/_config_docs/${DOCS_VERSION}/yaml/*; do
  cat > docs/content/docs/config/${DOCS_VERSION}/$(basename $dir).md <<EOF
---
menu:
  docs:
    parent: "config"
    weight: 23
    params:
      hide: true
title: "$(basename $dir) Config"
---

{{% config-docs-nav %}}

{{% config-doc config="$(basename $dir)" version="${DOCS_VERSION}" %}}

EOF
done