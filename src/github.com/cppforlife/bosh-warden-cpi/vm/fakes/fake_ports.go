package fakes

import (
	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FakePorts struct{}

func (f FakePorts) Forward(string, string, []bwcvm.VMPropsPort) error { return nil }
func (f FakePorts) RemoveForwarded(string) error                      { return nil }
