package vm_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bosh-warden-cpi/vm"
)

type portRangeTest struct {
	String   string
	Error    string
	Expected vm.PortRange
}

func MustPortRange(start, end int) vm.PortRange {
	r, err := vm.NewPortRange(start, end)
	if err != nil {
		panic(err)
	}
	return r
}

var _ = Describe("NewPortMapping", func() {
	It("returns error if host/container ranges dont have same len", func() {
		_, err := vm.NewPortMapping(MustPortRange(1, 1), MustPortRange(1, 2), "tcp")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Host and container port ranges must have same length"))
	})

	It("returns error if host/container ranges are not the same (only if range len > 1)", func() {
		_, err := vm.NewPortMapping(MustPortRange(1, 2), MustPortRange(5, 6), "tcp")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Host and container port ranges must be same"))

		_, err = vm.NewPortMapping(MustPortRange(2, 2), MustPortRange(4, 4), "tcp")
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns error if host/container ranges len > 1 and protocol isnt udp or tcp", func() {
		_, err := vm.NewPortMapping(MustPortRange(1, 2), MustPortRange(1, 2), "other")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Port ranges can only be used with tcp or udp protocol"))

		_, err = vm.NewPortMapping(MustPortRange(1, 2), MustPortRange(1, 2), "tcp")
		Expect(err).ToNot(HaveOccurred())

		_, err = vm.NewPortMapping(MustPortRange(1, 2), MustPortRange(1, 2), "udp")
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns error if protocol is empty", func() {
		_, err := vm.NewPortMapping(MustPortRange(1, 1), MustPortRange(1, 1), "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Protocol must be specified"))
	})

	It("succeeds", func() {
		mapping, err := vm.NewPortMapping(MustPortRange(1, 1), MustPortRange(2, 2), "tcp")
		Expect(err).ToNot(HaveOccurred())
		Expect(mapping.Host()).To(Equal(MustPortRange(1, 1)))
		Expect(mapping.Container()).To(Equal(MustPortRange(2, 2)))
		Expect(mapping.Protocol()).To(Equal("tcp"))
	})
})

var _ = Describe("NewPortRangeFromString", func() {
	tests := []portRangeTest{
		{
			String: "0",
			Error:  "Port range must match",
		},
		{
			String: "0-0",
			Error:  "Port range must match",
		},
		{
			String:   "1",
			Expected: MustPortRange(1, 1),
		},
		{
			String:   "10",
			Expected: MustPortRange(10, 10),
		},
		{
			String: "10+10",
			Error:  "Port range must match",
		},
		{
			String:   "10-10",
			Expected: MustPortRange(10, 10),
		},
		{
			String:   "10-12",
			Expected: MustPortRange(10, 12),
		},
		{
			String:   "10:12",
			Expected: MustPortRange(10, 12),
		},
		{
			String: "10     :	12",
			Expected: MustPortRange(10, 12),
		},
		{
			String: "10+12",
			Error:  "Port range must match",
		},
		{
			String: "65536",
			Error:  "Port range start must be > 0 and <= 65535",
		},
		{
			String: "1-65536",
			Error:  "Port range end must be > 0 and <= 65535",
		},
		{
			String: "12-10",
			Error:  "Port range start must be <= end",
		},
	}

	for _, test := range tests {
		test := test // copy
		It(fmt.Sprintf("works or fails with '%s'", test.String), func() {
			r, err := vm.NewPortRangeFromString(test.String)
			if len(test.Error) > 0 {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(test.Error))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(r.Same(test.Expected)).To(BeTrue())
			}
		})
	}
})

var _ = Describe("PortRange", func() {
	Describe("Len", func() {
		It("returns lenght including start and end", func() {
			Expect(MustPortRange(5, 5).Len()).To(Equal(1))
			Expect(MustPortRange(5, 6).Len()).To(Equal(2))
			Expect(MustPortRange(5, 10).Len()).To(Equal(6))
		})
	})

	Describe("Same", func() {
		It("returns true if range is exactly same", func() {
			Expect(MustPortRange(5, 5).Same(MustPortRange(5, 5))).To(BeTrue())
			Expect(MustPortRange(5, 10).Same(MustPortRange(5, 10))).To(BeTrue())
			Expect(MustPortRange(5, 10).Same(MustPortRange(5, 11))).To(BeFalse())
			Expect(MustPortRange(5, 10).Same(MustPortRange(4, 10))).To(BeFalse())
			Expect(MustPortRange(5, 10).Same(MustPortRange(50, 60))).To(BeFalse())
		})
	})
})
