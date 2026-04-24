package vm_test

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	wrdn "code.cloudfoundry.org/garden"
	wrdnclient "code.cloudfoundry.org/garden/client"
	fakewrdnconn "code.cloudfoundry.org/garden/client/connection/connectionfakes"
	fakewrdn "code.cloudfoundry.org/garden/gardenfakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "bosh-warden-cpi/vm"
)

var _ = Describe("WardenFileService", func() {
	var (
		wardenConn   *fakewrdnconn.FakeConnection
		wardenClient wrdnclient.Client

		wardenFileService WardenFileService
	)

	BeforeEach(func() {
		wardenConn = &fakewrdnconn.FakeConnection{}
		wardenClient = wrdnclient.New(wardenConn)

		wardenConn.CreateReturns("fake-vm-id", nil)

		containerSpec := wrdn.ContainerSpec{
			Handle:     "fake-vm-id",
			RootFSPath: "fake-root-fs-path",
		}

		container, err := wardenClient.Create(containerSpec)
		Expect(err).ToNot(HaveOccurred())

		logger := boshlog.NewLogger(boshlog.LevelNone)
		wardenFileService = NewWardenFileService(container, logger)
	})

	Describe("Upload", func() {
		var (
			runProcess *fakewrdn.FakeProcess
		)

		BeforeEach(func() {
			runProcess = &fakewrdn.FakeProcess{}
			runProcess.WaitReturns(0, nil)
			wardenConn.RunReturns(runProcess, nil)
		})

		It("places content into the container at the destination directory", func() {
			err := wardenFileService.Upload("/var/vcap/file.ext", []byte("fake-contents"))
			Expect(err).ToNot(HaveOccurred())

			count := wardenConn.StreamInCallCount()
			Expect(count).To(Equal(1))

			handle, spec := wardenConn.StreamInArgsForCall(0)
			Expect(handle).To(Equal("fake-vm-id"))
			Expect(spec.Path).To(Equal("/var/vcap/"))
			Expect(spec.User).To(Equal("root"))

			tarStream := tar.NewReader(spec.TarStream)

			header, err := tarStream.Next()
			Expect(err).ToNot(HaveOccurred())
			Expect(header.Name).To(Equal("file.ext"))

			contentBytes := make([]byte, header.Size)

			tarStream.Read(contentBytes) //nolint:errcheck

			Expect(contentBytes).To(Equal([]byte("fake-contents")))

			_, err = tarStream.Next()
			Expect(err).To(HaveOccurred())
		})

		Context("when streaming into the container succeeds", func() {
			It("verifies the file exists at the destination", func() {
				err := wardenFileService.Upload("/var/vcap/file.ext", []byte("fake-contents"))
				Expect(err).ToNot(HaveOccurred())

				// Should make 2 Run calls: sync + file check
				count := wardenConn.RunCallCount()
				Expect(count).To(Equal(2))

				// First call: sync
				handle, processSpec, _ := wardenConn.RunArgsForCall(0)
				Expect(handle).To(Equal("fake-vm-id"))
				Expect(processSpec.Path).To(Equal("bash"))
				Expect(processSpec.User).To(Equal("root"))
				Expect(processSpec.Args).To(HaveLen(2))
				Expect(processSpec.Args[0]).To(Equal("-c"))
				Expect(processSpec.Args[1]).To(Equal("sync"))

				// Second call: file existence check
				handle, processSpec, _ = wardenConn.RunArgsForCall(1)
				Expect(handle).To(Equal("fake-vm-id"))
				Expect(processSpec.Path).To(Equal("bash"))
				Expect(processSpec.User).To(Equal("root"))
				Expect(processSpec.Args).To(HaveLen(2))
				Expect(processSpec.Args[0]).To(Equal("-c"))
				Expect(processSpec.Args[1]).To(Equal("[ -f '/var/vcap/file.ext' ]"))
			})

			Context("when verifying the file fails because command exits with non-0 code", func() {
				BeforeEach(func() {
					runProcess.WaitReturns(1, nil)
				})

				It("returns error", func() {
					err := wardenFileService.Upload("/var/vcap/file.ext", []byte("fake-contents"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Script exited with non-0 exit code"))
				})
			})

			Context("when verifying the file fails", func() {
				BeforeEach(func() {
					runProcess.WaitReturns(0, errors.New("fake-wait-err"))
				})

				It("returns error", func() {
					err := wardenFileService.Upload("/var/vcap/file.ext", []byte("fake-contents"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-wait-err"))
				})
			})

			Context("when verifying the file cannot start", func() {
				BeforeEach(func() {
					wardenConn.RunReturns(nil, errors.New("fake-run-err"))
				})

				It("returns error", func() {
					err := wardenFileService.Upload("/var/vcap/file.ext", []byte("fake-contents"))
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				})
			})
		})

		Context("when container fails to stream in", func() {
			BeforeEach(func() {
				wardenConn.StreamInReturns(errors.New("fake-stream-in-err"))
			})

			It("returns error", func() {
				err := wardenFileService.Upload("/var/vcap/file.ext", []byte("fake-contents"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-stream-in-err"))
			})
		})

		Context("when file is not immediately visible (retry logic)", func() {
			var (
				callCount int
			)

			BeforeEach(func() {
				callCount = 0
				wardenConn.RunStub = func(handle string, spec wrdn.ProcessSpec, io wrdn.ProcessIO) (wrdn.Process, error) {
					callCount++
					process := &fakewrdn.FakeProcess{}

					// sync always succeeds (odd call numbers)
					if callCount%2 == 1 {
						process.WaitReturns(0, nil)
						return process, nil
					}

					// First 2 file checks fail (call 2, 4)
					if callCount <= 4 {
						process.WaitReturns(1, nil) // file not found
						return process, nil
					}

					// Third file check succeeds (call 6)
					process.WaitReturns(0, nil)
					return process, nil
				}
			})

			It("retries verification until file is visible", func() {
				err := wardenFileService.Upload("/var/vcap/file.ext", []byte("fake-contents"))
				Expect(err).ToNot(HaveOccurred())

				// Should have: sync(1) + check(2) fail, sync(3) + check(4) fail, sync(5) + check(6) success = 6 calls
				count := wardenConn.RunCallCount()
				Expect(count).To(Equal(6))

				for i := 0; i < 6; i++ {
					handle, processSpec, _ := wardenConn.RunArgsForCall(i)
					Expect(handle).To(Equal("fake-vm-id"))
					Expect(processSpec.Path).To(Equal("bash"))
					Expect(processSpec.User).To(Equal("root"))

					if i%2 == 0 {
						// sync commands (even indices: 0, 2, 4)
						Expect(processSpec.Args[1]).To(Equal("sync"))
					} else {
						// file check commands (odd indices: 1, 3, 5)
						Expect(processSpec.Args[1]).To(Equal("[ -f '/var/vcap/file.ext' ]"))
					}
				}
			})
		})

		Context("when file never becomes visible (retry exhaustion)", func() {
			BeforeEach(func() {
				callCount := 0
				wardenConn.RunStub = func(handle string, spec wrdn.ProcessSpec, io wrdn.ProcessIO) (wrdn.Process, error) {
					callCount++
					process := &fakewrdn.FakeProcess{}

					// sync always succeeds (odd call numbers)
					if callCount%2 == 1 {
						process.WaitReturns(0, nil)
						return process, nil
					}

					// All file checks fail
					process.WaitReturns(1, nil)
					return process, nil
				}
			})

			It("returns error after exhausting retries", func() {
				err := wardenFileService.Upload("/var/vcap/file.ext", []byte("fake-contents"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Script exited with non-0 exit code"))

				// Should have attempted 10 times: 10 syncs + 10 checks = 20 calls
				count := wardenConn.RunCallCount()
				Expect(count).To(Equal(20))
			})
		})
	})

	Describe("Download", func() {
		var (
			runProcess *fakewrdn.FakeProcess
		)

		BeforeEach(func() {
			runProcess = &fakewrdn.FakeProcess{}
			runProcess.WaitReturns(0, nil)
			wardenConn.RunReturns(runProcess, nil)
		})

		makeValidAgentEnvTar := func() io.ReadCloser {
			tarBytes := &bytes.Buffer{}

			tarWriter := tar.NewWriter(tarBytes)

			contents := []byte("fake-contents")

			fileHeader := &tar.Header{
				Name: "warden-cpi-agent-env.json",
				Size: int64(len(contents)),
			}

			err := tarWriter.WriteHeader(fileHeader)
			Expect(err).ToNot(HaveOccurred())

			_, err = tarWriter.Write(contents)
			Expect(err).ToNot(HaveOccurred())

			err = tarWriter.Close()
			Expect(err).ToNot(HaveOccurred())

			return io.NopCloser(tarBytes)
		}

		BeforeEach(func() {
			wardenConn.StreamOutReturns(makeValidAgentEnvTar(), nil)
		})

		It("copies agent env into temporary location with a unique name", func() {
			_, err := wardenFileService.Download("/fake-download-path/file.ext")
			Expect(err).ToNot(HaveOccurred())

			count := wardenConn.RunCallCount()
			Expect(count).To(Equal(1))

			handle, processSpec, _ := wardenConn.RunArgsForCall(0)
			Expect(handle).To(Equal("fake-vm-id"))
			Expect(processSpec.Path).To(Equal("bash"))
			Expect(processSpec.User).To(Equal("root"))
			Expect(processSpec.Args).To(HaveLen(2))
			Expect(processSpec.Args[0]).To(Equal("-c"))
			Expect(processSpec.Args[1]).To(MatchRegexp(
				`^cp /fake-download-path/file\.ext /tmp/file-[0-9a-f]+\.ext && chown vcap:vcap /tmp/file-[0-9a-f]+\.ext$`,
			))
		})

		Context("when copying agent env into temporary location succeeds", func() {
			Context("when container succeeds to stream out agent env", func() {
				It("returns agent env from temporary location in the container", func() {
					contents, err := wardenFileService.Download("/fake-download-path/file.ext")
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal([]byte("fake-contents")))

					count := wardenConn.StreamOutCallCount()
					Expect(count).To(Equal(1))

					handle, spec := wardenConn.StreamOutArgsForCall(0)
					Expect(handle).To(Equal("fake-vm-id"))
					Expect(spec.Path).To(MatchRegexp(`^/tmp/file-[0-9a-f]+\.ext$`))
					Expect(spec.User).To(Equal("root"))
				})
			})

			Context("when container fails to stream out because tar stream contains bad header", func() {
				BeforeEach(func() {
					wardenConn.StreamOutReturns(io.NopCloser(&bytes.Buffer{}), nil)
				})

				It("returns error", func() {
					contents, err := wardenFileService.Download("/fake-download-path/file.ext")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Reading tar header for 'file.ext'"))
					Expect(contents).To(Equal([]byte{}))
				})
			})

			Context("when container fails to stream out", func() {
				BeforeEach(func() {
					wardenConn.StreamOutReturns(nil, errors.New("fake-stream-out-err"))
				})

				It("returns error", func() {
					contents, err := wardenFileService.Download("/fake-download-path/file.ext")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-stream-out-err"))
					Expect(contents).To(Equal([]byte{}))
				})
			})
		})

		Context("when copying file into temporary location fails because command exits with non-0 code", func() {
			BeforeEach(func() {
				runProcess.WaitReturns(1, nil)
			})

			It("returns error", func() {
				contents, err := wardenFileService.Download("/fake-download-path/file.ext")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Script exited with non-0 exit code"))
				Expect(contents).To(Equal([]byte{}))
			})
		})

		Context("when copying file into temporary location fails", func() {
			BeforeEach(func() {
				runProcess.WaitReturns(0, errors.New("fake-wait-err"))
			})

			It("returns error", func() {
				contents, err := wardenFileService.Download("/fake-download-path/file.ext")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-wait-err"))
				Expect(contents).To(Equal([]byte{}))
			})
		})

		Context("when copying file into temporary location cannot start", func() {
			BeforeEach(func() {
				wardenConn.RunReturns(nil, errors.New("fake-run-err"))
			})

			It("returns error", func() {
				contents, err := wardenFileService.Download("/fake-download-path/file.ext")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-run-err"))
				Expect(contents).To(Equal([]byte{}))
			})
		})
	})
})
