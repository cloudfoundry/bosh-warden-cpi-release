package stemcell

import (
	"strings"

	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type LightStemcell struct {
	cid            apiv1.StemcellCID
	imageReference string

	logTag string
	logger boshlog.Logger
}

func NewLightStemcell(
	cid apiv1.StemcellCID,
	imageReference string,
	logger boshlog.Logger,
) LightStemcell {
	return LightStemcell{
		cid:            cid,
		imageReference: imageReference,

		logTag: "LightStemcell",
		logger: logger,
	}
}

func (s LightStemcell) ID() apiv1.StemcellCID { return s.cid }

func (s LightStemcell) DirPath() string {
	// Garden requires docker:// scheme for OCI images from registry
	// The image reference may already have a scheme, or be a bare image reference
	if !strings.HasPrefix(s.imageReference, "docker://") {
		return "docker://" + s.imageReference
	}
	return s.imageReference
}

func (s LightStemcell) Delete() error {
	s.logger.Debug(s.logTag, "Delete light stemcell '%s'", s.cid)
	return nil
}
