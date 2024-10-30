#!/usr/bin/env bash
set -e
set -x

jumpbox_url=${JUMPBOX_URL:-${JUMPBOX_IP}:22}
jumpbox_private_key_path=$(mktemp)
chmod 600 "${jumpbox_private_key_path}"
echo "${JUMPBOX_PRIVATE_KEY}" > "${jumpbox_private_key_path}"
export BOSH_ALL_PROXY="ssh+socks5://${JUMPBOX_USERNAME}@${jumpbox_url}?private-key=${jumpbox_private_key_path}"

run_bats_on_vm() {
  stemcell_url=$1
  iaas_stemcell_url=$2
  iaas_stemcell_version=$3
  bosh_release_path=$4
  cpi_release_path=$5
  garden_linux_release_path=$6
  BOSH_CLI_VERSION=$7

  vars_store_path=$(mktemp)
  deploy_director "${iaas_stemcell_url}" "${iaas_stemcell_version}" "${bosh_release_path}" "${cpi_release_path}" "${garden_linux_release_path}" "${vars_store_path}"
  lite_director_ip="$(bosh -d bosh-warden-cpi-bats-director instances --json | jq -r '.Tables[].Rows[] | select( .instance | contains("bosh"))'.ips)"

  BOSH_LITE_CA_CERT="$(bosh int --path /default_ca/ca "${vars_store_path}")"
  bosh -d bosh-warden-cpi-bats-director ssh -c "set -e -x; $(declare -f install_bats_prereqs); install_bats_prereqs ${BOSH_CLI_VERSION}"
  bosh -d bosh-warden-cpi-bats-director ssh -c "set -e -x; $(declare -f run_bats); run_bats $lite_director_ip '$stemcell_url' '${BOSH_LITE_CA_CERT}'"
  bosh -d bosh-warden-cpi-bats-director delete-deployment -n
}

deploy_director() {
  iaas_stemcell_url=$1
  iaas_stemcell_version=$2
  bosh_release_path=$3
  cpi_release_path=$4
  garden_linux_release_path=$5
  vars_store_path=$6

  # Upload specific dependencies
  bosh upload-release $bosh_release_path
  bosh upload-release $cpi_release_path
  bosh upload-release $garden_linux_release_path
  bosh upload-stemcell $iaas_stemcell_url

  # Deploy empty VM so we can get the IP
  empty_manifest=$(mktemp)
  cat <<EOF > "${empty_manifest}"
instance_groups:
- azs:
  - az1
  instances: 1
  jobs: []
  name: bosh
  networks:
  - name: default
  stemcell: default
  vm_type: large
name: bosh-warden-cpi-bats-director
releases: []
stemcells:
- alias: default
  os: ubuntu-jammy
  version: $iaas_stemcell_version
update:
  canaries: 0
  canary_watch_time: 60000
  max_in_flight: 2
  update_watch_time: 60000
EOF

  bosh -d bosh-warden-cpi-bats-director -n deploy "${empty_manifest}"

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
    value: $iaas_stemcell_version
  - type: replace
    path: /name
    value: bosh-warden-cpi-bats-director
  - type: replace
    path: /instance_groups/name=bosh/vm_type?
    value: large
  "

  bosh -d bosh-warden-cpi-bats-director -n deploy ./bosh-deployment/bosh.yml \
      -o ./bosh-deployment/bosh-lite.yml \
      -o ./bosh-deployment/misc/bosh-dev.yml \
      -o <(echo -e "${ops}") \
      -v internal_ip="$lite_director_ip" \
      -v director_name="dev-director" \
      -v admin_password=admin \
      -v local-bosh-warden-cpi-release="$cpi_release_path" \
      --vars-store "${vars_store_path}"
}

