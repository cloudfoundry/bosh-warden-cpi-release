#!/usr/bin/env bash

set -eu

fly -t bosh set-pipeline -p bosh-warden-cpi \
    -c ci/pipeline.yml
