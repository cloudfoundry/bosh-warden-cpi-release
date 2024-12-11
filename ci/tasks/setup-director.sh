#!/usr/bin/env bash

set -e

export HOME=/home/non-root-user
sudo chown -R non-root-user $(pwd)

source bosh-cpi-src/ci/tasks/utils.sh

check_param google_test_bucket_name
check_param google_subnetwork_range
check_param google_json_key_data

creds_file="${PWD}/director-creds/creds.yml"
state_file="${PWD}/director-state/manifest-state.json"
jumpbox_creds_file="${PWD}/jumpbox-creds/jumpbox-creds.yml"
jumpbox_state_file="${PWD}/jumpbox-state/jumpbox-manifest-state.json"
cpi_release_name=bosh-google-cpi
infrastructure_metadata="${PWD}/infrastructure/metadata"
deployment_dir="${PWD}/deployment"
google_json_key=$HOME/.config/gcloud/application_default_credentials.json

echo "Creating google json key..."
mkdir -p $HOME/.config/gcloud/
echo "${google_json_key_data}" > ${google_json_key}

read_infrastructure

export BOSH_CLI=${deployment_dir}/bosh

echo "Configuring google account..."
gcloud auth activate-service-account --key-file $HOME/.config/gcloud/application_default_credentials.json
gcloud config set project ${google_project}
gcloud config set compute/region ${google_region}
gcloud config set compute/zone ${google_zone}

export BOSH_CONFIG=${deployment_dir}/.boshconfig

echo "Creating ops files..."
# Use the locally built CPI
cat > "${deployment_dir}/ops_local_cpi.yml" <<EOF
---
- type: replace
  path: /releases/name=${cpi_release_name}?
  value:
    name: ${cpi_release_name}
    url: file://${deployment_dir}/${cpi_release_name}.tgz
EOF

# Use locally sourced stemcell
cat > "${deployment_dir}/ops_local_stemcell.yml" <<EOF
---
- type: replace
  path: /resource_pools/name=vms/stemcell?
  value:
    url: file://${deployment_dir}/stemcell.tgz
EOF

cat > "${deployment_dir}/enable_gcp_external_ip.yml" <<EOF
---
- path: /networks/name=default/subnets/0/cloud_properties/ephemeral_external_ip
  type: replace
  value: true
EOF

echo "Deploying Jumpbox..."
bosh create-env "jumpbox-deployment/jumpbox.yml" \
  -o "jumpbox-deployment/gcp/cpi.yml" \
  --state=${jumpbox_state_file} \
  --vars-store=${jumpbox_creds_file} \
  -v external_ip=${google_jumpbox_ip} \
  -v zone=${google_zone} \
  -v network=${google_network} \
  -v subnetwork=${google_subnetwork} \
  -v project_id=${google_project} \
  -v internal_cidr=${google_internal_cidr} \
  -v internal_gw=${google_internal_gw} \
  -v internal_ip=${google_internal_jumpbox_ip} \
  -v "tags=["jumpbox"]" \
  --var-file gcp_credentials_json=${google_json_key}

jumpbox_private_key=$(mktemp)
bosh int "${jumpbox_creds_file}" --path=/jumpbox_ssh/private_key > "${jumpbox_private_key}"
export BOSH_ALL_PROXY="ssh+socks5://jumpbox@${google_jumpbox_ip}:22?private-key=${jumpbox_private_key}"

pushd ${deployment_dir}
  function finish {
    echo "Final state of director deployment:"
    echo "=========================================="
    cat ${state_file}
    echo "=========================================="

    cp -r $HOME/.bosh ./
  }
  trap finish ERR

  echo "Using bosh version..."
  ${BOSH_CLI} --version

  echo "Deploying BOSH Director..."
  ${BOSH_CLI} create-env bosh-deployment/bosh.yml \
      --state=${state_file} \
      --vars-store=${creds_file} \
      -o bosh-deployment/gcp/cpi.yml \
      -o bosh-deployment/gcp/gcs-blobstore.yml \
      -o enable_gcp_external_ip.yml \
      -o bosh-deployment/jumpbox-user.yml \
      -o ops_local_cpi.yml \
      -o ops_local_stemcell.yml \
      -v director_name=micro-google \
      -v internal_cidr=${google_subnetwork_range} \
      -v internal_gw=${google_subnetwork_gateway} \
      -v internal_ip=${google_address_director_internal_ip} \
      -v external_ip=${google_address_director_ip} \
      --var-file gcp_credentials_json=${google_json_key} \
      -v project_id=${google_project} \
      -v zone=${google_zone} \
      -v "tags=[${google_firewall_internal}, ${google_firewall_external}]" \
      -v network=${google_network} \
      -v subnetwork=${google_subnetwork} \
      -v bucket_name=${google_test_bucket_name} \
     --var-file director_gcs_credentials_json=${google_json_key} \
     --var-file agent_gcs_credentials_json=${google_json_key}

  echo "Smoke testing connection to BOSH Director"
  export BOSH_ENVIRONMENT="${google_address_director_internal_ip}"
  export BOSH_CLIENT="admin"
  export BOSH_CLIENT_SECRET="$(${BOSH_CLI} interpolate ${creds_file} --path /admin_password)"
  export BOSH_CA_CERT="$(${BOSH_CLI} interpolate ${creds_file} --path /director_ssl/ca)"
  ${BOSH_CLI} env
  ${BOSH_CLI} login

  cloud_config=$(mktemp)
  cat <<EOF > "${cloud_config}"
azs:
- cloud_properties:
    zone: us-east1-b
  name: az1
vm_types:
- cloud_properties:
    machine_type: e2-micro
    root_disk_size_gb: 20
    root_disk_type: pd-ssd
  name: default
- cloud_properties:
    machine_type: e2-standard-2
    root_disk_size_gb: 50
    root_disk_type: pd-ssd
  name: large
vm_extensions:
- cloud_properties:
    root_disk_size_gb: 50
    root_disk_type: pd-ssd
  name: 50GB_ephemeral_disk
compilation:
  az: az1
  network: default
  reuse_compilation_vms: true
  vm_type: large
  workers: 5
networks:
- name: default
  subnets:
  - azs:
    - az1
    cloud_properties:
      ephemeral_external_ip: true
      network_name: wardencpi-manual
      subnetwork_name: wardencpi
      tags:
      - bosh-deployed
    dns:
    - 8.8.8.8
    gateway: 10.0.0.1
    range: 10.0.0.0/24
    reserved:
    - ${google_internal_jumpbox_ip}
    - ${google_address_director_internal_ip}
  type: manual
disk_types:
- disk_size: 3000
  name: default
- disk_size: 50000
  name: large
stemcells:
- alias: default
  os: ubuntu-jammy
  version: '1.639'
releases:
- name: bosh-dns
  sha1: fcdd6a0c9818da11d9dc081d7c4f3f97fd690035
  url: https://bosh.io/d/github.com/cloudfoundry/bosh-dns-release?v=1.39.0
  version: 1.39.0
update:
  canaries: 0
  canary_watch_time: 60000
  max_in_flight: 2
  update_watch_time: 60000
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
EOF

  ${BOSH_CLI} -n update-cloud-config "${cloud_config}"

  trap - ERR
  finish
popd
