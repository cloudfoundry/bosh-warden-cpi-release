package action_test

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	"github.com/cppforlife/bosh-warden-cpi/action"
	fakesc "github.com/cppforlife/bosh-warden-cpi/stemcell/fakes"
	fakevm "github.com/cppforlife/bosh-warden-cpi/vm/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateVM", func() {
	var (
		createVmMethod     action.CreateVMMethod
		agentId            = apiv1.AgentID{}
		stemcellCid        = apiv1.StemcellCID{}
		cloudProps         = apiv1.CloudPropsImpl{[]byte(`{}`)}
		networks           = apiv1.Networks{}
		associatedDiskCids = []apiv1.DiskCID{}
		env                = apiv1.NewVMEnv(map[string]interface{}{"env1": "env1-val"})
		apiVersions        apiv1.ApiVersions
	)

	JustBeforeEach(func() {
		fakeStemcellFinder := &fakesc.FakeFinder{}
		fakeStemcellFinder.FindReturns(nil, true, nil)

		fakeVM := &fakevm.FakeVM{}
		fakeVM.IDReturns(apiv1.NewVMCID("test"))

		fakeCreator := &fakevm.FakeCreator{}
		fakeCreator.CreateReturns(fakeVM, nil)

		createVmMethod = action.NewCreateVMMethod(fakeStemcellFinder, fakeCreator, apiVersions)
	})

	Context("when the api contract is 1", func() {
		BeforeEach(func() {
			apiVersions = apiv1.ApiVersions{Contract: 1, Stemcell: 1}
		})

		It("responds with v1 contract", func() {
			result, err := createVmMethod.CreateVM(agentId, stemcellCid, cloudProps, networks, associatedDiskCids, env)
			Expect(err).To(BeNil())
			Expect(result).To(BeAssignableToTypeOf(apiv1.VMCID{}))
		})
	})

	Context("when the api contract is 2", func() {
		BeforeEach(func() {
			apiVersions = apiv1.ApiVersions{Contract: 2, Stemcell: 1}
		})

		It("responds with v2 contract", func() {
			result, err := createVmMethod.CreateVM(agentId, stemcellCid, cloudProps, networks, associatedDiskCids, env)
			Expect(err).To(BeNil())
			Expect(result).To(BeAssignableToTypeOf([]interface{}{}))
		})
	})
})
