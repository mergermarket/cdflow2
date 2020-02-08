#!/bin/bash

set -e

export TEST_ROOT="$PWD"

prefix=cdflow2-test-$RANDOM

# run a registry - this is required because of https://github.com/moby/moby/issues/32016

registry_id=$prefix-registry

docker run -d -p 5000 --name $registry_id registry:2 >/dev/null

function finish {
    docker stop $registry_id >/dev/null
    docker rm $registry_id >/dev/null
}
trap finish EXIT 

registry="localhost:$(docker inspect --format='{{(index (index .NetworkSettings.Ports "5000/tcp") 0).HostPort}}' "$registry_id")"

# build supporting containers

export TEST_TERRAFORM_IMAGE="$registry/$prefix-terraform"
echo "
    building and pushing test terraform image...
"
docker build -q -t "$TEST_TERRAFORM_IMAGE" test/terraform
docker push "$TEST_TERRAFORM_IMAGE"

export TEST_TERRAFORM_REPO_DIGEST="$(docker image inspect "$TEST_TERRAFORM_IMAGE" -f '{{index .RepoDigests 0}}')"

echo "
    building and pushing test config image...
"
export TEST_CONFIG_IMAGE="$registry/$prefix-config"
docker build -q -t "$TEST_CONFIG_IMAGE" test/config
docker push "$TEST_CONFIG_IMAGE"

echo "
    building and pushing test release image...
"
export TEST_RELEASE_IMAGE="$registry/$prefix-release"
docker build -q -t "$TEST_RELEASE_IMAGE" test/release
docker push "$TEST_RELEASE_IMAGE"

echo "
    running tests...
"

tests="$(go list ./... | grep -v 'cdflow2$' | grep -v cdflow2/test)"
if [[ ! -z "$1" ]]; then
    tests="$(echo "$tests" | grep "$1")"
fi
set +e
go test $tests

# go test ouput doesn't make it that obvious
status=$?
if [ "$status" -ne "0" ]; then
    echo "
!! TEST FAILURE !!
" >&2
    exit $status
fi