install_bats_prereqs() {
  BOSH_CLI_VERSION=$1
  sudo apt-get -y update
  sudo apt-get install -y git libmysqlclient-dev libpq-dev libsqlite3-dev

  export PATH=$PATH:/var/vcap/store/ruby/bin:/var/vcap/store/bosh/bin

  if bosh --version |  grep "${BOSH_CLI_VERSION}"; then
    echo "found bosh, skipping download"
  else
    sudo mkdir -p /var/vcap/store/bosh/bin || true
    sudo wget -O /var/vcap/store/bosh/bin/bosh \
      "https://github.com/cloudfoundry/bosh-cli/releases/download/v${BOSH_CLI_VERSION}/bosh-cli-${BOSH_CLI_VERSION}-linux-amd64"
    sudo chmod +x /var/vcap/store/bosh/bin/bosh
  fi
  sudo rm -rf /tmp/bosh
  git clone --depth=1 https://github.com/cloudfoundry/bosh.git /tmp/bosh

  git clone https://github.com/postmodern/ruby-install
  sudo mkdir -p /var/vcap/store/ruby
  sudo ruby-install/bin/ruby-install \
    --install-dir /var/vcap/store/ruby "$(cat /tmp/bosh/src/.ruby-version)"

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
  export BATS_ENV_FILE=/tmp/bats_env_file.sh

  # Download specific stemcell
  sudo wget -O /var/vcap/store/stemcell.tgz "${stemcell_url}"
  cat << EOF > "${BATS_ENV_FILE}"
  export BAT_BOSH_CLI=/var/vcap/store/bosh/bin/bosh
  export BAT_DEPLOYMENT_SPEC=/tmp/bosh-acceptance-tests/bats.spec
  export BAT_DIRECTOR=$lite_director_ip
  export BAT_DNS_HOST=$lite_director_ip
  export BAT_INFRASTRUCTURE=warden
  export BAT_STEMCELL=/var/vcap/store/stemcell.tgz
  export BAT_RSPEC_FLAGS=( --tag ~multiple_manual_networks --tag ~raw_instance_storage --tag ~os --tag ~reboot )
  export PATH=${PATH}:/var/vcap/store/bosh/bin/:/var/vcap/store/ruby/bin/

  export BOSH_ENVIRONMENT=$lite_director_ip
  export BOSH_CLIENT=admin
  export BOSH_CLIENT_SECRET=admin
  export BOSH_CA_CERT="$BOSH_CA_CERT"
EOF

  rm -rf /tmp/bosh-acceptance-tests
  git clone https://github.com/cloudfoundry/bosh-acceptance-tests.git /tmp/bosh-acceptance-tests

  pushd /tmp/bosh-acceptance-tests

  ssh_public_key="$( cat ~/.ssh/id_rsa.pub )"
  ssh_private_key="$( sed 's/$/\\n/' ~/.ssh/id_rsa | tr -d '\n' )"
  cat > bats.spec << EOF
---
cpi: warden
properties:
  instances: 1
  ssh_key_pair:
    public_key: "${ssh_public_key}"
    private_key: "${ssh_private_key}"
  stemcell:
    name: bosh-warden-boshlite-ubuntu-jammy-go_agent
    version: latest
  persistent_disk: 1024
  networks:
  - name: default
    type: manual
    static_ip: 10.244.0.34
  second_static_ip: 10.244.0.35
EOF
    # shellcheck disable=SC1090
    source "${BATS_ENV_FILE}"
    sudo --preserve-env bundle install
    sudo --preserve-env bundle exec rspec "${BAT_RSPEC_FLAGS[@]}"

  popd
}

warden_stemcell_url=$(cat warden-ubuntu-jammy-stemcell/url)
iaas_stemcell_url=$(cat iaas-stemcell/url)
iaas_stemcell_version=$(cat iaas-stemcell/version)
bosh_release_path=$(ls "${PWD}"/bosh-release/*.tgz)
cpi_release_path=$(ls "${PWD}"/pipeline-bosh-warden-cpi-tarball/*.tgz)
garden_linux_release_path=$(ls "${PWD}"/garden-linux-release/*.tgz)
bosh_cli_version=$(cat bosh-cli-github-release/version)

run_bats_on_vm \
  "${warden_stemcell_url}" \
  "${iaas_stemcell_url}" \
  "${iaas_stemcell_version}" \
  "${bosh_release_path}" \
  "${cpi_release_path}" \
  "${garden_linux_release_path}" \
  "${bosh_cli_version}"