package action_test

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/action"
)

var _ = Describe("FactoryOpts", func() {
	var (
		opts FactoryOpts

		validOptions = FactoryOpts{
			StemcellsDir: "/tmp/stemcells",
			DisksDir:     "/tmp/disks",

			HostEphemeralBindMountsDir:  "/tmp/host-ephemeral-bind-mounts-dir",
			HostPersistentBindMountsDir: "/tmp/host-persistent-bind-mounts-dir",

			GuestEphemeralBindMountPath:  "/tmp/guest-ephemeral-bind-mount-path",
			GuestPersistentBindMountsDir: "/tmp/guest-persistent-bind-mounts-dir",

			Agent: apiv1.AgentOptions{
				Mbus: "fake-mbus",
				NTP:  []string{},

				Blobstore: apiv1.BlobstoreOptions{
					Type: "fake-blobstore-type",
				},
			},
		}
	)

	Describe("Validate", func() {
		BeforeEach(func() {
			opts = validOptions
		})

		It("does not return error if all fields are valid", func() {
			err := opts.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error if StemcellsDir is empty", func() {
			opts.StemcellsDir = ""

			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Must provide non-empty StemcellsDir"))
		})

		It("returns error if DisksDir is empty", func() {
			opts.DisksDir = ""

			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Must provide non-empty DisksDir"))
		})

		It("returns error if HostEphemeralBindMountsDir is empty", func() {
			opts.HostEphemeralBindMountsDir = ""

			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Must provide non-empty HostEphemeralBindMountsDir"))
		})

		It("returns error if HostPersistentBindMountsDir is empty", func() {
			opts.HostPersistentBindMountsDir = ""

			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Must provide non-empty HostPersistentBindMountsDir"))
		})

		It("returns error if GuestEphemeralBindMountPath is empty", func() {
			opts.GuestEphemeralBindMountPath = ""

			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Must provide non-empty GuestEphemeralBindMountPath"))
		})

		It("returns error if GuestPersistentBindMountsDir is empty", func() {
			opts.GuestPersistentBindMountsDir = ""

			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"Must provide non-empty GuestPersistentBindMountsDir"))
		})

		It("returns error if agent section is not valid", func() {
			opts.Agent.Mbus = ""

			err := opts.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Validating Agent configuration"))
		})
	})
})
