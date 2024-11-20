#!/usr/bin/env bash
set -euo pipefail

cd bosh-cpi-src

bosh create-release \
  --timestamp-version \
  --force \
  --tarball=../releases/"bosh-warden-cpi-dev-release.tgz"
