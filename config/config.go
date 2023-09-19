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
	"regexp"

	"cloud.google.com/go/compute/metadata"
	"github.com/cloudprober/cloudprober/common/file"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"sigs.k8s.io/yaml"
)

var (
	configFile = flag.String("config_file", "", "Config file")
)

var envRegex = regexp.MustCompile(`{{ \$([^$]+) }}`)

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

func configToProto(configStr, configFormat string) (*configpb.ProberConfig, error) {
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

func ConfigTest(fileName string, baseVars map[string]string) error {
	if fileName == "" {
		fileName = *configFile
	}
	content, configFormat, err := readConfigFile(fileName)
	if err != nil {
		return err
	}

	configStr, err := ParseTemplate(content, baseVars, func(v string) (string, error) {
		return v + "-test-value", nil
	})

	if err != nil {
		return err
	}

	_, err = configToProto(configStr, configFormat)
	return err
}

func DumpConfig(fileName, outFormat string, baseVars map[string]string) ([]byte, error) {
	if fileName == "" {
		fileName = *configFile
	}

	content, configFormat, err := readConfigFile(fileName)
	if err != nil {
		return nil, err
	}

	cfg, _, err := ParseConfig(content, configFormat, baseVars)
	if err != nil {
		return nil, err
	}

	switch outFormat {
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
		return nil, fmt.Errorf("unknown format: %s", outFormat)
	}
}

// substEnvVars substitutes environment variables in the config string.
func substEnvVars(configStr string) (string, error) {
	m := envRegex.FindAllStringSubmatch(configStr, -1)
	if len(m) == 0 {
		return configStr, nil
	}

	var envVars []string
	// Collect all the environment variables that need to be substituted.
	for _, word := range m {
		if len(word) != 2 {
			continue
		}
		envVars = append(envVars, word[1])
	}

	for _, v := range envVars {
		if os.Getenv(v) == "" {
			return "", fmt.Errorf("environment variable %s not defined", v)
		}
		configStr = envRegex.ReplaceAllString(configStr, os.Getenv(v))
	}

	return configStr, nil
}

func ParseConfig(content, format string, vars map[string]string) (*configpb.ProberConfig, string, error) {
	parsedConfig, err := ParseTemplate(content, vars, nil)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing config file as Go template. Err: %v", err)
	}

	finalConfigStr, err := substEnvVars(parsedConfig)
	if err != nil {
		return nil, "", fmt.Errorf("error substituting secret environment variables in the config string. Err: %v", err)
	}

	cfg, err := configToProto(finalConfigStr, format)
	return cfg, parsedConfig, err
}
