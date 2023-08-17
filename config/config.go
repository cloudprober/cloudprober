// Copyright 2017-2023 The Cloudprober Authors.
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
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"cloud.google.com/go/compute/metadata"
	"github.com/cloudprober/cloudprober/common/file"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/sysvars"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"sigs.k8s.io/yaml"
)

var (
	configFile = flag.String("config_file", "", "Config file")
)

const (
	configMetadataKeyName = "cloudprober_config"
	defaultConfigFile     = "/etc/cloudprober.cfg"
)

func readConfigFile(fileName string) (string, string, error) {
	b, err := file.ReadFile(fileName)
	if err != nil {
		return "", "", err
	}

	switch filepath.Ext(fileName) {
	case ".pb.txt", ".cfg", ".textpb":
		return string(b), "textpb", nil
	case ".json":
		return string(b), "json", nil
	case ".yaml", ".yml":
		return string(b), "yaml", nil
	}

	return string(b), "", nil
}

func GetConfig(confFile string, l *logger.Logger) (content string, format string, err error) {
	if confFile != "" {
		return readConfigFile(confFile)
	}

	if *configFile != "" {
		return readConfigFile(*configFile)
	}

	// On GCE first check if there is a config in custom metadata
	// attributes.
	if metadata.OnGCE() {
		if config, err := ReadFromGCEMetadata(configMetadataKeyName); err != nil {
			l.Infof("Error reading config from metadata. Err: %v", err)
		} else {
			return config, "", nil
		}
	}

	// If config not found in metadata, check default config on disk
	if _, err := os.Stat(defaultConfigFile); !os.IsNotExist(err) {
		return readConfigFile(defaultConfigFile)
	}

	l.Warningf("Config file %s not found. Using default config.", defaultConfigFile)
	return DefaultConfig(), "textpb", nil
}

func debugConfigYAML(cfg *configpb.ProberConfig, format string) {
	if format == "json" || format == "yaml" {
		fmt.Println("Textproto format:\n\n", prototext.Format(cfg))
		return
	}

	// Dealing with proto textpb format.
	jsonCfg, err := protojson.Marshal(cfg)
	if err != nil {
		panic("error converting config to json: " + err.Error())
	}
	fmt.Println("Config to JSON:\n\n", protojson.Format(cfg))
	y, err := yaml.JSONToYAML(jsonCfg)
	if err != nil {
		panic("error converting config to yaml: " + err.Error())
	}
	fmt.Println("Config to YAML:\n\n", string(y))
}

func ConfigToProto(configStr, configFormat string) (*configpb.ProberConfig, error) {
	cfg := &configpb.ProberConfig{}
	switch configFormat {
	case "yaml":
		jsonCfg, err := yaml.YAMLToJSON([]byte(configStr))
		if err != nil {
			return nil, fmt.Errorf("error converting YAML config to JSON: %v", err)
		}
		if err := protojson.Unmarshal(jsonCfg, cfg); err != nil {
			return nil, fmt.Errorf("error unmarshaling intermediate JSON to proto: %v", err)
		}
	case "json":
		if err := protojson.Unmarshal([]byte(configStr), cfg); err != nil {
			return nil, err
		}
	default:
		if err := prototext.Unmarshal([]byte(configStr), cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func parseConfig(f string, vars map[string]string) (*configpb.ProberConfig, error) {
	content, configFormat, err := readConfigFile(f)
	if err != nil {
		return nil, err
	}

	configStr, err := ParseTemplate(content, vars, func(v string) (string, error) {
		return v + "-test-value", nil
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing config file as Go template. Err: %v", err)
	}

	return ConfigToProto(configStr, configFormat)
}

func ConfigTest(baseVars map[string]string) error {
	_, err := parseConfig(*configFile, baseVars)
	return err
}

func DumpConfig(file, format string) ([]byte, error) {
	if file == "" {
		file = *configFile
	}

	cfg, err := parseConfig(file, sysvars.Vars())
	if err != nil {
		return nil, err
	}

	switch format {
	case "yaml":
		jsonCfg, err := protojson.Marshal(cfg)
		if err != nil {
			return nil, fmt.Errorf("error converting config to json: %v", err)
		}
		return yaml.JSONToYAML(jsonCfg)
	case "json":
		return protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(cfg)
	case "textpb":
		return prototext.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(cfg)
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}
