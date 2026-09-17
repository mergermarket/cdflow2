#!/bin/bash

set -e

version=$1
sha256=$(shasum -a 256 cdflow2-darwin-amd64 | cut -f 1 -d " ")
sha256arm=$(shasum -a 256 cdflow2-darwin-arm64 | cut -f 1 -d " ")

git clone git@github.com:mergermarket/homebrew-tap

cd homebrew-tap

cat <<END > cdflow2.rb
class Cdflow2 < Formula
  desc     "Deployment tooling for continuous delivery"
  homepage "https://github.com/mergermarket/cdflow2"
  version  "$version"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/mergermarket/cdflow2/releases/download/$version/cdflow2-darwin-amd64"
      sha256 "$sha256"
    elsif Hardware::CPU.arm?
      url "https://github.com/mergermarket/cdflow2/releases/download/$version/cdflow2-darwin-arm64"
      sha256 "$sha256arm"
    end
  end
  
  def install
    if Hardware::CPU.intel?
      bin.install "cdflow2-darwin-amd64" => "cdflow2"
    elsif Hardware::CPU.arm?
      bin.install "cdflow2-darwin-arm64" => "cdflow2"
    end
  end
end
END

git config --global user.email "platform@acuris.com"
git config --global user.name "cdflow2 publish action"

git commit -am "Update cdflow2 to $version"
git push
