package vm_test

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	fakebwcvm "github.com/cppforlife/bosh-warden-cpi/vm/fakes"

	. "github.com/cppforlife/bosh-warden-cpi/vm"
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

		Context("when agent env service is registry", func() {
			BeforeEach(func() {
				metadataService = NewMetadataService("registry", registryOptions, logger)
			})

			It("saves registry endpoint as URL to registry", func() {
				err := metadataService.Save(fakeWardenFileService)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeWardenFileService.UploadDestinationPath).To(Equal("/var/vcap/bosh/warden-cpi-user-data.json"))

				userDataContents := UserDataContentsType{
					Registry: RegistryType{
						Endpoint: "http://fake-registry-username:fake-registry-password@fake-registry-host:1234",
					},
				}

				expectedContents, err := json.Marshal(userDataContents)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeWardenFileService.UploadContents).To(Equal(expectedContents))
			})
		})

		Context("when agent env service is file", func() {
			BeforeEach(func() {
				metadataService = NewMetadataService("file", registryOptions, logger)
			})

			It("saves registry endpoint as file path", func() {
				err := metadataService.Save(fakeWardenFileService)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeWardenFileService.UploadDestinationPath).To(Equal("/var/vcap/bosh/warden-cpi-user-data.json"))

				userDataContents := UserDataContentsType{
					Registry: RegistryType{
						Endpoint: "/var/vcap/bosh/warden-cpi-agent-env.json",
					},
				}

				expectedContents, err := json.Marshal(userDataContents)
				Expect(err).ToNot(HaveOccurred())
				Expect(fakeWardenFileService.UploadContents).To(Equal(expectedContents))
			})
		})

		Context("when uploading user data file fails", func() {
			BeforeEach(func() {
				fakeWardenFileService.UploadErr = errors.New("fake-upload-error")
			})

			It("returns an error", func() {
				err := metadataService.Save(fakeWardenFileService)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-upload-error"))
			})
		})
	})
})
