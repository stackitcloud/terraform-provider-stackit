#!/usr/bin/env bash
# Copyright (c) STACKIT


# ./tf.sh apply > >(tee -a stdout.log) 2> >(tee -a stderr.log >&2)

# copy or rename sample.tfrc.example and adjust it
TERRAFORM_CONFIG=$(pwd)/sample.tfrc
export TERRAFORM_CONFIG

parsed_options=$(
  getopt -n "$0" -o l -- "$@"
) || exit
eval "set -- $parsed_options"
while [ "$#" -gt 0 ]; do
  case $1 in
    (-l) TF_LOG=TRACE
         export TF_LOG
         shift;;
    (--) shift; break;;
    (*) echo "Unknown option ${1}" # should never be reached.
  esac
done

terraform "$*"
