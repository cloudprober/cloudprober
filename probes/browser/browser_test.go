// Copyright 2024 The Cloudprober Authors.
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

package browser

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestProbePrepareCommand(t *testing.T) {
	os.Setenv("PLAYWRIGHT_DIR", "/playwright")
	defer os.Unsetenv("PLAYWRIGHT_DIR")

	baseEnvVars := func(pwDir string) []string {
		return []string{"NODE_PATH=" + pwDir + "/node_modules", "PLAYWRIGHT_HTML_REPORT={OUTPUT_DIR}/report", "PLAYWRIGHT_HTML_OPEN=never"}
	}

	cmdLine := func(npxPath string) []string {
		return []string{npxPath, "playwright", "test", "--config={WORKDIR}/playwright.config.ts", "--output=${OUTPUT_DIR}/results", "--reporter=html,{WORKDIR}/cloudprober-reporter.ts"}
	}

	baseWantEMLabels := [][2]string{{"ptype", "browser"}, {"probe", "test_browser"}, {"dst", ""}}

	tests := []struct {
		name               string
		disableAggregation bool
		npxPath            string
		playwrightDir      string
		testSpec           []string
		target             endpoint.Endpoint
		wantCmdLine        []string
		wantEnvVars        []string
		wantWorkDir        string
		wantEMLabels       [][2]string
	}{
		{
			name:         "default",
			wantCmdLine:  cmdLine("npx"),
			wantEnvVars:  baseEnvVars("/playwright"),
			wantWorkDir:  "/playwright",
			wantEMLabels: baseWantEMLabels,
		},
		{
			name:         "with_target",
			target:       endpoint.Endpoint{Name: "test_target", IP: net.ParseIP("12.12.12.12"), Port: 9313, Labels: map[string]string{"env": "prod"}},
			wantCmdLine:  cmdLine("npx"),
			wantEnvVars:  append(baseEnvVars("/playwright"), "target_name=test_target", "target_ip=12.12.12.12", "target_port=9313", "target_label_env=prod"),
			wantWorkDir:  "/playwright",
			wantEMLabels: [][2]string{{"ptype", "browser"}, {"probe", "test_browser"}, {"dst", "test_target:9313"}},
		},
		{
			name:               "disable_aggregation",
			disableAggregation: true,
			wantCmdLine:        cmdLine("npx"),
			wantEnvVars:        baseEnvVars("/playwright"),
			wantWorkDir:        "/playwright",
			wantEMLabels:       append(baseWantEMLabels, [2]string{"run_id", "0"}),
		},
		{
			name:          "with_playwright_dir",
			playwrightDir: "/app",
			wantCmdLine:   cmdLine("npx"),
			wantEnvVars:   baseEnvVars("/app"),
			wantWorkDir:   "/app",
			wantEMLabels:  baseWantEMLabels,
		},
		{
			name:         "with_npx_path",
			npxPath:      "/usr/bin/npx",
			wantCmdLine:  cmdLine("/usr/bin/npx"),
			wantEnvVars:  baseEnvVars("/playwright"),
			wantWorkDir:  "/playwright",
			wantEMLabels: baseWantEMLabels,
		},
		{
			name:         "with_test_spec",
			testSpec:     []string{"test_spec_1", "test_spec_2"},
			wantCmdLine:  append(cmdLine("npx"), "test_spec_1", "test_spec_2"),
			wantEnvVars:  baseEnvVars("/playwright"),
			wantWorkDir:  "/playwright",
			wantEMLabels: baseWantEMLabels,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &configpb.ProbeConf{
				TestSpec: tt.testSpec,
				TestMetricsOptions: &configpb.TestMetricsOptions{
					DisableAggregation: &tt.disableAggregation,
				},
			}
			if tt.playwrightDir != "" {
				conf.PlaywrightDir = &tt.playwrightDir
			}
			if tt.npxPath != "" {
				conf.NpxPath = proto.String(filepath.FromSlash(tt.npxPath))
			}

			opts := options.DefaultOptions()
			opts.ProbeConf = conf
			p := &Probe{}
			if err := p.Init("test_browser", opts); err != nil {
				t.Fatalf("Error in probe initialization: %v", err)
			}

			ts := time.Now()
			cmd, _ := p.prepareCommand(tt.target, ts)

			outputDir := p.outputDirPath(tt.target, ts)
			for i, arg := range tt.wantCmdLine {
				tt.wantCmdLine[i] = strings.ReplaceAll(arg, "{WORKDIR}", p.workdir)
				tt.wantCmdLine[i] = filepath.FromSlash(strings.ReplaceAll(tt.wantCmdLine[i], "${OUTPUT_DIR}", outputDir))
			}
			for i, envVar := range tt.wantEnvVars {
				tt.wantEnvVars[i] = filepath.FromSlash(strings.ReplaceAll(envVar, "{OUTPUT_DIR}", outputDir))
			}

			assert.Equal(t, tt.wantCmdLine, cmd.CmdLine)
			assert.Equal(t, tt.wantEnvVars, cmd.EnvVars)
			assert.Equal(t, tt.wantWorkDir, cmd.WorkDir)

			p.dataChan = make(chan *metrics.EventMetrics, 10)
			cmd.ProcessStreamingOutput([]byte("test_1_succeeded 1\n"))
			em := <-p.dataChan
			assert.Len(t, em.LabelsKeys(), len(tt.wantEMLabels))
			for _, label := range tt.wantEMLabels {
				assert.Equal(t, label[1], em.Label(label[0]), "label %s", label[0])
			}
		})
	}
}

func TestProbeOutputDirPath(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		target    endpoint.Endpoint
		ts        time.Time
		want      string
	}{
		{
			name:      "default",
			outputDir: "/tmp/output",
			ts:        time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			want:      "/tmp/output/2024-01-01/1704067200000",
		},
		{
			name:      "with_target",
			outputDir: "/tmp/output",
			target:    endpoint.Endpoint{Name: "test_target"},
			ts:        time.Date(2024, time.February, 2, 12, 30, 45, 0, time.UTC),
			want:      "/tmp/output/2024-02-02/1706877045000/test_target",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{outputDir: tt.outputDir}
			assert.Equal(t, filepath.FromSlash(tt.want), p.outputDirPath(tt.target, tt.ts))
		})
	}
}
