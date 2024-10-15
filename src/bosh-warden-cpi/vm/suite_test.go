package vm_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "vm")
}

type NonJSONMarshable struct{}

func (m NonJSONMarshable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("fake-marshal-err")
}

type FailingWriteCloser struct {
	WriteErr error
	CloseErr error
}

func (wc FailingWriteCloser) Write(data []byte) (int, error) { return len(data), wc.WriteErr }
func (wc FailingWriteCloser) Close() error                   { return wc.CloseErr }
