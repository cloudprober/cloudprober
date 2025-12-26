#!/bin/bash
# Script to generate protobuf files for Consul surfacer

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_FILE="$SCRIPT_DIR/config.proto"

echo "Generating protobuf files for Consul surfacer..."

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed"
    echo "Please install protoc from https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# Check if protoc-gen-go is installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Error: protoc-gen-go is not installed"
    echo "Install it with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

# Generate the Go code
protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    --proto_path="$SCRIPT_DIR" \
    --proto_path="$(go list -m -f '{{.Dir}}' github.com/cloudprober/cloudprober)" \
    "$PROTO_FILE"

echo "Successfully generated: config.pb.go"
