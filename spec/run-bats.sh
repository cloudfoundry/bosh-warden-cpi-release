#!/bin/bash

set -e

director_target=192.168.56.4
director_user=admin
director_password=admin

bosh -n target $director_target
bosh -n login $director_user $director_password

director_uuid=$(bosh -n status --uuid)

spec_path=$PWD/bat.spec

# Create bat.spec used by BATS to generate BOSH manifest
cat > $spec_path << EOF
---
cpi: warden
properties:
  static_ip: 10.244.0.2
  uuid: $director_uuid
  pool_size: 1
  instances: 1
  stemcell:
    name: bosh-warden-boshlite-ubuntu-trusty-go_agent
    version: latest
  mbus: nats://nats:0b450ada9f830085e2cdeff6@10.42.49.80:4222
EOF

# Set up env vars used by BATS
export BAT_DEPLOYMENT_SPEC=$spec_path
export BAT_STEMCELL=$HOME/Downloads/bosh-stemcell-3-warden-boshlite-ubuntu-trusty-go_agent.tgz
export BAT_DIRECTOR=$director_target
export BAT_DNS_HOST=$director_target
export BAT_VCAP_PASSWORD=c1oudc0w

cd $HOME/workspace/bosh/bat

bundle exec rake bat
