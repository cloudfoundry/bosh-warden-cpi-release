package apiv1_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-cpi-go/apiv1"
)

var _ = Describe("DiskHintImpl", func() {
	It("marshals given data from string", func() {
		cps := NewDiskHintFromString("val1")

		bytes, err := json.Marshal(cps)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(bytes)).To(Equal(`"val1"`))
	})

	It("marshals given data from map", func() {
		cps := NewDiskHintFromMap(map[string]interface{}{"cp1": "cp1-val"})

		bytes, err := json.Marshal(cps)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(bytes)).To(Equal(`{"cp1":"cp1-val"}`))

		wrongCPs := NewDiskHintFromMap(map[string]interface{}{"func": func() {}})

		_, err = json.Marshal(wrongCPs)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("json: error calling MarshalJSON for type apiv1.DiskHint: json: unsupported type: func()"))
	})

	It("can marshal empty disk hint", func() {
		cps := DiskHint{}

		bytes, err := json.Marshal(cps)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(bytes)).To(Equal(`null`))
	})
})
