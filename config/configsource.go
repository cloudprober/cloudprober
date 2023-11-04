// Copyright 2023 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"log/slog"
	"os"

	"cloud.google.com/go/compute/metadata"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
)

const (
	sysvarsModuleName = "sysvars"
)

type ConfigSource interface {
	GetConfig() (*configpb.ProberConfig, error)
	RawConfig() string
	ParsedConfig() string
}

type DefaultConfigSource struct {
	OverrideConfigFile   string
	BaseVars             map[string]string
	GetGCECustomMetadata func(string) (string, error)
	l                    *logger.Logger

	parsedConfig string
	rawConfig    string
	cfg          *configpb.ProberConfig
}

func (dcs *DefaultConfigSource) getConfigContent() (content string, format string, err error) {
	if dcs.OverrideConfigFile != "" {
		return readConfigFile(dcs.OverrideConfigFile)
	}

	if *configFile != "" {
		return readConfigFile(*configFile)
	}

	// On GCE first check if there is a config in custom metadata
	// attributes.
	if metadata.OnGCE() {
		if config, err := readFromGCEMetadata(configMetadataKeyName); err != nil {
			dcs.l.Infof("Error reading config from metadata. Err: %v", err)
		} else {
			return config, "", nil
		}
	}

	// If config not found in metadata, check default config on disk
	if _, err := os.Stat(defaultConfigFile); !os.IsNotExist(err) {
		return readConfigFile(defaultConfigFile)
	}

	dcs.l.Warningf("Config file %s not found. Using default config.", defaultConfigFile)
	return DefaultConfig(), "textpb", nil
}

func (dcs *DefaultConfigSource) parseConfig(configStr, configFormat string) (*configpb.ProberConfig, string, error) {
	parsedConfig, err := ParseTemplate(configStr, dcs.BaseVars, nil)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing config file as Go template. Err: %v", err)
	}

	cfg, err := configTextToProto(substEnvVars(parsedConfig, dcs.l), configFormat)
	return cfg, parsedConfig, err
}

func (dcs *DefaultConfigSource) GetConfig() (*configpb.ProberConfig, error) {
	if dcs.BaseVars == nil {
		// Initialize sysvars module
		if err := sysvars.Init(logger.NewWithAttrs(slog.String("component", sysvarsModuleName)), nil); err != nil {
			return nil, err
		}
		dcs.BaseVars = sysvars.Vars()
	}

	configStr, configFormat, err := dcs.getConfigContent()
	if err != nil {
		return nil, err
	}

	cfg, parsedConfigStr, err := dcs.parseConfig(configStr, configFormat)
	if err != nil {
		return nil, err
	}

	dcs.parsedConfig = parsedConfigStr
	dcs.rawConfig = configStr
	dcs.cfg = cfg

	return cfg, nil
}

func (dcs *DefaultConfigSource) RawConfig() string {
	return dcs.rawConfig
}

func (dcs *DefaultConfigSource) ParsedConfig() string {
	return dcs.parsedConfig
}
