#!/usr/bin/env bash

set -e -x

source $(dirname $0)/lib/bats.sh

if $DEV_RELEASE; then
  git clone https://github.com/cloudfoundry/bosh-warden-cpi-release.git bosh-warden-cpi-release-dev
  pushd bosh-warden-cpi-release-dev
    cpi_release_path=`pwd`/dev-release.tgz
    bosh create-release --force --tarball $cpi_release_path
  popd

else
  cpi_release_path=$PWD/pipeline-bosh-warden-cpi-tarball/*.tgz
fi

#credhub login --skip-tls-validation
warden_stemcell_url=`cat warden-ubuntu-bionic-stemcell/url`
iaas_stemcell_url=`cat iaas-stemcell/url`
iaas_stemcell_version=`cat iaas-stemcell/version`
bosh_release_path=$PWD/bosh-release/*.tgz

garden_linux_release_path=$PWD/garden-linux-release/*.tgz

run_bats_on_vm $warden_stemcell_url $iaas_stemcell_url $iaas_stemcell_version $bosh_release_path $cpi_release_path $garden_linux_release_path $SKIP_RUBY_INSTALL $BOSH_CLI_VERSION
