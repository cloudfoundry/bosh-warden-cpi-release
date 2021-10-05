#!/usr/bin/env bash

set -e -x

dev_version=`cat dev-version/number`

cd bosh-warden-cpi-release

source .envrc

pushd src/bosh-warden-cpi
./bin/test
popd

bosh create-release --version $dev_version --force --tarball=release.tgz

mv *.tgz ../releases/
