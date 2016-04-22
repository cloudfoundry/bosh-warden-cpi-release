package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/action"
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

var _ = Describe("Cloud Properties", func() {
	var (
		cloudProperties VMCloudProperties
	)

	BeforeEach(func() {
		cloudProperties = VMCloudProperties{
			LaunchUpstart: true,
		}
	})

	Describe("AsVMCloudProperties", func() {
		It("returns cloud properties for VM", func() {
			expectedVMCloudProperties := bwcvm.CloudProperties{
				LaunchUpstart: true,
			}

			Expect(cloudProperties.AsVMCloudProperties()).To(Equal(expectedVMCloudProperties))
		})
	})
})
