package vm

import (
	wrdnclient "code.cloudfoundry.org/garden/client"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	bwcdisk "bosh-warden-cpi/disk"
)

type WardenVM struct {
	id apiv1.VMCID

	wardenClient    wrdnclient.Client
	agentEnvService AgentEnvService

	ports           Ports
	hostBindMounts  HostBindMounts
	guestBindMounts GuestBindMounts

	logger boshlog.Logger

	containerExists bool
}

func NewWardenVM(
	id apiv1.VMCID,
	wardenClient wrdnclient.Client,
	agentEnvService AgentEnvService,
	ports Ports,
	hostBindMounts HostBindMounts,
	guestBindMounts GuestBindMounts,
	logger boshlog.Logger,
	containerExists bool,
) WardenVM {
	return WardenVM{
		id: id,

		wardenClient:    wardenClient,
		agentEnvService: agentEnvService,

		ports:           ports,
		hostBindMounts:  hostBindMounts,
		guestBindMounts: guestBindMounts,

		logger:          logger,
		containerExists: containerExists,
	}
}

func (vm WardenVM) ID() apiv1.VMCID { return vm.id }

func (vm WardenVM) Delete() error {
	if vm.containerExists {
		err := vm.wardenClient.Destroy(vm.id.AsString())
		if err != nil {
			return bosherr.WrapError(err, "Deleting container")
		}
	}

	err := vm.ports.RemoveForwarded(vm.id)
	if err != nil {
		return bosherr.WrapError(err, "Removing forwarded ports")
	}

	err = vm.hostBindMounts.DeleteEphemeral(vm.id)
	if err != nil {
		return bosherr.WrapError(err, "Deleting ephemeral bind mounts")
	}

	err = vm.hostBindMounts.DeletePersistent(vm.id)
	if err != nil {
		return bosherr.WrapError(err, "Deleting persistent bind mounts")
	}

	return nil
}

func (vm WardenVM) AttachDisk(disk bwcdisk.Disk) (apiv1.DiskHint, error) {
	if !vm.containerExists {
		return apiv1.DiskHint{}, bosherr.Error("VM does not exist")
	}

	agentEnv, err := vm.agentEnvService.Fetch()
	if err != nil {
		return apiv1.DiskHint{}, bosherr.WrapError(err, "Fetching agent env")
	}

	err = vm.hostBindMounts.MountPersistent(vm.id, disk.ID(), disk.Path())
	if err != nil {
		return apiv1.DiskHint{}, bosherr.WrapError(err, "Mounting persistent bind mounts dir")
	}

	diskHint := apiv1.NewDiskHintFromString(vm.guestBindMounts.MountPersistent(disk.ID()))

	agentEnv.AttachPersistentDisk(disk.ID(), diskHint)

	err = vm.agentEnvService.Update(agentEnv)
	if err != nil {
		return apiv1.DiskHint{}, bosherr.WrapError(err, "Updating agent env")
	}

	return diskHint, nil
}

func (vm WardenVM) DetachDisk(disk bwcdisk.Disk) error {
	if !vm.containerExists {
		return bosherr.Error("VM does not exist")
	}

	agentEnv, err := vm.agentEnvService.Fetch()
	if err != nil {
		return bosherr.WrapError(err, "Fetching agent env")
	}

	err = vm.hostBindMounts.UnmountPersistent(vm.id, disk.ID())
	if err != nil {
		return bosherr.WrapError(err, "Unmounting persistent bind mounts dir")
	}

	agentEnv.DetachPersistentDisk(disk.ID())

	err = vm.agentEnvService.Update(agentEnv)
	if err != nil {
		return bosherr.WrapError(err, "Updating agent env")
	}

	return nil
}
