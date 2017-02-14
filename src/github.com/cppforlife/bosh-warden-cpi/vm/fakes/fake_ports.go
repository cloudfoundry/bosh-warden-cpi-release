package fakes

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcvm "github.com/cppforlife/bosh-warden-cpi/vm"
)

type FakePorts struct{}

func (f FakePorts) Forward(apiv1.VMCID, string, []bwcvm.PortMapping) error { return nil }
func (f FakePorts) RemoveForwarded(apiv1.VMCID) error                      { return nil }
