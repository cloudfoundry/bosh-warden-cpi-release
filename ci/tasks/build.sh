#!/usr/bin/env bash
set -euo pipefail

cd bosh-cpi-src-in

bosh create-release \
  --timestamp-version \
  --force \
  --tarball=../releases/"bosh-warden-cpi-dev-release.tgz"
