package action_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
	. "github.com/cppforlife/bosh-warden-cpi/action"
	bwcutil "github.com/cppforlife/bosh-warden-cpi/util"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
	fakevm "github.com/cppforlife/bosh-warden-cpi/vm/fakes"
)

var _ = Describe("DeleteVM", func() {
	var (
		vmFinder       *fakevm.FakeFinder
		action         DeleteVM
		fs             *fakesys.FakeFileSystem
		cmdRunner      *fakesys.FakeCmdRunner
		sleeper        bwcutil.Sleeper
		logger         boshlog.Logger
		hostBindMounts bwcvm.FSHostBindMounts
	)

	BeforeEach(func() {
		vmFinder = &fakevm.FakeFinder{}

		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		sleeper = bwcutil.RealSleeper{}
		logger = boshlog.NewLogger(boshlog.LevelNone)
		hostBindMounts = bwcvm.NewFSHostBindMounts(
			"/tmp/host-ephemeral-bind-mounts-dir",
			"/tmp/host-persistent-bind-mounts-dir",
			sleeper,
			fs,
			cmdRunner,
			logger,
		)

		action = NewDeleteVM(vmFinder, hostBindMounts)
	})

	Describe("Run", func() {
		var (
			vm *fakevm.FakeVM
		)

		BeforeEach(func() {
			vm = fakevm.NewFakeVM("fake-vm-id")
			vmFinder.FindVM = vm
		})

		It("tries to find vm with given vm cid", func() {
			_, err := action.Run("fake-vm-id")
			Expect(err).ToNot(HaveOccurred())

			Expect(vmFinder.FindID).To(Equal("fake-vm-id"))
		})

		Context("when vm is found with given vm cid", func() {
			BeforeEach(func() {
				vmFinder.FindFound = true
			})

			It("deletes vm", func() {
				_, err := action.Run("fake-vm-id")
				Expect(err).ToNot(HaveOccurred())

				Expect(vm.DeleteCalled).To(BeTrue())
			})

			It("returns error if deleting vm fails", func() {
				vm.DeleteErr = errors.New("fake-delete-err")

				_, err := action.Run("fake-vm-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-delete-err"))
			})
		})

		Context("when vm is not found with given cid", func() {
			BeforeEach(func() {
				vmFinder.FindFound = false
			})

			It("does not return error", func() {
				_, err := action.Run("fake-vm-id")
				Expect(err).ToNot(HaveOccurred())
			})

			It("still deletes the vm data", func() {
				_, err := action.Run("fake-vm-id")
				Expect(err).ToNot(HaveOccurred())
				Expect(vm.DeleteCalled).To(BeTrue())
			})
		})

		Context("when vm finding fails", func() {
			It("does not return error", func() {
				vmFinder.FindErr = errors.New("fake-find-err")

				_, err := action.Run("fake-vm-id")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-find-err"))
			})
		})
	})
})
