package vm

import (
	wrdnclient "github.com/cloudfoundry-incubator/garden/client"
	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"

	bwcdisk "github.com/cppforlife/bosh-warden-cpi/disk"
)

type WardenVM struct {
	id string

	wardenClient    wrdnclient.Client
	agentEnvService AgentEnvService

	hostBindMounts  HostBindMounts
	guestBindMounts GuestBindMounts

	logger boshlog.Logger

	containerExists bool
}

func NewWardenVM(
	id string,
	wardenClient wrdnclient.Client,
	agentEnvService AgentEnvService,
	hostBindMounts HostBindMounts,
	guestBindMounts GuestBindMounts,
	logger boshlog.Logger,
	containerExists bool,
) WardenVM {
	return WardenVM{
		id: id,

		wardenClient:    wardenClient,
		agentEnvService: agentEnvService,

		hostBindMounts:  hostBindMounts,
		guestBindMounts: guestBindMounts,

		logger:          logger,
		containerExists: containerExists,
	}
}

func (vm WardenVM) ID() string { return vm.id }

func (vm WardenVM) Delete() error {
	// Destroy container before deleting bind mounts to avoid 'device is busy' error
	if vm.containerExists {
		err := vm.wardenClient.Destroy(vm.id)
		if err != nil {
			return err
		}
	}

	vm.logger.Debug("zaksoup", "vm is deleting ephemeral")
	err := vm.hostBindMounts.DeleteEphemeral(vm.id)
	if err != nil {
		return err
	}

	// No need to unmount since DetachDisk should have been called before this
	err = vm.hostBindMounts.DeletePersistent(vm.id)
	if err != nil {
		return err
	}

	return nil
}

func (vm WardenVM) AttachDisk(disk bwcdisk.Disk) error {

	if !vm.containerExists {
		return bosherr.New("VM does not exist")
	}

	agentEnv, err := vm.agentEnvService.Fetch()
	if err != nil {
		return bosherr.WrapError(err, "Fetching agent env")
	}

	err = vm.hostBindMounts.MountPersistent(vm.id, disk.ID(), disk.Path())
	if err != nil {
		return bosherr.WrapError(err, "Mounting persistent bind mounts dir")
	}

	diskHintPath := vm.guestBindMounts.MountPersistent(disk.ID())

	agentEnv = agentEnv.AttachPersistentDisk(disk.ID(), diskHintPath)

	err = vm.agentEnvService.Update(agentEnv)
	if err != nil {
		return bosherr.WrapError(err, "Updating agent env")
	}

	return nil
}

func (vm WardenVM) DetachDisk(disk bwcdisk.Disk) error {

	if !vm.containerExists {
		return bosherr.New("VM does not exist")
	}

	agentEnv, err := vm.agentEnvService.Fetch()
	if err != nil {
		return bosherr.WrapError(err, "Fetching agent env")
	}

	err = vm.hostBindMounts.UnmountPersistent(vm.id, disk.ID())
	if err != nil {
		return bosherr.WrapError(err, "Unmounting persistent bind mounts dir")
	}

	agentEnv = agentEnv.DetachPersistentDisk(disk.ID())

	err = vm.agentEnvService.Update(agentEnv)
	if err != nil {
		return bosherr.WrapError(err, "Updating agent env")
	}

	return nil
}
