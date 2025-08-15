#!/usr/bin/env bash

# This script is used to manage the project, only used for installing the required tools for now
# Usage: ./project.sh [action]
# * tools: Install required tools to run the project
set -eo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)

action=$1

if [ "$action" = "help" ]; then
    [ -f "$0".man ] && man "$0".man || echo "No help, please read the script in ${script}, we will add help later"
elif [ "$action" = "tools" ]; then
    cd ${ROOT_DIR}

    go mod download

    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0
    go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.21.0
else
    echo "Invalid action: '$action', please use $0 help for help"
fi
