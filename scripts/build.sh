#!/bin/bash

set -e

version=$1
if [ -z "$version" ]; then
    echo version is required >&2
    exit 1
fi

rm -f cdflow2*

git archive --format=tar.gz --prefix=cdflow2-$version/ --output=cdflow2-$version.tar.gz $version

GOOS=linux GOARCH=amd64 go build -o cdflow2-linux-amd64 -ldflags="-X main.version=$version" .
GOOS=darwin GOARCH=amd64 go build -o cdflow2-darwin-amd64 -ldflags="-X main.version=$version" .
GOOS=windows GOARCH=amd64 go build -o cdflow2-windows-amd64 -ldflags="-X main.version=$version" .
GOOS=linux GOARCH=arm64 go build -o cdflow2-linux-arm64 -ldflags="-X main.version=$version" .
