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
	"runtime"
	"strings"
	"testing"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	probespb "github.com/cloudprober/cloudprober/probes/proto"
	surfacerspb "github.com/cloudprober/cloudprober/surfacers/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestDefaultConfigSource(t *testing.T) {
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
	wantCfgStr := `probe {
    name: "dns_k8s"
    type: DNS
    targets {
        host_names: "10.0.0.1"
    }
}

surfacer {
    type: STACKDRIVER
}`

	if runtime.GOOS == "windows" {
		wantCfgStr = strings.ReplaceAll(wantCfgStr, "\n", "\r\n")
	}

	tests := []struct {
		name              string
		filename          string
		surfacersCfgFile  string
		configFile        string
		defaultConfigFile string
		baseVars          map[string]any
		want              *configpb.ProberConfig
		wantRawConfig     string
		wantParsedConfig  string
		wantErr           bool
	}{
		{
			name:             "filename_provided",
			filename:         "testdata/cloudprober.cfg",
			want:             wantCfg,
			wantRawConfig:    wantCfgStr,
			wantParsedConfig: wantCfgStr,
		},
		{
			name:             "config_file_flag",
			configFile:       "testdata/cloudprober.cfg",
			want:             wantCfg,
			wantRawConfig:    wantCfgStr,
			wantParsedConfig: wantCfgStr,
		},
		{
			name:             "config_and_surfacers_file",
			configFile:       "testdata/surfacers_config/cloudprober_no_surfacers.cfg",
			surfacersCfgFile: "testdata/surfacers_config/cloudprober_only_surfacers.cfg",
			want:             wantCfg,
			wantRawConfig:    wantCfgStr,
			wantParsedConfig: wantCfgStr,
		},
		{
			name:              "default_config_file",
			defaultConfigFile: "testdata/cloudprober.cfg",
			want:              wantCfg,
			wantRawConfig:     wantCfgStr,
			wantParsedConfig:  wantCfgStr,
		},
		{
			name:             "config_with_template_vars",
			filename:         "testdata/cloudprober_tmpl.cfg",
			baseVars:         map[string]any{"ProbeName": "dns_k8s", "ProbeType": "DNS"},
			want:             wantCfg,
			wantRawConfig:    strings.ReplaceAll(strings.ReplaceAll(wantCfgStr, "dns_k8s", "{{.ProbeName}}"), "DNS", "{{.ProbeType}}"),
			wantParsedConfig: wantCfgStr,
		},
		{
			name:             "default_config",
			want:             &configpb.ProberConfig{},
			wantRawConfig:    "",
			wantParsedConfig: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*configFile = tt.configFile
			dcs := &defaultConfigSource{
				fileName:                tt.filename,
				surfacersConfigFileName: tt.surfacersCfgFile,
			}
			if tt.baseVars != nil {
				dcs.baseVars = tt.baseVars
			}

			if tt.defaultConfigFile != "" {
				oldDefaultConfigFile := defaultConfigFile
				defaultConfigFile = tt.defaultConfigFile
				defer func() { defaultConfigFile = oldDefaultConfigFile }()
			}

			got, err := dcs.GetConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigToProto() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fixNewLines := func(s string) string {
				return strings.ReplaceAll(s, "\r\n", "\n")
			}
			assert.Equal(t, tt.want.String(), got.String())
			assert.Equal(t, fixNewLines(tt.wantRawConfig), fixNewLines(dcs.RawConfig()))
			assert.Equal(t, fixNewLines(tt.wantParsedConfig), fixNewLines(dcs.ParsedConfig()))
		})
	}
}
