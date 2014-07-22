Vagrant.configure("2") do |config|
  config.vm.box = "precise64"
  config.vm.box_url = "http://files.vagrantup.com/precise64.box"

  config.vm.network "private_network", ip: "192.168.56.4"

  config.vm.network "forwarded_port", { # BOSH Director API
    guest: 25555, 
    host:  25555,
  }

  config.vm.provider "virtualbox" do |v|
    v.memory = 4096
    v.cpus = 2
  end

  config.vm.provision "bosh" do |c|
    c.manifest = File.read("manifests/vagrant-bosh.yml")
  end
end
