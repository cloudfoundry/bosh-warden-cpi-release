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
	imageRef := s.imageReference
	
	parts := strings.Split(imageRef, ":")
	if len(parts) > 2 {
		imageRef = strings.Join(parts[:len(parts)-1], ":")
	}
	
	if !strings.HasPrefix(imageRef, "docker://") {
		return "docker://" + imageRef
	}
	return imageRef
}

func (s LightStemcell) Delete() error {
	s.logger.Debug(s.logTag, "Delete light stemcell '%s'", s.cid)
	return nil
}
