#!/bin/bash

set -e

export TEST_ROOT="$PWD"

prefix=cdflow2-test-$RANDOM
export TEST_TERRAFORM_IMAGE=$prefix-terraform
export TEST_CONFIG_IMAGE=$prefix-config

# build supporting containers

set -x
docker build -t $TEST_TERRAFORM_IMAGE test/terraform
docker build -t $TEST_CONFIG_IMAGE test/config

go test -v .