#!/usr/bin/env bash

set -eo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)

# Create a new empty directory for the docs
if [ -d "${ROOT_DIR}/docs" ]; then
    rm -rf "${ROOT_DIR}/docs"
fi
mkdir -p "${ROOT_DIR}/docs"

echo ">> Generating documentation"
go tool tfplugindocs generate \
    --provider-name "stackit"
