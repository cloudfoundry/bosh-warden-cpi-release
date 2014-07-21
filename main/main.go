package main

import (
	"flag"
	"os"

	boshlog "bosh/logger"
	boshcmd "bosh/platform/commands"
	boshsys "bosh/system"
	boshuuid "bosh/uuid"
	wrdnclient "github.com/cloudfoundry-incubator/garden/client"
	wrdnconn "github.com/cloudfoundry-incubator/garden/client/connection"

	bwcaction "bosh-warden-cpi/action"
	bwcdisp "bosh-warden-cpi/api/dispatcher"
	bwctrans "bosh-warden-cpi/api/transport"
	bwcutil "bosh-warden-cpi/util"
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
