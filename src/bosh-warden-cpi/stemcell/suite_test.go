package stemcell_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestReg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "stemcell")
}
