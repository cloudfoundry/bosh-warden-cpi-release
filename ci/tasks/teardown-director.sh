#!/usr/bin/env bash

set -e

source bosh-warden-cpi-release/ci/tasks/utils.sh

creds_file="${PWD}/director-creds/creds.yml"
state_file="${PWD}/director-state/manifest-state.json"
jumpbox_creds_file="${PWD}/jumpbox-creds/jumpbox-creds.yml"
jumpbox_state_file="${PWD}/jumpbox-state/jumpbox-manifest-state.json"
cpi_release_name=bosh-google-cpi
deployment_dir="${PWD}/deployment"
google_json_key=${deployment_dir}/google_key.json
infrastructure_metadata="${PWD}/infrastructure/metadata"

echo "Creating google json key..."
echo "${google_json_key_data}" > ${google_json_key}
mkdir -p $HOME/.config/gcloud/
cp ${google_json_key} $HOME/.config/gcloud/application_default_credentials.json

read_infrastructure

echo "Setting up artifacts..."
cp ./bosh-cli/bosh-cli-* ${deployment_dir}/bosh && chmod +x ${deployment_dir}/bosh
export BOSH_CLI=${deployment_dir}/bosh

# Export prefixed variables so they are accessible
echo "Populating environment with BOSH_ prefixed vars"
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

echo "Using bosh version..."
${BOSH_CLI} --version

pushd ${deployment_dir}
  echo "Destroying BOSH Director..."
  ${BOSH_CLI} delete-env bosh-deployment/bosh.yml \
      --state=${state_file} \
      --vars-store=${creds_file} \
      -o bosh-deployment/gcp/cpi.yml \
      -o bosh-deployment/gcp/gcs-blobstore.yml \
      -o bosh-deployment/external-ip-not-recommended.yml \
      -o ops_local_cpi.yml \
      -o ops_local_stemcell.yml \
      -o enable_gcp_external_ip.yml \
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
popd

echo "Destroying Jumpbox..."
bosh delete-env "jumpbox-deployment/jumpbox.yml" \
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
