package state

import (
	configv1 "github.com/daishe/impostorcmd/config/v1"
)

func List() (*configv1.Config, error) {
	return nil, nil // TODO
}

func Add(cfg *configv1.Config) error {
	if err := abs(cfg); err != nil {
		return err
	}
	fileCfg, err := loadStored()
	if err != nil {
		return err
	}
	limit(fileCfg, cfg)

	return nil // TODO
}

func Remove(cfg *configv1.Config) error {
	return nil // TODO
}

func abs(cfg *configv1.Config) error {
	return nil // TODO
}

func limit(toLimit, reference *configv1.Config) *configv1.Config {
	return nil // TODO
}

func onlyRealImpostors(cfg *configv1.Config) (*configv1.Config, error) {
	return nil, nil // TODO
}
