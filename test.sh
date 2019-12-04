#!/bin/bash

set -e

export TEST_ROOT="$PWD"

prefix=cdflow-test-$RANDOM
export TEST_TERRAFORM_IMAGE=$prefix-teraform

# build supporting containers

set -x
docker build -t $TEST_TERRAFORM_IMAGE test/terraform

go test -v .