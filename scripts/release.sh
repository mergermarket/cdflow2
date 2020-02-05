#!/bin/bash

set -e

latest_tag="$(git describe --abbrev=0 --tags)"

echo latest version: $latest_tag

major=$(echo $latest_tag | sed -En 's/v([0-9]+).([0-9]+).([0-9])+/\1/p')
minor=$(echo $latest_tag | sed -En 's/v([0-9]+).([0-9]+).([0-9])+/\2/p')
patch=$(echo $latest_tag | sed -En 's/v([0-9]+).([0-9]+).([0-9])+/\3/p')

if [ "$latest_tag" != "v$major.$minor.$patch" ]; then
    echo error parsing version, major: "'$major'", minor: "'$minor'", patch: "'$patch'" >&2
    exit 1
fi

if [ "$1" == "major" ]; then
    major=$((major+1))
    minor=0
    patch=0
elif [ "$1" == "minor" ]; then
    minor=$((minor+1))
    patch=0
elif [ "$1" == "patch" ]; then
    patch=$((patch+1))
else
    echo 'Usage: script/release.sh major | minor | patch' >&2
    exit 1
fi

version=v$major.$minor.$patch
echo Releasing version $version...

./test.sh
git tag $version
git push
git push --tags