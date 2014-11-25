## Experimental `bosh-micro` usage

!!! `bosh-micro` CLI is still being worked on !!!

To start experimenting with bosh-warden-cpi release and new bosh-micro cli:

* Create a deployment directory

```
mkdir my-micro
```

* Create `manifest.yml` inside deployment directory. See examples manifest in [manifests](https://github.com/cppforlife/bosh-warden-cpi-release/tree/master/docs/manifests) folder. 

	* [without_registry.yml](https://github.com/cppforlife/bosh-warden-cpi-release/tree/master/docs/manifests/without_registry.yml) specifies default deployment manifest for warden CPI. By default warden CPI uses file injection to pass data to the agent on micro BOSH VM.
	* [with_registry.yml](https://github.com/cppforlife/bosh-warden-cpi-release/tree/master/docs/manifests/with_registry.yml) specified manifest that enables registry for warden CPI. Registry will be used by CPI to pass data to the agent. Registry server will be started on a server specified by registry properties. SSH tunnel will be started if ssh tunnel options are provided. SSH tunnel will start reverse ssh tunnel from the micro BOSH VM to the registry port. That makes registry available to the agent on remote machine.


* Set deployment manifest

```
bosh-micro deployment my-micro/manifest.yml
```

* Kick off a deploy

```
bosh-micro deploy ~/Downloads/bosh-warden-cpi-?.tgz ~/Downloads/stemcell.tgz
```

* See [micro CLI docs](https://github.com/cloudfoundry/bosh-micro-cli/blob/master/docs/cli_workflow.md) for current state of micro CLI project.
