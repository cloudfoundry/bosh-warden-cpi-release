package vm

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	bwcutil "bosh-warden-cpi/util"
)

type IPTablesPorts struct {
	sleeper   bwcutil.Sleeper
	cmdRunner boshsys.CmdRunner
}

func NewIPTablesPorts(sleeper bwcutil.Sleeper, cmdRunner boshsys.CmdRunner) IPTablesPorts {
	return IPTablesPorts{sleeper, cmdRunner}
}

func (p IPTablesPorts) Forward(id apiv1.VMCID, containerIP string, mappings []PortMapping) error {
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
			p.removeRulesWithID(id) //nolint:errcheck
			return bosherr.WrapErrorf(err, "Forwarding host port(s) '%v'", mapping.Host())
		}
	}

	return nil
}

func (p IPTablesPorts) RemoveForwarded(id apiv1.VMCID) error {
	return p.removeRulesWithID(id)
}

func (p IPTablesPorts) removeRulesWithID(id apiv1.VMCID) error {
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

func (IPTablesPorts) comment(id apiv1.VMCID) string {
	return "bosh-warden-cpi-" + id.AsString()
}

func (IPTablesPorts) fmtPortRange(portRange PortRange, delim string) string {
	if portRange.Len() > 1 {
		return strconv.Itoa(portRange.Start()) + delim + strconv.Itoa(portRange.End())
	}
	return strconv.Itoa(portRange.Start())
}

func (p IPTablesPorts) runCmd(action string, args []string) (string, string, int, error) {
	globalArgs := []string{"-w", "-t", "nat", action}
	globalArgs = append(globalArgs, args...)

	for i := 0; i < 60; i++ {
		stdout, stderr, code, err := p.cmdRunner.RunCommand("iptables", globalArgs...)
		if err != nil {
			if strings.Contains(stderr, "Resource temporarily unavailable") {
				p.sleeper.Sleep(500 * time.Millisecond)
				continue
			}
		}

		return stdout, stderr, code, err
	}

	return "", "", 0, bosherr.Errorf("Failed to add iptables rule")
}
