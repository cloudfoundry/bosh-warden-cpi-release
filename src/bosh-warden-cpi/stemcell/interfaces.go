package stemcell

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
)

type Importer interface {
	ImportFromPath(imagePath string) (Stemcell, error)
}

type Finder interface {
	Find(apiv1.StemcellCID) (Stemcell, bool, error)
}

type Stemcell interface {
	ID() apiv1.StemcellCID
	DirPath() string

	Delete() error
}
