#!/usr/bin/env bash
set -e

jumpbox_url=${JUMPBOX_URL:-${JUMPBOX_IP}:22}
jumpbox_private_key_path=$(mktemp)
chmod 600 ${jumpbox_private_key_path}
echo "${JUMPBOX_PRIVATE_KEY}" > ${jumpbox_private_key_path}
export BOSH_ALL_PROXY=ssh+socks5://${JUMPBOX_USERNAME}@${jumpbox_url}?private-key=${jumpbox_private_key_path}


run_bats_on_vm() {
  stemcell_url=$1
  bosh_release_path=$2
  cpi_release_path=$3
  garden_linux_release_path=$4
  lite_director_ip="10.0.0.10"

  export CREDHUB_PROXY=ssh+socks5://${JUMPBOX_USERNAME}@${jumpbox_url}?private-key=${jumpbox_private_key_path}
  export CREDHUB_SERVER=https://${BOSH_ENVIRONMENT}:8844
  export CREDHUB_CA_CERT=$(mktemp)
  echo "${CREDHUB_CA_PEM}" > ${CREDHUB_CA_CERT}

  credhub login --skip-tls-validation
  deploy_director $stemcell_url $bosh_release_path $cpi_release_path $garden_linux_release_path $lite_director_ip
  BOSH_LITE_CA_CERT="$(credhub get -n /concourse/bosh/default_ca -j | jq .value.ca -r)"
  bosh -d bosh ssh -c "set -e -x; $(declare -f install_bats_prereqs); install_bats_prereqs"
  bosh -d bosh ssh -c "set -e -x; $(declare -f run_bats); run_bats $lite_director_ip '$stemcell_url' '${BOSH_LITE_CA_CERT}'"
}

deploy_director() {
  stemcell_url=$1
  bosh_release_path=$2
  cpi_release_path=$3
  garden_linux_release_path=$4
  lite_director_ip=$5

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
    - type: replace
      path: /instance_groups/name=bosh/persistent_disk_type
      value: large
    '
    bosh int ${bd}/bosh.yml  \
        -o ${bd}/bosh-lite.yml \
        -o ${bd}/misc/bosh-dev.yml \
        -o <(echo -e "${ops}") \
        -v internal_ip="$lite_director_ip" \
        -v director_name="dev-director"
    if $RECREATE ; then
      RC='--recreate'
    fi
    if $SKIP_DRAIN ; then
      SD='--skip-drain'
    fi
    if $FIX ; then
      FF='--fix'
    fi
    bosh -d bosh -n deploy ${bd}/bosh.yml \
        -o ${bd}/bosh-lite.yml \
        -o ${bd}/misc/bosh-dev.yml \
        -o <(echo -e "${ops}") \
        -v internal_ip="$lite_director_ip" \
        -v director_name="dev-director" \
        -v admin_password=admin
        $RC $SD $FF
}

install_bats_prereqs() {
  sudo apt-get -y update
  sudo apt-get install -y git libmysqlclient-dev libpq-dev libsqlite3-dev 

}

run_bats() {
  lite_director_ip=$1
  stemcell_url=$2
  export BOSH_CA_CERT="$3"
  export BOSH_ENVIRONMENT=$lite_director_ip 
  if [ ! -f "$HOME/.ssh/id_rsa" ]; then
    # bosh_cli expects this key to exist
    ssh-keygen -t rsa -N "" -f ~/.ssh/id_rsa
  fi

  git clone --depth=1 https://github.com/cloudfoundry/bosh.git
  if $SKIP_RUBY_INSTALL ; then 
    echo "SKIPPING RUBY INSTALL, SINCE \$SKIP_RUBY_INSTALL IS SET"
  else
    git clone https://github.com/postmodern/ruby-install
    sudo mkdir -p /var/vcap/store/ruby
    sudo ruby-install/bin/ruby-install --install-dir /var/vcap/store/ruby $(cat bosh/src/Gemfile | grep '^ruby ' | cut -f2 -d"'")
  fi
  export PATH=$PATH:/var/vcap/store/ruby/bin:/var/vcap/store/bosh/bin
  if type bosh || bosh --version |  grep "$BOSH_CLI_VERSION"; then
    echo "found bosh, skipping download"
  else
    sudo mkdir -p /var/vcap/store/bosh/bin || true
    sudo wget -O /var/vcap/store/bosh/bin/bosh https://github.com/cloudfoundry/bosh-cli/releases/download/v$BOSH_CLI_VERSION/bosh-cli-$BOSH_CLI_VERSION-linux-amd64
    sudo chmod +x /var/vcap/store/bosh/bin/bosh
  fi

  sudo gem install bundler
  cd bosh/src

  # Pull in bat submodule
  git submodule update --init

  sudo gem install bundler

  rm -rf ./.bundle
  sudo bundler install

  rm -f ~/.bosh_config
  sudo -E bundle exec bosh -n login --client=admin --client-secret=admin
  export UUID=$(bosh env --column uuid)
  # 10.244.10.2 is specified as static IP in bat/templates/warden.yml.erb
  cat > bats.spec << EOF
---
cpi: warden
properties:
  static_ip: $lite_director_ip
  uuid: $UUID
  pool_size: 1
  persistent_disk: 100
  stemcell:
    name: bosh-warden-boshlite-ubuntu-trusty-go_agent
    version: latest
  instances: 1
  mbus: "huh?"
  networks:
  - type: manual
    static_ip: $lite_director_ip
EOF

  # Download specific stemcell
  sudo wget -O /var/vcap/store/stemcell.tgz $stemcell_url

  export BAT_DIRECTOR=$lite_director_ip
  export BAT_DNS_HOST=$lite_director_ip
  export BAT_STEMCELL=`pwd`/stemcell.tgz
  export BAT_DEPLOYMENT_SPEC=`pwd`/bats.spec
  export BAT_VCAP_PASSWORD=c1oudc0w
  export BAT_INFRASTRUCTURE=warden
  export BAT_NETWORKING=manual

  cd bats


  # All bats' VMs should be in 10.244.10.0/24
  sed -i.bak "s/10\.244\.0\./10\.244\.10\./g" templates/warden.yml.erb

  bundle exec rake bat
}
