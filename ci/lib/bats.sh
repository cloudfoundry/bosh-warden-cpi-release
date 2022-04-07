#!/usr/bin/env bash
set -e

jumpbox_url=${JUMPBOX_URL:-${JUMPBOX_IP}:22}
jumpbox_private_key_path=$(mktemp)
chmod 600 ${jumpbox_private_key_path}
echo "${JUMPBOX_PRIVATE_KEY}" > ${jumpbox_private_key_path}
export BOSH_ALL_PROXY=ssh+socks5://${JUMPBOX_USERNAME}@${jumpbox_url}?private-key=${jumpbox_private_key_path}

run_bats_on_vm() {
  stemcell_url=$1
  iaas_stemcell_url=$2
  iaas_stemcell_version=$3
  bosh_release_path=$4
  cpi_release_path=$5
  garden_linux_release_path=$6
  SKIP_RUBY_INSTALL=$7
  BOSH_CLI_VERSION=$8

  export CREDHUB_PROXY=ssh+socks5://${JUMPBOX_USERNAME}@${jumpbox_url}?private-key=${jumpbox_private_key_path}
  export CREDHUB_SERVER=https://${BOSH_ENVIRONMENT}:8844
  export CREDHUB_CA_CERT=$(mktemp)
  echo "${CREDHUB_CA_PEM}" > ${CREDHUB_CA_CERT}

  credhub login --skip-tls-validation
  deploy_director $iaas_stemcell_url $iaas_stemcell_version $bosh_release_path $cpi_release_path $garden_linux_release_path
  lite_director_ip="$(bosh -d bosh-warden-cpi-bats-director instances --json | jq -r '.Tables[].Rows[] | select( .instance | contains("bosh"))'.ips)"

  BOSH_LITE_CA_CERT="$(credhub get -n /concourse/bosh/default_ca -j | jq .value.ca -r)"
  bosh -d bosh-warden-cpi-bats-director ssh -c "set -e -x; $(declare -f install_bats_prereqs); install_bats_prereqs $SKIP_RUBY_INSTALL $BOSH_CLI_VERSION"
  bosh -d bosh-warden-cpi-bats-director ssh -c "set -e -x; $(declare -f run_bats); run_bats $lite_director_ip '$stemcell_url' '${BOSH_LITE_CA_CERT}'"
  bosh -d bosh-warden-cpi-bats-director delete-vm $(bosh is --details --column=VM_CID) -n
  bosh -d bosh-warden-cpi-bats-director delete-deployment -n
}

deploy_director() {
  iaas_stemcell_url=$1
  iaas_stemcell_version=$2
  bosh_release_path=$3
  cpi_release_path=$4
  garden_linux_release_path=$5

  # Upload specific dependencies
  bosh upload-release $bosh_release_path
  bosh upload-release $cpi_release_path
  bosh upload-release $garden_linux_release_path
  bosh upload-stemcell $iaas_stemcell_url

  # Deploy empty VM so we can get the IP
  empty_manifest=$(mktemp)
  cat <<EOF > $empty_manifest
instance_groups:
- azs:
  - az1
  instances: 1
  jobs: []
  name: bosh
  networks:
  - name: default
  stemcell: default
  vm_type: default
name: bosh
releases: []
stemcells:
- alias: default
  os: ubuntu-bionic
  version: $iaas_stemcell_version
update:
  canaries: 0
  canary_watch_time: 60000
  max_in_flight: 2
  update_watch_time: 60000
EOF

  bosh -d bosh-warden-cpi-bats-director -n deploy $empty_manifest

  lite_director_ip="$(bosh -d bosh-warden-cpi-bats-director instances --json | jq -r '.Tables[].Rows[] | select( .instance | contains("bosh"))'.ips)"

  git clone https://github.com/cloudfoundry/bosh-deployment.git
  ops="
  - type: replace
    path: /releases/name=bosh/version
    value: latest
  - type: replace
    path: /releases/name=bosh/version
    value: latest
  - type: replace
    path: /instance_groups/name=bosh/azs?
    value: [az1]
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
  - type: remove
    path: /instance_groups/name=bosh/networks/0/static_ips
  - type: replace
    path: /stemcells/0/version
    value: $iaas_stemcell_url
  - type: replace
    path: /name
    value: bosh-warden-cpi-bats-director
  "

  bosh -d bosh-warden-cpi-bats-director -n deploy ${bd}/bosh.yml \
      -o ./bosh-deployment/bosh-lite.yml \
      -o ./bosh-deployment/misc/bosh-dev.yml \
      -o <(echo -e "${ops}") \
      -v internal_ip="$lite_director_ip" \
      -v director_name="dev-director" \
      -v admin_password=admin \
      -v local-bosh-warden-cpi-release="$cpi_release_path"
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

  export BOSH_ENVIRONMENT=$lite_director_ip
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
