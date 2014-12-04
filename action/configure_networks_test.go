package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/action"
)

var _ = Describe("ConfigureNetworks", func() {
	var (
		configureNetworks ConfigureNetworks
	)

	BeforeEach(func() {
		configureNetworks = NewConfigureNetworks()
	})

	Describe("Run", func() {
		It("returns an NotSupportedError", func() {
			networks := Networks{}
			vmcid := VMCID("fake-vm-cid")

			_, err := configureNetworks.Run(vmcid, networks)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Not supported"))
		})

	})
})
