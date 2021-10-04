package action

import (
	"github.com/cloudfoundry/bosh-cpi-go/apiv1"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"

	bwcstem "bosh-warden-cpi/stemcell"
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
