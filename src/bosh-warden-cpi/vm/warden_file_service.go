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

const (
	// DefaultRetryAttempts is the default number of retry attempts used by the
	// file existence verification logic.
	DefaultRetryAttempts = 10
	// DefaultRetryDelay is the default delay between retry attempts.
	DefaultRetryDelay = 200 * time.Millisecond
)

type wardenFileService struct {
	container wrdn.Container

	logTag        string
	logger        boshlog.Logger
	retryAttempts int
	retryDelay    time.Duration
}

// NewWardenFileService creates a WardenFileService with default retry tunables.
func NewWardenFileService(container wrdn.Container, logger boshlog.Logger) WardenFileService {
	return NewWardenFileServiceWithRetry(container, logger, DefaultRetryAttempts, DefaultRetryDelay)
}

// NewWardenFileServiceWithRetry allows tests to control retry attempts and delay
// so slow retry loops can be avoided in unit tests.
func NewWardenFileServiceWithRetry(container wrdn.Container, logger boshlog.Logger, attempts int, delay time.Duration) WardenFileService {
	return &wardenFileService{
		container:     container,
		logTag:        "vm.wardenFileService",
		logger:        logger,
		retryAttempts: attempts,
		retryDelay:    delay,
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

	// Stream directly to the destination directory rather than staging via /tmp.
	//
	// Root cause: Garden bind-mounts the garden-init binary into /tmp/garden-init
	// inside every container (see guardiancmd/command_linux.go:initBindMountAndPath).
	// On Ubuntu Noble (kernel 6.8) with cgroup v2, nstar's setns+tar writes to /tmp
	// land on the bind-mount layer and are not visible to subsequent container.Run()
	// processes looking at the overlayfs upper layer. /var/vcap/bosh/ has no such
	// bind mounts and is a plain overlayfs directory, so writes there are stable.
	// This regression did not affect Jammy (kernel 5.15).
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

	// Verify the file exists with retries as a safety net: while streaming directly
	// to /var/vcap/bosh/ is stable on Noble, this guards against any transient
	// overlayfs propagation delay.
	s.logger.Debug(s.logTag, "Verifying file exists at %s with retry", destinationPath)

	retryable := boshretry.NewRetryable(func() (bool, error) {
		// Sync filesystem; if sync fails, log and continue to the existence
		// check anyway — a sync failure is unlikely to self-heal across a
		// 200ms sleep and shouldn't burn a retry attempt.
		if syncErr := s.runPrivilegedScript("sync"); syncErr != nil {
			s.logger.Debug(s.logTag, "Failed to sync filesystem: %s", syncErr.Error())
		}

		// Check if file exists
		checkScript := fmt.Sprintf("[ -f '%s' ]", destinationPath)
		err := s.runPrivilegedScript(checkScript)
		if err != nil {
			s.logger.Debug(s.logTag, "File not yet visible at %s", destinationPath)
			return true, err
		}

		s.logger.Debug(s.logTag, "File verified at %s", destinationPath)
		return false, nil
	})

	// Retry using configured attempts and delay
	retryStrategy := boshretry.NewAttemptRetryStrategy(s.retryAttempts, s.retryDelay, retryable, s.logger)
	err = retryStrategy.Try()
	if err != nil {
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
