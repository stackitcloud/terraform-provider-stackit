#!/usr/bin/env bash
set -e
. $(dirname ${0})/utility.sh

BINARY_NAME=golangci-lint
INSTALL_TO=${BIN_DIR}/${BINARY_NAME}

install() {
    echo "  installing ${BINARY_NAME} ${GOLANGCI_LINT_VERSION}"

    TYPE=windows
    if [[ "${OSTYPE}" == linux* ]]; then
        TYPE=linux
    elif [[ "${OSTYPE}" == darwin* ]]; then
        TYPE=darwin
    fi

    case $(uname -m) in
    arm64|aarch64)
        ARCH=arm64
        ;;
    *)
        ARCH=amd64
        ;;
    esac

    BASE_URL=https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}
    URL=${BASE_URL}/golangci-lint-${GOLANGCI_LINT_VERSION}-${TYPE}-${ARCH}.tar.gz
    echo "  Downloading: ${URL}"
    download ${URL} | tar --extract --gzip --strip-components 1 --preserve-permissions -C ${BIN_DIR} -f-

    # Ensure the binary has the correct name
    if [ -f "${BIN_DIR}/golangci-lint" ] && [ "${BIN_DIR}/golangci-lint" != "${INSTALL_TO}" ]; then
        mv "${BIN_DIR}/golangci-lint" "${INSTALL_TO}"
    fi
}

get_version() {
    ${INSTALL_TO} version 2>/dev/null | awk '{print $4}'
}

update_if_necessary ${GOLANGCI_LINT_VERSION}
