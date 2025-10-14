#!/usr/bin/env bash
# Add replace directives to local files to go.work
set -eo pipefail

while getopts "s:" option; do
    case "${option}" in
        s)
        SDK_DIR=${OPTARG}
        ;;
        
        *)
            echo "call: $0 [-s sdk-dir] <apis*>"
            exit 0
        ;;
    esac
done
shift $((OPTIND-1))

if [ -z "$SDK_DIR" ]; then
    SDK_DIR=../stackit-sdk-generator/sdk-repo-updated
    echo "No SDK_DIR set, using $SDK_DIR"
fi


if [ ! -f go.work ]; then
    go work init
    go work use .
else
    echo "go.work already exists"
fi

if [ $# -gt 0 ];then
    # modules passed via commandline
    for service in $*; do
        if [ ! -d $SDK_DIR/services/$service ]; then
            echo "service directory $SDK_DIR/services/$service does not exist"
            exit 1
        fi
        echo "replacing selected service $service"
        if [ "$service" = "core" ]; then
            go work edit -replace github.com/stackitcloud/stackit-sdk-go/core=$SDK_DIR/core
        else
            go work edit -replace github.com/stackitcloud/stackit-sdk-go/services/$service=$SDK_DIR/services/$service
        fi
    done
else
    # replace all modules
    echo "replacing all services"
    go work edit -replace github.com/stackitcloud/stackit-sdk-go/core=$SDK_DIR/core
    for n in $(find ${SDK_DIR}/services -name go.mod);do
        service=$(dirname $n)
        service=${service#${SDK_DIR}/services/}
        go work edit -replace github.com/stackitcloud/stackit-sdk-go/services/$service=$(dirname $n)
    done
fi
go work edit -fmt
go work sync
