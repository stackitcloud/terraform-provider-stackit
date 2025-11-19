#!/usr/bin/env bash
# Common utility functions for tool installation scripts

ROOT_DIR=$(git rev-parse --show-toplevel)
BIN_DIR="${ROOT_DIR}/bin"

# Ensure bin directory exists
mkdir -p "${BIN_DIR}"

# Download function using curl
download() {
    local URL=$1
    if command -v curl &> /dev/null; then
        curl -sSfL "${URL}"
    elif command -v wget &> /dev/null; then
        wget -qO- "${URL}"
    else
        echo "Error: Neither curl nor wget found. Please install one of them."
        exit 1
    fi
}

# Update tool if necessary
update_if_necessary() {
    local EXPECTED_VERSION=$1

    if [ -x "${INSTALL_TO}" ]; then
        CURRENT_VERSION=$(get_version 2>/dev/null || echo "")
        if [ "${CURRENT_VERSION}" = "${EXPECTED_VERSION}" ]; then
            echo "  ${BINARY_NAME} ${EXPECTED_VERSION} already installed"
            return 0
        else
            echo "  ${BINARY_NAME} version mismatch (current: ${CURRENT_VERSION}, expected: ${EXPECTED_VERSION})"
            echo "  updating to ${EXPECTED_VERSION}..."
        fi
    fi

    install

    INSTALLED_VERSION=$(get_version 2>/dev/null || echo "unknown")
    if [ "${INSTALLED_VERSION}" = "${EXPECTED_VERSION}" ]; then
        echo "  ${BINARY_NAME} ${EXPECTED_VERSION} installed successfully"
    else
        echo "  Warning: installed version (${INSTALLED_VERSION}) does not match expected version (${EXPECTED_VERSION})"
    fi
}
