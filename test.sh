#!/bin/bash

set -e

export TEST_ROOT="$PWD"

prefix=cdflow2-test-$RANDOM
export TEST_TERRAFORM_IMAGE=$prefix-terraform
export TEST_CONFIG_IMAGE=$prefix-config
export TEST_RELEASE_IMAGE=$prefix-release

# build supporting containers

set -x
docker build -t $TEST_TERRAFORM_IMAGE test/terraform
docker build -t $TEST_CONFIG_IMAGE test/config
docker build -t $TEST_RELEASE_IMAGE test/release

go test -v .