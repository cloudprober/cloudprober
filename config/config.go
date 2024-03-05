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
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/internal/file"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/google/go-jsonnet"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"sigs.k8s.io/yaml"
)

var (
	configFile       = flag.String("config_file", "", "Config file")
	testInstanceName = flag.String("test_instance_name", "ig-us-central1-a-01-0000", "Instance name example to be used in tests")
)

// EnvRegex is the regex used to find environment variable placeholders
// in the config file. The placeholders are of the form **$<env_var_name>**,
// and are added during Go template processing for envSecret functions.
var EnvRegex = regexp.MustCompile(`\*\*\$([^*\s]+)\*\*`)

const (
	configMetadataKeyName = "cloudprober_config"
)

var configTestVars = map[string]string{
	"zone":              "us-central1-a",
	"project":           "fake-domain.com:fake-project",
	"project_id":        "12345678",
	"instance":          *testInstanceName,
	"internal_ip":       "192.168.0.10",
	"external_ip":       "10.10.10.10",
	"instance_template": "ig-us-central1-a-01",
	"machine_type":      "e2-small",
}

func DefaultConfigSource() ConfigSource {
	return ConfigSourceWithFile(*configFile)
}

func ConfigSourceWithFile(fileName string) ConfigSource {
	return &defaultConfigSource{
		FileName: fileName,
	}
}

func formatFromFileName(fileName string) string {
	switch filepath.Ext(fileName) {
	case ".json":
		return "json"
	case ".jsonnet":
		return "jsonnet"
	case ".yaml", ".yml":
		return "yaml"
	default:
		return "textpb"
	}
}

// handleIncludes handles "include" statements in the config file. It handles
// nested includes in a depth-first manner.
func handleIncludes(baseDir string, content []byte) (string, error) {
	var final []string

	re := regexp.MustCompile(`(?m)^include\s+"([^"]+)"\s*$`)

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		m := re.FindStringSubmatch(line)
		if len(m) != 2 {
			final = append(final, line)
			continue
		}
		includedCfg, err := readConfigFile(filepath.Join(baseDir, m[1]))
		if err != nil {
			return "", err
		}
		final = append(final, includedCfg)
	}

	newline := "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}
	return strings.Join(final, newline), nil
}

func readConfigFile(fileName string) (string, error) {
	b, err := file.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	final, err := handleIncludes(filepath.Dir(fileName), b)
	return final, err
}

func unmarshalConfig(configStr, configFormat string) (*configpb.ProberConfig, error) {
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
	case "jsonnet":
		jsonCfg, err := jsonnet.MakeVM().EvaluateAnonymousSnippet("config", configStr)
		if err != nil {
			return nil, fmt.Errorf("error evaluating jsonnet config: %v", err)
		}
		if err := protojson.Unmarshal([]byte(jsonCfg), cfg); err != nil {
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

// substEnvVars substitutes environment variables in the config string.
func substEnvVars(configStr string, l *logger.Logger) string {
	m := EnvRegex.FindAllStringSubmatch(configStr, -1)
	if len(m) == 0 {
		return configStr
	}

	var envVars []string
	for _, match := range m {
		if len(match) != 2 {
			continue
		}
		fmt.Printf("Found env var: %v\n", match)
		envVars = append(envVars, match[1]) // match[0] is the whole string.
	}

	for _, v := range envVars {
		envVal := os.Getenv(v)
		if envVal == "" {
			l.Warningf("Environment variable %s not defined, skipping substitution.", v)
			continue
		}
		configStr = strings.ReplaceAll(configStr, "**$"+v+"**", envVal)
	}

	return configStr
}

func ConfigTest(cs ConfigSource) error {
	// cs is provided only for testing.
	if cs == nil {
		if *configFile == "" {
			return errors.New("config_file is required for testing")
		}
		cs = &defaultConfigSource{
			FileName: *configFile,
			BaseVars: configTestVars,
			GetGCECustomMetadata: func(v string) (string, error) {
				return v + "-test-value", nil
			},
		}
	}
	_, err := cs.GetConfig()
	return err
}

func DumpConfig(outFormat string, cs ConfigSource) ([]byte, error) {
	if cs == nil {
		cs = &defaultConfigSource{
			BaseVars: configTestVars,
		}
	}
	cfg, err := cs.GetConfig()
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
