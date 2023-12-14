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
	"os"

	"cloud.google.com/go/compute/metadata"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
)

const (
	sysvarsModuleName = "sysvars"
)

var defaultConfigFile = "/etc/cloudprober.cfg"

type ConfigSource interface {
	GetConfig() (*configpb.ProberConfig, error)
	RawConfig() string
	ParsedConfig() string
}

type defaultConfigSource struct {
	FileName             string
	BaseVars             map[string]string
	GetGCECustomMetadata func(string) (string, error)
	l                    *logger.Logger

	parsedConfig string
	rawConfig    string
	cfg          *configpb.ProberConfig
}

func (dcs *defaultConfigSource) configContent() (content string, format string, err error) {
	if dcs.FileName != "" {
		content, err := readConfigFile(dcs.FileName)
		return content, formatFromFileName(dcs.FileName), err
	}

	// On GCE first check if there is a config in custom metadata attributes.
	if metadata.OnGCE() {
		if config, err := dcs.GetGCECustomMetadata(configMetadataKeyName); err != nil {
			dcs.l.Infof("Error reading config from metadata. Err: %v", err)
		} else {
			return config, "", nil
		}
	}

	dcs.l.Warningf("Config file %s not found. Using default config.", defaultConfigFile)
	return DefaultConfig(), "textpb", nil
}

func (dcs *defaultConfigSource) GetConfig() (*configpb.ProberConfig, error) {
	// Figure out which file to read
	if dcs.FileName == "" {
		dcs.FileName = *configFile
	}

	if dcs.FileName == "" {
		if _, err := os.Stat(defaultConfigFile); !os.IsNotExist(err) {
			dcs.FileName = defaultConfigFile
		}
	}

	if dcs.BaseVars == nil {
		dcs.BaseVars = sysvars.Vars()
	}

	if dcs.GetGCECustomMetadata == nil {
		dcs.GetGCECustomMetadata = readFromGCEMetadata
	}

	configStr, configFormat, err := dcs.configContent()
	if err != nil {
		return nil, err
	}
	dcs.rawConfig = configStr

	dcs.parsedConfig, err = parseTemplate(dcs.rawConfig, dcs.BaseVars, nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file as Go template. Err: %v", err)
	}

	dcs.cfg, err = unmarshalConfig(substEnvVars(dcs.parsedConfig, dcs.l), configFormat)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config. Err: %v", err)
	}

	return dcs.cfg, nil
}

func (dcs *defaultConfigSource) RawConfig() string {
	return dcs.rawConfig
}

func (dcs *defaultConfigSource) ParsedConfig() string {
	return dcs.parsedConfig
}
