package vm_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bwcutil "bosh-warden-cpi/util"
	. "bosh-warden-cpi/vm"
)

var _ = Describe("FSHostBindMounts", func() {
	var (
		sleeper        *bwcutil.RecordingNoopSleeper
		fs             *fakesys.FakeFileSystem
		cmdRunner      *fakesys.FakeCmdRunner
		hostBindMounts FSHostBindMounts
	)

	BeforeEach(func() {
		sleeper = bwcutil.NewRecordingNoopSleeper()
		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		logger := boshlog.NewLogger(boshlog.LevelNone)

		hostBindMounts = NewFSHostBindMounts(
			"/fake-ephemeral-dir",
			"/fake-persistent-dir",
			sleeper,
			fs,
			cmdRunner,
			logger,
		)
	})

	Describe("MakeEphemeral", func() {
		It("creates directory for requested id", func() {
			path, err := hostBindMounts.MakeEphemeral(apiv1.NewVMCID("fake-id"))
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("/fake-ephemeral-dir/fake-id"))

			pathStat := fs.GetFileTestStat("/fake-ephemeral-dir/fake-id")
			Expect(pathStat.FileType).To(Equal(fakesys.FakeFileTypeDir))
			Expect(int(pathStat.FileMode)).To(Equal(0755)) // todo
		})

		It("returns error if creating directory fails", func() {
			fs.MkdirAllError = errors.New("fake-mkdir-all-err")

			path, err := hostBindMounts.MakeEphemeral(apiv1.NewVMCID("fake-id"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
			Expect(path).To(Equal(""))
		})

		Context("when creating directory succeeds", func() {
			It("makes the bind mount point private", func() {
				_, err := hostBindMounts.MakeEphemeral(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())

				Expect(cmdRunner.RunCommands).To(Equal([][]string{
					[]string{
						"mount", "--bind",
						"/fake-ephemeral-dir/fake-id",
						"/fake-ephemeral-dir/fake-id",
					},
					[]string{
						"mount", "--make-private",
						"/fake-ephemeral-dir/fake-id",
					},
				}))
			})

			Context("when making bind point private fails", func() {
				It("returns error if --bind fails", func() {
					cmdRunner.AddCmdResult(
						"mount --bind /fake-ephemeral-dir/fake-id /fake-ephemeral-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
					)

					_, err := hostBindMounts.MakeEphemeral(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})

				It("returns error if --make-private fails", func() {
					cmdRunner.AddCmdResult(
						"mount --make-private /fake-ephemeral-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
					)

					_, err := hostBindMounts.MakeEphemeral(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})
			})
		})
	})

	Describe("DeleteEphemeral", func() {
		Context("when directory for requested id exists", func() {
			var (
				path string
			)

			BeforeEach(func() {
				var err error

				path, err = hostBindMounts.MakeEphemeral(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when unmounting directory succeeds", func() {
				It("deletes directory for requested id", func() {
					err := hostBindMounts.DeleteEphemeral(apiv1.NewVMCID("fake-id"))
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.FileExists(path)).To(BeFalse())
				})

				It("returns error if deleting directory fails", func() {
					fs.RemoveAllStub = func(string) error {
						return errors.New("fake-remove-all-err")
					}

					err := hostBindMounts.DeleteEphemeral(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-remove-all-err"))
				})
			})

			Context("when unmounting fails because it is not mounted", func() {
				BeforeEach(func() {
					cmdRunner.AddCmdResult(
						"umount /fake-ephemeral-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("not mounted")},
					)
				})

				It("deletes directory for requested id", func() {
					err := hostBindMounts.DeleteEphemeral(apiv1.NewVMCID("fake-id"))
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.FileExists(path)).To(BeFalse())
				})
			})

			Context("when unmounting directory fails", func() {
				BeforeEach(func() {
					cmdRunner.AddCmdResult(
						"umount /fake-ephemeral-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
					)
				})

				It("returns error", func() {
					err := hostBindMounts.DeleteEphemeral(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})

				It("does not delete directory because unmounting failed", func() {
					err := hostBindMounts.DeleteEphemeral(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())

					Expect(fs.FileExists(path)).To(BeTrue())
				})
			})
		})

		Context("when directory for requested id does not exist", func() {
			It("does not return error", func() {
				err := hostBindMounts.DeleteEphemeral(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not unmount directory", func() {
				err := hostBindMounts.DeleteEphemeral(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())

				Expect(cmdRunner.RunCommands).To(BeEmpty())
			})
		})
	})

	Describe("MakePersistent", func() {
		It("creates directory for requested id", func() {
			path, err := hostBindMounts.MakePersistent(apiv1.NewVMCID("fake-id"))
			Expect(err).ToNot(HaveOccurred())
			Expect(path).To(Equal("/fake-persistent-dir/fake-id"))

			pathStat := fs.GetFileTestStat("/fake-persistent-dir/fake-id")
			Expect(pathStat.FileType).To(Equal(fakesys.FakeFileTypeDir))
			Expect(int(pathStat.FileMode)).To(Equal(0755)) // todo
		})

		Context("when creating directory succeeds", func() {
			It("makes the bind mount point shareable", func() {
				_, err := hostBindMounts.MakePersistent(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())

				Expect(cmdRunner.RunCommands).To(Equal([][]string{
					[]string{
						"mount", "--bind",
						"/fake-persistent-dir/fake-id",
						"/fake-persistent-dir/fake-id",
					},
					[]string{
						"mount", "--make-unbindable",
						"/fake-persistent-dir/fake-id",
					},
					[]string{
						"mount", "--make-shared",
						"/fake-persistent-dir/fake-id",
					},
				}))
			})

			Context("when making bind point shareable fails", func() {
				It("returns error if --bind fails", func() {
					cmdRunner.AddCmdResult(
						"mount --bind /fake-persistent-dir/fake-id /fake-persistent-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
					)

					_, err := hostBindMounts.MakePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})

				It("returns error if --make-unbindable fails", func() {
					cmdRunner.AddCmdResult(
						"mount --make-unbindable /fake-persistent-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
					)

					_, err := hostBindMounts.MakePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})

				It("returns error if --make-shared fails", func() {
					cmdRunner.AddCmdResult(
						"mount --make-shared /fake-persistent-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
					)

					_, err := hostBindMounts.MakePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})
			})
		})

		Context("when creating directory fails", func() {
			BeforeEach(func() {
				fs.MkdirAllError = errors.New("fake-mkdir-all-err")
			})

			It("returns error if creating directory fails", func() {
				path, err := hostBindMounts.MakePersistent(apiv1.NewVMCID("fake-id"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
				Expect(path).To(Equal(""))
			})

			It("does not run any mount comamnds (also implies that mount runs after creating dir)", func() {
				_, err := hostBindMounts.MakePersistent(apiv1.NewVMCID("fake-id"))
				Expect(err).To(HaveOccurred())
				Expect(cmdRunner.RunCommands).To(BeEmpty())
			})
		})
	})

	Describe("DeletePersistent", func() {
		Context("when directory for requested id exists", func() {
			var (
				path string
			)

			BeforeEach(func() {
				var err error

				path, err = hostBindMounts.MakePersistent(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())

				fs.SetGlob("/fake-persistent-dir/fake-id/*", []string{
					"/fake-persistent-dir/fake-id/fake-disk-id-1",
					"/fake-persistent-dir/fake-id/fake-disk-id-2",
				})

				cmdRunner.AddCmdResult(
					"mount",
					fakesys.FakeCmdResult{Stdout: "/fake-persistent-dir/fake-id/fake-disk-id-1\n/fake-persistent-dir/fake-id/fake-disk-id-2"},
				)
				cmdRunner.AddCmdResult(
					"mount",
					fakesys.FakeCmdResult{Stdout: "/fake-persistent-dir/fake-id/fake-disk-id-2"},
				)

				cmdRunner.RunCommands = [][]string{} // Reset cmd runner comamnds
			})

			It("unmounts all mount points in that directory and then directory itself", func() {
				err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())

				Expect(cmdRunner.RunCommands).To(Equal([][]string{
					[]string{"mount"},
					[]string{"umount", "/fake-persistent-dir/fake-id/fake-disk-id-1"},
					[]string{"mount"},
					[]string{"umount", "/fake-persistent-dir/fake-id/fake-disk-id-2"},
					[]string{"umount", "/fake-persistent-dir/fake-id"},
				}))
			})

			Context("when getting mounted disk paths fails", func() {
				BeforeEach(func() {
					fs.GlobErr = errors.New("fake-glob-error")
				})

				It("returns an error", func() {
					err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-glob-error"))
				})
			})

			Context("when unmounting directory succeeds", func() {
				It("deletes directory for requested id", func() {
					err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.FileExists(path)).To(BeFalse())
				})

				It("returns error if deleting directory fails", func() {
					fs.RemoveAllStub = func(string) error {
						return errors.New("fake-remove-all-err")
					}

					err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-remove-all-err"))
				})
			})

			Context("when unmounting fails because it is not mounted", func() {
				BeforeEach(func() {
					cmdRunner.AddCmdResult(
						"umount /fake-persistent-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("not mounted")},
					)
				})

				It("deletes directory for requested id", func() {
					err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).ToNot(HaveOccurred())

					Expect(fs.FileExists(path)).To(BeFalse())
				})
			})

			Context("when unmounting directory fails", func() {
				BeforeEach(func() {
					cmdRunner.AddCmdResult(
						"umount /fake-persistent-dir/fake-id",
						fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
					)
				})

				It("returns error", func() {
					err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})

				It("does not delete directory because unmounting failed", func() {
					err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
					Expect(err).To(HaveOccurred())

					Expect(fs.FileExists(path)).To(BeTrue())
				})
			})
		})

		Context("when directory for requested id does not exist", func() {
			It("does not return error", func() {
				err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not unmount directory", func() {
				err := hostBindMounts.DeletePersistent(apiv1.NewVMCID("fake-id"))
				Expect(err).ToNot(HaveOccurred())

				Expect(cmdRunner.RunCommands).To(BeEmpty())
			})
		})
	})

	Describe("MountPersistent", func() {
		It("creates directory for mount point for that requested id and disk id", func() {
			err := hostBindMounts.MountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"), "/fake-disk-path")
			Expect(err).ToNot(HaveOccurred())

			pathStat := fs.GetFileTestStat("/fake-persistent-dir/fake-id/fake-disk-id")
			Expect(pathStat.FileType).To(Equal(fakesys.FakeFileTypeDir))
			Expect(int(pathStat.FileMode)).To(Equal(0755)) // todo
		})

		Context("when creating directory succeeds", func() {
			It("mounts disk path as a loop back device", func() {
				err := hostBindMounts.MountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"), "/fake-disk-path")
				Expect(err).ToNot(HaveOccurred())

				Expect(cmdRunner.RunCommands).To(HaveLen(1))
				Expect(cmdRunner.RunCommands[0]).To(Equal(
					[]string{"mount", "/fake-disk-path", "/fake-persistent-dir/fake-id/fake-disk-id", "-o", "loop"},
				))
			})

			Context("when mounting fails", func() {
				It("returns error", func() {
					cmdRunner.AddCmdResult(
						"mount /fake-disk-path /fake-persistent-dir/fake-id/fake-disk-id -o loop",
						fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
					)

					err := hostBindMounts.MountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"), "/fake-disk-path")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})
			})
		})

		Context("when creating directory fails", func() {
			BeforeEach(func() {
				fs.MkdirAllError = errors.New("fake-mkdir-all-err")
			})

			It("returns error if creating directory fails", func() {
				err := hostBindMounts.MountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"), "/fake-disk-path")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mkdir-all-err"))
			})

			It("does not run any mount comamnds (also implies that mount runs after creating dir)", func() {
				err := hostBindMounts.MountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"), "/fake-disk-path")
				Expect(err).To(HaveOccurred())
				Expect(cmdRunner.RunCommands).To(BeEmpty())
			})
		})
	})

	Describe("UnmountPersistent", func() {
		It("unmounts disk path if disk path is mounted", func() {
			cmdRunner.AddCmdResult("mount", fakesys.FakeCmdResult{
				Stdout: `/dev/sda1 on / type ext4 (rw)
/fake-persistent-dir/fake-id/fake-disk-id on /fake-disk-path type none (rw,bind)`,
			})

			err := hostBindMounts.UnmountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"))
			Expect(err).ToNot(HaveOccurred())

			Expect(cmdRunner.RunCommands).To(HaveLen(2))
			Expect(cmdRunner.RunCommands).To(Equal([][]string{
				[]string{"mount"},
				[]string{"umount", "/fake-persistent-dir/fake-id/fake-disk-id"},
			}))
		})

		It("does not try to unmount disk path if it is not mounted", func() {
			cmdRunner.AddCmdResult("mount", fakesys.FakeCmdResult{
				Stdout: "/dev/sda1 on / type ext4 (rw)",
			})

			err := hostBindMounts.UnmountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"))
			Expect(err).ToNot(HaveOccurred())

			Expect(cmdRunner.RunCommands).To(HaveLen(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"mount"}))
		})

		It("returns error if checking mount information fails", func() {
			cmdRunner.AddCmdResult("mount", fakesys.FakeCmdResult{
				Error: errors.New("fake-run-err"),
			})

			err := hostBindMounts.UnmountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-run-err"))

			// Does not try to unmount or run mount command again
			Expect(cmdRunner.RunCommands).To(HaveLen(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"mount"}))
		})

		It("tries to unmount disk path up to 60 times", func() {
			for i := 0; i < 60; i++ {
				cmdRunner.AddCmdResult("mount", fakesys.FakeCmdResult{
					Stdout: "/fake-persistent-dir/fake-id/fake-disk-id",
				})
			}

			for i := 0; i < 59; i++ {
				cmdRunner.AddCmdResult(
					"umount /fake-persistent-dir/fake-id/fake-disk-id",
					fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
				)
			}

			// 60th time
			cmdRunner.AddCmdResult(
				"umount /fake-persistent-dir/fake-id/fake-disk-id",
				fakesys.FakeCmdResult{},
			)

			err := hostBindMounts.UnmountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"))
			Expect(err).ToNot(HaveOccurred())

			// Mount check and unmount operations performed
			Expect(cmdRunner.RunCommands).To(HaveLen(120))

			for i := 0; i < 120; i += 2 {
				Expect(cmdRunner.RunCommands[i]).To(Equal([]string{"mount"}))
				Expect(cmdRunner.RunCommands[i+1]).To(Equal(
					[]string{"umount", "/fake-persistent-dir/fake-id/fake-disk-id"},
				))
			}

			// Times slept in between unmount operations
			Expect(sleeper.SleptTimes()).To(HaveLen(59))

			for _, d := range sleeper.SleptTimes() {
				Expect(d).To(Equal(3 * time.Second))
			}
		})

		It("returns error if unmounting disk path fails at 60th time", func() {
			for i := 0; i < 60; i++ {
				cmdRunner.AddCmdResult("mount", fakesys.FakeCmdResult{
					Stdout: "/fake-persistent-dir/fake-id/fake-disk-id",
				})

				cmdRunner.AddCmdResult(
					"umount /fake-persistent-dir/fake-id/fake-disk-id",
					fakesys.FakeCmdResult{Error: errors.New("fake-run-err")},
				)
			}

			err := hostBindMounts.UnmountPersistent(apiv1.NewVMCID("fake-id"), apiv1.NewDiskCID("fake-disk-id"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-run-err"))

			Expect(cmdRunner.RunCommands).To(HaveLen(120))

			for i := 0; i < 120; i += 2 {
				Expect(cmdRunner.RunCommands[i]).To(Equal([]string{"mount"}))
				Expect(cmdRunner.RunCommands[i+1]).To(Equal(
					[]string{"umount", "/fake-persistent-dir/fake-id/fake-disk-id"},
				))
			}
		})
	})
})
