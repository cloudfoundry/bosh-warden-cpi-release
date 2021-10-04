package fakes

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type FakeAgentEnvService struct {
	FetchCalled   bool
	FetchAgentEnv apiv1.AgentEnv
	FetchErr      error

	UpdateAgentEnv apiv1.AgentEnv
	UpdateErr      error
}

func (s *FakeAgentEnvService) Fetch() (apiv1.AgentEnv, error) {
	s.FetchCalled = true
	return s.FetchAgentEnv, s.FetchErr
}

func (s *FakeAgentEnvService) Update(agentEnv apiv1.AgentEnv) error {
	s.UpdateAgentEnv = agentEnv
	return s.UpdateErr
}
