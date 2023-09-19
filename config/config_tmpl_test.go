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
	"testing"

	"cloud.google.com/go/compute/metadata"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
)

func testParse(config string, sysVars map[string]string) (*configpb.ProberConfig, error) {
	textConfig, err := ParseTemplate(config, sysVars, nil)
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
		sysVars     map[string]string
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
			sysVars:     map[string]string{"region": "testRegion"},
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
			wantProbes:  []string{"{{ secret:$SECRET_PROBE_NAME }}"},
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
			if tt.sysVars == nil {
				tt.sysVars = map[string]string{}
			}

			cfg, err := testParse(tt.config, tt.sysVars)
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
	textConfig, err := ParseTemplate(testConfig, map[string]string{}, func(key string) (string, error) {
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
