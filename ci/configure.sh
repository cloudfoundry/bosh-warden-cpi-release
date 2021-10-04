#!/usr/bin/env bash

set -eu

fly -t bosh-ecosystem set-pipeline -p bosh-warden-cpi \
    -c ci/pipeline.yml \
    -l <( lpass show --notes "warden-cpi-concourse-secrets" )