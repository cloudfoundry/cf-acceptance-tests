#!/usr/bin/env bash

# This script re-generates the Go code in this directory from the `.proto` files
# in the `api`. [protoc](https://github.com/protocolbuffers/protobuf/releases)
# must be installed beforehand.
# Usage: `rpc/logcache_v1/generate.sh`.

set -euxo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

git clone --depth 1 "https://github.com/googleapis/googleapis.git" "${TMP_DIR}/google-api"
mv "${TMP_DIR}/google-api/google" "${TMP_DIR}/google"
git clone --depth 1 "https://github.com/cloudfoundry/loggregator-api" "${TMP_DIR}/loggregator-api"

export GOBIN="${TMP_DIR}/hack/bin"
export PATH="${GOBIN}:${PATH}"
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

pushd "${REPO_ROOT}/.." > /dev/null
  mkdir -p "${TMP_DIR}/go-log-cache/api"
  cp -R "${REPO_ROOT}/api/v1" "${TMP_DIR}/go-log-cache/api/v1"

  protoc \
    -I="${TMP_DIR}" \
    --go_out=$TMP_DIR \
    --go-grpc_out=$TMP_DIR \
    --grpc-gateway_out=$TMP_DIR \
    $TMP_DIR/go-log-cache/api/v1/*.proto

  mv $TMP_DIR/code.cloudfoundry.org/go-log-cache/v2/rpc/logcache_v1/* $REPO_ROOT/rpc/logcache_v1
popd > /dev/null
