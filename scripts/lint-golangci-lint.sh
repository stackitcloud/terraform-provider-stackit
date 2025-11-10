#!/usr/bin/env bash
# This script lints the SDK modules and the internal examples
# Pre-requisites: golangci-lint (provided by Makefile or system)
set -eo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)
GOLANG_CI_YAML_PATH="${ROOT_DIR}/golang-ci.yaml"
GOLANG_CI_ARGS="--allow-parallel-runners --timeout=5m --config=${GOLANG_CI_YAML_PATH}"

# Use provided golangci-lint binary or fallback to system installation
GOLANGCI_LINT_BIN="${1:-golangci-lint}"

if [ ! -x "${GOLANGCI_LINT_BIN}" ] && ! type -p "${GOLANGCI_LINT_BIN}" >/dev/null; then
    echo "golangci-lint not found at ${GOLANGCI_LINT_BIN} and not installed in PATH, unable to proceed."
    exit 1
fi

cd ${ROOT_DIR}
${GOLANGCI_LINT_BIN} run ${GOLANG_CI_ARGS}
