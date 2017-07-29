package main

import (
	"flag"
	"os"

	wrdnclient "code.cloudfoundry.org/garden/client"
	wrdnconn "code.cloudfoundry.org/garden/client/connection"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
	"github.com/cppforlife/bosh-cpi-go/rpc"

	bwcaction "github.com/cppforlife/bosh-warden-cpi/action"
	"github.com/cppforlife/bosh-warden-cpi/util"
)

var (
	configPathOpt = flag.String("configPath", "", "Path to configuration file")
)

func main() {
	logger, fs, cmdRunner, uuidGen := basicDeps()
	defer logger.HandlePanic("Main")

	flag.Parse()

	config, err := NewConfigFromPath(*configPathOpt, fs)
	if err != nil {
		logger.Error("main", "Loading config %s", err.Error())
		os.Exit(1)
	}

	wardenConn := wrdnconn.New(config.Warden.ConnectNetwork, config.Warden.ConnectAddress)
	wardenClient := wrdnclient.New(wardenConn)

	compressor := boshcmd.NewTarballCompressor(cmdRunner, fs)
	decompressor := util.NewTarDecompressor(fs, compressor)

	cpiFactory := bwcaction.NewFactory(
		wardenClient, fs, cmdRunner, uuidGen, decompressor, config.Actions, logger)

	cli := rpc.NewFactory(logger).NewCLI(cpiFactory)

	err = cli.ServeOnce()
	if err != nil {
		logger.Error("main", "Serving once %s", err)
		os.Exit(1)
	}
}

func basicDeps() (boshlog.Logger, boshsys.FileSystem, boshsys.CmdRunner, boshuuid.Generator) {
	logger := boshlog.NewWriterLogger(boshlog.LevelDebug, os.Stderr, os.Stderr)
	fs := boshsys.NewOsFileSystem(logger)
	cmdRunner := boshsys.NewExecCmdRunner(logger)
	uuidGen := boshuuid.NewGenerator()
	return logger, fs, cmdRunner, uuidGen
}
