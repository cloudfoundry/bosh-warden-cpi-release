package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/action"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

var _ = Describe("Networks", func() {
	var (
		networks Networks
	)

	BeforeEach(func() {
		networks = Networks{
			"fake-net1-name": Network{
				Type: "fake-net1-type",

				IP:      "fake-net1-ip",
				Netmask: "fake-net1-netmask",
				Gateway: "fake-net1-gateway",

				DNS:     []string{"fake-net1-dns"},
				Default: []string{"fake-net1-default"},

				CloudProperties: map[string]interface{}{
					"fake-net1-cp-key": "fake-net1-cp-value",
				},
			},
			"fake-net2-name": Network{
				Type: "fake-net2-type",
				IP:   "fake-net2-ip",
			},
		}
	})

	Describe("AsVMNetworks", func() {
		It("returns networks for VM", func() {
			expectedVMNetworks := bwcvm.Networks{
				"fake-net1-name": bwcvm.Network{
					Type: "fake-net1-type",

					IP:      "fake-net1-ip",
					Netmask: "fake-net1-netmask",
					Gateway: "fake-net1-gateway",

					DNS:     []string{"fake-net1-dns"},
					Default: []string{"fake-net1-default"},

					CloudProperties: map[string]interface{}{
						"fake-net1-cp-key": "fake-net1-cp-value",
					},
				},
				"fake-net2-name": bwcvm.Network{
					Type: "fake-net2-type",
					IP:   "fake-net2-ip",
				},
			}

			Expect(networks.AsVMNetworks()).To(Equal(expectedVMNetworks))
		})
	})
})
