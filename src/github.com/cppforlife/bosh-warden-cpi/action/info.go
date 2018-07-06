package action

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"
)

type InfoMethod struct{}

func NewInfoMethod() InfoMethod {
	return InfoMethod{}
}

func (a InfoMethod) Info() (apiv1.Info, error) {
	return apiv1.Info{
		StemcellFormats: []string{"warden-tar", "general-tar"},
		ApiVersion:      apiv1.MaxSupportedApiVersion, // TODO: What if the library is more than we handle?
	}, nil
}
