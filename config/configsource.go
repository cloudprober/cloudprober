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
	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
)

var defaultConfigFile = "/etc/cloudprober.cfg"

type ConfigSource interface {
	GetConfig() (*configpb.ProberConfig, error)
	RawConfig() string
	ParsedConfig() string
}

type Option func(ConfigSource) ConfigSource

func WithBaseVars(vars map[string]any) Option {
	return func(cs ConfigSource) ConfigSource {
		dcs, ok := cs.(*defaultConfigSource)
		if !ok {
			return cs
		}
		dcs.BaseVars = vars
		return dcs
	}
}

func WithSurfacerConfig(sconfig string) Option {
	return func(cs ConfigSource) ConfigSource {
		dcs, ok := cs.(*defaultConfigSource)
		if !ok {
			return cs
		}
		dcs.SurfacersConfigFileName = sconfig
		return dcs
	}
}

type defaultConfigSource struct {
	FileName                string
	SurfacersConfigFileName string
	BaseVars                map[string]any
	GetGCECustomMetadata    func(string) (string, error)
	l                       *logger.Logger

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

	// Set the config file path in runconfig. This can be used to find files
	// relative to the config file.
	runconfig.SetConfigFilePath(dcs.FileName)

	if dcs.BaseVars == nil {
		dcs.BaseVars = make(map[string]any, len(sysvars.Vars()))
		for k, v := range sysvars.Vars() {
			dcs.BaseVars[k] = v
		}
	}

	if dcs.GetGCECustomMetadata == nil {
		dcs.GetGCECustomMetadata = readFromGCEMetadata
	}

	configStr, configFormat, err := dcs.configContent()
	if err != nil {
		return nil, err
	}
	dcs.rawConfig = configStr

	dcs.cfg = &configpb.ProberConfig{}
	if dcs.parsedConfig, err = processConfigText(dcs.rawConfig, configFormat, dcs.BaseVars, dcs.cfg, dcs.l); err != nil {
		return nil, fmt.Errorf("error processing config. Err: %v", err)
	}

	if dcs.SurfacersConfigFileName != "" {
		var sConfigText string
		var err error

		sConfigText, err = readConfigFile(dcs.SurfacersConfigFileName)
		if err != nil {
			return nil, fmt.Errorf("error reading surfacers config file: %v", err)
		}
		dcs.rawConfig += "\n\n" + sConfigText

		sConfig, fileFmt := &configpb.SurfacersConfig{}, formatFromFileName(dcs.SurfacersConfigFileName)
		parsedSConfig, err := processConfigText(sConfigText, fileFmt, dcs.BaseVars, sConfig, dcs.l)
		if err != nil {
			return nil, fmt.Errorf("error processing surfacers config. Err: %v", err)
		}
		dcs.parsedConfig += "\n\n" + parsedSConfig
		dcs.cfg.Surfacer = append(dcs.cfg.Surfacer, sConfig.GetSurfacer()...)
	}

	// Set the config file path in runconfig. This can be used to find files
	// relative to the config file.
	runconfig.SetConfigFilePath(dcs.FileName)

	return dcs.cfg, nil
}

func (dcs *defaultConfigSource) RawConfig() string {
	return dcs.rawConfig
}

func (dcs *defaultConfigSource) ParsedConfig() string {
	return dcs.parsedConfig
}
