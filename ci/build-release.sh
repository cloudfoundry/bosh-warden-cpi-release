#!/usr/bin/env bash

set -e -x

dev_version=`cat dev-version/number`

cd bosh-warden-cpi-release

pushd src/bosh-warden-cpi
./bin/test
popd

bosh create-release --version $dev_version --force --tarball=bosh-warden-cpi-$dev_version.tgz

mv *.tgz ../releases/
