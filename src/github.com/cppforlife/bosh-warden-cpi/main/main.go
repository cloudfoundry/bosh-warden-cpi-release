package main

import (
	"flag"
	"os"

	wrdnclient "github.com/cloudfoundry-incubator/garden/client"
	wrdnconn "github.com/cloudfoundry-incubator/garden/client/connection"
	boshcmd "github.com/cloudfoundry/bosh-utils/fileutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"

	bwcaction "github.com/cppforlife/bosh-warden-cpi/action"
	bwcdisp "github.com/cppforlife/bosh-warden-cpi/api/dispatcher"
	bwctrans "github.com/cppforlife/bosh-warden-cpi/api/transport"
	bwcutil "github.com/cppforlife/bosh-warden-cpi/util"
)

const mainLogTag = "main"

var (
	configPathOpt = flag.String("configPath", "", "Path to configuration file")
)

func main() {
	logger, fs, cmdRunner, uuidGen := basicDeps()

	defer logger.HandlePanic("Main")

	flag.Parse()

	config, err := NewConfigFromPath(*configPathOpt, fs)
	if err != nil {
		logger.Error(mainLogTag, "Loading config %s", err.Error())
		os.Exit(1)
	}

	dispatcher := buildDispatcher(config, logger, fs, cmdRunner, uuidGen)

	cli := bwctrans.NewCLI(os.Stdin, os.Stdout, dispatcher, logger)

	err = cli.ServeOnce()
	if err != nil {
		logger.Error(mainLogTag, "Serving once %s", err)
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

func buildDispatcher(
	config Config,
	logger boshlog.Logger,
	fs boshsys.FileSystem,
	cmdRunner boshsys.CmdRunner,
	uuidGen boshuuid.Generator,
) bwcdisp.Dispatcher {
	wardenConn := wrdnconn.New(
		config.Warden.ConnectNetwork,
		config.Warden.ConnectAddress,
	)

	wardenClient := wrdnclient.New(wardenConn)

	compressor := boshcmd.NewTarballCompressor(cmdRunner, fs)

	sleeper := bwcutil.RealSleeper{}

	actionFactory := bwcaction.NewConcreteFactory(
		wardenClient,
		fs,
		cmdRunner,
		uuidGen,
		compressor,
		sleeper,
		config.Actions,
		logger,
	)

	caller := bwcdisp.NewJSONCaller()

	return bwcdisp.NewJSON(actionFactory, caller, logger)
}
