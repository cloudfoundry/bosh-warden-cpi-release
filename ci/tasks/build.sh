#!/usr/bin/env bash
set -euo pipefail

dev_version=$(cat dev-version/number)

cd bosh-cpi-src-in

bosh create-release \
  --version "${dev_version}" \
  --force \
  --tarball=../releases/"bosh-warden-cpi-${dev_version}.tgz"
