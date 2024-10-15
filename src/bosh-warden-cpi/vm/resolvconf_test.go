package vm_test

import (
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "bosh-warden-cpi/vm"
)

var _ = Describe("ResolvConf", func() {
	var (
		fs *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
	})

	Describe("NewSystemResolvConfFromPath", func() {
		examples := map[string][]string{
			"\n\n\n": []string{},

			`
nameserver 10.0.2.3`: []string{"10.0.2.3"},

			`
nameserver 10.0.2.3
nameserver 10.0.2.3`: []string{"10.0.2.3", "10.0.2.3"},

			`
nameserver 10.0.2.3
search domain.com
nameserver 8.8.8.8`: []string{"10.0.2.3", "8.8.8.8"},

			`# Generated by bosh-agent
nameserver 10.0.2.3
# Some other comment
nameserver 8.8.8.8`: []string{"10.0.2.3", "8.8.8.8"},

			`
      nameserver       10.0.2.3
  nameserver 8.8.8.8        `: []string{"10.0.2.3", "8.8.8.8"},
		}

		for content, servers := range examples {
			It("parses /etc/resolv.conf", func() {
				fs.WriteFileString("/etc/resolv.conf", content)

				resolvConf, err := NewSystemResolvConfFromPath(fs)
				Expect(err).ToNot(HaveOccurred())

				Expect(resolvConf.Nameservers).To(Equal(servers))
			})
		}

		It("returns error if /etc/resolv.conf cannot be read", func() {
			_, err := NewSystemResolvConfFromPath(fs)
			Expect(err).To(HaveOccurred())
		})
	})
})
