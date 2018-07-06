package stemcell

import (
	"github.com/cppforlife/bosh-cpi-go/apiv1"
)

type Importer interface {
	ImportFromPath(imagePath string) (Stemcell, error)
}

//go:generate counterfeiter -o fakes/fake_finder.go . Finder
type Finder interface {
	Find(apiv1.StemcellCID) (Stemcell, bool, error)
}

type Stemcell interface {
	ID() apiv1.StemcellCID
	DirPath() string

	Delete() error
}
