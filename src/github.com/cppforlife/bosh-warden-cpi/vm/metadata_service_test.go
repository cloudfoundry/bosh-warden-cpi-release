package vm_test

import (
	"encoding/json"
	"errors"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/vm"
	fakebwcvm "github.com/cppforlife/bosh-warden-cpi/vm/fakes"
)

var _ = Describe("MetadataService", func() {
	var (
		metadataService       MetadataService
		fakeWardenFileService *fakebwcvm.FakeWardenFileService
		registryOptions       RegistryOptions
		logger                boshlog.Logger
	)

	Describe("Save", func() {
		BeforeEach(func() {
			fakeWardenFileService = fakebwcvm.NewFakeWardenFileService()
			registryOptions = RegistryOptions{
				Host:     "fake-registry-host",
				Port:     1234,
				Username: "fake-registry-username",
				Password: "fake-registry-password",
			}
			logger = boshlog.NewLogger(boshlog.LevelNone)
		})

		It("saves instance id to metadata", func() {
			metadataService = NewMetadataService("registry", registryOptions, logger)

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

		Context("when agent env service is registry", func() {
			BeforeEach(func() {
				metadataService = NewMetadataService("registry", registryOptions, logger)
			})

			It("saves registry endpoint as URL to registry", func() {
				err := metadataService.Save(fakeWardenFileService, apiv1.NewVMCID("fake-instance-id"))
				Expect(err).ToNot(HaveOccurred())

				userDataContents := UserDataContentsType{
					Registry: RegistryType{
						Endpoint: "http://fake-registry-username:fake-registry-password@fake-registry-host:1234",
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

		Context("when agent env service is file", func() {
			BeforeEach(func() {
				metadataService = NewMetadataService("file", registryOptions, logger)
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
