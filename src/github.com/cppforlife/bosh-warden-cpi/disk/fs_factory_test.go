package disk_test

import (
	"errors"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	"github.com/cppforlife/bosh-cpi-go/apiv1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/bosh-warden-cpi/disk"
)

var _ = Describe("FSFactory", func() {
	var (
		fs        *fakesys.FakeFileSystem
		uuidGen   *fakeuuid.FakeGenerator
		cmdRunner *fakesys.FakeCmdRunner
		logger    boshlog.Logger
		factory   FSFactory
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		uuidGen = &fakeuuid.FakeGenerator{}
		cmdRunner = fakesys.NewFakeCmdRunner()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		factory = NewFSFactory("/fake-disks-dir", fs, uuidGen, cmdRunner, logger)
	})

	Describe("Create", func() {
		It("returns unique disk id", func() {
			uuidGen.GeneratedUUID = "fake-uuid"

			disk, err := factory.Create(40)
			Expect(err).ToNot(HaveOccurred())

			expectedDisk := NewFSDisk(apiv1.NewDiskCID("fake-uuid"), "/fake-disks-dir/fake-uuid", fs, logger)
			Expect(disk).To(Equal(expectedDisk))
		})

		Context("when generating unique id succeeds", func() {
			BeforeEach(func() {
				uuidGen.GeneratedUUID = "fake-uuid"
			})

			It("touches disk path in disks directory", func() {
				uuidGen.GeneratedUUID = "fake-uuid"

				_, err := factory.Create(40)
				Expect(err).ToNot(HaveOccurred())

				bytes, err := fs.ReadFile("/fake-disks-dir/fake-uuid")
				Expect(err).ToNot(HaveOccurred())
				Expect(bytes).To(BeEmpty())
			})

			Context("when touching disk path succeeds", func() {
				It("increases size of the file to given size in MB", func() {
					_, err := factory.Create(40)
					Expect(err).ToNot(HaveOccurred())

					Expect(len(cmdRunner.RunCommands)).To(BeNumerically(">", 0))
					Expect(cmdRunner.RunCommands[0]).To(Equal(
						[]string{"truncate", "-s", "40MB", "/fake-disks-dir/fake-uuid"},
					))
				})

				ItDestroysFile := func(errMsg string) {
					It("deletes file since it was not turned into a filesystem", func() {
						disk, err := factory.Create(40)
						Expect(err).To(HaveOccurred())
						Expect(disk).To(BeNil())

						Expect(fs.FileExists("/fake-disks-dir/fake-uuid")).To(BeFalse())
					})

					Context("when deleting file fails", func() {
						BeforeEach(func() {
							fs.RemoveAllStub = func(string) error {
								return errors.New("fake-remove-all-err")
							}
						})

						It("returns running error and not destroy error", func() {
							disk, err := factory.Create(40)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring(errMsg))
							Expect(disk).To(BeNil())
						})
					})
				}

				Context("when increasing file size succeeds", func() {
					It("turns file into a filesystem", func() {
						_, err := factory.Create(40)
						Expect(err).ToNot(HaveOccurred())

						Expect(cmdRunner.RunCommands).To(HaveLen(2))
						Expect(cmdRunner.RunCommands[1]).To(Equal(
							[]string{"/sbin/mkfs", "-t", "ext4", "-F", "/fake-disks-dir/fake-uuid"},
						))
					})

					Context("when turning file into a filesystem fails", func() {
						BeforeEach(func() {
							cmdRunner.AddCmdResult(
								"/sbin/mkfs -t ext4 -F /fake-disks-dir/fake-uuid",
								fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
							)
						})

						It("returns an error", func() {
							disk, err := factory.Create(40)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(ContainSubstring("fake-run-err"))
							Expect(disk).To(BeNil())
						})

						ItDestroysFile("fake-run-err")
					})
				})

				Context("when increasing file size fails", func() {
					BeforeEach(func() {
						cmdRunner.AddCmdResult(
							"truncate -s 40MB /fake-disks-dir/fake-uuid",
							fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
						)
					})

					It("returns an error", func() {
						disk, err := factory.Create(40)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("fake-run-err"))
						Expect(disk).To(BeNil())
					})

					ItDestroysFile("fake-run-err")
				})
			})

			Context("when touching disk path fails", func() {
				It("returns error if touching disk path fails", func() {
					fs.WriteFileError = errors.New("fake-write-file-err")

					disk, err := factory.Create(40)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-write-file-err"))
					Expect(disk).To(BeNil())
				})
			})
		})

		Context("when generating unique id fails", func() {
			It("returns error if generating disk id fails", func() {
				uuidGen.GenerateError = errors.New("fake-generate-err")

				disk, err := factory.Create(40)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-generate-err"))
				Expect(disk).To(BeNil())
			})
		})
	})

	Describe("Find", func() {
		It("returns disk", func() {
			err := fs.WriteFile("/fake-disks-dir/fake-disk-id", []byte{})
			Expect(err).ToNot(HaveOccurred())

			disk, err := factory.Find(apiv1.NewDiskCID("fake-disk-id"))
			Expect(err).ToNot(HaveOccurred())
			Expect(disk).To(Equal(NewFSDisk(apiv1.NewDiskCID("fake-disk-id"), "/fake-disks-dir/fake-disk-id", fs, logger)))
		})
	})
})
