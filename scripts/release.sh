#!/bin/bash

set -e

require_clean_work_tree () {
    # Update the index
    git update-index -q --ignore-submodules --refresh
    err=0

    # Disallow unstaged changes in the working tree
    if ! git diff-files --quiet --ignore-submodules --
    then
        echo >&2 "cannot $1: you have unstaged changes."
        git diff-files --name-status -r --ignore-submodules -- >&2
        err=1
    fi

    # Disallow uncommitted changes in the index
    if ! git diff-index --cached --quiet HEAD --ignore-submodules --
    then
        echo >&2 "cannot $1: your index contains uncommitted changes."
        git diff-index --cached --name-status -r --ignore-submodules HEAD -- >&2
        err=1
    fi

    if [ $err = 1 ]
    then
        echo >&2 "Please commit or stash them."
        exit 1
    fi
}

require_clean_work_tree release

if [ "$2" == "" ]; then
    latest_tag="$(git describe --abbrev=0 --tags)"
else
    latest_tag="$2"
fi

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

set +e
git tag $version
if [ "$?" -ne "0" ]; then
    echo Tag failed, this can happen when there are two tags from the same >&2
    echo release. To work around add an additional parameter of the latest tag. >&2
    exit 1
fi
set -e

git push
git push --tags