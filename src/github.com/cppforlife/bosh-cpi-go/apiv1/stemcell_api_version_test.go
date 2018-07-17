package apiv1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-cpi-go/apiv1"
)

var _ = Describe("StemcellAPIVersion", func() {
	It("retrieves stemcell version", func() {
		version := NewStemcellAPIVersion(CloudPropsImpl{[]byte(`{"vm":{"stemcell":{"api_version":1}}}`)})

		val, err := version.Value()
		Expect(err).ToNot(HaveOccurred())
		Expect(val).To(Equal(1))
	})

	It("defaults to 0", func() {
		version := NewStemcellAPIVersion(CloudPropsImpl{[]byte(`{}`)})

		val, err := version.Value()
		Expect(err).ToNot(HaveOccurred())
		Expect(val).To(Equal(0))
	})

	It("returns error if cannot parse", func() {
		version := NewStemcellAPIVersion(CloudPropsImpl{[]byte(`{"vm":{"stemcell":{"api_version":"val"}}}`)})

		val, err := version.Value()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Expected to unmarshal stemcell API version"))
		Expect(val).To(Equal(0))
	})
})
