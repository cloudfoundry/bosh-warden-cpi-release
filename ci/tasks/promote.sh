#!/usr/bin/env bash
set -e
set -x

integer_version=$(cat dev-version/number | cut -f1 -d'.')

pushd bosh-cpi-src
  echo "${PRIVATE_YML}" > config/private.yml

  bosh finalize-release --version "${integer_version}" ../pipeline-bosh-warden-cpi-tarball/*.tgz

  # Be extra careful about not committing private.yml
  rm config/private.yml

  final_version=$(git diff releases/*/index.yml | grep -E "^\+.+version" | sed s/[^0-9]*//g)
  git diff | cat
  git add .

  git config --global user.email "cf-bosh-eng@pivotal.io"
  git config --global user.name "CI"
  git commit -m "New final release v${final_version}"

  echo "${final_version}" > final_version
popd
