#!/usr/bin/env bash

set -e -x

mkdir out

cd bosh-warden-cpi-release

# todo remove installation
gem install bosh_cli --no-ri --no-rdoc

bosh -n create release --with-tarball

mv dev_releases/bosh-warden-cpi/*.tgz ../out/

cd ../out

# todo s3 concourse resource does not properly handle plus sign
mv *.tgz `echo *.tgz|tr '+dev' '-dev'`
