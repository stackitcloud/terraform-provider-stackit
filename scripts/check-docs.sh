#!/usr/bin/env bash

# This script is used to ensure for PRs the docs are up-to-date via the CI pipeline
# Usage: ./check-docs.sh
set -eo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)

before_hash=$(find docs -type f -exec sha256sum {} \; | sort | sha256sum | awk '{print $1}')

# re-generate the docs
$ROOT_DIR/scripts/tfplugindocs.sh

after_hash=$(find docs -type f -exec sha256sum {} \; | sort | sha256sum | awk '{print $1}')

if [[ "$before_hash" == "$after_hash" ]]; then
	echo "Docs are up-to-date"
else 
	echo "Changes detected. Docs are *not* up-to-date."
	exit 1
fi
