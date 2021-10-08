#!/usr/bin/env bash
set -e

jumpbox_url=${JUMPBOX_URL:-${JUMPBOX_IP}:22}
jumpbox_private_key_path=$(mktemp)
chmod 600 ${jumpbox_private_key_path}
echo "${JUMPBOX_PRIVATE_KEY}" > ${jumpbox_private_key_path}
export BOSH_ALL_PROXY=ssh+socks5://${JUMPBOX_USERNAME}@${jumpbox_url}?private-key=${jumpbox_private_key_path}
export CREDHUB_PROXY=ssh+socks5://${JUMPBOX_USERNAME}@${jumpbox_url}?private-key=${jumpbox_private_key_path}
export CREDHUB_SERVER=https://${BOSH_ENVIRONMENT}:8844
export CREDHUB_CA_CERT=$(mktemp)
echo "${CREDHUB_CA_PEM}" > ${CREDHUB_CA_CERT}

run_bats_on_vm() {
  stemcell_url=$1
  bosh_release_path=$2
  cpi_release_path=$3
  garden_linux_release_path=$4


  deploy_director $stemcell_url $bosh_release_path $cpi_release_path $garden_linux_release_path
  vagrant_ssh "set -e -x; $(declare -f install_bats_prereqs); install_bats_prereqs"
  vagrant_ssh "set -e -x; $(declare -f run_bats); run_bats 10.244.8.2 '$stemcell_url'"
}

deploy_director() {
  stemcell_url=$1
  bosh_release_path=$2
  cpi_release_path=$3
  garden_linux_release_path=$4

  # Upload specific dependencies
  bosh upload-release $bosh_release_path
  bosh upload-release $cpi_release_path
  bosh upload-release $garden_linux_release_path


  git clone https://github.com/cloudfoundry/bosh-deployment.git
  bd=./bosh-deployment
    ops='

    - type: replace
      path: /releases/name=bosh/version
      value: latest
    - type: replace
      path: /instance_groups/name=bosh/azs?
      value: [az1]
    - type: replace
      path: /stemcells/alias=default/os
      value: ubuntu-bionic
    - type: replace
      path: /instance_groups/name=bosh/jobs/-
      value:
        name: registry
        release: bosh
    - path: /instance_groups/name=bosh/properties/registry?
      type: replace
      value:
        address: ((internal_ip))
        db:
          adapter: postgres
          database: bosh
          host: 127.0.0.1
          password: ((postgres_password))
          user: postgres
        host: ((internal_ip))
        http:
          password: ((registry_password))
          port: 25777
          user: registry
        password: ((registry_password))
        port: 25777
        username: registry
    - path: /variables/-
      type: replace
      value:
        name: registry_password
        type: password
    - type: remove
      path: /instance_groups/name=bosh/jobs/name=disable_agent
    '
    bosh -d bosh -n deploy ${bd}/bosh.yml \
        -o ${bd}/bosh-lite.yml \
        -o ${bd}/misc/bosh-dev.yml \
        -o <(echo -e "${ops}") \
        -v internal_ip=10.0.0.10 \
        -v director_name="dev-director"
}

install_bats_prereqs() {
  sudo apt-get -y update
  sudo apt-get install -y git libmysqlclient-dev libpq-dev libsqlite3-dev
  sudo gem install bundler --no-ri --no-rdoc
}

run_bats() {
  director_ip=$1
  stemcell_url=$2

  if [ ! -f "$HOME/.ssh/id_rsa" ]; then
    # bosh_cli expects this key to exist
    ssh-keygen -t rsa -N "" -f ~/.ssh/id_rsa
  fi

  git clone --depth=1 https://github.com/cloudfoundry/bosh.git

  cd bosh

  # Pull in bat submodule
  git submodule update --init

  sudo gem install bundler

  rm -rf ./.bundle
  bundle install

  rm -f ~/.bosh_config
  bundle exec bosh -n target $director_ip
  bundle exec bosh -n login admin admin

  # 10.244.10.2 is specified as static IP in bat/templates/warden.yml.erb
  cat > bats.spec << EOF
---
cpi: warden
properties:
  static_ip: 10.244.10.2
  uuid: $(bundle exec bosh -u admin -p admin status --uuid | tail -n 1)
  pool_size: 1
  persistent_disk: 100
  stemcell:
    name: bosh-warden-boshlite-ubuntu-trusty-go_agent
    version: latest
  instances: 1
  mbus: "huh?"
  networks:
  - type: manual
    static_ip: 10.244.10.2
EOF

  # Download specific stemcell
  wget -O stemcell.tgz $stemcell_url

  export BAT_DIRECTOR=$director_ip
  export BAT_DNS_HOST=$director_ip
  export BAT_STEMCELL=`pwd`/stemcell.tgz
  export BAT_DEPLOYMENT_SPEC=`pwd`/bats.spec
  export BAT_VCAP_PASSWORD=c1oudc0w
  export BAT_INFRASTRUCTURE=warden
  export BAT_NETWORKING=manual

  cd bat

  # 10.244.10.0/24 is inside the Director container, so route .10.0/24 traffic into it
  sudo route add -net 10.244.10.0/24 gw $director_ip

  # All bats' VMs should be in 10.244.10.0/24
  sed -i.bak "s/10\.244\.0\./10\.244\.10\./g" templates/warden.yml.erb

  bundle exec rake bat
}
