#!/usr/bin/env bash

# copy or rename sample.tfrc.example and adjust it
TERRAFORM_CONFIG=$(pwd)/sample.tfrc
export TERRAFORM_CONFIG

tofu "$1"
