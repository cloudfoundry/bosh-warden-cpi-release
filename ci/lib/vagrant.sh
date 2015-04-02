#!/usr/bin/env bash

set -e -x

# todo only cleanup when did vagrant up?
trap clean_vagrant EXIT

vagrant_dir=$(mktemp -d /tmp/ssh_key.XXXXXXXXXX)

vagrant_up() {
	set_up_vagrant_private_key

	(
		set -e
		cd $vagrant_dir
		cat > Vagrantfile << EOF
Vagrant.configure("2") do |config|
  config.vm.box = "cloudfoundry/bosh-lite"
  config.vm.provider :aws do |aws, _|
  end
end
EOF

		vagrant up --provider aws
	)
}

set_up_vagrant_private_key() {
  if [ ! -f "$BOSH_LITE_PRIVATE_KEY" ]; then
    key_path=$(mktemp -d /tmp/ssh_key.XXXXXXXXXX)/value
    echo "$BOSH_LITE_PRIVATE_KEY" > $key_path
    chmod 600 $key_path
    export BOSH_LITE_PRIVATE_KEY=$key_path
  fi
}

vagrant_ip() {
	( cd $vagrant_dir && vagrant ssh-config | grep HostName | awk '{print $2}' )
}

vagrant_ssh() {
	(
		set -e
		cd $vagrant_dir
		vagrant ssh -c "$@"
	)
}

clean_vagrant() {
  ( cd $vagrant_dir && vagrant destroy -f || true )
}
