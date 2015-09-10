package vm_test

import (
	"github.com/cppforlife/bosh-warden-cpi/vm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network", func() {
	var (
		networks                     vm.Networks
		networkA, networkB, networkC vm.Network
	)
	BeforeEach(func() {
		networks = make(map[string]vm.Network)
		networkA = vm.Network{Type: "A"}
		networkB = vm.Network{Type: "B"}
		networkC = vm.Network{Type: "C"}
	})

	Describe("Default", func() {
		Context("when there are no networks defined", func() {
			It("Returns and empty network", func() {
				Expect(networks.Default()).To(Equal(vm.Network{}))
			})
		})

		Context("when is only one network defined", func() {
			It("Returns the deinfed network", func() {
				networks["A"] = networkA
				Expect(networks.Default()).To(Equal(networkA))
			})
		})

		Context("when are multiple networks defined", func() {
			var allNetworks []vm.Network
			It("Returns the one that has the default gateway", func() {
				networkB.Default = []string{"gateway"}
				networks["A"] = networkA
				networks["B"] = networkB
				networks["C"] = networkC
				Expect(networks.Default()).To(Equal(networkB))
			})

			It("Returns the one that has the default gateway even with others have other defaults", func() {
				networkA.Default = []string{"dns"}
				networkB.Default = []string{"other"}
				networkC.Default = []string{"gateway"}
				networks["A"] = networkA
				networks["B"] = networkB
				networks["C"] = networkC
				Expect(networks.Default()).To(Equal(networkC))
			})

			It("returns one of the networks when none have the default gateway set", func() {
				networks["A"] = networkA
				networks["B"] = networkB
				networks["C"] = networkC
				for _, net := range networks {
					allNetworks = append(allNetworks, net)
				}
				Expect(allNetworks).To(ContainElement(networks.Default()))
			})

			It("returns one of the networks when none have the default gateway set but have other defaults", func() {
				networkA.Default = []string{"dns", "foo"}
				networkB.Default = []string{"bar"}
				networks["A"] = networkA
				networks["B"] = networkB
				networks["C"] = networkC
				for _, net := range networks {
					allNetworks = append(allNetworks, net)
				}
				Expect(allNetworks).To(ContainElement(networks.Default()))
			})
		})
	})

	Describe("IsDynamic", func() {
		It("returns true if the type is 'dynamic'", func() {
			Expect(vm.Network{Type: "A"}.IsDynamic()).To(BeFalse())
			Expect(vm.Network{Type: "manual"}.IsDynamic()).To(BeFalse())
			Expect(vm.Network{Type: "Dynamic"}.IsDynamic()).To(BeFalse())
			Expect(vm.Network{Type: "dynamic"}.IsDynamic()).To(BeTrue())
		})
	})

	Describe("IPWithSubnetMask", func() {
		It("returns 12.18.3.4/24 when IP is 12.18.3.4 and netmask is 255.255.255.0", func() {
			net := vm.Network{IP: "12.18.3.4", Netmask: "255.255.255.0"}
			Expect(net.IPWithSubnetMask()).To(Equal("12.18.3.4/24"))
		})
	})
})
