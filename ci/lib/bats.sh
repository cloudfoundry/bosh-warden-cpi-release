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
  SKIP_RUBY_INSTALL=$5
  BOSH_CLI_VERSION=$6
  lite_director_ip="10.0.0.10"

  export CREDHUB_PROXY=ssh+socks5://${JUMPBOX_USERNAME}@${jumpbox_url}?private-key=${jumpbox_private_key_path}
  export CREDHUB_SERVER=https://${BOSH_ENVIRONMENT}:8844
  export CREDHUB_CA_CERT=$(mktemp)
  echo "${CREDHUB_CA_PEM}" > ${CREDHUB_CA_CERT}

  credhub login --skip-tls-validation
  deploy_director $stemcell_url $bosh_release_path $cpi_release_path $garden_linux_release_path $lite_director_ip
  BOSH_LITE_CA_CERT="$(credhub get -n /concourse/bosh/default_ca -j | jq .value.ca -r)"
  bosh -d bosh ssh -c "set -e -x; $(declare -f install_bats_prereqs); install_bats_prereqs $SKIP_RUBY_INSTALL $BOSH_CLI_VERSION"
  bosh -d bosh ssh -c "set -e -x; $(declare -f run_bats); run_bats $lite_director_ip '$stemcell_url' '${BOSH_LITE_CA_CERT}'"
  bosh -d bosh delete-vm $(bosh is --details --column=VM_CID) -n
  bosh -d bosh delete-deployment -n

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
      path: /releases/name=bosh/version
      value: latest
    - type: replace
      path: /instance_groups/name=bosh/azs?
      value: [az1]
    - type: replace
      path: /stemcells/alias=default/os
      value: ubuntu-bionic
    - type: remove
      path: /instance_groups/name=bosh/jobs/name=disable_agent
    - type: replace
      path: /instance_groups/name=bosh/persistent_disk_type
      value: large
    - type: replace
      path: /releases/name=bosh-warden-cpi/url
      value: file://((local-bosh-warden-cpi-release))

    - type: remove
      path: /releases/name=bosh-warden-cpi/sha1

    - type: replace
      path: /releases/name=bosh-warden-cpi/version
      value: latest
    '
    bosh int ${bd}/bosh.yml  \
        -o ${bd}/bosh-lite.yml \
        -o ${bd}/misc/bosh-dev.yml \
        -o <(echo -e "${ops}") \
        -v internal_ip="$lite_director_ip" \
        -v director_name="dev-director" \
        -v local-bosh-warden-cpi-release="$cpi_release_path"
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
        -v admin_password=admin \
        -v local-bosh-warden-cpi-release="$cpi_release_path" \
        $RC $SD $FF
}

install_bats_prereqs() {
  
  SKIP_RUBY_INSTALL=$1
  BOSH_CLI_VERSION=$2
  sudo apt-get -y update
  sudo apt-get install -y git libmysqlclient-dev libpq-dev libsqlite3-dev 

  export PATH=$PATH:/var/vcap/store/ruby/bin:/var/vcap/store/bosh/bin

  if bosh --version |  grep "$BOSH_CLI_VERSION"; then
    echo "found bosh, skipping download"
  else
    sudo mkdir -p /var/vcap/store/bosh/bin || true
    sudo wget -O /var/vcap/store/bosh/bin/bosh https://github.com/cloudfoundry/bosh-cli/releases/download/v$BOSH_CLI_VERSION/bosh-cli-$BOSH_CLI_VERSION-linux-amd64
    sudo chmod +x /var/vcap/store/bosh/bin/bosh
  fi
  sudo rm -rf /tmp/bosh
  git clone --depth=1 https://github.com/cloudfoundry/bosh.git /tmp/bosh

  if $SKIP_RUBY_INSTALL; then 
    echo "SKIPPING RUBY INSTALL, SINCE \$SKIP_RUBY_INSTALL=$SKIP_RUBY_INSTALL IS SET"
  else
    git clone https://github.com/postmodern/ruby-install
    sudo mkdir -p /var/vcap/store/ruby
    sudo ruby-install/bin/ruby-install --install-dir /var/vcap/store/ruby $(cat /tmp/bosh/src/Gemfile | grep '^ruby ' | cut -f2 -d"'")
  fi

  pushd /tmp/bosh/src
    # Pull in bat submodule
    git submodule update --init

    rm -f ~/.bosh_config
  popd
}

run_bats() {
  lite_director_ip=$1
  stemcell_url=$2
  export BOSH_CA_CERT="$3"
  if [ ! -f "$HOME/.ssh/id_rsa" ]; then
    # bosh_cli expects this key to exist
    ssh-keygen -t rsa -N "" -f ~/.ssh/id_rsa
    sudo cp ~/.ssh/id_rsa /tmp/id_rsa
  fi

  export PATH=$PATH:/var/vcap/store/ruby/bin:/var/vcap/store/bosh/bin



  # Download specific stemcell
  sudo wget -O /var/vcap/store/stemcell.tgz $stemcell_url
  cat << EOF > /tmp/debug.rc
  export BAT_BOSH_CLI=/var/vcap/store/bosh/bin/bosh
  export BAT_DEPLOYMENT_SPEC=/tmp/bosh-acceptance-tests/bats.spec
  export BAT_DIRECTOR=$lite_director_ip
  export BAT_DNS_HOST=$lite_director_ip
  export BAT_INFRASTRUCTURE=warden
  export BAT_NETWORKING=manual
  export BAT_STEMCELL=/var/vcap/store/stemcell.tgz
  export BAT_VCAP_PASSWORD=c1oudc0w
  export BAT_PRIVATE_KEY=~/tmp/id_rsa
  export BAT_RSPEC_FLAGS=( --tag ~multiple_manual_networks --tag ~raw_instance_storage )
  export PATH=${PATH}:/var/vcap/store/bosh/bin/:/var/vcap/store/ruby/bin/

  export BOSH_ENVIRONMENT=10.0.0.10
  export BOSH_CLIENT=admin
  export BOSH_CLIENT_SECRET=admin
  export BOSH_CA_CERT="$BOSH_CA_CERT"

EOF

  rm -rf /tmp/bosh-acceptance-tests
  git clone https://github.com/cloudfoundry/bosh-acceptance-tests.git /tmp/bosh-acceptance-tests

  pushd /tmp/bosh-acceptance-tests

    cat > bats.spec << EOF
---
cpi: warden
properties:
  instances: 1
  stemcell:
    name: bosh-warden-boshlite-ubuntu-bionic-go_agent
    version: latest
  persistent_disk: 1024
  networks:
  - name: default
    type: manual
    static_ip: 10.244.0.34
  second_static_ip: 10.244.0.35
EOF
    source /tmp/debug.rc
    sudo -E bundle install 
    sudo -E bundle exec rspec "${BAT_RSPEC_FLAGS[@]}"

  popd
}
