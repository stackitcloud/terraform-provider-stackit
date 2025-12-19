#!/usr/bin/env bash
# Copyright (c) STACKIT


# copy or rename sample.tfrc.example and adjust it
TERRAFORM_CONFIG=$(pwd)/sample.tfrc
export TERRAFORM_CONFIG

tofu "$1"
