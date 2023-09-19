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
	"encoding/json"
	"os"
	"strings"
	"testing"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	probespb "github.com/cloudprober/cloudprober/probes/proto"
	surfacerspb "github.com/cloudprober/cloudprober/surfacers/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func testConfigToProto(t *testing.T, fileName string) (*configpb.ProberConfig, error) {
	t.Helper()

	configStr, configFormat, err := readConfigFile(fileName)
	if err != nil {
		t.Error(err)
	}
	return configToProto(configStr, configFormat)
}

func TestConfigToProto(t *testing.T) {
	wantCfg := &configpb.ProberConfig{
		Probe: []*probespb.ProbeDef{
			{
				Name: proto.String("dns_k8s"),
				Type: probespb.ProbeDef_DNS.Enum(),
				Targets: &targetspb.TargetsDef{
					Type: &targetspb.TargetsDef_HostNames{
						HostNames: "10.0.0.1",
					},
				},
			},
		},
		Surfacer: []*surfacerspb.SurfacerDef{
			{
				Type: surfacerspb.Type_STACKDRIVER.Enum(),
			},
		},
	}
	tests := []struct {
		name           string
		configFile     string
		baseConfigFile string
		want           *configpb.ProberConfig
		wantErr        bool
	}{
		{
			name:       "textpb",
			configFile: "testdata/cloudprober_base.cfg",
			want:       wantCfg,
		},
		{
			name:           "yaml",
			configFile:     "testdata/cloudprober.yaml",
			baseConfigFile: "testdata/cloudprober.cfg",
		},
		{
			name:           "json",
			configFile:     "testdata/cloudprober.json",
			baseConfigFile: "testdata/cloudprober.cfg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testConfigToProto(t, tt.configFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigToProto() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil {
				cfg, err := testConfigToProto(t, tt.baseConfigFile)
				if err != nil {
					t.Errorf("Error reading the base config itself: %v", err)
				}
				tt.want = cfg
			}
			assert.Equal(t, tt.want.String(), got.String())
		})
	}
}

func TestConfigTest(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
		baseVars   map[string]string
		wantErr    bool
	}{
		{
			name: "invalid_without_vars",
			baseVars: map[string]string{
				"az": "us-east-1a",
			},
			configFile: "testdata/cloudprober_invalid.cfg",
			wantErr:    true,
		},
		{
			name: "valid_with_vars",
			baseVars: map[string]string{
				"probetype": "HTTP",
			},
			configFile: "testdata/cloudprober_invalid.cfg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ConfigTest(tt.configFile, tt.baseVars); (err != nil) != tt.wantErr {
				t.Errorf("ConfigTest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDumpConfig(t *testing.T) {
	tests := []struct {
		configFile string
		format     string
		want       string
		wantErr    bool
	}{
		{
			configFile: "testdata/cloudprober_base.cfg",
			format:     "yaml",
			want: `
probe:
- name: dns_k8s
  targets:
    hostNames: 10.0.0.1
  type: DNS
surfacer:
- type: STACKDRIVER
`,
		},
		{

			configFile: "testdata/cloudprober_base.cfg",
			format:     "json",
			want: `
{
	"probe": [{
			"name": "dns_k8s",
			"type": "DNS",
			"targets": {"hostNames": "10.0.0.1"}
	}],
    "surfacer": [{
		"type": "STACKDRIVER"
	}]
}`,
		},
		{

			configFile: "testdata/cloudprober_base.cfg",
			format:     "textpb",
			want: `
probe: {
  name: "dns_k8s"
  type: DNS
  targets: {
    host_names: "10.0.0.1"
  }
}
surfacer: {
  type: STACKDRIVER
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got, err := DumpConfig(tt.configFile, tt.format, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("DumpConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			switch tt.format {
			case "json":
				var g interface{}
				var w interface{}
				assert.NoError(t, json.Unmarshal(got, &g))
				assert.NoError(t, json.Unmarshal([]byte(tt.want), &w))
				assert.Equal(t, w, g)
			case "textpb":
				var g configpb.ProberConfig
				var w configpb.ProberConfig
				assert.NoError(t, prototext.Unmarshal(got, &g))
				assert.NoError(t, prototext.Unmarshal([]byte(tt.want), &w))
				assert.Equal(t, w.String(), g.String())
			default:
				tt.want = strings.TrimLeft(tt.want, "\n")
				assert.Equal(t, tt.want, string(got))
			}
		})
	}
}

func TestSubstEnvVars(t *testing.T) {
	os.Setenv("SECRET_PROBE_NAME", "probe-x")
	// Make sure this env var is not set, for error behavior testing.
	os.Unsetenv("SECRET_PROBEX_NAME")

	tests := []struct {
		name      string
		configStr string
		want      string
		wantErr   bool
	}{
		{
			name:      "no_env_vars",
			configStr: `probe {name: "dns_k8s"}`,
			want:      `probe {name: "dns_k8s"}`,
		},
		{
			name:      "env_var",
			configStr: `probe {name: "{{ secret:$SECRET_PROBE_NAME }}"}`,
			want:      `probe {name: "probe-x"}`,
		},
		{
			name:      "env_var_not_defined",
			configStr: `probe {name: "{{ secret:$SECRET_PROBEX_NAME }}"}`,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := substEnvVars(tt.configStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("substEnvVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("substEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}
