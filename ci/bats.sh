#!/usr/bin/env bash

set -e -x

source $(dirname $0)/lib/bats.sh

stemcell_url=`cat warden-ubuntu-bionic-stemcell/url`
bosh_release_path=$PWD/bosh-release/*.tgz
cpi_release_path=$PWD/pipeline-bosh-warden-cpi-tarball/*.tgz
garden_linux_release_path=$PWD/garden-linux-release/*.tgz

run_bats_on_vm $stemcell_url $bosh_release_path $cpi_release_path $garden_linux_release_path
