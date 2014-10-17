package fakes

type FakeWardenFileService struct {
	UploadDestinationPath string
	UploadContents        []byte
	UploadErr             error

	DownloadSourcePath string
	DownloadContents   []byte
	DownloadErr        error
}

func NewFakeWardenFileService() *FakeWardenFileService {
	return &FakeWardenFileService{}
}

func (s *FakeWardenFileService) Upload(destinationPath string, contents []byte) error {
	s.UploadDestinationPath = destinationPath
	s.UploadContents = contents

	return s.UploadErr
}

func (s *FakeWardenFileService) Download(sourcePath string) ([]byte, error) {
	s.DownloadSourcePath = sourcePath

	return s.DownloadContents, s.DownloadErr
}
