package server_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"
	fakes "code.cloudfoundry.org/garden/gardenfakes"
	"code.cloudfoundry.org/garden/server"
)

var _ = Describe("When connecting directly to the server", func() {
	var (
		apiServer                *server.GardenServer
		logger                   *lagertest.TestLogger
		fakeBackend              *fakes.FakeBackend
		fakeContainer            *fakes.FakeContainer
		serverContainerGraceTime time.Duration
		sink                     *lagertest.TestSink
		port                     int
		client                   *http.Client
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		sink = lagertest.NewTestSink()
		logger.RegisterSink(sink)
		fakeBackend = new(fakes.FakeBackend)
		serverContainerGraceTime = 42 * time.Second
		client = &http.Client{}

		fakeContainer = new(fakes.FakeContainer)
		fakeContainer.HandleReturns("some-handle")

		fakeBackend.CreateReturns(fakeContainer, nil)
		port = 8000 + config.GinkgoConfig.ParallelNode
		apiServer = server.New(
			"tcp",
			fmt.Sprintf(":%d", port),
			serverContainerGraceTime,
			fakeBackend,
			logger,
		)
		listenAndServe(apiServer, "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	})

	AfterEach(func() {
		apiServer.Stop()
	})

	Context("when not specifing the content type", func() {
		It("handles the request", func() {
			request, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/containers", port), strings.NewReader("{}"))
			Expect(err).NotTo(HaveOccurred())
			response, err := client.Do(request)
			Expect(err).NotTo(HaveOccurred())

			body, err := ioutil.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(ContainSubstring("some-handle"))
			Expect(response.StatusCode).To(Equal(http.StatusOK))
		})
	})
})

var _ = Describe("When a client connects", func() {
	var socketPath string
	var tmpdir string

	var serverBackend *fakes.FakeBackend

	var serverContainerGraceTime time.Duration

	var logger *lagertest.TestLogger
	var sink *lagertest.TestSink

	var apiServer *server.GardenServer
	var apiClient garden.Client
	var isRunning bool

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		sink = lagertest.NewTestSink()
		logger.RegisterSink(sink)

		var err error
		tmpdir, err = ioutil.TempDir(os.TempDir(), "api-server-test")
		Expect(err).ToNot(HaveOccurred())

		socketPath = path.Join(tmpdir, "api.sock")
		serverBackend = new(fakes.FakeBackend)
		serverContainerGraceTime = 42 * time.Second

		apiServer = server.New(
			"unix",
			socketPath,
			serverContainerGraceTime,
			serverBackend,
			logger,
		)
		listenAndServe(apiServer, "unix", socketPath)

		isRunning = true

		apiClient = client.New(connection.New("unix", socketPath))

		Eventually(apiClient.Ping).Should(Succeed())
	})

	AfterEach(func() {
		if isRunning {
			apiServer.Stop()
		}
		if tmpdir != "" {
			os.RemoveAll(tmpdir)
		}
	})

	Context("and the client sends a PingRequest", func() {
		Context("and the backend ping succeeds", func() {
			It("does not error", func() {
				Expect(apiClient.Ping()).ToNot(HaveOccurred())
			})
		})

		Context("when the backend ping fails", func() {
			BeforeEach(func() {
				serverBackend.PingReturns(errors.New("oh no!"))
			})

			It("returns an error", func() {
				Expect(apiClient.Ping()).To(HaveOccurred())
			})
		})

		Context("when the backend ping fails with an UnrecoverableError", func() {
			BeforeEach(func() {
				serverBackend.PingReturns(garden.NewUnrecoverableError("ermahgahd!"))
			})

			It("returns an UnrecoverableError", func() {
				Expect(apiClient.Ping()).To(BeAssignableToTypeOf(garden.UnrecoverableError{}))
			})
		})

		Context("when the server is not up", func() {
			BeforeEach(func() {
				isRunning = false
				apiServer.Stop()
			})

			It("returns an error", func() {
				Expect(apiClient.Ping()).To(HaveOccurred())
			})
		})
	})

	Context("and the client sends a CapacityRequest", func() {
		BeforeEach(func() {
			serverBackend.CapacityReturns(garden.Capacity{
				MemoryInBytes: 1111,
				DiskInBytes:   2222,
				MaxContainers: 42,
			}, nil)
		})

		It("returns the backend's reported capacity", func() {
			capacity, err := apiClient.Capacity()
			Expect(err).ToNot(HaveOccurred())

			Expect(capacity.MemoryInBytes).To(Equal(uint64(1111)))
			Expect(capacity.DiskInBytes).To(Equal(uint64(2222)))
			Expect(capacity.MaxContainers).To(Equal(uint64(42)))
		})

		Context("when getting the capacity fails", func() {
			BeforeEach(func() {
				serverBackend.CapacityReturns(garden.Capacity{}, errors.New("oh no!"))
			})

			It("returns an error", func() {
				_, err := apiClient.Capacity()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("and the client sends a CreateRequest", func() {
		var fakeContainer *fakes.FakeContainer

		BeforeEach(func() {
			fakeContainer = new(fakes.FakeContainer)
			fakeContainer.HandleReturns("some-handle")

			serverBackend.CreateReturns(fakeContainer, nil)
		})

		It("returns a container with the created handle", func() {
			container, err := apiClient.Create(garden.ContainerSpec{
				Handle: "some-handle",
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(container.Handle()).To(Equal("some-handle"))
		})

		It("should not log any container spec properties", func() {
			_, err := apiClient.Create(garden.ContainerSpec{
				Handle:     "some-handle",
				Properties: garden.Properties{"CONTAINER_PROPERTY": "CONTAINER_SECRET"},
			})
			Expect(err).ToNot(HaveOccurred())

			buffer := sink.Buffer()
			Expect(buffer).ToNot(gbytes.Say("CONTAINER_PROPERTY"))
			Expect(buffer).ToNot(gbytes.Say("CONTAINER_SECRET"))
		})

		It("should not log any environment variables", func() {
			_, err := apiClient.Create(garden.ContainerSpec{
				Handle: "some-handle",
				Env:    []string{"PASSWORD=MY_SECRET"},
			})
			Expect(err).ToNot(HaveOccurred())

			buffer := sink.Buffer()
			Expect(buffer).ToNot(gbytes.Say("PASSWORD"))
			Expect(buffer).ToNot(gbytes.Say("MY_SECRET"))
		})

		It("creates the container with the spec from the request", func() {
			_, err := apiClient.Create(garden.ContainerSpec{
				Handle:     "some-handle",
				GraceTime:  42 * time.Second,
				Network:    "some-network",
				RootFSPath: "/path/to/rootfs",
				BindMounts: []garden.BindMount{
					{
						SrcPath: "/bind/mount/src",
						DstPath: "/bind/mount/dst",
						Mode:    garden.BindMountModeRW,
						Origin:  garden.BindMountOriginContainer,
					},
				},
				Properties: garden.Properties{
					"prop-a": "val-a",
					"prop-b": "val-b",
				},
				Env: []string{"env1=env1Value", "env2=env2Value"},
				Limits: garden.Limits{
					Bandwidth: garden.BandwidthLimits{
						RateInBytesPerSecond:      42,
						BurstRateInBytesPerSecond: 68,
					},
					Disk: garden.DiskLimits{
						InodeSoft: 1,
						InodeHard: 2,
						ByteSoft:  3,
						ByteHard:  4,
						Scope:     garden.DiskLimitScopeExclusive,
					},
					Memory: garden.MemoryLimits{
						LimitInBytes: 1024,
					},
					CPU: garden.CPULimits{
						LimitInShares: 5,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(serverBackend.CreateArgsForCall(0)).To(Equal(garden.ContainerSpec{
				Handle:     "some-handle",
				GraceTime:  time.Duration(42 * time.Second),
				Network:    "some-network",
				RootFSPath: "/path/to/rootfs",
				BindMounts: []garden.BindMount{
					{
						SrcPath: "/bind/mount/src",
						DstPath: "/bind/mount/dst",
						Mode:    garden.BindMountModeRW,
						Origin:  garden.BindMountOriginContainer,
					},
				},
				Properties: map[string]string{
					"prop-a": "val-a",
					"prop-b": "val-b",
				},
				Env: []string{"env1=env1Value", "env2=env2Value"},
				Limits: garden.Limits{
					Bandwidth: garden.BandwidthLimits{
						RateInBytesPerSecond:      42,
						BurstRateInBytesPerSecond: 68,
					},
					Disk: garden.DiskLimits{
						InodeSoft: 1,
						InodeHard: 2,
						ByteSoft:  3,
						ByteHard:  4,
						Scope:     garden.DiskLimitScopeExclusive,
					},
					Memory: garden.MemoryLimits{
						LimitInBytes: 1024,
					},
					CPU: garden.CPULimits{
						LimitInShares: 5,
					},
				},
			}))
		})

		Context("when a grace time is given", func() {
			var graceTime time.Duration

			BeforeEach(func() {
				graceTime = time.Second

				fakeContainer = new(fakes.FakeContainer)
				fakeContainer.HandleReturns("doomed-handle")
				serverBackend.CreateReturns(fakeContainer, nil)
				serverBackend.LookupReturns(fakeContainer, nil)
			})

			JustBeforeEach(func() {
				serverBackend.GraceTimeReturns(graceTime)
			})

			It("destroys the container after it has been idle for the grace time", func() {
				before := time.Now()

				_, err := apiClient.Create(garden.ContainerSpec{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(serverBackend.DestroyCallCount, 2*time.Second).Should(Equal(1))
				Expect(serverBackend.DestroyArgsForCall(0)).To(Equal("doomed-handle"))

				Expect(time.Since(before)).To(BeNumerically(">", graceTime), "should not destroy before the grace time expires")
			})

			Context("and a process is running", func() {
				It("destroys the container after it has been idle for the grace time", func() {
					fakeProcess := new(fakes.FakeProcess)
					fakeProcess.IDReturns("doomed-handle")
					fakeProcess.WaitStub = func() (int, error) {
						select {}
					}
					fakeContainer.RunReturns(fakeProcess, nil)

					var clientConnection net.Conn

					apiConnection := connection.NewWithDialerAndLogger(func(string, string) (net.Conn, error) {
						var err error
						clientConnection, err = net.DialTimeout("unix", socketPath, 2*time.Second)
						return clientConnection, err
					}, lagertest.NewTestLogger("api-conn-dialer"))

					apiClient = client.New(apiConnection)

					container, err := apiClient.Create(garden.ContainerSpec{})
					Expect(err).ToNot(HaveOccurred())

					before := time.Now()

					_, err = container.Run(garden.ProcessSpec{}, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					clientConnection.Close()

					Eventually(serverBackend.DestroyCallCount, 2*time.Second).Should(Equal(1))
					Expect(serverBackend.DestroyArgsForCall(0)).To(Equal("doomed-handle"))

					duration := time.Since(before)
					Expect(duration).To(BeNumerically(">=", graceTime))
					Expect(duration).To(BeNumerically("<", graceTime+time.Second))
				})
			})

			Context("but it expires during an API destroy request", func() {
				BeforeEach(func() {
					// increase the grace time so that we can API create/destroy before expiration
					graceTime = time.Second * 3

					serverBackend.DestroyStub = func(string) error {
						// sleep for longer than the grace time
						time.Sleep(graceTime * 2)
						return nil
					}
				})

				It("does not reap the container", func() {
					_, err := apiClient.Create(garden.ContainerSpec{})
					Expect(err).ToNot(HaveOccurred())

					err = apiClient.Destroy("doomed-handle")
					Expect(err).ToNot(HaveOccurred())

					Expect(serverBackend.DestroyArgsForCall(0)).To(Equal("doomed-handle"))
					Expect(serverBackend.DestroyCallCount()).To(Equal(1))
				})
			})
		})

		Context("when a grace time is not given", func() {
			It("defaults it to the server's grace time", func() {
				_, err := apiClient.Create(garden.ContainerSpec{
					Handle: "some-handle",
				})
				Expect(err).ToNot(HaveOccurred())

				spec := serverBackend.CreateArgsForCall(0)
				Expect(spec.GraceTime).To(Equal(serverContainerGraceTime))
			})
		})

		Context("when creating the container fails", func() {
			BeforeEach(func() {
				serverBackend.CreateReturns(nil, errors.New("oh no!"))
			})

			It("returns an error", func() {
				_, err := apiClient.Create(garden.ContainerSpec{
					Handle: "some-handle",
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the container fails with a ServiceUnavailableError", func() {
			var err error

			BeforeEach(func() {
				serverBackend.CreateReturns(nil, garden.NewServiceUnavailableError("special error"))

				_, err = apiClient.Create(garden.ContainerSpec{
					Handle: "some-handle",
				})
			})

			It("client returns an error with a well formed error msg", func() {
				Expect(err).To(MatchError("special error"))
			})

			It("client returns an error of type ServiceUnavailableError", func() {
				_, ok := err.(garden.ServiceUnavailableError)
				Expect(ok).To(BeTrue())
			})
		})
	})

	Context("and the client sends a destroy request", func() {
		It("destroys the container", func() {
			err := apiClient.Destroy("some-handle")
			Expect(err).ToNot(HaveOccurred())

			Expect(serverBackend.DestroyArgsForCall(0)).To(Equal("some-handle"))
		})

		Context("concurrent with other destroy requests", func() {
			var destroying chan struct{}

			BeforeEach(func() {
				destroying = make(chan struct{})

				serverBackend.DestroyStub = func(string) error {
					close(destroying)
					time.Sleep(time.Second)
					return nil
				}
			})

			It("only destroys once", func() {
				go apiClient.Destroy("some-handle")

				<-destroying

				err := apiClient.Destroy("some-handle")
				Expect(err).To(HaveOccurred())

				Expect(serverBackend.DestroyCallCount()).To(Equal(1))
			})
		})

		Context("when the container cannot be found", func() {
			var theError = garden.ContainerNotFoundError{Handle: "some-handle"}

			BeforeEach(func() {
				serverBackend.DestroyReturns(theError)
			})

			It("returns an ContainerNotFoundError", func() {
				err := apiClient.Destroy("some-handle")
				Expect(err).To(MatchError(garden.ContainerNotFoundError{Handle: "some-handle"}))
			})
		})

		Context("when destroying the container fails", func() {
			var theError = errors.New("o no")

			BeforeEach(func() {
				serverBackend.DestroyReturns(theError)
			})

			It("returns an error with the same message", func() {
				err := apiClient.Destroy("some-handle")
				Expect(err).To(MatchError("o no"))
			})

			Context("and destroying is attempted again", func() {
				BeforeEach(func() {
					err := apiClient.Destroy("some-handle")
					Expect(err).To(HaveOccurred())

					serverBackend.DestroyReturns(nil)
				})

				It("tries to destroy again", func() {
					err := apiClient.Destroy("some-handle")
					Expect(err).ToNot(HaveOccurred())

					Expect(serverBackend.DestroyArgsForCall(0)).To(Equal("some-handle"))
				})
			})
		})
	})

	Context("and the client sends a ListRequest", func() {
		BeforeEach(func() {
			c1 := new(fakes.FakeContainer)
			c1.HandleReturns("some-handle")

			c2 := new(fakes.FakeContainer)
			c2.HandleReturns("another-handle")

			c3 := new(fakes.FakeContainer)
			c3.HandleReturns("super-handle")

			serverBackend.ContainersReturns([]garden.Container{c1, c2, c3}, nil)
		})

		It("returns the containers from the backend", func() {
			containers, err := apiClient.Containers(nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(containers).To(HaveLen(3))

			handles := make([]string, 3)
			for i, c := range containers {
				handles[i] = c.Handle()
			}

			Expect(handles).To(ContainElement("some-handle"))
			Expect(handles).To(ContainElement("another-handle"))
			Expect(handles).To(ContainElement("super-handle"))
		})

		Context("when getting the containers fails", func() {
			BeforeEach(func() {
				serverBackend.ContainersReturns(nil, errors.New("oh no!"))
			})

			It("returns an error", func() {
				_, err := apiClient.Containers(nil)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the client sends a ListRequest with a property filter", func() {
			It("forwards the filter to the backend", func() {
				_, err := apiClient.Containers(garden.Properties{
					"foo": "bar",
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(serverBackend.ContainersArgsForCall(serverBackend.ContainersCallCount() - 1)).To(Equal(
					garden.Properties{
						"foo": "bar",
					},
				))
			})

			It("should not log the properties", func() {
				_, err := apiClient.Containers(garden.Properties{
					"hello": "banana",
				})
				Expect(err).ToNot(HaveOccurred())

				buffer := sink.Buffer()
				Expect(buffer).ToNot(gbytes.Say("hello"))
				Expect(buffer).ToNot(gbytes.Say("banana"))
			})

			Context("when getting the containers fails", func() {
				BeforeEach(func() {
					serverBackend.ContainersReturns(nil, errors.New("oh no!"))
				})

				It("should not log the properties", func() {
					apiClient.Containers(garden.Properties{
						"hello": "banana",
					})

					buffer := sink.Buffer()
					Expect(buffer).ToNot(gbytes.Say("hello"))
					Expect(buffer).ToNot(gbytes.Say("banana"))
				})
			})

		})
	})

	Context("when a container has been created", func() {
		var (
			container garden.Container
			graceTime time.Duration

			fakeContainer *fakes.FakeContainer
		)

		BeforeEach(func() {
			fakeContainer = new(fakes.FakeContainer)
			fakeContainer.HandleReturns("some-handle")

			serverBackend.CreateReturns(fakeContainer, nil)
			serverBackend.LookupReturns(fakeContainer, nil)
		})

		JustBeforeEach(func() {
			var err error

			container, err = apiClient.Create(garden.ContainerSpec{})
			Expect(err).ToNot(HaveOccurred())
		})

		itResetsGraceTimeWhenHandling := func(fn func(time.Duration)) {
			graceTime := 500 * time.Millisecond

			BeforeEach(func() {
				serverBackend.GraceTimeReturns(graceTime)
			})

			It("resets grace time when handling", func() {
				fn(graceTime * 2)
				Expect(serverBackend.DestroyCallCount()).To(Equal(0))
				Eventually(serverBackend.DestroyCallCount, graceTime+(1000*time.Millisecond)).Should(Equal(1))
			})
		}

		itFailsWhenTheContainerIsNotFound := func(example func() error) {
			Context("when the container is not found", func() {
				It("fails", func() {
					serverBackend.LookupReturns(nil, errors.New("not found"))
					Expect(example()).To(MatchError("not found"))
				})
			})
		}

		Describe("stopping", func() {
			It("stops the container and sends a StopResponse", func() {
				err := container.Stop(true)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeContainer.StopArgsForCall(0)).To(Equal(true))
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				return container.Stop(true)
			})

			Context("when stopping the container fails", func() {
				BeforeEach(func() {
					fakeContainer.StopReturns(errors.New("oh no!"))
				})

				It("returns an error", func() {
					err := container.Stop(true)
					Expect(err).To(HaveOccurred())
				})
			})

			itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
				fakeContainer.StopStub = func(_ bool) error { time.Sleep(timeToSleep); return nil }
				container.Stop(false)
			})
		})

		Describe("metrics", func() {

			containerMetrics := garden.Metrics{
				MemoryStat: garden.ContainerMemoryStat{
					Cache:                   1,
					Rss:                     2,
					MappedFile:              3,
					Pgpgin:                  4,
					Pgpgout:                 5,
					Swap:                    6,
					Pgfault:                 7,
					Pgmajfault:              8,
					InactiveAnon:            9,
					ActiveAnon:              10,
					InactiveFile:            11,
					ActiveFile:              12,
					Unevictable:             13,
					HierarchicalMemoryLimit: 14,
					HierarchicalMemswLimit:  15,
					TotalCache:              16,
					TotalRss:                17,
					TotalMappedFile:         18,
					TotalPgpgin:             19,
					TotalPgpgout:            20,
					TotalSwap:               21,
					TotalPgfault:            22,
					TotalPgmajfault:         23,
					TotalInactiveAnon:       24,
					TotalActiveAnon:         25,
					TotalInactiveFile:       26,
					TotalActiveFile:         27,
					TotalUnevictable:        28,
					TotalUsageTowardLimit:   7, // TotalRss+(TotalCache-TotalInactiveFile)
				},
				CPUStat: garden.ContainerCPUStat{
					Usage:  1,
					User:   2,
					System: 3,
				},
				DiskStat: garden.ContainerDiskStat{
					TotalBytesUsed:  1,
					TotalInodesUsed: 2,
				},
			}

			Context("when getting the metrics succeeds", func() {
				BeforeEach(func() {
					fakeContainer.MetricsReturns(
						containerMetrics,
						nil,
					)
				})

				It("returns the metrics from the container", func() {
					value, err := container.Metrics()
					Expect(err).ToNot(HaveOccurred())

					Expect(value).To(Equal(containerMetrics))
				})

				itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
					fakeContainer.MetricsStub = func() (garden.Metrics, error) { time.Sleep(timeToSleep); return garden.Metrics{}, nil }
					_, err := container.Metrics()
					Expect(err).ToNot(HaveOccurred())
				})

				itFailsWhenTheContainerIsNotFound(func() error {
					_, err := container.Metrics()
					return err
				})
			})

			Context("when getting the metrics fails", func() {
				BeforeEach(func() {
					fakeContainer.MetricsReturns(garden.Metrics{}, errors.New("o no"))
				})

				It("returns an error", func() {
					metrics, err := container.Metrics()
					Expect(err).To(HaveOccurred())
					Expect(metrics).To(Equal(garden.Metrics{}))
				})
			})
		})

		Describe("properties", func() {
			Describe("getting all", func() {
				Context("when getting the properties succeeds", func() {
					BeforeEach(func() {
						fakeContainer.PropertiesReturns(garden.Properties{"foo": "bar"}, nil)
					})

					It("returns the properties from the container", func() {
						value, err := container.Properties()
						Expect(err).ToNot(HaveOccurred())

						Expect(value).To(Equal(garden.Properties{"foo": "bar"}))
					})

					itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
						fakeContainer.PropertiesStub = func() (garden.Properties, error) { time.Sleep(timeToSleep); return nil, nil }
						_, err := container.Properties()
						Expect(err).ToNot(HaveOccurred())
					})

					itFailsWhenTheContainerIsNotFound(func() error {
						_, err := container.Properties()
						return err
					})

					It("should not log any properties", func() {
						_, err := container.Properties()
						Expect(err).ToNot(HaveOccurred())

						buffer := sink.Buffer()
						Expect(buffer).ToNot(gbytes.Say("foo"))
						Expect(buffer).ToNot(gbytes.Say("bar"))
					})
				})

				Context("when getting the properties fails", func() {
					BeforeEach(func() {
						fakeContainer.PropertiesReturns(nil, errors.New("o no"))
					})

					It("returns an error", func() {
						properties, err := container.Properties()
						Expect(err).To(HaveOccurred())
						Expect(properties).To(BeEmpty())
					})
				})
			})

			Describe("getting", func() {
				Context("when getting the property succeeds", func() {
					BeforeEach(func() {
						fakeContainer.PropertyReturns("some-property-value", nil)
					})

					It("returns the property from the container", func() {
						value, err := container.Property("some-property")
						Expect(err).ToNot(HaveOccurred())

						Expect(value).To(Equal("some-property-value"))

						Expect(fakeContainer.PropertyCallCount()).To(Equal(1))

						name := fakeContainer.PropertyArgsForCall(0)
						Expect(name).To(Equal("some-property"))
					})

					itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
						fakeContainer.PropertyStub = func(string) (string, error) { time.Sleep(timeToSleep); return "", nil }
						_, err := container.Property("some-property")
						Expect(err).ToNot(HaveOccurred())
					})

					itFailsWhenTheContainerIsNotFound(func() error {
						_, err := container.Property("some-property")
						return err
					})

					It("should not log any properties", func() {
						_, err := container.Property("some-property")
						Expect(err).ToNot(HaveOccurred())

						buffer := sink.Buffer()
						Expect(buffer).ToNot(gbytes.Say("some-property"))
						Expect(buffer).ToNot(gbytes.Say("some-property-value"))
					})
				})

				Context("when getting the property fails", func() {
					BeforeEach(func() {
						fakeContainer.PropertyReturns("", errors.New("oh no!"))
					})

					It("returns an error", func() {
						value, err := container.Property("some-property")
						Expect(err).To(HaveOccurred())
						Expect(value).To(BeZero())
					})
				})
			})

			Describe("setting", func() {
				Context("when setting the property succeeds", func() {
					BeforeEach(func() {
						fakeContainer.SetPropertyReturns(nil)
					})

					It("sets the property on the container", func() {
						err := container.SetProperty("some-property", "some-value")
						Expect(err).ToNot(HaveOccurred())

						Expect(fakeContainer.SetPropertyCallCount()).To(Equal(1))

						name, value := fakeContainer.SetPropertyArgsForCall(0)
						Expect(name).To(Equal("some-property"))
						Expect(value).To(Equal("some-value"))
					})

					itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
						fakeContainer.SetPropertyStub = func(string, string) error { time.Sleep(timeToSleep); return nil }
						err := container.SetProperty("some-property", "some-value")
						Expect(err).ToNot(HaveOccurred())
					})

					itFailsWhenTheContainerIsNotFound(func() error {
						return container.SetProperty("some-property", "some-value")
					})

					It("should not log any properties", func() {
						err := container.SetProperty("some-property", "some-value")
						Expect(err).ToNot(HaveOccurred())

						buffer := sink.Buffer()
						Expect(buffer).ToNot(gbytes.Say("some-property"))
						Expect(buffer).ToNot(gbytes.Say("some-value"))
					})
				})

				Context("when setting the property fails", func() {
					BeforeEach(func() {
						fakeContainer.SetPropertyReturns(errors.New("oh no!"))
					})

					It("returns an error", func() {
						err := container.SetProperty("some-property", "some-value")
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Describe("removing", func() {
				Context("when removing the property succeeds", func() {
					BeforeEach(func() {
						fakeContainer.RemovePropertyReturns(nil)
					})

					It("returns the property from the container", func() {
						err := container.RemoveProperty("some-property")
						Expect(err).ToNot(HaveOccurred())

						Expect(fakeContainer.RemovePropertyCallCount()).To(Equal(1))

						name := fakeContainer.RemovePropertyArgsForCall(0)
						Expect(name).To(Equal("some-property"))
					})

					itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
						fakeContainer.RemovePropertyStub = func(string) error { time.Sleep(timeToSleep); return nil }
						err := container.RemoveProperty("some-property")
						Expect(err).ToNot(HaveOccurred())
					})

					itFailsWhenTheContainerIsNotFound(func() error {
						return container.RemoveProperty("some-property")
					})

					It("should not log any properties", func() {
						err := container.RemoveProperty("some-property")
						Expect(err).ToNot(HaveOccurred())

						buffer := sink.Buffer()
						Expect(buffer).ToNot(gbytes.Say("some-property"))
						Expect(buffer).ToNot(gbytes.Say("some-value"))
					})
				})

				Context("when setting the property fails", func() {
					BeforeEach(func() {
						fakeContainer.RemovePropertyReturns(errors.New("oh no!"))
					})

					It("returns an error", func() {
						err := container.RemoveProperty("some-property")
						Expect(err).To(HaveOccurred())
					})
				})
			})
		})

		Describe("streaming in", func() {
			It("streams the file in, waits for completion, and succeeds", func() {
				data := bytes.NewBufferString("chunk-1;chunk-2;chunk-3;")

				fakeContainer.StreamInStub = func(spec garden.StreamInSpec) error {
					Expect(spec.Path).To(Equal("/dst/path"))
					Expect(spec.User).To(Equal("frank"))
					Expect(ioutil.ReadAll(spec.TarStream)).To(Equal([]byte("chunk-1;chunk-2;chunk-3;")))
					return nil
				}

				err := container.StreamIn(garden.StreamInSpec{User: "frank", Path: "/dst/path", TarStream: data})
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeContainer.StreamInCallCount()).To(Equal(1))
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				return container.StreamIn(garden.StreamInSpec{Path: "/dst/path"})
			})

			Context("when copying in to the container fails", func() {
				BeforeEach(func() {
					fakeContainer.StreamInReturns(errors.New("oh no!"))
				})

				It("fails", func() {
					err := container.StreamIn(garden.StreamInSpec{User: "bob", Path: "/dst/path"})
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("streaming out", func() {
			var streamOut io.ReadCloser

			BeforeEach(func() {
				streamOut = ioutil.NopCloser(bytes.NewBuffer([]byte("hello-world!")))
			})

			JustBeforeEach(func() {
				fakeContainer.StreamOutReturns(streamOut, nil)
			})

			It("streams the bits out and succeeds", func() {
				reader, err := container.StreamOut(garden.StreamOutSpec{User: "frank", Path: "/src/path"})
				Expect(err).ToNot(HaveOccurred())
				Expect(reader).ToNot(BeZero())

				streamedContent, err := ioutil.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())

				Expect(string(streamedContent)).To(Equal("hello-world!"))

				Expect(fakeContainer.StreamOutArgsForCall(0)).To(Equal(garden.StreamOutSpec{User: "frank", Path: "/src/path"}))
			})

			Context("when the connection dies as we're streaming", func() {
				var closer *closeChecker

				BeforeEach(func() {
					closer = &closeChecker{}

					streamOut = closer
				})

				It("closes the backend's stream", func() {
					reader, err := container.StreamOut(garden.StreamOutSpec{User: "frank", Path: "/src/path"})
					Expect(err).ToNot(HaveOccurred())

					err = reader.Close()
					Expect(err).ToNot(HaveOccurred())

					Eventually(closer.Closed).Should(BeTrue())
				})
			})

			itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
				fakeContainer.StreamOutStub = func(garden.StreamOutSpec) (io.ReadCloser, error) {
					time.Sleep(timeToSleep)
					return gbytes.NewBuffer(), nil
				}
				reader, err := container.StreamOut(garden.StreamOutSpec{User: "frank", Path: "/src/path"})
				Expect(err).ToNot(HaveOccurred())

				_, err = ioutil.ReadAll(reader)
				Expect(err).ToNot(HaveOccurred())
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				_, err := container.StreamOut(garden.StreamOutSpec{User: "frank", Path: "/src/path"})
				return err
			})

			Context("when streaming out of the container fails", func() {
				JustBeforeEach(func() {
					fakeContainer.StreamOutReturns(nil, errors.New("oh no!"))
				})

				It("returns an error", func() {
					_, err := container.StreamOut(garden.StreamOutSpec{User: "frank", Path: "/src/path"})
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("getting the current bandwidth limits", func() {
			It("returns the limits returned by the backend", func() {
				effectiveLimits := garden.BandwidthLimits{
					RateInBytesPerSecond:      1230,
					BurstRateInBytesPerSecond: 4560,
				}

				fakeContainer.CurrentBandwidthLimitsReturns(effectiveLimits, nil)

				limits, err := container.CurrentBandwidthLimits()
				Expect(err).ToNot(HaveOccurred())

				Expect(limits).To(Equal(effectiveLimits))
			})

			Context("when getting the current limits fails", func() {
				BeforeEach(func() {
					fakeContainer.CurrentBandwidthLimitsReturns(garden.BandwidthLimits{}, errors.New("oh no!"))
				})

				It("fails", func() {
					_, err := container.CurrentBandwidthLimits()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("getting memory limits", func() {
			It("obtains the current limits", func() {
				effectiveLimits := garden.MemoryLimits{LimitInBytes: 2048}
				fakeContainer.CurrentMemoryLimitsReturns(effectiveLimits, nil)

				limits, err := container.CurrentMemoryLimits()
				Expect(err).ToNot(HaveOccurred())
				Expect(limits).ToNot(BeZero())

				Expect(limits).To(Equal(effectiveLimits))
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				_, err := container.CurrentMemoryLimits()
				return err
			})

			Context("when getting the current memory limits fails", func() {
				BeforeEach(func() {
					fakeContainer.CurrentMemoryLimitsReturns(garden.MemoryLimits{}, errors.New("oh no!"))
				})

				It("fails", func() {
					_, err := container.CurrentMemoryLimits()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("getting the current disk limits", func() {
			currentLimits := garden.DiskLimits{
				InodeSoft: 3333,
				InodeHard: 4444,

				ByteSoft: 5555,
				ByteHard: 6666,
			}

			It("returns the limits returned by the backend", func() {
				fakeContainer.CurrentDiskLimitsReturns(currentLimits, nil)

				limits, err := container.CurrentDiskLimits()
				Expect(err).ToNot(HaveOccurred())

				Expect(limits).To(Equal(currentLimits))
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				_, err := container.CurrentDiskLimits()
				return err
			})

			Context("when getting the current disk limits fails", func() {
				BeforeEach(func() {
					fakeContainer.CurrentDiskLimitsReturns(garden.DiskLimits{}, errors.New("oh no!"))
				})

				It("fails", func() {
					_, err := container.CurrentDiskLimits()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("get the current cpu limits", func() {
			effectiveLimits := garden.CPULimits{LimitInShares: 456}

			It("gets the current limits", func() {
				fakeContainer.CurrentCPULimitsReturns(effectiveLimits, nil)

				limits, err := container.CurrentCPULimits()
				Expect(err).ToNot(HaveOccurred())

				Expect(limits).To(Equal(effectiveLimits))
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				_, err := container.CurrentCPULimits()
				return err
			})

			Context("when getting the current CPU limits fails", func() {
				BeforeEach(func() {
					fakeContainer.CurrentCPULimitsReturns(garden.CPULimits{}, errors.New("oh no!"))
				})

				It("fails", func() {
					_, err := container.CurrentCPULimits()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("setting the grace time", func() {
			BeforeEach(func() {
				graceTime = time.Second
				serverBackend.GraceTimeReturns(graceTime)
			})

			It("destroys the container after it has been idle for the grace time", func() {
				before := time.Now()
				Expect(container.SetGraceTime(graceTime)).To(Succeed())

				Eventually(serverBackend.DestroyCallCount, 2*time.Second).Should(Equal(1))
				Expect(serverBackend.DestroyArgsForCall(0)).To(Equal("some-handle"))

				Expect(time.Since(before)).To(BeNumerically(">=", graceTime))
				Expect(time.Since(before)).To(BeNumerically("<", graceTime+time.Second))
			})
		})

		Describe("net in", func() {
			It("maps the ports and returns them", func() {
				fakeContainer.NetInReturns(111, 222, nil)

				hostPort, containerPort, err := container.NetIn(123, 456)
				Expect(err).ToNot(HaveOccurred())

				hp, cp := fakeContainer.NetInArgsForCall(0)
				Expect(hp).To(Equal(uint32(123)))
				Expect(cp).To(Equal(uint32(456)))

				Expect(hostPort).To(Equal(uint32(111)))
				Expect(containerPort).To(Equal(uint32(222)))
			})

			itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
				fakeContainer.NetInStub = func(uint32, uint32) (uint32, uint32, error) { time.Sleep(timeToSleep); return 0, 0, nil }
				_, _, err := container.NetIn(123, 456)
				Expect(err).ToNot(HaveOccurred())
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				_, _, err := container.NetIn(123, 456)
				return err
			})

			Context("when mapping the port fails", func() {
				BeforeEach(func() {
					fakeContainer.NetInReturns(0, 0, errors.New("oh no!"))
				})

				It("fails", func() {
					_, _, err := container.NetIn(123, 456)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("net out", func() {
			Context("when a zero-value NetOutRule is supplied", func() {
				It("permits all TCP traffic to everywhere, with logging not enabled", func() {
					Expect(container.NetOut(garden.NetOutRule{})).To(Succeed())
					rule := fakeContainer.NetOutArgsForCall(0)

					Expect(rule.Protocol).To(Equal(garden.ProtocolAll))
					Expect(rule.Networks).To(BeNil())
					Expect(rule.Ports).To(BeNil())
					Expect(rule.ICMPs).To(BeNil())
					Expect(rule.Log).To(Equal(false))
				})
			})

			Context("when protocol is specified", func() {
				Context("as TCP", func() {
					It("permits TCP traffic", func() {
						Expect(container.NetOut(garden.NetOutRule{
							Protocol: garden.ProtocolTCP,
						})).To(Succeed())
						rule := fakeContainer.NetOutArgsForCall(0)
						Expect(rule.Protocol).To(Equal(garden.ProtocolTCP))
					})
				})

				Context("as UDP", func() {
					It("permits UDP traffic", func() {
						Expect(container.NetOut(garden.NetOutRule{
							Protocol: garden.ProtocolUDP,
						})).To(Succeed())
						rule := fakeContainer.NetOutArgsForCall(0)
						Expect(rule.Protocol).To(Equal(garden.ProtocolUDP))
					})
				})

				Context("as ICMP", func() {
					It("permits ICMP traffic", func() {
						Expect(container.NetOut(garden.NetOutRule{
							Protocol: garden.ProtocolICMP,
						})).To(Succeed())
						rule := fakeContainer.NetOutArgsForCall(0)
						Expect(rule.Protocol).To(Equal(garden.ProtocolICMP))
					})
				})

				Context("as ALL", func() {
					It("permits ALL traffic", func() {
						Expect(container.NetOut(garden.NetOutRule{
							Protocol: garden.ProtocolAll,
						})).To(Succeed())

						rule := fakeContainer.NetOutArgsForCall(0)
						Expect(rule.Protocol).To(Equal(garden.ProtocolAll))
					})
				})
			})

			Context("when network is specified", func() {
				It("permits traffic to that network", func() {
					Expect(container.NetOut(garden.NetOutRule{
						Networks: []garden.IPRange{
							{Start: net.ParseIP("1.3.5.7"), End: net.ParseIP("9.9.7.6")},
						},
					})).To(Succeed())

					rule := fakeContainer.NetOutArgsForCall(0)
					Expect(rule.Networks).To(Equal([]garden.IPRange{
						{
							Start: net.ParseIP("1.3.5.7"),
							End:   net.ParseIP("9.9.7.6"),
						},
					}))
				})
			})

			Context("when multiple networks are specified", func() {
				It("permits traffic to those networks", func() {
					Expect(container.NetOut(garden.NetOutRule{
						Networks: []garden.IPRange{
							{Start: net.ParseIP("1.3.5.7"), End: net.ParseIP("9.9.7.6")},
							{Start: net.ParseIP("2.4.6.8"), End: net.ParseIP("8.6.4.2")},
						},
					})).To(Succeed())

					rule := fakeContainer.NetOutArgsForCall(0)
					Expect(rule.Networks).To(ConsistOf(
						garden.IPRange{
							Start: net.ParseIP("1.3.5.7"),
							End:   net.ParseIP("9.9.7.6"),
						},
						garden.IPRange{
							Start: net.ParseIP("2.4.6.8"),
							End:   net.ParseIP("8.6.4.2"),
						},
					))
				})
			})

			Context("when ports are specified", func() {
				It("permits traffic to those ports", func() {
					Expect(container.NetOut(garden.NetOutRule{
						Ports: []garden.PortRange{
							{Start: 4, End: 44},
						},
					})).To(Succeed())

					rule := fakeContainer.NetOutArgsForCall(0)
					Expect(rule.Ports).To(Equal([]garden.PortRange{
						{
							Start: 4,
							End:   44,
						},
					}))
				})
			})

			Context("when multiple ports are specified", func() {
				It("permits traffic to those ports", func() {
					Expect(container.NetOut(garden.NetOutRule{
						Ports: []garden.PortRange{
							{Start: 4, End: 44},
							{Start: 563, End: 3944},
						},
					})).To(Succeed())

					rule := fakeContainer.NetOutArgsForCall(0)
					Expect(rule.Ports).To(ConsistOf(
						garden.PortRange{
							Start: 4,
							End:   44,
						},
						garden.PortRange{
							Start: 563,
							End:   3944,
						},
					))
				})
			})

			Context("when icmps are specified without a code", func() {
				It("permits traffic matching those icmps", func() {
					Expect(container.NetOut(garden.NetOutRule{
						ICMPs: &garden.ICMPControl{Type: 4},
					})).To(Succeed())

					rule := fakeContainer.NetOutArgsForCall(0)
					Expect(rule.ICMPs).To(Equal(&garden.ICMPControl{Type: 4}))
				})
			})

			Context("when icmps are specified with a code", func() {
				It("permits traffic matching those icmps", func() {
					var code garden.ICMPCode = 34
					Expect(container.NetOut(garden.NetOutRule{
						ICMPs: &garden.ICMPControl{Type: 4, Code: &code},
					})).To(Succeed())

					rule := fakeContainer.NetOutArgsForCall(0)
					Expect(rule.ICMPs).To(Equal(&garden.ICMPControl{Type: 4, Code: &code}))
				})
			})

			Context("when log is true", func() {
				It("requests that the rule logs", func() {
					Expect(container.NetOut(garden.NetOutRule{Log: true})).To(Succeed())

					rule := fakeContainer.NetOutArgsForCall(0)
					Expect(rule.Log).To(BeTrue())
				})
			})

			itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
				fakeContainer.NetOutStub = func(garden.NetOutRule) error { time.Sleep(timeToSleep); return nil }
				err := container.NetOut(garden.NetOutRule{})
				Expect(err).ToNot(HaveOccurred())
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				return container.NetOut(garden.NetOutRule{})
			})

			Context("when permitting traffic fails", func() {
				BeforeEach(func() {
					fakeContainer.NetOutReturns(errors.New("oh no!"))
				})

				It("fails", func() {
					err := container.NetOut(garden.NetOutRule{})
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("bulk net out", func() {
			It("calls bulk net out with rules provided", func() {
				rules := []garden.NetOutRule{
					garden.NetOutRule{Protocol: garden.ProtocolTCP},
					garden.NetOutRule{Protocol: garden.ProtocolUDP},
				}

				Expect(container.BulkNetOut(rules)).To(Succeed())
				Expect(fakeContainer.BulkNetOutCallCount()).To(Equal(1))
				Expect(fakeContainer.BulkNetOutArgsForCall(0)).To(Equal(rules))
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				return container.BulkNetOut([]garden.NetOutRule{})
			})

			Context("when permitting traffic fails", func() {
				BeforeEach(func() {
					fakeContainer.BulkNetOutReturns(errors.New("oh no!"))
				})

				It("fails", func() {
					err := container.BulkNetOut([]garden.NetOutRule{})
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("info", func() {
			containerInfo := garden.ContainerInfo{
				State:         "active",
				Events:        []string{"oom", "party"},
				HostIP:        "host-ip",
				ContainerIP:   "container-ip",
				ExternalIP:    "external-ip",
				ContainerPath: "/path/to/container",
				ProcessIDs:    []string{"process-handle-1", "process-handle-2"},
				Properties: garden.Properties{
					"foo": "bar",
					"a":   "b",
				},
				MappedPorts: []garden.PortMapping{
					{HostPort: 1234, ContainerPort: 5678},
					{HostPort: 1235, ContainerPort: 5679},
				},
			}

			It("reports information about the container", func() {
				fakeContainer.InfoReturns(containerInfo, nil)

				info, err := container.Info()
				Expect(err).ToNot(HaveOccurred())

				Expect(info).To(Equal(containerInfo))
			})

			itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
				fakeContainer.InfoStub = func() (garden.ContainerInfo, error) { time.Sleep(timeToSleep); return garden.ContainerInfo{}, nil }
				_, err := container.Info()
				Expect(err).ToNot(HaveOccurred())
			})

			itFailsWhenTheContainerIsNotFound(func() error {
				_, err := container.Info()
				return err
			})

			Context("when getting container info fails", func() {
				BeforeEach(func() {
					fakeContainer.InfoReturns(garden.ContainerInfo{}, errors.New("oh no!"))
				})

				It("fails", func() {
					_, err := container.Info()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("BulkInfo", func() {

			handles := []string{"handle1", "handle2"}

			expectedBulkInfo := map[string]garden.ContainerInfoEntry{
				"handle1": garden.ContainerInfoEntry{
					Info: garden.ContainerInfo{
						State: "container1state",
					},
				},
				"handle2": garden.ContainerInfoEntry{
					Info: garden.ContainerInfo{
						State: "container2state",
					},
				},
			}

			It("calls BulkInfo with empty slice when handles is empty", func() {
				handles = nil
				serverBackend.BulkInfoReturns(nil, nil)
				apiClient.BulkInfo(handles)
				Expect(serverBackend.BulkInfoArgsForCall(0)).To(BeEmpty())
			})

			It("reports information about containers by list of handles", func() {
				serverBackend.BulkInfoReturns(expectedBulkInfo, nil)

				bulkInfo, err := apiClient.BulkInfo(handles)
				Expect(err).ToNot(HaveOccurred())
				Expect(bulkInfo).To(Equal(expectedBulkInfo))
			})

			Context("when retrieving bulk info fails", func() {
				It("returns the error", func() {
					serverBackend.BulkInfoReturns(
						make(map[string]garden.ContainerInfoEntry),
						errors.New("Oh noes!"),
					)

					_, err := apiClient.BulkInfo(handles)
					Expect(err).To(MatchError("Oh noes!"))
				})
			})

			Context("when a container is in error state", func() {
				It("return single container error", func() {

					expectedBulkInfo := map[string]garden.ContainerInfoEntry{
						"error": garden.ContainerInfoEntry{
							Err: &garden.Error{Err: errors.New("Oopps!")},
						},
						"success": garden.ContainerInfoEntry{
							Info: garden.ContainerInfo{
								State: "container2state",
							},
						},
					}

					serverBackend.BulkInfoReturns(expectedBulkInfo, nil)

					bulkInfo, err := apiClient.BulkInfo(handles)
					Expect(err).ToNot(HaveOccurred())
					Expect(bulkInfo).To(Equal(expectedBulkInfo))
				})
			})
		})

		Describe("BulkMetrics", func() {

			handles := []string{"handle1", "handle2"}

			expectedBulkMetrics := map[string]garden.ContainerMetricsEntry{
				"handle1": garden.ContainerMetricsEntry{
					Metrics: garden.Metrics{
						DiskStat: garden.ContainerDiskStat{
							TotalInodesUsed: 1,
						},
					},
				},
				"handle2": garden.ContainerMetricsEntry{
					Metrics: garden.Metrics{
						DiskStat: garden.ContainerDiskStat{
							TotalInodesUsed: 2,
						},
					},
				},
			}

			It("reports information about containers by list of handles", func() {
				serverBackend.BulkMetricsReturns(expectedBulkMetrics, nil)

				bulkMetrics, err := apiClient.BulkMetrics(handles)
				Expect(err).ToNot(HaveOccurred())
				Expect(bulkMetrics).To(Equal(expectedBulkMetrics))
			})

			It("calls BulkMetrics with empty slice when handles is empty", func() {
				handles = nil
				serverBackend.BulkMetricsReturns(nil, nil)
				apiClient.BulkMetrics(handles)
				Expect(serverBackend.BulkMetricsArgsForCall(0)).To(BeEmpty())
			})

			Context("when retrieving bulk info fails", func() {
				It("returns the error", func() {
					serverBackend.BulkMetricsReturns(
						make(map[string]garden.ContainerMetricsEntry),
						errors.New("Oh noes!"),
					)

					_, err := apiClient.BulkMetrics(handles)
					Expect(err).To(MatchError("Oh noes!"))
				})
			})

			Context("when a container has an error", func() {
				It("returns the error for the container", func() {

					errorBulkMetrics := map[string]garden.ContainerMetricsEntry{
						"error": garden.ContainerMetricsEntry{
							Err: &garden.Error{Err: errors.New("Oh noes!")},
						},
						"success": garden.ContainerMetricsEntry{
							Metrics: garden.Metrics{
								DiskStat: garden.ContainerDiskStat{
									TotalInodesUsed: 1,
								},
							},
						},
					}

					serverBackend.BulkMetricsReturns(errorBulkMetrics, nil)

					bulkMetrics, err := apiClient.BulkMetrics(handles)
					Expect(err).ToNot(HaveOccurred())
					Expect(bulkMetrics).To(Equal(errorBulkMetrics))
				})
			})
		})

		Describe("attaching", func() {
			Context("when attaching succeeds", func() {
				BeforeEach(func() {
					fakeContainer.AttachStub = func(processID string, io garden.ProcessIO) (garden.Process, error) {
						writing := new(sync.WaitGroup)
						writing.Add(1)

						go func() {
							defer writing.Done()
							defer GinkgoRecover()

							_, err := fmt.Fprintf(io.Stdout, "stdout data")
							Expect(err).ToNot(HaveOccurred())

							in, err := ioutil.ReadAll(io.Stdin)
							Expect(err).ToNot(HaveOccurred())

							_, err = fmt.Fprintf(io.Stdout, "mirrored %s", string(in))
							Expect(err).ToNot(HaveOccurred())

							_, err = fmt.Fprintf(io.Stderr, "stderr data")
							Expect(err).ToNot(HaveOccurred())
						}()

						process := new(fakes.FakeProcess)

						process.IDReturns("process-handle")

						process.WaitStub = func() (int, error) {
							writing.Wait()
							return 123, nil
						}

						return process, nil
					}
				})

				It("responds with a ProcessPayload for every chunk", func() {
					stdout := gbytes.NewBuffer()
					stderr := gbytes.NewBuffer()

					processIO := garden.ProcessIO{
						Stdin:  bytes.NewBufferString("stdin data"),
						Stdout: stdout,
						Stderr: stderr,
					}

					process, err := container.Attach("process-handle", processIO)
					Expect(err).ToNot(HaveOccurred())

					pid, _ := fakeContainer.AttachArgsForCall(0)
					Expect(pid).To(Equal("process-handle"))

					Eventually(stdout).Should(gbytes.Say("stdout data"))
					Eventually(stdout).Should(gbytes.Say("mirrored stdin data"))
					Eventually(stderr).Should(gbytes.Say("stderr data"))

					status, err := process.Wait()
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(123))
				})

				itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
					fakeContainer.AttachStub = func(string, garden.ProcessIO) (garden.Process, error) {
						time.Sleep(timeToSleep)
						return nil, errors.New("blam")
					}
					container.Attach("process-handle", garden.ProcessIO{})
				})
			})

			Context("when the container is not found", func() {
				It("fails", func() {
					serverBackend.LookupReturns(nil, errors.New("not found"))
					_, err := container.Attach("process-handle", garden.ProcessIO{})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when waiting on the process fails server-side", func() {
				BeforeEach(func() {
					fakeContainer.AttachStub = func(id string, io garden.ProcessIO) (garden.Process, error) {
						process := new(fakes.FakeProcess)

						process.IDReturns("process-handle")
						process.WaitReturns(0, errors.New("oh no!"))

						return process, nil
					}
				})

				It("bubbles the error up", func() {
					process, err := container.Attach("process-handle", garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					_, err = process.Wait()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("oh no!"))
				})
			})

			Context("when the process is not found", func() {
				BeforeEach(func() {
					fakeContainer.AttachReturns(nil, garden.ProcessNotFoundError{
						ProcessID: "not-existing-process"},
					)
				})

				It("returns a ProcessNotFoundError", func() {
					_, err := container.Attach("not-existing-process", garden.ProcessIO{})
					Expect(err).To(MatchError(garden.ProcessNotFoundError{
						ProcessID: "not-existing-process",
					}))
				})
			})

			Context("when attaching fails", func() {
				BeforeEach(func() {
					fakeContainer.AttachReturns(nil, errors.New("oh no!"))
				})

				It("fails", func() {
					_, err := container.Attach("process-handle", garden.ProcessIO{})
					Expect(err).To(HaveOccurred())
				})

				It("closes the stdin writer", func() {
					container.Attach("process-handle", garden.ProcessIO{})

					_, processIO := fakeContainer.AttachArgsForCall(0)
					_, err := processIO.Stdin.Read([]byte{})
					Expect(err).To(Equal(io.EOF))
				})
			})
		})

		Describe("running", func() {
			processSpec := garden.ProcessSpec{
				ID:   "some-process-id",
				Path: "/some/script",
				Args: []string{"arg1", "arg2"},
				Dir:  "/some/dir",
				Env: []string{
					"FLAVOR=chocolate",
					"TOPPINGS=sprinkles",
				},
				User: "root",
				Limits: garden.ResourceLimits{
					As:         uint64ptr(1),
					Core:       uint64ptr(2),
					Cpu:        uint64ptr(3),
					Data:       uint64ptr(4),
					Fsize:      uint64ptr(5),
					Locks:      uint64ptr(6),
					Memlock:    uint64ptr(7),
					Msgqueue:   uint64ptr(8),
					Nice:       uint64ptr(9),
					Nofile:     uint64ptr(10),
					Nproc:      uint64ptr(11),
					Rss:        uint64ptr(12),
					Rtprio:     uint64ptr(13),
					Sigpending: uint64ptr(14),
					Stack:      uint64ptr(15),
				},
				TTY: &garden.TTYSpec{
					WindowSize: &garden.WindowSize{
						Columns: 80,
						Rows:    24,
					},
				},
				Image: garden.ImageRef{
					URI:      "some-uri",
					Username: "registry-username",
					Password: "registry-password",
				},
			}

			Context("when running succeeds", func() {
				BeforeEach(func() {
					fakeContainer.RunStub = func(spec garden.ProcessSpec, io garden.ProcessIO) (garden.Process, error) {
						writing := new(sync.WaitGroup)
						writing.Add(1)

						go func() {
							defer writing.Done()
							defer GinkgoRecover()

							_, err := fmt.Fprintf(io.Stdout, "stdout data")
							Expect(err).ToNot(HaveOccurred())

							in, err := ioutil.ReadAll(io.Stdin)
							Expect(err).ToNot(HaveOccurred())

							_, err = fmt.Fprintf(io.Stdout, "mirrored %s", string(in))
							Expect(err).ToNot(HaveOccurred())

							_, err = fmt.Fprintf(io.Stderr, "stderr data")
							Expect(err).ToNot(HaveOccurred())
						}()

						process := new(fakes.FakeProcess)

						process.IDReturns("process-handle")

						process.WaitStub = func() (int, error) {
							writing.Wait()
							return 123, nil
						}

						return process, nil
					}
				})

				It("should not log any environment variables and command line args", func() {
					process, err := container.Run(garden.ProcessSpec{
						User: "alice",
						Path: "echo",
						Args: []string{"-username", "banana"},
						Env:  []string{"PASSWORD=MY_SECRET"},
					}, garden.ProcessIO{
						Stdin:  bytes.NewBufferString("stdin data"),
						Stdout: GinkgoWriter,
						Stderr: GinkgoWriter,
					})
					Expect(err).ToNot(HaveOccurred())

					exitCode, err := process.Wait()
					Expect(exitCode).To(Equal(123))
					Expect(err).ToNot(HaveOccurred())

					buffer := sink.Buffer()

					Expect(buffer).ToNot(gbytes.Say("PASSWORD"))
					Expect(buffer).ToNot(gbytes.Say("MY_SECRET"))
					Expect(buffer).ToNot(gbytes.Say("-username"))
					Expect(buffer).ToNot(gbytes.Say("banana"))
				})

				It("runs the process and streams the output", func() {
					stdout := gbytes.NewBuffer()
					stderr := gbytes.NewBuffer()

					processIO := garden.ProcessIO{
						Stdin:  bytes.NewBufferString("stdin data"),
						Stdout: stdout,
						Stderr: stderr,
					}

					process, err := container.Run(processSpec, processIO)
					Expect(err).ToNot(HaveOccurred())

					ranSpec, processIO := fakeContainer.RunArgsForCall(0)
					Expect(ranSpec).To(Equal(processSpec))

					Eventually(stdout).Should(gbytes.Say("stdout data"))
					Eventually(stdout).Should(gbytes.Say("mirrored stdin data"))
					Eventually(stderr).Should(gbytes.Say("stderr data"))

					status, err := process.Wait()
					Expect(err).ToNot(HaveOccurred())
					Expect(status).To(Equal(123))

					_, err = processIO.Stdin.Read([]byte{})
					Expect(err).To(Equal(io.EOF))
				})

				itResetsGraceTimeWhenHandling(func(timeToSleep time.Duration) {
					fakeContainer.RunStub = func(garden.ProcessSpec, garden.ProcessIO) (garden.Process, error) {
						time.Sleep(timeToSleep)
						return nil, errors.New("boom")
					}

					container.Run(processSpec, garden.ProcessIO{})
				})
			})

			Context("when the backend returns an error", func() {
				It("returns the error message", func() {
					fakeContainer.RunReturns(nil, errors.New("o no!"))
					_, err := container.Run(garden.ProcessSpec{}, garden.ProcessIO{})
					Expect(err).To(MatchError(ContainSubstring("o no!")))
				})
			})

			Describe("when the server is shut down while there is a process running", func() {
				BeforeEach(func() {
					fakeContainer.RunStub = func(spec garden.ProcessSpec, io garden.ProcessIO) (garden.Process, error) {
						writing := new(sync.WaitGroup)
						writing.Add(1)

						go func() {
							defer writing.Done()
							defer GinkgoRecover()

							for {
								_, err := fmt.Fprintf(io.Stdout, "stdout data")
								Expect(err).ToNot(HaveOccurred())
							}
						}()

						process := new(fakes.FakeProcess)
						process.IDReturns("process-handle")

						process.WaitStub = func() (int, error) {
							select {}
						}

						return process, nil
					}
				})

				It("does not close the process's stdin", func() {
					pipeR, _ := io.Pipe()
					_, err := container.Run(processSpec, garden.ProcessIO{Stdin: pipeR})
					Expect(err).ToNot(HaveOccurred())

					apiServer.Stop()
					isRunning = false

					_, processIO := fakeContainer.RunArgsForCall(0)

					readExited := make(chan struct{})
					go func() {
						_, err = ioutil.ReadAll(processIO.Stdin)
						close(readExited)
					}()

					Eventually(readExited).Should(BeClosed())
					Expect(err).To(HaveOccurred())
					Expect(err).ToNot(Equal(io.EOF))
				})
			})

			Context("when the process is killed", func() {
				var fakeProcess *fakes.FakeProcess

				BeforeEach(func() {
					fakeProcess = new(fakes.FakeProcess)
					fakeProcess.IDReturns("process-handle")
					fakeProcess.WaitStub = func() (int, error) {
						select {}
					}

					fakeContainer.RunReturns(fakeProcess, nil)
				})

				It("is eventually killed in the backend", func() {
					process, err := container.Run(processSpec, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					err = process.Signal(garden.SignalKill)
					Expect(err).ToNot(HaveOccurred())

					Eventually(fakeProcess.SignalCallCount).Should(Equal(1))
					Expect(fakeProcess.SignalArgsForCall(0)).To(Equal(garden.SignalKill))
				})
			})

			Context("when the process is terminated", func() {
				var fakeProcess *fakes.FakeProcess

				BeforeEach(func() {
					fakeProcess = new(fakes.FakeProcess)
					fakeProcess.IDReturns("process-handle")
					fakeProcess.WaitStub = func() (int, error) {
						select {}
					}

					fakeContainer.RunReturns(fakeProcess, nil)
				})

				It("is eventually terminated in the backend", func() {
					process, err := container.Run(processSpec, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					err = process.Signal(garden.SignalTerminate)
					Expect(err).ToNot(HaveOccurred())

					Eventually(fakeProcess.SignalCallCount).Should(Equal(1))
					Expect(fakeProcess.SignalArgsForCall(0)).To(Equal(garden.SignalTerminate))
				})
			})

			Context("when the process's window size is set", func() {
				var fakeProcess *fakes.FakeProcess

				BeforeEach(func() {
					fakeProcess = new(fakes.FakeProcess)
					fakeProcess.IDReturns("process-handle")
					fakeProcess.WaitStub = func() (int, error) {
						select {}
					}

					fakeContainer.RunReturns(fakeProcess, nil)
				})

				It("is eventually set in the backend", func() {
					process, err := container.Run(processSpec, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					ttySpec := garden.TTYSpec{
						WindowSize: &garden.WindowSize{
							Columns: 80,
							Rows:    24,
						},
					}

					err = process.SetTTY(ttySpec)
					Expect(err).ToNot(HaveOccurred())

					Eventually(fakeProcess.SetTTYCallCount).Should(Equal(1))

					Expect(fakeProcess.SetTTYArgsForCall(0)).To(Equal(ttySpec))
				})
			})

			Context("when waiting on the process fails server-side", func() {
				BeforeEach(func() {
					fakeContainer.RunStub = func(spec garden.ProcessSpec, io garden.ProcessIO) (garden.Process, error) {
						process := new(fakes.FakeProcess)

						process.IDReturns("process-handle")
						process.WaitReturns(0, errors.New("oh no!"))

						return process, nil
					}
				})

				It("bubbles the error up", func() {
					process, err := container.Run(processSpec, garden.ProcessIO{})
					Expect(err).ToNot(HaveOccurred())

					_, err = process.Wait()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("oh no!"))
				})
			})

			Context("when the container is not found", func() {
				It("fails", func() {
					serverBackend.LookupReturns(nil, errors.New("not found"))
					_, err := container.Run(processSpec, garden.ProcessIO{})
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when running fails", func() {
				BeforeEach(func() {
					fakeContainer.RunReturns(nil, errors.New("oh no!"))
				})

				It("fails", func() {
					_, err := container.Run(processSpec, garden.ProcessIO{})
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})

type closeChecker struct {
	closed bool
	sync.Mutex
}

func (checker *closeChecker) Read(b []byte) (int, error) {
	b[0] = 'x'
	return 1, nil
}

func (checker *closeChecker) Close() error {
	checker.Lock()
	checker.closed = true
	checker.Unlock()
	return nil
}

func (checker *closeChecker) Closed() bool {
	checker.Lock()
	defer checker.Unlock()
	return checker.closed
}
