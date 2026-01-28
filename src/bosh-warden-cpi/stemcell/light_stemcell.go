package stemcell

import (
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

func (s LightStemcell) URI() string {
	return "docker://" + s.imageReference
}

func (s LightStemcell) Delete() error {
	s.logger.Debug(s.logTag, "Delete light stemcell '%s'", s.cid)
	return nil
}
