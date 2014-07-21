package fakes

import (
	bwcaction "bosh-warden-cpi/action"
)

type FakeCaller struct {
	CallAction bwcaction.Action
	CallArgs   []interface{}
	CallResult interface{}
	CallErr    error
}

func (caller *FakeCaller) Call(action bwcaction.Action, args []interface{}) (interface{}, error) {
	caller.CallAction = action
	caller.CallArgs = args
	return caller.CallResult, caller.CallErr
}
