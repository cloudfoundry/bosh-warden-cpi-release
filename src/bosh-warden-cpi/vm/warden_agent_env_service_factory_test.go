package vm_test

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "bosh-warden-cpi/vm"
	fakebwcvm "bosh-warden-cpi/vm/fakes"
)

var _ = Describe("WardenAgentEnvServiceFactory", func() {
	Describe("New", func() {
		var (
			logger                boshlog.Logger
			fakeWardenFileService *fakebwcvm.FakeWardenFileService
		)

		BeforeEach(func() {
			fakeWardenFileService = fakebwcvm.NewFakeWardenFileService()

			logger = boshlog.NewLogger(boshlog.LevelNone)
		})

		It("returns a NewFSAgentEnvService", func() {
			expectedAgentEnvService := NewFSAgentEnvService(fakeWardenFileService, logger)
			wardenAgentEnvServiceFactory := NewWardenAgentEnvServiceFactory(logger)
			agentEnvService := wardenAgentEnvServiceFactory.New(fakeWardenFileService, apiv1.NewVMCID("fake-instance-id"))
			Expect(agentEnvService).To(Equal(expectedAgentEnvService))
		})
	})
})
