package action

import (
	bosherr "bosh/errors"

	bwcstem "bosh-warden-cpi/stemcell"
)

type CreateStemcell struct {
	stemcellImporter bwcstem.Importer
}

type CreateStemcellCloudProps struct{}

func NewCreateStemcell(stemcellImporter bwcstem.Importer) CreateStemcell {
	return CreateStemcell{stemcellImporter: stemcellImporter}
}

func (a CreateStemcell) Run(imagePath string, _ CreateStemcellCloudProps) (StemcellCID, error) {
	stemcell, err := a.stemcellImporter.ImportFromPath(imagePath)
	if err != nil {
		return "", bosherr.WrapError(err, "Importing stemcell from '%s'", imagePath)
	}

	return StemcellCID(stemcell.ID()), nil
}
