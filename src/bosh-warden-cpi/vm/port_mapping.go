package vm

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var (
	portRangeRegexp = regexp.MustCompile(`\A([1-9][0-9]*)(\s*[\-:]\s*([1-9][0-9]*))?\z`)
)

type PortMapping struct {
	host      PortRange
	container PortRange
	protocol  string
}

func NewPortMapping(host, container PortRange, protocol string) (PortMapping, error) {
	if host.Len() != container.Len() {
		return PortMapping{}, errors.New("Host and container port ranges must have same length")
	}
	if len(protocol) == 0 {
		return PortMapping{}, errors.New("Protocol must be specified")
	}
	if host.Len() > 1 {
		if !host.Same(container) {
			return PortMapping{}, errors.New("Host and container port ranges must be same") //nolint:staticcheck
		}
		if protocol != "tcp" && protocol != "udp" {
			return PortMapping{}, errors.New("Port ranges can only be used with tcp or udp protocol") //nolint:staticcheck
		}
	}
	return PortMapping{host: host, container: container, protocol: protocol}, nil
}

func (m PortMapping) Host() PortRange      { return m.host }
func (m PortMapping) Container() PortRange { return m.container }
func (m PortMapping) Protocol() string     { return m.protocol }

type PortRange struct {
	start, end int // both ends are inclusive
}

func NewPortRange(start, end int) (PortRange, error) {
	if start <= 0 || start > 65535 {
		return PortRange{}, errors.New("Port range start must be > 0 and <= 65535") //nolint:staticcheck
	}
	if end <= 0 || end > 65535 {
		return PortRange{}, errors.New("Port range end must be > 0 and <= 65535") //nolint:staticcheck
	}
	if start > end {
		return PortRange{}, errors.New("Port range start must be <= end") //nolint:staticcheck
	}
	return PortRange{start: start, end: end}, nil
}

func NewPortRangeFromString(s string) (PortRange, error) {
	matches := portRangeRegexp.FindStringSubmatch(s)
	if len(matches) == 0 {
		return PortRange{}, fmt.Errorf("Port range must match '%s'", portRangeRegexp) //nolint:staticcheck
	}

	if len(matches) != 4 {
		panic("Internal inconsistency: port range regexp mismatch")
	}

	start, err := strconv.Atoi(matches[1])
	if err != nil {
		panic(fmt.Sprintf("Expected '%s' to be an int", matches[1]))
	}

	// matches[2] is delim

	switch { //nolint:staticcheck
	case matches[3] == "":
		return NewPortRange(start, start)

	default:
		end, err := strconv.Atoi(matches[3])
		if err != nil {
			panic(fmt.Sprintf("Expected '%s' to be an int", matches[3]))
		}
		return NewPortRange(start, end)
	}
}

func (r PortRange) Start() int { return r.start }
func (r PortRange) End() int   { return r.end }
func (r PortRange) Len() int   { return r.end - r.start + 1 }

func (r PortRange) Same(other PortRange) bool {
	return r.start == other.start && r.end == other.end
}
