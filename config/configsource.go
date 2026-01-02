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
	"github.com/cloudprober/cloudprober/state"
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
		dcs.baseVars = vars
		return dcs
	}
}

func WithSurfacerConfig(sconfig string) Option {
	return func(cs ConfigSource) ConfigSource {
		dcs, ok := cs.(*defaultConfigSource)
		if !ok {
			return cs
		}
		dcs.surfacersConfigFileName = sconfig
		return dcs
	}
}

type defaultConfigSource struct {
	fileName                string
	surfacersConfigFileName string
	baseVars                map[string]any
	getGCECustomMetadata    func(string) (string, error)
	l                       *logger.Logger

	parsedConfig string
	rawConfig    string
	cfg          *configpb.ProberConfig
}

func (dcs *defaultConfigSource) configContent() (content string, format string, err error) {
	if dcs.fileName != "" {
		dcs.l.Debugf("configContent fileName is not empty will read from path %s", dcs.fileName)
		content, err := readConfigFile(dcs.fileName)
		return content, formatFromFileName(dcs.fileName), err
	}

	// On GCE first check if there is a config in custom metadata attributes.
	if metadata.OnGCE() {
		dcs.l.Debug("don't know why would get here")
		if config, err := dcs.getGCECustomMetadata(configMetadataKeyName); err != nil {
			dcs.l.Infof("Error reading config from metadata. Err: %v", err)
		} else {
			return config, "", nil
		}
	}

	dcs.l.Warningf("Config file %s not found. Using default config.", defaultConfigFile)
	return DefaultConfig(), "textpb", nil
}

func (dcs *defaultConfigSource) GetConfig() (*configpb.ProberConfig, error) {
	dcs.l.Debugf("dcs %+v", dcs)
	// Figure out which file to read
	if dcs.fileName == "" {
		dcs.l.Errorf("filename was empty gross")
		dcs.fileName = *configFile
	}

	if dcs.fileName == "" {
		dcs.l.Errorf("filename was empty twice going to set to default")
		if _, err := os.Stat(defaultConfigFile); !os.IsNotExist(err) {
			dcs.fileName = defaultConfigFile
		}
	}

	// Set the config file path in state. This can be used to find files
	// relative to the config file.
	state.SetConfigFilePath(dcs.fileName)

	tmplVars := make(map[string]any)
	for k, v := range dcs.baseVars {
		tmplVars[k] = v
	}
	for k, v := range sysvars.Vars() {
		tmplVars[k] = v
	}

	if dcs.getGCECustomMetadata == nil {
		dcs.getGCECustomMetadata = readFromGCEMetadata
	}

	configStr, configFormat, err := dcs.configContent()
	if err != nil {
		return nil, err
	}
	dcs.rawConfig = configStr

	dcs.cfg = &configpb.ProberConfig{}
	if dcs.parsedConfig, err = processConfigText(dcs.rawConfig, configFormat, tmplVars, dcs.cfg, dcs.l); err != nil {
		return nil, fmt.Errorf("error processing config. Err: %v", err)
	}

	if dcs.surfacersConfigFileName != "" {
		var sConfigText string
		var err error

		sConfigText, err = readConfigFile(dcs.surfacersConfigFileName)
		if err != nil {
			return nil, fmt.Errorf("error reading surfacers config file: %v", err)
		}
		dcs.rawConfig += "\n\n" + sConfigText

		sConfig, fileFmt := &configpb.SurfacersConfig{}, formatFromFileName(dcs.surfacersConfigFileName)
		parsedSConfig, err := processConfigText(sConfigText, fileFmt, tmplVars, sConfig, dcs.l)
		if err != nil {
			return nil, fmt.Errorf("error processing surfacers config. Err: %v", err)
		}
		dcs.parsedConfig += "\n\n" + parsedSConfig
		dcs.cfg.Surfacer = append(dcs.cfg.Surfacer, sConfig.GetSurfacer()...)
	}

	return dcs.cfg, nil
}

func (dcs *defaultConfigSource) RawConfig() string {
	return dcs.rawConfig
}

func (dcs *defaultConfigSource) ParsedConfig() string {
	return dcs.parsedConfig
}
