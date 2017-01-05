package vm

import (
	"net"
	"strconv"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type IPTablesPorts struct {
	cmdRunner boshsys.CmdRunner
}

func NewIPTablesPorts(cmdRunner boshsys.CmdRunner) IPTablesPorts {
	return IPTablesPorts{cmdRunner}
}

func (p IPTablesPorts) Forward(id, containerIP string, mappings []PortMapping) error {
	for _, mapping := range mappings {
		forwardArgs := []string{
			"PREROUTING",
			"-p", mapping.Protocol(),
			"!", "-i", "w+", // non-warden interfaces // todo wont work if cpi is nested
			"--dport", p.fmtPortRange(mapping.Host(), ":"),
			"-j", "DNAT", "--to", net.JoinHostPort(containerIP, p.fmtPortRange(mapping.Container(), "-")),
			"-m", "comment", "--comment", p.comment(id),
		}

		_, _, _, err := p.runCmd("-A", forwardArgs)
		if err != nil {
			p.removeRulesWithID(id)
			return bosherr.WrapErrorf(err, "Forwarding host port(s) '%s'", mapping.Host())
		}
	}

	return nil
}

func (p IPTablesPorts) RemoveForwarded(id string) error {
	return p.removeRulesWithID(id)
}

func (p IPTablesPorts) removeRulesWithID(id string) error {
	stdout, _, _, err := p.cmdRunner.RunCommand("iptables-save", "-t", "nat")
	if err != nil {
		return bosherr.WrapErrorf(err, "Listing nat table rules to remove rules")
	}

	var lastErr error

	for _, line := range strings.Split(stdout, "\n") {
		if !strings.Contains(line, p.comment(id)) {
			continue
		}

		forwardArgs := strings.Split(line, " ")[1:] // skip -A

		_, stderr, _, err := p.runCmd("-D", forwardArgs)
		if err != nil {
			// Ignore missing rule errors as there may be race conditions (though unlikely)
			if !strings.Contains(stderr, "No chain/target/match by that name") {
				lastErr = err
			}
		}
	}

	return lastErr
}

func (IPTablesPorts) comment(id string) string { return "bosh-warden-cpi-" + id }

func (IPTablesPorts) fmtPortRange(portRange PortRange, delim string) string {
	if portRange.Len() > 1 {
		return strconv.Itoa(portRange.Start()) + delim + strconv.Itoa(portRange.End())
	}
	return strconv.Itoa(portRange.Start())
}

func (p IPTablesPorts) runCmd(action string, args []string) (string, string, int, error) {
	globalArgs := []string{"-w", "-t", "nat", action}
	globalArgs = append(globalArgs, args...)
	return p.cmdRunner.RunCommand("iptables", globalArgs...)
}
