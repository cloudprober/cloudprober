#!/bin/bash -eu
#
# Copyright 2017-2023 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script generates Go code for the config protobufs.

PROTOC_VERSION="33.5"

GOPATH=$(go env GOPATH)

if [ -z "$GOPATH" ]; then
  echo "Go environment is not setup correctly. Please look at"
  echo "https://golang.org/doc/code.html to set up Go environment."
  exit 1
fi
echo GOPATH=${GOPATH}

if [ -z ${PROJECTROOT+x} ]; then
  # If PROJECTROOT is not set, try to determine it from script's location
  SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
  if [[ $SCRIPTDIR == *"/tools"* ]]; then
    PROJECTROOT="${SCRIPTDIR}/.."
  else
    echo "PROJECTROOT is not set and we are not able to determine PROJECTROOT"
    echo "from script's path."
    exit 1
  fi
fi
echo PROJECTROOT=${PROJECTROOT}

# Make sure protobuf compilation is set up correctly.
echo "Install protoc from the internet."
echo "============================================"
sleep 1

if [ "$(uname -s)" == "Darwin" ]; then
  os="osx"
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
  os="linux"
else
  echo "OS unsupported by this this build script. Please install protoc manually."
fi

arch=$(uname -m)

# On Apple arm64, protoc is called aarch64.
if [ "${os}" == "osx" ] && [ "${arch}" == "arm64" ]; then
  arch="aarch_64"
fi

protoc_package="protoc-${PROTOC_VERSION}-${os}-${arch}.zip"
protoc_package_url="https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/${protoc_package}"

TMPDIR=$(mktemp -d)
cd $TMPDIR
echo -e "Downloading protobuf compiler from..\n${protoc_package_url}"
echo "======================================================================"
wget "${protoc_package_url}"
unzip "${protoc_package}"
export protoc_path=${PWD}/bin/protoc
cd -

function cleanup {
  echo "Removing temporary directory used for protoc installation: ${TMPDIR}"
  rm  -r "${TMPDIR}"
}
trap cleanup EXIT

# Get go plugin for protoc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo "Generating Go code for protobufs.."
echo "======================================================================"
# Generate protobuf code from the root directory to ensure proper import paths.
cd $PROJECTROOT

# Create a temporary director to generate protobuf Go files.
TMPDIR=$(mktemp -d)
echo $TMPDIR
mkdir -p ${TMPDIR}/github.com/cloudprober/cloudprober
rsync -mr --exclude='.git' --include='*/' --include='*.proto' --include='*.cue' --exclude='*' . $TMPDIR/github.com/cloudprober/cloudprober

cd $TMPDIR

MODULE=github.com/cloudprober/cloudprober

# Generate Go code for proto
find ${MODULE} -type d | \
  while read -r dir
  do
    # Ignore directories with no proto files.
    ls ${dir}/*.proto > /dev/null 2>&1 || continue
    ${protoc_path} --go-grpc_out=. --go_out=. ${dir}/*.proto
  done

# Split external config proto into a separate package.
EXTERNAL_PROTO_DIR=${MODULE}/probes/external/proto
echo -e "syntax = \"proto2\";\n\npackage cloudprober.probes.external;" > ${EXTERNAL_PROTO_DIR}/server.proto
sed -n "/^\/\/ SERVER_MESSAGES_START/,\$p" \
  ${EXTERNAL_PROTO_DIR}/config.proto > ${EXTERNAL_PROTO_DIR}/server.proto
${protoc_path} --python_out=. ${MODULE}/probes/external/proto/server.proto
PY_SRC_DIR=${MODULE}/probes/external/serverutils/py/src/cloudprober/external
mkdir -p ${PY_SRC_DIR}
mv github/com/cloudprober/cloudprober/probes/external/proto/server_pb2.py ${PY_SRC_DIR}

# Copy generated files back to their original location.
find ${MODULE} \( -name *.pb.go -o -name *.py \) | \
  while read -r pbgofile
  do
    dst=${PROJECTROOT}/${pbgofile/github.com\/cloudprober\/cloudprober\//}
    cp "$pbgofile" "$dst"
  done

cd -

go generate ./pkg/protos/...
