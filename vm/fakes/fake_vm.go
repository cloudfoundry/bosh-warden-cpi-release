package fakes

import (
	bwcdisk "bosh-warden-cpi/disk"
)

type FakeVM struct {
	id string

	DeleteCalled bool
	DeleteErr    error

	AttachDiskDisk bwcdisk.Disk
	AttachDiskErr  error

	DetachDiskDisk bwcdisk.Disk
	DetachDiskErr  error
}

func NewFakeVM(id string) *FakeVM {
	return &FakeVM{id: id}
}

func (vm FakeVM) ID() string { return vm.id }

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
