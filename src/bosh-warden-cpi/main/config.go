package main

import (
	"encoding/json"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	bwcaction "bosh-warden-cpi/action"
)

type Config struct {
	Warden WardenConfig

	Actions bwcaction.FactoryOpts
}

type WardenConfig struct {
	// e.g. tcp, udp, unix
	ConnectNetwork string

	// Could be file path to sock file or an IP address
	ConnectAddress string
}

func NewConfigFromPath(path string, fs boshsys.FileSystem) (Config, error) {
	var config Config

	bytes, err := fs.ReadFile(path)
	if err != nil {
		return config, bosherr.WrapErrorf(err, "Reading config '%s'", path)
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return config, bosherr.WrapError(err, "Unmarshalling config")
	}

	err = config.Validate()
	if err != nil {
		return config, bosherr.WrapError(err, "Validating config")
	}

	return config, nil
}

func (c Config) Validate() error {
	err := c.Warden.Validate()
	if err != nil {
		return bosherr.WrapError(err, "Validating Warden configuration")
	}

	err = c.Actions.Validate()
	if err != nil {
		return bosherr.WrapError(err, "Validating Actions configuration")
	}

	return nil
}

func (c WardenConfig) Validate() error {
	if c.ConnectNetwork == "" {
		return bosherr.Error("Must provide non-empty ConnectNetwork")
	}

	if c.ConnectAddress == "" {
		return bosherr.Error("Must provide non-empty ConnectAddress")
	}

	return nil
}
