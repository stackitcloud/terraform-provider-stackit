#!/usr/bin/env bash


# ./tf.sh apply > >(tee -a stdout.log) 2> >(tee -a stderr.log >&2)

usage() {
  echo "$0 usage:" && grep "[[:space:]].)\ #" "$0" | sed 's/#//' | sed -r 's/([a-z])\)/-\1/';
  exit 0;
}

[ $# -eq 0 ] && usage

CONFIG_FOLDER=$(dirname "$0")
BINARY=terraform

while getopts ":b:hdit" arg; do
  case $arg in
    b) # Set binary (default is terraform).
      BINARY=${OPTARG}
      shift 2
      ;;
    d) # Set log level to DEBUG.
      TF_LOG=DEBUG
      export TF_LOG
      shift
      ;;
    i) # Set log level to INFO.
      TF_LOG=INFO
      export TF_LOG
      shift
      ;;
    t) # Set log level to TRACE.
      TF_LOG=TRACE
      export TF_LOG
      shift
      ;;
    h | *) # Display help.
      usage
      ;;
  esac
done

TERRAFORM_CONFIG=${CONFIG_FOLDER}/config.tfrc
export TERRAFORM_CONFIG

${BINARY} "$@"
