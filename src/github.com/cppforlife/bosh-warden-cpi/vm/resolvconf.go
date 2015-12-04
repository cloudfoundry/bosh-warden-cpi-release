package vm

// Adapted from https://github.com/docker/libnetwork/blob/master/resolvconf/resolvconf.go

import (
	gobytes "bytes"
	"regexp"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

var (
	ipv4NumBlock = `(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`
	ipv4Address  = `(` + ipv4NumBlock + `\.){3}` + ipv4NumBlock
	// This is not an IPv6 address verifier as it will accept a super-set of IPv6, and also
	// will *not match* IPv4-Embedded IPv6 Addresses (RFC6052), but that and other variants
	// -- e.g. other link-local types -- either won't work in containers or are unnecessary.
	// For readability and sufficiency for Docker purposes this seemed more reasonable than a
	// 1000+ character regexp with exact and complete IPv6 validation
	ipv6Address = `([0-9A-Fa-f]{0,4}:){2,7}([0-9A-Fa-f]{0,4})`
	nsRegexp    = regexp.MustCompile(`^\s*nameserver\s*((` + ipv4Address + `)|(` + ipv6Address + `))\s*$`)
)

type ResolvConf struct {
	Nameservers []string
}

func NewSystemResolvConfFromPath(fs boshsys.FileSystem) (ResolvConf, error) {
	var conf ResolvConf

	bytes, err := fs.ReadFile("/etc/resolv.conf")
	if err != nil {
		return conf, bosherr.WrapErrorf(err, "Reading resolv.conf")
	}

	conf.Nameservers = resolvConfGetNameservers(resolvConfGetLines(bytes))

	return conf, nil
}

func resolvConfGetNameservers(lines [][]byte) []string {
	nameservers := []string{}

	for _, line := range lines {
		ns := nsRegexp.FindSubmatch(line)
		if len(ns) > 0 {
			nameservers = append(nameservers, string(ns[1]))
		}
	}

	return nameservers
}

func resolvConfGetLines(input []byte) [][]byte {
	output := [][]byte{}
	lines := gobytes.Split(input, []byte("\n"))

	for _, currentLine := range lines {
		var commentIndex = gobytes.Index(currentLine, []byte("#"))
		if commentIndex == -1 {
			output = append(output, currentLine)
		} else {
			output = append(output, currentLine[:commentIndex])
		}
	}

	return output
}
