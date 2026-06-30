package vm_test

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherrfakes "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bwcutil "bosh-warden-cpi/util"
	. "bosh-warden-cpi/vm"
)

var _ = Describe("IPTablesPorts", func() {
	var (
		cmdRunner *bosherrfakes.FakeCmdRunner
		sleeper   *bwcutil.RecordingNoopSleeper
		ports     IPTablesPorts
		vmID      apiv1.VMCID
	)

	BeforeEach(func() {
		cmdRunner = bosherrfakes.NewFakeCmdRunner()
		sleeper = bwcutil.NewRecordingNoopSleeper()
		ports = NewIPTablesPorts(sleeper, cmdRunner)
		vmID = apiv1.NewVMCID("test-vm-id")
	})

	Describe("RemoveForwarded", func() {
		Context("when iptables-save fails (e.g. running without CAP_NET_ADMIN)", func() {
			BeforeEach(func() {
				cmdRunner.AddCmdResult(
					"iptables-save -t nat",
					bosherrfakes.FakeCmdResult{
						Stderr: "iptables-save: Permission denied (you must be root)",
						Error:  errors.New("iptables-save failed"),
					},
				)
			})

			It("returns nil without error", func() {
				err := ports.RemoveForwarded(vmID)
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not attempt to delete any iptables rules", func() {
				_ = ports.RemoveForwarded(vmID)
				Expect(cmdRunner.RunCommands).To(HaveLen(1))
				Expect(cmdRunner.RunCommands[0]).To(ConsistOf("iptables-save", "-t", "nat"))
			})
		})

		Context("when iptables-save succeeds but has no rules for this VM", func() {
			BeforeEach(func() {
				cmdRunner.AddCmdResult(
					"iptables-save -t nat",
					bosherrfakes.FakeCmdResult{
						Stdout: "-A PREROUTING -p tcp -j DNAT --to 10.0.0.1:8080 -m comment --comment bosh-warden-cpi-other-vm\n",
					},
				)
			})

			It("returns nil without error", func() {
				err := ports.RemoveForwarded(vmID)
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not attempt to delete any iptables rules", func() {
				_ = ports.RemoveForwarded(vmID)
				Expect(cmdRunner.RunCommands).To(HaveLen(1))
			})
		})

		Context("when iptables-save succeeds and has rules for this VM", func() {
			BeforeEach(func() {
				natRule := fmt.Sprintf(
					"-A PREROUTING -p tcp -j DNAT --to 10.0.0.1:8080 -m comment --comment bosh-warden-cpi-%s",
					vmID.AsString(),
				)
				cmdRunner.AddCmdResult(
					"iptables-save -t nat",
					bosherrfakes.FakeCmdResult{Stdout: natRule + "\n"},
				)
				cmdRunner.AddCmdResult(
					fmt.Sprintf("iptables -w -t nat -D PREROUTING -p tcp -j DNAT --to 10.0.0.1:8080 -m comment --comment bosh-warden-cpi-%s", vmID.AsString()),
					bosherrfakes.FakeCmdResult{},
				)
			})

			It("removes the matching iptables rule", func() {
				err := ports.RemoveForwarded(vmID)
				Expect(err).NotTo(HaveOccurred())
				Expect(cmdRunner.RunCommands).To(HaveLen(2))
			})
		})
	})
})
