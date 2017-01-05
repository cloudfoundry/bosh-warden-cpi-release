package vm

import (
	wrdnclient "github.com/cloudfoundry-incubator/garden/client"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

const wardenFinderLogTag = "WardenFinder"

type WardenFinder struct {
	wardenClient           wrdnclient.Client
	agentEnvServiceFactory AgentEnvServiceFactory

	ports           Ports
	hostBindMounts  HostBindMounts
	guestBindMounts GuestBindMounts

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

		logger: logger,
	}
}

func (f WardenFinder) Find(id string) (VM, bool, error) {
	f.logger.Debug(wardenFinderLogTag, "Finding container with ID '%s'", id)

	// Cannot just use Lookup(id) since we need to differentiate between error and not found
	containers, err := f.wardenClient.Containers(nil)
	if err != nil {
		return nil, false, bosherr.WrapError(err, "Listing all containers")
	}

	for _, container := range containers {
		if container.Handle() == id {
			f.logger.Debug(wardenFinderLogTag, "Found container with ID '%s'", id)

			wardenFileService := NewWardenFileService(container, f.logger)
			agentEnvService := f.agentEnvServiceFactory.New(wardenFileService, id)

			vm := NewWardenVM(id, f.wardenClient, agentEnvService, f.ports, f.hostBindMounts, f.guestBindMounts, f.logger, true)

			return vm, true, nil
		}
	}

	f.logger.Debug(wardenFinderLogTag, "Did not find container with ID '%s'", id)

	vm := NewWardenVM(id, f.wardenClient, nil, f.ports, f.hostBindMounts, f.guestBindMounts, f.logger, false)

	return vm, false, nil
}
