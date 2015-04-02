#!/usr/bin/env bash

set -e -x

cd bosh-warden-cpi-release

# todo remove installation
ls -la /root/.gem
rm -rf /root/.gem
gem install bosh_cli --no-ri --no-rdoc

cat > config/private.yml << EOF
---
blobstore:
  s3:
    access_key_id: $BOSH_AWS_ACCESS_KEY_ID
    secret_access_key: $BOSH_AWS_SECRET_ACCESS_KEY
EOF

bosh finalize release `echo ../pipeline-bosh-warden-cpi-tarball/*.tgz`

# Be extra careful about not committing private.yml
rm config/private.yml

git diff | cat
git add .

git config --global user.email "cf-bosh-eng@pivotal.io"
git config --global user.name "CI"
git commit -m "New final release"
