package vm_test

import (
	"errors"

	wrdnclient "github.com/cloudfoundry-incubator/garden/client"
	fakewrdnconn "github.com/cloudfoundry-incubator/garden/client/connection/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/vm"
	fakevm "github.com/cppforlife/bosh-warden-cpi/vm/fakes"
)

var _ = Describe("WardenFinder", func() {
	var (
		wardenConn   *fakewrdnconn.FakeConnection
		wardenClient wrdnclient.Client

		agentEnvServiceFactory *fakevm.FakeAgentEnvServiceFactory
		hostBindMounts         *fakevm.FakeHostBindMounts
		guestBindMounts        *fakevm.FakeGuestBindMounts
		logger                 boshlog.Logger
		finder                 WardenFinder
	)

	BeforeEach(func() {
		wardenConn = &fakewrdnconn.FakeConnection{}
		wardenClient = wrdnclient.New(wardenConn)

		agentEnvServiceFactory = &fakevm.FakeAgentEnvServiceFactory{}
		hostBindMounts = &fakevm.FakeHostBindMounts{}
		guestBindMounts = &fakevm.FakeGuestBindMounts{}
		logger = boshlog.NewLogger(boshlog.LevelNone)

		finder = NewWardenFinder(
			wardenClient,
			agentEnvServiceFactory,
			hostBindMounts,
			guestBindMounts,
			logger,
		)
	})

	Describe("Find", func() {
		It("returns VM and found as true if warden has container with VM ID as its handle", func() {
			agentEnvService := &fakevm.FakeAgentEnvService{}
			agentEnvServiceFactory.NewAgentEnvService = agentEnvService

			wardenConn.ListReturns([]string{"non-matching-vm-id", "fake-vm-id"}, nil)

			expectedVM := NewWardenVM(
				"fake-vm-id",
				wardenClient,
				agentEnvService,
				hostBindMounts,
				guestBindMounts,
				logger,
				true,
			)

			vm, found, err := finder.Find("fake-vm-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(vm).To(Equal(expectedVM))

			Expect(agentEnvServiceFactory.NewInstanceID).To(Equal("fake-vm-id"))

			Expect(wardenConn.ListCallCount()).To(Equal(1))
			Expect(wardenConn.ListArgsForCall(0)).To(BeNil())
		})

		It("returns found as false if warden does not have container with VM ID as its handle", func() {
			wardenConn.ListReturns([]string{"non-matching-vm-id"}, nil)

			expectedVM := NewWardenVM(
				"fake-vm-id",
				wardenClient,
				nil,
				hostBindMounts,
				guestBindMounts,
				logger,
				false,
			)

			vm, found, err := finder.Find("fake-vm-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
			Expect(vm).To(Equal(expectedVM))
		})

		It("returns error if warden container listing fails", func() {
			wardenConn.ListReturns(nil, errors.New("fake-list-err"))

			vm, found, err := finder.Find("fake-vm-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-list-err"))
			Expect(found).To(BeFalse())
			Expect(vm).To(BeNil())
		})
	})
})
