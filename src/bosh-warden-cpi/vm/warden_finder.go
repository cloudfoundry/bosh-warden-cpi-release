package vm

import (
	wrdnclient "code.cloudfoundry.org/garden/client"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type WardenFinder struct {
	wardenClient           wrdnclient.Client
	agentEnvServiceFactory AgentEnvServiceFactory

	ports           Ports
	hostBindMounts  HostBindMounts
	guestBindMounts GuestBindMounts

	logTag string
	logger boshlog.Logger
}

func NewWardenFinder(
	wardenClient wrdnclient.Client,
	agentEnvServiceFactory AgentEnvServiceFactory,
	ports Ports,
	hostBindMounts HostBindMounts,
	guestBindMounts GuestBindMounts,
	logger boshlog.Logger,
) WardenFinder {
	return WardenFinder{
		wardenClient:           wardenClient,
		agentEnvServiceFactory: agentEnvServiceFactory,

		ports:           ports,
		hostBindMounts:  hostBindMounts,
		guestBindMounts: guestBindMounts,

		logTag: "vm.WardenFinder",
		logger: logger,
	}
}

func (f WardenFinder) Find(id apiv1.VMCID) (VM, bool, error) {
	f.logger.Debug(f.logTag, "Finding container with ID '%s'", id)

	// Cannot just use Lookup(id) since we need to differentiate between error and not found
	containers, err := f.wardenClient.Containers(nil)
	if err != nil {
		return nil, false, bosherr.WrapError(err, "Listing all containers")
	}

	for _, container := range containers {
		if container.Handle() == id.AsString() {
			f.logger.Debug(f.logTag, "Found container with ID '%s'", id)

			wardenFileService := NewWardenFileService(container, f.logger)
			agentEnvService := f.agentEnvServiceFactory.New(wardenFileService, id)

			vm := NewWardenVM(id, f.wardenClient, agentEnvService, f.ports, f.hostBindMounts, f.guestBindMounts, f.logger, true)

			return vm, true, nil
		}
	}

	f.logger.Debug(f.logTag, "Did not find container with ID '%s'", id)

	vm := NewWardenVM(id, f.wardenClient, nil, f.ports, f.hostBindMounts, f.guestBindMounts, f.logger, false)

	return vm, false, nil
}
