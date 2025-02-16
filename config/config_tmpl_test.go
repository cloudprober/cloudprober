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
	"fmt"
	"runtime"
	"testing"

	"cloud.google.com/go/compute/metadata"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/state"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
)

func testParse(config string, tmplVars map[string]any) (*configpb.ProberConfig, error) {
	textConfig, err := parseTemplate(config, tmplVars, nil)
	if err != nil {
		return nil, err
	}

	cfg := &configpb.ProberConfig{}
	if err = prototext.Unmarshal([]byte(textConfig), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func TestParseTemplate(t *testing.T) {
	tests := []struct {
		desc        string
		config      string
		tmplVars    map[string]any
		wantProbes  []string
		wantTargets []string
		wantErrStr  string
	}{
		{
			desc: "config-with-extract-substring",
			config: `
				{{ $shard := "ig-us-east1-a-02-afgx" | extractSubstring "[^-]+-[^-]+-[^-]+-[^-]+-([^-]+)-.*" 1 }}
				probe {
				type: PING
				name: "vm-to-google-{{$shard}}-{{.region}}"
				targets {
					host_names: "www.google.com"
				}
				}
			`,
			tmplVars:    map[string]any{"region": "testRegion"},
			wantProbes:  []string{"vm-to-google-02-testRegion"},
			wantTargets: []string{"host_names:\"www.google.com\""},
		},
		{
			desc: "config-with-nested-tmpl-vars",
			config: `
				probe {
				type: PING
				name: "vm-to-google-{{.shard}}-{{.config.region}}"
				targets {
					host_names: "{{.config.targets}}"
				}
				}
			`,
			tmplVars:    map[string]any{"shard": "02", "config": map[string]string{"region": "testRegion", "targets": "www.google.com"}},
			wantProbes:  []string{"vm-to-google-02-testRegion"},
			wantTargets: []string{"host_names:\"www.google.com\""},
		},
		{
			desc: "config-with-secret-env",
			config: `
				probe {
				name: "{{ envSecret "SECRET_PROBE_NAME" }}"
				type: PING
				targets {
					host_names: "www.google.com"
				}
				}
			`,
			wantProbes:  []string{"**$SECRET_PROBE_NAME**"},
			wantTargets: []string{"host_names:\"www.google.com\""},
		},
		{
			desc: "config-with-map-and-template",
			config: `
				{{define "probeTmpl"}}
				probe {
					type: {{.typ}}
					name: "{{.name}}"
					targets {
					host_names: "{{.target}}"
					}
				}
				{{end}}

				{{template "probeTmpl" mkMap "typ" "PING" "name" "ping_google" "target" "www.google.com"}}
				{{template "probeTmpl" mkMap "typ" "PING" "name" "ping_facebook" "target" "www.facebook.com"}}
			`,
			wantProbes:  []string{"ping_google", "ping_facebook"},
			wantTargets: []string{"host_names:\"www.google.com\"", "host_names:\"www.facebook.com\""},
		},
		{
			desc: "config-with-template-error",
			config: `
				probe {
				type: PING
				name: "vm-to-google-{{$shard}}"
				targets {
					host_names: "www.google.com"
				}
				}
			`,
			wantErrStr: "template",
		},
		{
			desc: "config-with-missing-required-field",
			config: `
				probe {
				type: PING
				targets {
					host_names: "www.google.com"
				}
				}
			`,
			wantErrStr: "proto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.tmplVars == nil {
				tt.tmplVars = map[string]any{}
			}

			cfg, err := testParse(tt.config, tt.tmplVars)
			if err != nil {
				if tt.wantErrStr == "" {
					t.Errorf("Got unexpected error: %v", err)
				}
				assert.ErrorContains(t, err, tt.wantErrStr, "error string")
				return
			}
			if tt.wantErrStr != "" {
				t.Errorf("Expected an error, got none")
			}

			var probeNames, targets []string
			for _, probe := range cfg.GetProbe() {
				probeNames = append(probeNames, probe.GetName())
				targets = append(targets, probe.GetTargets().String())
			}
			assert.Equal(t, tt.wantProbes, probeNames, "probe names")
			assert.Equal(t, tt.wantTargets, targets, "targets")
		})
	}
}

func TestParseTemplateConfigDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}
	configFilePath := state.ConfigFilePath()
	defer state.SetConfigFilePath(configFilePath)
	state.SetConfigFilePath("/cfg/cloudprober.cfg")

	testConfig := `
probe {
  name: "test_x"
  type: EXTERNAL
  probe_external {
    # test_x.sh is in the same directory as the config file.
	command: "{{configDir}}/scripts/test_x.sh"
  }
}
`
	wantConfig := `
probe {
  name: "test_x"
  type: EXTERNAL
  probe_external {
    # test_x.sh is in the same directory as the config file.
	command: "/cfg/scripts/test_x.sh"
  }
}
`
	textConfig, err := parseTemplate(testConfig, map[string]any{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, wantConfig, textConfig, "config")
}

func TestParseGCEMetadata(t *testing.T) {
	testConfig := `
probe {
  type: PING
  name: "{{gceCustomMetadata "google-probe-name"}}-{{gceCustomMetadata "cluster"}}"
  targets {
    host_names: "www.google.com"
  }
}
`
	textConfig, err := parseTemplate(testConfig, map[string]any{}, func(key string) (string, error) {
		if key == "google-probe-name" {
			return "google_dot_com_from", nil
		}
		if key == "cluster" {
			return "", metadata.NotDefinedError("not defined")
		}
		return "", fmt.Errorf("not-implemented")
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &configpb.ProberConfig{}
	if err = prototext.Unmarshal([]byte(textConfig), cfg); err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cfg.GetProbe(), 1, "number of probes")
	assert.Equal(t, "google_dot_com_from-undefined", cfg.GetProbe()[0].GetName(), "probe name")
}
