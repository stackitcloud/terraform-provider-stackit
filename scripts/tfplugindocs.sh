#!/usr/bin/env bash
# Pre-requisites: tfplugindocs
set -eo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)
EXAMPLES_DIR="${ROOT_DIR}/examples"
PROVIDER_NAME="stackit"

# Create a new empty directory for the docs
if [ -d ${ROOT_DIR}/docs ]; then
    rm -rf ${ROOT_DIR}/docs
fi
mkdir -p ${ROOT_DIR}/docs

echo ">> Generating documentation"
tfplugindocs generate \
    --provider-name "stackit"
