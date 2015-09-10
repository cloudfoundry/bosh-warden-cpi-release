#!/usr/bin/env bash

set -e -x

mkdir out

dev_version=`cat dev-version/number`

cd bosh-warden-cpi-release

source .envrc

./src/github.com/cppforlife/bosh-warden-cpi/bin/test

# todo remove installation
gem install bosh_cli --no-ri --no-rdoc

bosh -n create release --version $dev_version --with-tarball

mv dev_releases/bosh-warden-cpi/*.tgz ../out/
