#!/usr/bin/env bash

set -e -x
integer_version=$(cat dev-version/number | cut -f1 -d'.')

pushd bosh-warden-cpi-release
  cat > config/private.yml << EOF
---
blobstore:
  options:
    access_key_id: $BOSH_AWS_ACCESS_KEY_ID
    secret_access_key: $BOSH_AWS_SECRET_ACCESS_KEY
    assume_role_arn: $BOSH_AWS_ROLE_ARN
EOF

  bosh finalize-release --version $integer_version ../pipeline-bosh-warden-cpi-tarball/*.tgz

  # Be extra careful about not committing private.yml
  rm config/private.yml

  final_version=`git diff releases/*/index.yml | grep -E "^\+.+version" | sed s/[^0-9]*//g`
  git diff | cat
  git add .

  git config --global user.email "cf-bosh-eng@pivotal.io"
  git config --global user.name "CI"
  git commit -m "New final release v$final_version"

  echo $final_version > final_version
popd
