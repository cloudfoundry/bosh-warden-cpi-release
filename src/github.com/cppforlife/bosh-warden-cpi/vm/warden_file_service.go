package vm

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	wrdn "github.com/cloudfoundry-incubator/garden"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type wardenFileService struct {
	container wrdn.Container
	logger    boshlog.Logger
	logTag    string
}

func NewWardenFileService(container wrdn.Container, logger boshlog.Logger) WardenFileService {
	return &wardenFileService{
		container: container,
		logger:    logger,
		logTag:    "wardenFileService",
	}
}

func (s *wardenFileService) Download(sourcePath string) ([]byte, error) {
	sourceFileName := filepath.Base(sourcePath)
	tmpFilePath := filepath.Join("/tmp", sourceFileName)

	s.logger.Debug(s.logTag, "Downloading file at %s", sourcePath)

	// Copy settings file to a temporary directory
	// so that tar (running as vcap) has permission to readdir.
	// (/var/vcap/bosh is owned by root.)
	script := fmt.Sprintf(
		"cp %s %s && chown vcap:vcap %s",
		sourcePath,
		tmpFilePath,
		tmpFilePath,
	)

	err := s.runPrivilegedScript(script)
	if err != nil {
		return []byte{}, bosherr.WrapError(err, "Running copy source file script")
	}

	spec := wrdn.StreamOutSpec{
		Path: tmpFilePath,
		User: "root",
	}

	streamOut, err := s.container.StreamOut(spec)
	if err != nil {
		return []byte{}, bosherr.WrapErrorf(err, "Streaming out file '%s'", sourceFileName)
	}

	tarReader := tar.NewReader(streamOut)

	_, err = tarReader.Next()
	if err != nil {
		return []byte{}, bosherr.WrapErrorf(err, "Reading tar header for '%s'", sourceFileName)
	}

	return ioutil.ReadAll(tarReader)
}

func (s *wardenFileService) Upload(destinationPath string, contents []byte) error {
	s.logger.Debug(s.logTag, "Uploading file to %s", destinationPath)

	destinationFileName := filepath.Base(destinationPath)

	// Stream in settings file to a temporary directory
	// so that tar (running as vcap) has permission to unpack into dir.
	tarReader, err := s.tarReader(destinationFileName, contents)
	if err != nil {
		return bosherr.WrapError(err, "Creating tar")
	}

	spec := wrdn.StreamInSpec{
		Path:      "/tmp/",
		User:      "root",
		TarStream: tarReader,
	}

	err = s.container.StreamIn(spec)
	if err != nil {
		return bosherr.WrapError(err, "Streaming in tar")
	}

	tmpFilePath := filepath.Join("/tmp", destinationFileName)
	// Move settings file to its final location
	script := fmt.Sprintf(
		"mv %s %s",
		tmpFilePath,
		destinationPath,
	)

	err = s.runPrivilegedScript(script)
	if err != nil {
		return bosherr.WrapErrorf(err, "Moving temporary file to destination '%s'", destinationPath)
	}

	return nil
}

func (s *wardenFileService) runPrivilegedScript(script string) error {
	processSpec := wrdn.ProcessSpec{
		Path: "bash",
		Args: []string{"-c", script},
		User: "root",
	}

	// Collect output for debugging
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	processIO := wrdn.ProcessIO{Stdout: stdout, Stderr: stderr}

	process, err := s.container.Run(processSpec, processIO)
	if err != nil {
		return bosherr.WrapError(err, "Running script")
	}

	exitCode, err := process.Wait()
	if err != nil {
		return bosherr.WrapError(err, "Waiting for script")
	}

	if exitCode != 0 {
		return bosherr.Errorf("Script exited with non-0 exit code, stdout: '%s' stderr: '%s'", stdout.String(), stderr.String())
	}

	return nil
}

func (s *wardenFileService) tarReader(fileName string, contents []byte) (io.Reader, error) {
	tarBytes := &bytes.Buffer{}

	tarWriter := tar.NewWriter(tarBytes)

	fileHeader := &tar.Header{
		Name: fileName,
		Size: int64(len(contents)),
		Mode: 0640,
	}

	err := tarWriter.WriteHeader(fileHeader)
	if err != nil {
		return nil, bosherr.WrapError(err, "Writing tar header")
	}

	_, err = tarWriter.Write(contents)
	if err != nil {
		return nil, bosherr.WrapError(err, "Writing file to tar")
	}

	err = tarWriter.Close()
	if err != nil {
		return nil, bosherr.WrapError(err, "Closing tar writer")
	}

	return tarBytes, nil
}
