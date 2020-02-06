#!/bin/bash

set -e

version=$1
sha256=$(shasum -a 256 cdflow2-darwin-amd64 | cut -f 1 -d " ")

git clone git@github.com:mergermarket/homebrew-tap

cd homebrew-tap

cat <<END > cdflow2.rb
class Cdflow2 < Formula
  desc     "Deployment tooling for continuous delivery"
  homepage "https://github.com/mergermarket/cdflow2"
  version  "$version"
  url      "https://github.com/mergermarket/cdflow2/releases/download/$version/cdflow2-darwin-amd64"
  sha256   "$sha256"
  
  def install
    bin.install "cdflow2-darwin-amd64"
    mv bin/"cdflow2-darwin-amd64", bin/"cdflow2"
  end
end
END

git commit -am "Update cdflow2 to $version"
git push
