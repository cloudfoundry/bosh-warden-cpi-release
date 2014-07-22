#!/bin/bash

echo "Adding route entry to your local route table to enable direct container access. Your sudo password may be required."

# Vagrant VM IP address as specified by private_network in Vagrantfile
vagrant=192.168.56.4

# Container network pool as specified by warden.network_pool
vms=10.244.0.0/22

if [ `uname` = "Darwin" ]; then
  sudo route delete -net $vms $vagrant > /dev/null 2>&1
  sudo route add    -net $vms $vagrant
elif [ `uname` = "Linux" ]; then
  sudo route add    -net $vms gw $vagrant
fi
