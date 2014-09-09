Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/trusty64"

  config.vm.network "private_network", ip: "192.168.56.4"

  config.vm.network "forwarded_port", guest: 25555, host: 25555 # BOSH Director API

  config.vm.provider "virtualbox" do |v|
    v.memory = 4096
    v.cpus = 2
  end

  config.vm.provision "shell", inline: "apt-get -y install linux-image-extra-$(uname -r)" # aufs

  config.vm.provision "bosh" do |c|
    c.manifest = File.read("manifests/vagrant-bosh.yml")
  end
end
