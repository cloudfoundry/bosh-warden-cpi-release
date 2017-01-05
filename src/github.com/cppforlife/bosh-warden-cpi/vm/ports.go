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

func (p IPTablesPorts) Forward(id, containerIP string, ports []VMPropsPort) error {
	for _, port := range ports {
		forwardArgs := []string{
			"-w",
			"-t", "nat",
			"-A", "PREROUTING",
			"-p", port.Protocol,
			"!", "-i", "w+", // non-warden interfaces // todo wont work if cpi is nested
			"--dport", strconv.Itoa(port.Host),
			"-j", "DNAT", "--to", net.JoinHostPort(containerIP, strconv.Itoa(port.Container)),
			"-m", "comment", "--comment", id,
		}

		_, _, _, err := p.cmdRunner.RunCommand("iptables", forwardArgs...)
		if err != nil {
			p.removeRulesWithComment(id)
			return bosherr.WrapErrorf(err, "Forwarding port '%d'", port.Host)
		}
	}

	return nil
}

func (p IPTablesPorts) RemoveForwarded(id string) error {
	return p.removeRulesWithComment(id)
}

func (p IPTablesPorts) removeRulesWithComment(id string) error {
	stdout, _, _, err := p.cmdRunner.RunCommand("iptables-save", "-t", "nat")
	if err != nil {
		return bosherr.WrapErrorf(err, "Listing nat table rules to remove rules")
	}

	var lastErr error

	for _, line := range strings.Split(stdout, "\n") {
		if !strings.Contains(line, id) {
			continue
		}

		forwardArgs := []string{"-t", "nat", "-D"}
		forwardArgs = append(forwardArgs, strings.Split(line, " ")[1:]...)

		_, stderr, _, err := p.cmdRunner.RunCommand("iptables", forwardArgs...)
		if err != nil {
			// Ignore missing rule errors as there may be race conditions (though unlikely)
			if !strings.Contains(stderr, "No chain/target/match by that name") {
				lastErr = err
			}
		}
	}

	return lastErr
}
