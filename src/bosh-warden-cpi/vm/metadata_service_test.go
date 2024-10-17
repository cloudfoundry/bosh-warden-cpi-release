package vm_test

import (
	"encoding/json"
	"errors"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "bosh-warden-cpi/vm"
	fakebwcvm "bosh-warden-cpi/vm/fakes"
)

var _ = Describe("MetadataService", func() {
	var (
		metadataService       MetadataService
		fakeWardenFileService *fakebwcvm.FakeWardenFileService
		logger                boshlog.Logger
	)

	Describe("Save", func() {
		BeforeEach(func() {
			fakeWardenFileService = fakebwcvm.NewFakeWardenFileService()
			logger = boshlog.NewLogger(boshlog.LevelNone)
		})

		It("saves instance id to metadata", func() {
			metadataService = NewMetadataService(logger)

			err := metadataService.Save(fakeWardenFileService, apiv1.NewVMCID("fake-instance-id"))
			Expect(err).ToNot(HaveOccurred())

			metadataContents := MetadataContentsType{
				InstanceID: "fake-instance-id",
			}

			expectedContents, err := json.Marshal(metadataContents)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeWardenFileService.UploadInputs).To(ContainElement(fakebwcvm.UploadInput{
				DestinationPath: "/var/vcap/bosh/warden-cpi-metadata.json",
				Contents:        expectedContents,
			}))
		})

		Context("when agent env service is file", func() {
			BeforeEach(func() {
				metadataService = NewMetadataService(logger)
			})

			It("saves registry endpoint as file path", func() {
				err := metadataService.Save(fakeWardenFileService, apiv1.NewVMCID("fake-instance-id"))
				Expect(err).ToNot(HaveOccurred())

				userDataContents := UserDataContentsType{
					Registry: RegistryType{
						Endpoint: "/var/vcap/bosh/warden-cpi-agent-env.json",
					},
				}

				expectedContents, err := json.Marshal(userDataContents)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeWardenFileService.UploadInputs).To(ContainElement(fakebwcvm.UploadInput{
					DestinationPath: "/var/vcap/bosh/warden-cpi-user-data.json",
					Contents:        expectedContents,
				}))
			})
		})

		Context("when uploading user data file fails", func() {
			BeforeEach(func() {
				fakeWardenFileService.UploadErr = errors.New("fake-upload-error")
			})

			It("returns an error", func() {
				err := metadataService.Save(fakeWardenFileService, apiv1.NewVMCID("fake-instance-id"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-upload-error"))
			})
		})
	})
})
