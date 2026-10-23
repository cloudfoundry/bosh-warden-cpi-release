package action

import (
	"strings"

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
	// Light stemcells have nothing on disk to delete. Skip the finder so that
	// older light CIDs, which it no longer accepts, can still be deleted.
	if strings.HasPrefix(cid.AsString(), "light://") {
		return nil
	}

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
