## bosh-warden-cpi-release

A [BOSH](https://github.com/cloudfoundry/bosh) release for `bosh-warden-cpi` written in Go.


### Example Vagrant environment

`bosh-warden-cpi` release can be deployed with any BOSH Director 
just like any other BOSH release. It can also be deployed with 
`vagrant-bosh` Vagrant plugin for local usage and development.

1. Install Vagrant dependencies

```
vagrant plugin install vagrant-bosh
gem install bosh_cli --no-ri --no-rdoc
```

1. Create a new VM with BOSH Director and BOSH Warden CPI releases

```
git submodule update --init
vagrant up
```

Note: See [deployment manifest](manifests/vagrant-bosh.yml) 
to see how bosh and bosh-warden-cpi releases are collocated.

1. Target deployed BOSH Director

```
bosh target localhost:25555
bosh status
```


### Configuring private network

1. Make sure `Vagrantfile` is configured with a private network 
  (`vagrant reload` if necessary)

1. Run `./add-private-network-route.sh`

1. You should be able to directly access Warden containers from the host via their IP


### Running tests

1. Follow instructions above to set up Vagrant environment and configure private network

1. Clone BOSH repository into `$HOME/workspace/bosh` to get BATS source code

1. Download Warden stemcell #3 to `$HOME/Downloads/bosh-stemcell-3-warden-boshlite-ubuntu-trusty-go_agent.tgz`
   from [BOSH Artifacts](https://s3.amazonaws.com/bosh-jenkins-artifacts/bosh-stemcell/warden/bosh-stemcell-3-warden-boshlite-ubuntu-trusty-go_agent.tgz)

1. Run BOSH Acceptance Tests via `spec/run-bats.sh`


### Experimental `bosh-micro` usage

See [bosh-micro usage doc](docs/bosh-micro-usage.md)


### Todo

- Use standalone BATS and CPI lifecycle tests
- Use BATS errand for running tests
