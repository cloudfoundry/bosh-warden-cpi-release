package vm_test

import (
	"errors"

	wrdnclient "code.cloudfoundry.org/garden/client"
	fakewrdnconn "code.cloudfoundry.org/garden/client/connection/connectionfakes"
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "bosh-warden-cpi/vm"
	fakevm "bosh-warden-cpi/vm/fakes"
)

var _ = Describe("WardenFinder", func() {
	var (
		wardenConn   *fakewrdnconn.FakeConnection
		wardenClient wrdnclient.Client

		agentEnvServiceFactory *fakevm.FakeAgentEnvServiceFactory
		ports                  *fakevm.FakePorts
		hostBindMounts         *fakevm.FakeHostBindMounts
		guestBindMounts        *fakevm.FakeGuestBindMounts
		logger                 boshlog.Logger
		finder                 WardenFinder
	)

	BeforeEach(func() {
		wardenConn = &fakewrdnconn.FakeConnection{}
		wardenClient = wrdnclient.New(wardenConn)

		agentEnvServiceFactory = &fakevm.FakeAgentEnvServiceFactory{}
		ports = &fakevm.FakePorts{}
		hostBindMounts = &fakevm.FakeHostBindMounts{}
		guestBindMounts = &fakevm.FakeGuestBindMounts{}
		logger = boshlog.NewLogger(boshlog.LevelNone)
		finder = NewWardenFinder(wardenClient, agentEnvServiceFactory, ports, hostBindMounts, guestBindMounts, logger)
	})

	Describe("Find", func() {
		It("returns VM and found as true if warden has container with VM ID as its handle", func() {
			agentEnvService := &fakevm.FakeAgentEnvService{}
			agentEnvServiceFactory.NewAgentEnvService = agentEnvService

			wardenConn.ListReturns([]string{"non-matching-vm-id", "fake-vm-id"}, nil)

			expectedVM := NewWardenVM(
				apiv1.NewVMCID("fake-vm-id"), wardenClient, agentEnvService,
				ports, hostBindMounts, guestBindMounts, logger, true)

			vm, found, err := finder.Find(apiv1.NewVMCID("fake-vm-id"))
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(vm).To(Equal(expectedVM))

			Expect(agentEnvServiceFactory.NewInstanceID).To(Equal(apiv1.NewVMCID("fake-vm-id")))

			Expect(wardenConn.ListCallCount()).To(Equal(1))
			Expect(wardenConn.ListArgsForCall(0)).To(BeNil())
		})

		It("returns found as false if warden does not have container with VM ID as its handle", func() {
			wardenConn.ListReturns([]string{"non-matching-vm-id"}, nil)

			expectedVM := NewWardenVM(
				apiv1.NewVMCID("fake-vm-id"), wardenClient, nil,
				ports, hostBindMounts, guestBindMounts, logger, false)

			vm, found, err := finder.Find(apiv1.NewVMCID("fake-vm-id"))
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
			Expect(vm).To(Equal(expectedVM))
		})

		It("returns error if warden container listing fails", func() {
			wardenConn.ListReturns(nil, errors.New("fake-list-err"))

			vm, found, err := finder.Find(apiv1.NewVMCID("fake-vm-id"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-list-err"))
			Expect(found).To(BeFalse())
			Expect(vm).To(BeNil())
		})
	})
})
