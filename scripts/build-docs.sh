#!/bin/bash

set -e

name=cdflow2-docs-build-$RANDOM

docker pull mergermarket/docz-site-builder

function cleanup {
  echo "Removing $name"
  docker rm $name >/dev/null
}
trap cleanup EXIT

docker run \
    --init -i \
    --name $name \
    -v $PWD/docs/doczrc.js:/app/doczrc.js \
    -v $PWD/docs/src:/app/src \
    mergermarket/docz-site-builder \
    docz build

docker cp $name:/app/.docz/dist docs/dist

echo Wrote docs/dist
