package action

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cppforlife/bosh-cpi-go/apiv1"

	bwcstem "github.com/cppforlife/bosh-warden-cpi/stemcell"
)

type DeleteStemcellMethod struct {
	stemcellFinder bwcstem.Finder
}

func NewDeleteStemcellMethod(stemcellFinder bwcstem.Finder) DeleteStemcellMethod {
	return DeleteStemcellMethod{stemcellFinder: stemcellFinder}
}

func (a DeleteStemcellMethod) DeleteStemcell(cid apiv1.StemcellCID) error {
	stemcell, found, err := a.stemcellFinder.Find(cid)
	if err != nil {
		return bosherr.WrapErrorf(err, "Finding stemcell '%s'", cid)
	}

	if found {
		err := stemcell.Delete()
		if err != nil {
			return bosherr.WrapErrorf(err, "Deleting stemcell '%s'", cid)
		}
	}

	return nil
}
