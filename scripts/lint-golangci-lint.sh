#!/bin/bash
# This script lints the SDK modules and the internal examples
# Pre-requisites: golangci-lint
set -eo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)
GOLANG_CI_YAML_PATH="${ROOT_DIR}/golang-ci.yaml"
GOLANG_CI_ARGS="--allow-parallel-runners --timeout=5m --config=${GOLANG_CI_YAML_PATH}"

cd "${ROOT_DIR}"
go tool golangci-lint run ${GOLANG_CI_ARGS}
