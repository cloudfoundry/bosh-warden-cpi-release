package fakes

type FakeWardenFileService struct {
	UploadInputs []UploadInput
	UploadErr    error

	DownloadSourcePath string
	DownloadContents   []byte
	DownloadErr        error
}

type UploadInput struct {
	DestinationPath string
	Contents        []byte
}

func NewFakeWardenFileService() *FakeWardenFileService {
	return &FakeWardenFileService{
		UploadInputs: []UploadInput{},
	}
}

func (s *FakeWardenFileService) Upload(destinationPath string, contents []byte) error {
	s.UploadInputs = append(s.UploadInputs, UploadInput{
		DestinationPath: destinationPath,
		Contents:        contents,
	})

	return s.UploadErr
}

func (s *FakeWardenFileService) Download(sourcePath string) ([]byte, error) {
	s.DownloadSourcePath = sourcePath

	return s.DownloadContents, s.DownloadErr
}
