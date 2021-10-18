#!/bin/bash

set -e # -x

cpi_path=$PWD/cpi

rm -f creds.yml

echo "-----> `date`: Create dev release"
bosh create-release --force --dir ./../ --tarball $cpi_path

echo "-----> `date`: Create env"
bosh create-env ~/workspace/bosh-deployment/bosh.yml \
  --state ./state.json \
  -o ~/workspace/bosh-deployment/virtualbox/cpi.yml \
  -o ~/workspace/bosh-deployment/virtualbox/outbound-network.yml \
  -o ~/workspace/bosh-deployment/bosh-lite.yml \
  -o ~/workspace/bosh-deployment/bosh-lite-runc.yml \
  -o ~/workspace/bosh-deployment/jumpbox-user.yml \
  -o ../manifests/dev.yml \
  --vars-store ./creds.yml \
  -v warden_cpi_path=$cpi_path \
  -v director_name="Bosh Lite Director" \
  -v internal_ip=192.168.50.16 \
  -v internal_gw=192.168.50.1 \
  -v internal_cidr=192.168.50.0/24 \
  -v outbound_network_name=NatNetwork

export BOSH_ENVIRONMENT=192.168.50.16
export BOSH_CA_CERT="$(bosh int creds.yml --path /director_ssl/ca)"
export BOSH_CLIENT=admin
export BOSH_CLIENT_SECRET="$(bosh int creds.yml --path /admin_password)"

echo "-----> `date`: Update cloud config"
bosh -n update-cloud-config ~/workspace/bosh-deployment/warden/cloud-config.yml

echo "-----> `date`: Upload stemcell"
bosh upload-stemcell --sha1 f399044d2ebe3351f0f1b0b3f97ef11464d283b4 \
  https://bosh.io/d/stemcells/bosh-warden-boshlite-ubuntu-xenial-go_agent?v=621.125

echo "-----> `date`: Delete previous deployment"
bosh -n -d zookeeper delete-deployment --force

echo "-----> `date`: Deploy"
bosh -n -d zookeeper deploy <(wget -O- https://raw.githubusercontent.com/cppforlife/zookeeper-release/master/manifests/zookeeper.yml)

echo "-----> `date`: Recreate all VMs"
bosh -n -d zookeeper recreate

echo "-----> `date`: Exercise deployment"
bosh -n -d zookeeper run-errand smoke-tests

echo "-----> `date`: Restart deployment"
bosh -n -d zookeeper restart

echo "-----> `date`: Report any problems"
bosh -n -d zookeeper cck --report

echo "-----> `date`: Delete random VM"
bosh -n -d zookeeper delete-vm `bosh -d zookeeper vms|sort|cut -f5|head -1`

echo "-----> `date`: Fix deleted VM"
bosh -n -d zookeeper cck --auto

echo "-----> `date`: Delete deployment"
bosh -n -d zookeeper delete-deployment

echo "-----> `date`: Clean up disks, etc."
bosh -n -d zookeeper clean-up --all

echo "-----> `date`: Deleting env"
bosh delete-env ~/workspace/bosh-deployment/bosh.yml \
  --state ./state.json \
  -o ~/workspace/bosh-deployment/virtualbox/cpi.yml \
  -o ~/workspace/bosh-deployment/virtualbox/outbound-network.yml \
  -o ~/workspace/bosh-deployment/bosh-lite.yml \
  -o ~/workspace/bosh-deployment/bosh-lite-runc.yml \
  -o ~/workspace/bosh-deployment/jumpbox-user.yml \
  -o ../manifests/dev.yml \
  --vars-store ./creds.yml \
  -v warden_cpi_path=$cpi_path \
  -v director_name="Bosh Lite Director" \
  -v internal_ip=192.168.50.16 \
  -v internal_gw=192.168.50.1 \
  -v internal_cidr=192.168.50.0/24 \
  -v outbound_network_name=NatNetwork

echo "-----> `date`: Done"
