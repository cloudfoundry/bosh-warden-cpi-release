package fakes

import (
	bwcvm "bosh-warden-cpi/vm"
)

type FakeAgentEnvService struct {
	FetchCalled   bool
	FetchAgentEnv bwcvm.AgentEnv
	FetchErr      error

	UpdateAgentEnv bwcvm.AgentEnv
	UpdateErr      error
}

func (s *FakeAgentEnvService) Fetch() (bwcvm.AgentEnv, error) {
	s.FetchCalled = true
	return s.FetchAgentEnv, s.FetchErr
}

func (s *FakeAgentEnvService) Update(agentEnv bwcvm.AgentEnv) error {
	s.UpdateAgentEnv = agentEnv
	return s.UpdateErr
}
