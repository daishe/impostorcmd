package config

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/encoding/protojson"

	configv1 "github.com/daishe/impostorcmd/config/v1"
)

type VersionEntity interface {
	GetVersion() string
}

func checkStrictVersionString(v string) error {
	if strings.IndexFunc(v, unicode.IsSpace) != -1 {
		return fmt.Errorf("version cannot contain whitespace characters")
	} else if v == "" {
		return fmt.Errorf("unset version is unsupported")
	} else if v != "v1" {
		return fmt.Errorf("version %s is unsupported", v)
	}
	return nil
}

func checkRelaxedVersionString(v string) error {
	if v == "" { // allow empty version
		return nil
	}
	return checkStrictVersionString(v)
}

func UnmarshalAndValidateVersionEntity(p []byte) (VersionEntity, error) {
	ve := &configv1.VersionEntity{}
	if err := (protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}).Unmarshal(p, ve); err != nil {
		return nil, fmt.Errorf("parsing version: %w", err)
	}
	return ve, checkStrictVersionString(ve.Version)
}

func ValidateVersionEntity(ve VersionEntity) error {
	return checkStrictVersionString(ve.GetVersion())
}

func UnmarshalAndValidateTarget(targetBytes []byte) (*configv1.Target, error) {
	if _, err := UnmarshalAndValidateVersionEntity(targetBytes); err != nil {
		return nil, fmt.Errorf("unmarshalling target information: %w", err)
	}
	target := &configv1.Target{}
	if err := (protojson.UnmarshalOptions{AllowPartial: false, DiscardUnknown: false}).Unmarshal(targetBytes, target); err != nil {
		return nil, fmt.Errorf("unmarshalling target information: %w", err)
	}
	return target, nil
}

func UnmarshalAndValidateConfiguration(cfgBytes []byte) (*configv1.Config, error) {
	if _, err := UnmarshalAndValidateVersionEntity(cfgBytes); err != nil {
		return nil, fmt.Errorf("unmarshalling configuration: %w", err)
	}
	cfg := &configv1.Config{}
	if err := (protojson.UnmarshalOptions{AllowPartial: false, DiscardUnknown: false}).Unmarshal(cfgBytes, cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling configuration: %w", err)
	}
	for i, t := range cfg.Targets {
		if err := checkRelaxedVersionString(t.GetVersion()); err != nil {
			return nil, fmt.Errorf("unmarshalling configuration: target #%d (%s): %w", i+1, t.Cmd, err)
		}
	}
	return cfg, nil
}
