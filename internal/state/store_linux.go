//go:build linux

package state

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	configv1 "github.com/daishe/impostorcmd/config/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func storagePath() string {
	switch runtime.GOOS {
	case "windows":
		return "%PROGRAMDATA%/impostorcmd/impostorcmd_storage.json"
	case "darwin":
		return "/Library/Application/impostorcmd/impostorcmd_storage.json"
	case "linux":
		return "/var/impostorcmd/impostorcmd_storage.json"
	default:
		panic(fmt.Errorf("unsupported platform %s", runtime.GOOS))
	}
}

func checkVersion(p []byte) error {
	ver := &configv1.VersionEntity{}
	if err := (protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}).Unmarshal(p, ver); err != nil {
		return fmt.Errorf("parsing configuration version: %w", err)
	}
	return checkVersionString(ver.Version)
}

func checkVersionString(v string) error {
	if v != "v1" {
		return fmt.Errorf("configuration version %s is unsupported", v)
	}
	return nil
}

func loadStoredRaw() ([]byte, error) {
	p, err := os.ReadFile(storagePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read internal storage: %w", err)
	}

	if err := checkVersion(p); err != nil {
		return nil, fmt.Errorf("unmarshalling internal storage: %w", err)
	}
	return p, nil
}

func loadStored() (*configv1.Config, error) {
	p, err := loadStoredRaw()
	if err != nil {
		return nil, err
	}

	cfg := &configv1.Config{}
	if err := (protojson.UnmarshalOptions{AllowPartial: false, DiscardUnknown: false}).Unmarshal(p, cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling internal storage: %w", err)
	}
	return cfg, nil
}

func saveToStorageRaw(p []byte) error {
	return nil // TODO
}

func saveToStorage(cfg *configv1.Config) error {
	if err := checkVersionString(cfg.Version); err != nil {
		return err
	}
	_, err := loadStoredRaw()
	if err != nil {
		return err
	}

	p, err := (protojson.MarshalOptions{AllowPartial: false}).Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling internal storage: %w", err)
	}
	return saveToStorageRaw(p)
}
