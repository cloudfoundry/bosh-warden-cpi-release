package vm

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"path/filepath"
	"time"

	wrdn "code.cloudfoundry.org/garden"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

type wardenFileService struct {
	container wrdn.Container

	logTag string
	logger boshlog.Logger
}

func NewWardenFileService(container wrdn.Container, logger boshlog.Logger) WardenFileService {
	return &wardenFileService{
		container: container,

		logTag: "vm.wardenFileService",
		logger: logger,
	}
}

func (s *wardenFileService) Download(sourcePath string) ([]byte, error) {
	sourceFileName := filepath.Base(sourcePath)
	tmpFileName := uniqueTmpFileName(sourceFileName)
	tmpFilePath := filepath.Join("/tmp", tmpFileName)

	s.logger.Debug(s.logTag, "Downloading file at %s", sourcePath)

	// Copy settings file to a temporary directory for streaming out.
	// We use /tmp as an intermediate location because StreamOut requires
	// the file to be readable. The process runs as root.
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

	return io.ReadAll(tarReader)
}

func (s *wardenFileService) Upload(destinationPath string, contents []byte) error {
	s.logger.Debug(s.logTag, "Uploading file to %s", destinationPath)

	destinationFileName := filepath.Base(destinationPath)
	destinationDir := filepath.Dir(destinationPath)

	// Stream directly to destination directory to avoid overlayfs issues
	// with /tmp on Ubuntu Noble with cgroup v2.
	tarReader, err := s.tarReader(destinationFileName, contents)
	if err != nil {
		return bosherr.WrapError(err, "Creating tar")
	}

	spec := wrdn.StreamInSpec{
		Path:      destinationDir + "/",
		User:      "root",
		TarStream: tarReader,
	}

	s.logger.Debug(s.logTag, "Streaming in file %s to container %s/", destinationFileName, destinationDir)
	err = s.container.StreamIn(spec)
	if err != nil {
		return bosherr.WrapError(err, "Streaming in tar")
	}
	s.logger.Debug(s.logTag, "Successfully streamed in file %s", destinationFileName)

	// Verify the file exists with retries (for overlayfs sync issues on Ubuntu Noble with cgroup v2).
	// StreamIn may report success before the file is visible due to filesystem sync issues
	// with the overlayfs upper layer.
	s.logger.Debug(s.logTag, "Verifying file exists at %s with retry", destinationPath)

	retryable := boshretry.NewRetryable(func() (bool, error) {
		// Sync filesystem first
		syncScript := "sync"
		err := s.runPrivilegedScript(syncScript)
		if err != nil {
			s.logger.Debug(s.logTag, "Failed to sync filesystem: %s", err.Error())
			return true, err
		}

		// Check if file exists
		checkScript := fmt.Sprintf("[ -f %s ]", destinationPath)
		err = s.runPrivilegedScript(checkScript)
		if err != nil {
			s.logger.Debug(s.logTag, "File not yet visible at %s", destinationPath)
			return true, err
		}

		s.logger.Debug(s.logTag, "File verified at %s", destinationPath)
		return false, nil
	})

	// Retry for up to 2 seconds with 200ms delay between attempts (10 attempts)
	retryStrategy := boshretry.NewAttemptRetryStrategy(10, 200*time.Millisecond, retryable, s.logger)
	err = retryStrategy.Try()
	if err != nil {
		// If all retries failed, list directory contents for debugging
		listScript := fmt.Sprintf("ls -la %s/", destinationDir)
		listErr := s.runPrivilegedScript(listScript)
		if listErr == nil {
			s.logger.Debug(s.logTag, "Directory listing after failed verification")
		}
		return bosherr.WrapErrorf(err, "Verifying file at destination '%s' after streaming", destinationPath)
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

func uniqueTmpFileName(baseName string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	ext := filepath.Ext(baseName)
	name := baseName[:len(baseName)-len(ext)]
	return fmt.Sprintf("%s-%x%s", name, b, ext)
}

func (s *wardenFileService) tarReader(fileName string, contents []byte) (io.Reader, error) {
	tarBytes := &bytes.Buffer{}

	tarWriter := tar.NewWriter(tarBytes)

	fileHeader := &tar.Header{
		Name: fileName,
		Size: int64(len(contents)),
		Mode: 0644, // readable by owner, group, and others so vcap user can read
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
