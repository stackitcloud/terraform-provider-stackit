#!/usr/bin/env bash
# This script lints the SDK modules and the internal examples
# Pre-requisites: golangci-lint
set -eo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)
GOLANG_CI_YAML_PATH="${ROOT_DIR}/golang-ci.yaml"
GOLANG_CI_ARGS="--allow-parallel-runners --timeout=5m --config=${GOLANG_CI_YAML_PATH}"

if type -p golangci-lint >/dev/null; then
    :
else
    echo "golangci-lint not installed, unable to proceed."
    exit 1
fi

cd ${ROOT_DIR}
golangci-lint run ${GOLANG_CI_ARGS}
