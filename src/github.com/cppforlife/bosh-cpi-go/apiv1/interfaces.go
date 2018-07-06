package apiv1

//go:generate counterfeiter -o apiv1fakes/fake_cpi.go . CPI
//go:generate counterfeiter -o apiv1fakes/fake_cpifactory.go . CPIFactory

const MaxSupportedApiVersion = 2

type CPIFactory interface {
	New(CallContext, ApiVersions) (CPI, error)
}

type CallContext interface {
	As(interface{}) error
}

type CPI interface {
	Info() (Info, error)
	Stemcells
	VMs
	Disks
	Snapshots
}

type Info struct {
	StemcellFormats []string `json:"stemcell_formats"`
	ApiVersion      int      `json:"api_version,omitempty"`
}

type ApiVersions struct {
	Stemcell int
	Contract int
}
