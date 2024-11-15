#!/bin/bash

# To generate config docs for a specific version, copy this script to a
# different location and checkout cloudprober repo at the desired version.
# Then run the script with the version as an argument. For example:
#   cp tools/gen_config_docs.sh /tmp
#   git checkout v0.13.7
#   /tmp/gen_config_docs.sh v0.13.7
DOCS_VERSION=$1
RELEASE=$1

if [ -z "${DOCS_VERSION}" ]; then
    DOCS_VERSION=$(git describe --exact-match --exclude tip --tags HEAD 2>/dev/null || /bin/true)

    if [ -z "${DOCS_VERSION}" ]; then
        DOCS_VERSION=$(git rev-parse --abbrev-ref HEAD)
    fi
fi

DOCS_VERSION=${DOCS_VERSION//\//_}

ORIGINAL_DIR=$(pwd)

if [ "${RELEASE}" == "latest" ]; then
  RELEASE=$(curl -s https://api.github.com/repos/cloudprober/cloudprober/releases/latest | grep 'tag_name' | cut -d '"' -f4)
fi

if [ ! -z "${RELEASE}" ]; then
  TEMPDIR=$(mktemp -d) && cd $TEMPDIR
  wget https://github.com/cloudprober/cloudprober/archive/refs/tags/${RELEASE}.tar.gz
  tar -xzf ${RELEASE}.tar.gz
  mv cloudprober-${RELEASE/v/} cloudprober
  cd cloudprober
fi

go install github.com/manugarg/protodoc/cmd/protodoc@latest
protodoc --proto_root_dir=. --package_prefix=github.com/cloudprober/cloudprober \
    --format=yaml --out_dir=${ORIGINAL_DIR}/docs/_config_docs/${DOCS_VERSION}/yaml --extra_msgs=cloudprober.rds.file.FileResources
protodoc --proto_root_dir=. --package_prefix=github.com/cloudprober/cloudprober \
    --format=textpb --out_dir=${ORIGINAL_DIR}/docs/_config_docs/${DOCS_VERSION}/textpb  --extra_msgs=cloudprober.rds.file.FileResources

BASE_PATH=${ORIGINAL_DIR}/docs/content/docs/config/${DOCS_VERSION}
mkdir -p ${BASE_PATH}

MENU_HDR="menu:
  docs:
    parent: \"config\"
    weight: 23
    params:
      hide: true
"
TITLE_VERSION=""

if [ "${DOCS_VERSION}" != "latest" ]; then
  MENU_HDR=""
  TITLE_VERSION=" (${DOCS_VERSION})"
fi

generate_config_files() {
  local base_path="$1"
  local menu_hdr="$2"
  for dir in ${ORIGINAL_DIR}/docs/_config_docs/${DOCS_VERSION}/textpb/*; do
    baseName=$(basename $dir)
    if [ ! -d $dir ]; then
      continue
    fi
    cat > ${base_path}/${baseName}.md <<EOF
---
${menu_hdr}title: "${baseName^} Config${TITLE_VERSION}"
---

{{% config-docs-nav version="${DOCS_VERSION}" %}}

{{% config-doc config="$(basename $dir)" version="${DOCS_VERSION}" %}}

EOF
  done
}

# Call the function with BASE_PATH
generate_config_files "${BASE_PATH}" "${MENU_HDR}"

cp ${ORIGINAL_DIR}/docs/content/docs/config/_index.md ${BASE_PATH}/

# Copy latest configs to non-versioned path as well to make sure
# we don't break existing links.
if [ "${DOCS_VERSION}" == "latest" ]; then
  echo "Copying latest configs to non-versioned path as well."
  NON_VERSIONED_BASE_PATH=${ORIGINAL_DIR}/docs/content/docs/config
  mkdir -p ${NON_VERSIONED_BASE_PATH}
  generate_config_files "${NON_VERSIONED_BASE_PATH}" ""
fi
