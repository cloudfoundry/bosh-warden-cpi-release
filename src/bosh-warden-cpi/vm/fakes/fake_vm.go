package fakes

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"

	bwcdisk "bosh-warden-cpi/disk"
)

type FakeVM struct {
	id apiv1.VMCID

	DeleteCalled bool
	DeleteErr    error

	AttachDiskDisk bwcdisk.Disk
	AttachDiskErr  error

	DetachDiskDisk bwcdisk.Disk
	DetachDiskErr  error
}

func NewFakeVM(id apiv1.VMCID) *FakeVM {
	return &FakeVM{id: id}
}

func (vm FakeVM) ID() apiv1.VMCID { return vm.id }

func (vm *FakeVM) Delete() error {
	vm.DeleteCalled = true
	return vm.DeleteErr
}

func (vm *FakeVM) AttachDisk(disk bwcdisk.Disk) error {
	vm.AttachDiskDisk = disk
	return vm.AttachDiskErr
}

func (vm *FakeVM) DetachDisk(disk bwcdisk.Disk) error {
	vm.DetachDiskDisk = disk
	return vm.DetachDiskErr
}
