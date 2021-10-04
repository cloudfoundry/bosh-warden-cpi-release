package vm_test

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "bosh-warden-cpi/vm"
	fakebwcvm "bosh-warden-cpi/vm/fakes"
)

var _ = Describe("WardenAgentEnvServiceFactory", func() {
	Describe("New", func() {
		var (
			logger                boshlog.Logger
			fakeWardenFileService *fakebwcvm.FakeWardenFileService
			registryOptions       RegistryOptions
		)

		BeforeEach(func() {
			fakeWardenFileService = fakebwcvm.NewFakeWardenFileService()

			logger = boshlog.NewLogger(boshlog.LevelNone)

			registryOptions = RegistryOptions{
				Host: "fake-host",
			}
		})

		Context("when agentEnvService is registry", func() {
			It("returns a NewRegistryAgentEnvService", func() {
				expectedAgentEnvService := NewRegistryAgentEnvService(registryOptions, apiv1.NewVMCID("fake-instance-id"), logger)
				wardenAgentEnvServiceFactory := NewWardenAgentEnvServiceFactory("registry", registryOptions, logger)
				agentEnvService := wardenAgentEnvServiceFactory.New(fakeWardenFileService, apiv1.NewVMCID("fake-instance-id"))
				Expect(agentEnvService).To(Equal(expectedAgentEnvService))
			})
		})

		Context("when agentEnvService is not registry", func() {
			It("returns a NewFSAgentEnvService", func() {
				expectedAgentEnvService := NewFSAgentEnvService(fakeWardenFileService, logger)
				wardenAgentEnvServiceFactory := NewWardenAgentEnvServiceFactory("file", registryOptions, logger)
				agentEnvService := wardenAgentEnvServiceFactory.New(fakeWardenFileService, apiv1.NewVMCID("fake-instance-id"))
				Expect(agentEnvService).To(Equal(expectedAgentEnvService))
			})
		})
	})
})
