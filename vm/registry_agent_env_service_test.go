package vm_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"

	. "github.com/cppforlife/bosh-warden-cpi/vm"
)

var _ = Describe("RegistryAgentEnvService", func() {
	var (
		logger               boshlog.Logger
		agentEnvService      AgentEnvService
		registryServer       *registryServer
		expectedAgentEnv     AgentEnv
		expectedAgentEnvJSON []byte
	)

	BeforeEach(func() {
		registryOptions := RegistryOptions{
			Host:     "127.0.0.1",
			Port:     6307,
			Username: "fake-username",
			Password: "fake-password",
		}
		registryServer = NewRegistryServer(registryOptions)
		readyCh := make(chan struct{})
		go registryServer.Start(readyCh)
		<-readyCh

		instanceID := "fake-instance-id"
		logger = boshlog.NewLogger(boshlog.LevelNone)
		agentEnvService = NewRegistryAgentEnvService(registryOptions, instanceID, logger)

		expectedAgentEnv = AgentEnv{AgentID: "fake-agent-id"}
		var err error
		expectedAgentEnvJSON, err = json.Marshal(expectedAgentEnv)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		registryServer.Stop()
	})

	Describe("Fetch", func() {
		Context("when settings for the instance exist in the registry", func() {
			BeforeEach(func() {
				registryServer.InstanceSettings = expectedAgentEnvJSON
			})

			It("fetches settings from the registry", func() {
				agentEnv, err := agentEnvService.Fetch()
				Expect(err).ToNot(HaveOccurred())
				Expect(agentEnv).To(Equal(expectedAgentEnv))
			})
		})

		Context("when settings for instance do not exist", func() {
			It("returns an error", func() {
				agentEnv, err := agentEnvService.Fetch()
				Expect(err).To(HaveOccurred())
				Expect(agentEnv).To(Equal(AgentEnv{}))
			})
		})
	})

	Describe("Update", func() {
		It("updates settings in the registry", func() {
			Expect(registryServer.InstanceSettings).To(Equal([]byte{}))
			err := agentEnvService.Update(expectedAgentEnv)
			Expect(err).ToNot(HaveOccurred())
			Expect(registryServer.InstanceSettings).To(Equal(expectedAgentEnvJSON))
		})
	})
})

type registryServer struct {
	InstanceSettings []byte
	options          RegistryOptions
	listener         net.Listener
}

func NewRegistryServer(options RegistryOptions) *registryServer {
	return &registryServer{
		InstanceSettings: []byte{},
		options:          options,
	}
}

func (s *registryServer) Start(readyCh chan struct{}) error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", s.options.Host, s.options.Port))
	if err != nil {
		return err
	}

	readyCh <- struct{}{}

	httpServer := http.Server{}
	mux := http.NewServeMux()
	httpServer.Handler = mux
	mux.HandleFunc("/instances/fake-instance-id/settings", s.instanceHandler)

	return httpServer.Serve(s.listener)
}

func (s *registryServer) Stop() error {
	// if client keeps connection alive, server will still be running
	s.InstanceSettings = nil

	err := s.listener.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *registryServer) instanceHandler(w http.ResponseWriter, req *http.Request) {
	if !s.isAuthorized(req) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if req.Method == "GET" {
		if s.InstanceSettings != nil {
			w.Write(s.InstanceSettings)
			return
		}
		http.NotFound(w, req)
		return
	}

	if req.Method == "PUT" {
		reqBody, _ := ioutil.ReadAll(req.Body)
		s.InstanceSettings = reqBody

		w.WriteHeader(http.StatusOK)
		return
	}
}

func (s *registryServer) isAuthorized(req *http.Request) bool {
	auth := s.options.Username + ":" + s.options.Password
	expectedAuthorizationHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	return expectedAuthorizationHeader == req.Header.Get("Authorization")
}
