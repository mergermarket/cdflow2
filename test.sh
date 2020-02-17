#!/bin/bash

set -e

export TEST_ROOT="$PWD"

prefix=cdflow2-test-$RANDOM

# run a registry - this is required because of https://github.com/moby/moby/issues/32016

set +e
registry_id=$(cat .test-registry-id 2>/dev/null)
set -e

if [ -z "$PERSIST_REGISTRY" ]; then
    if docker inspect "$registry_id" 2>/dev/null >/dev/null; then
        docker stop $registry_id >/dev/null
        docker rm $registry_id >/dev/null
        rm .test-registry-id
        registry_id=""
    fi
fi

if [ -z "$registry_id" ]; then
    registry_id=$prefix-registry
    docker run -d -p 5000 --name $registry_id registry:2 >/dev/null
    echo $registry_id > .test-registry-id
fi

if [ "$PERSIST_REGISTRY" == "" ]; then
    function finish {
        docker stop $registry_id >/dev/null
        docker rm $registry_id >/dev/null
        rm .test-registry-id
    }
    trap finish EXIT
fi

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

goversion="$(go version)"
if [[ "$(go version)" != *"go1.13."* ]]; then
    echo "wrong go version - want 1.13, got $(go version)" >&2
    exit 1
fi

tests="$(go list ./... | grep -v 'cdflow2$' | grep -v cdflow2/test | sort)"
if [[ ! -z "$1" ]]; then
    tests="$(echo "$tests" | grep "$1")"
fi

go test $tests
