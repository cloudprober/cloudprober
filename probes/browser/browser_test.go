// Copyright 2024-2025 The Cloudprober Authors.
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
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestProbePrepareCommand(t *testing.T) {
	// npx is a real, resolvable executable that stands in for npx so the Init
	// preflight (exec.LookPath) passes. playwrightDir must contain
	// node_modules/@playwright/test for the preflight to pass too.
	npx := os.Args[0]
	pwDir, appDir := t.TempDir(), t.TempDir()
	for _, d := range []string{pwDir, appDir} {
		if err := os.MkdirAll(filepath.Join(d, "node_modules", "@playwright", "test"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PLAYWRIGHT_DIR", pwDir)

	baseEnvVars := func(pwDir string) []string {
		return []string{"NODE_PATH=" + pwDir + "/node_modules", "PLAYWRIGHT_HTML_REPORT={OUTPUT_DIR}/" + playwrightReportDir, "PLAYWRIGHT_HTML_OPEN=never"}
	}

	cmdLine := func(npxPath string) []string {
		return []string{npxPath, "playwright", "test", "--config={WORKDIR}/playwright.config.ts", "--output=${OUTPUT_DIR}/results", "--reporter=html,{WORKDIR}/cloudprober-reporter.ts"}
	}

	baseWantEMLabels := [][2]string{{"ptype", "browser"}, {"probe", "test_browser"}, {"dst", ""}}

	testDir := "/tests"

	tests := []struct {
		name               string
		disableAggregation bool
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
			wantCmdLine:  cmdLine(npx),
			wantEnvVars:  baseEnvVars(pwDir),
			wantWorkDir:  pwDir,
			wantEMLabels: baseWantEMLabels,
		},
		{
			name:         "with_target",
			target:       endpoint.Endpoint{Name: "test_target", IP: net.ParseIP("12.12.12.12"), Port: 9313, Labels: map[string]string{"env": "prod"}},
			wantCmdLine:  cmdLine(npx),
			wantEnvVars:  append(baseEnvVars(pwDir), "target_name=test_target", "target_ip=12.12.12.12", "target_port=9313", "target_label_env=prod"),
			wantWorkDir:  pwDir,
			wantEMLabels: [][2]string{{"ptype", "browser"}, {"probe", "test_browser"}, {"dst", "test_target:9313"}},
		},
		{
			name:               "disable_aggregation",
			disableAggregation: true,
			wantCmdLine:        cmdLine(npx),
			wantEnvVars:        baseEnvVars(pwDir),
			wantWorkDir:        pwDir,
			wantEMLabels:       append(baseWantEMLabels, [2]string{"run_id", "0"}),
		},
		{
			name:          "with_playwright_dir",
			playwrightDir: appDir,
			wantCmdLine:   cmdLine(npx),
			wantEnvVars:   baseEnvVars(appDir),
			wantWorkDir:   appDir,
			wantEMLabels:  baseWantEMLabels,
		},
		{
			name:         "with_test_spec",
			testSpec:     []string{"test_spec_1", "test_spec_2"},
			wantCmdLine:  append(cmdLine(npx), "^.*/test_spec_1$", "^.*/test_spec_2$"),
			wantEnvVars:  baseEnvVars(pwDir),
			wantWorkDir:  pwDir,
			wantEMLabels: baseWantEMLabels,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &configpb.ProbeConf{
				TestSpec: tt.testSpec,
				TestDir:  &testDir,
				NpxPath:  proto.String(npx),
				TestMetricsOptions: &configpb.TestMetricsOptions{
					DisableAggregation: &tt.disableAggregation,
				},
			}
			if tt.playwrightDir != "" {
				conf.PlaywrightDir = &tt.playwrightDir
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
				if runtime.GOOS == "windows" {
					// For test specs, backslashes get escaped again by regexp.QuoteMeta.
					tt.wantCmdLine[i] = strings.ReplaceAll(tt.wantCmdLine[i], `.*\`, `.*\\`)
				}
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

func TestInitPreflight(t *testing.T) {
	// A playwright dir that has the @playwright/test package installed.
	pwOK := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pwOK, "node_modules", "@playwright", "test"), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		npxPath       string
		playwrightDir string
		wantErr       string
	}{
		{
			name:          "npx_missing",
			npxPath:       "cloudprober-nonexistent-npx-xyz",
			playwrightDir: pwOK,
			wantErr:       "npx not found",
		},
		{
			name:          "playwright_missing_is_not_fatal",
			npxPath:       os.Args[0],
			playwrightDir: t.TempDir(), // no node_modules/@playwright/test; warns, doesn't fail
			wantErr:       "",
		},
		{
			name:          "ok",
			npxPath:       os.Args[0],
			playwrightDir: pwOK,
			wantErr:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &configpb.ProbeConf{
				NpxPath:       proto.String(tt.npxPath),
				PlaywrightDir: proto.String(tt.playwrightDir),
			}
			opts := options.DefaultOptions()
			opts.ProbeConf = conf

			err := (&Probe{}).Init("test_browser", opts)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestRunProbeInternalError drives a full probe run through runProbe with a
// stub npx that fails with a Playwright missing-browser error on stderr, and
// asserts the run is counted as an internal_error: total advances, success
// stays put, internal_errors increments, and LastRun records the failure.
func TestRunProbeInternalError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub npx is a /bin/sh script; skip on Windows")
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "@playwright", "test"), 0755); err != nil {
		t.Fatal(err)
	}
	// Stub npx: ignore args, emit Playwright's missing-browser error to stderr,
	// exit non-zero.
	npx := filepath.Join(dir, "npx")
	script := "#!/bin/sh\n" +
		"echo \"browserType.launch: Executable doesn't exist at /root/.cache/ms-playwright/chromium/chrome\" 1>&2\n" +
		"exit 1\n"
	if err := os.WriteFile(npx, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	opts := options.DefaultOptions()
	opts.ProbeConf = &configpb.ProbeConf{
		NpxPath:       proto.String(npx),
		PlaywrightDir: proto.String(dir),
	}
	p := &Probe{}
	if err := p.Init("test_browser", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	p.startCtx = context.Background()
	p.dataChan = make(chan *metrics.EventMetrics, 10)

	runReq := &sched.RunProbeForTargetRequest{
		Target:  endpoint.Endpoint{Name: "t"},
		LastRun: &sched.LastRunResult{},
	}
	p.runProbe(context.Background(), runReq)

	result := runReq.Result.(*probeRunResult)
	assert.Equal(t, int64(1), result.total.Int64(), "total")
	assert.Equal(t, int64(0), result.success.Int64(), "success")
	assert.Equal(t, int64(1), result.internalErrors.Int64(), "internal_errors")
	assert.False(t, runReq.LastRun.Success, "LastRun.Success")
	assert.Error(t, runReq.LastRun.Error, "LastRun.Error")
}

func TestInternalErrorRe(t *testing.T) {
	browserMissing := "browserType.launch: Executable doesn't exist at /root/.cache/ms-playwright/chromium-1091/chrome-linux/chrome\n" +
		"Please run the following command to download new browsers:\nnpx playwright install"
	playwrightMissing := "npm error could not determine executable to run"

	assert.True(t, internalErrorRe.MatchString(browserMissing))
	assert.True(t, internalErrorRe.MatchString(playwrightMissing))
	assert.False(t, internalErrorRe.MatchString("Error: expect(received).toBe(expected)"))
}

func TestProbeOutputDirPath(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		target    endpoint.Endpoint
		targets   []endpoint.Endpoint
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
			name:      "single_target",
			outputDir: "/tmp/output",
			target:    endpoint.Endpoint{Name: "test_target"},
			targets:   []endpoint.Endpoint{{Name: "test_target"}},
			ts:        time.Date(2024, time.February, 2, 12, 30, 45, 0, time.UTC),
			want:      "/tmp/output/2024-02-02/1706877045000",
		},
		{
			name:      "multiple_targets",
			outputDir: "/tmp/output",
			target:    endpoint.Endpoint{Name: "test_target"},
			targets:   []endpoint.Endpoint{{Name: "test_target"}, {Name: "test_target_2"}},
			ts:        time.Date(2024, time.February, 2, 12, 30, 45, 0, time.UTC),
			want:      "/tmp/output/2024-02-02/1706877045000/test_target",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{outputDir: tt.outputDir, targets: tt.targets}
			assert.Equal(t, filepath.FromSlash(tt.want), p.outputDirPath(tt.target, tt.ts))
		})
	}
}

func TestProbeInitTemplates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows, path issues - not worth it")
	}

	tmpDir := t.TempDir()

	oldConfigFilePath := state.ConfigFilePath()
	defer state.SetConfigFilePath(oldConfigFilePath)
	state.SetConfigFilePath("/cfg/cloudprober.cfg")

	defaultConfigContains := []string{
		"testDir: \"/cfg\"",
		"screenshot: \"only-on-failure\"",
		"trace: \"off\"",
		"retries: 0",
	}
	reporterContainTestLevel := []string{
		"print(`test_status",
		"print(`test_latency",
	}
	reporterContainStepLevel := []string{
		"print(`test_step_status",
		"print(`test_step_latency",
	}

	tests := []struct {
		name                string
		conf                *configpb.ProbeConf
		configContains      []string
		reporterContains    []string
		reporterNotContains []string
	}{
		{
			name: "default",
			conf: &configpb.ProbeConf{
				Workdir: proto.String(tmpDir),
			},
			configContains:      defaultConfigContains,
			reporterContains:    reporterContainTestLevel,
			reporterNotContains: reporterContainStepLevel,
		},
		{
			name: "with_config_dir",
			conf: &configpb.ProbeConf{
				TestDir: proto.String("/cfg/tests"),
				Workdir: proto.String(tmpDir),
			},
			configContains: []string{
				"testDir: \"/cfg/tests\"",
				"screenshot: \"only-on-failure\"",
				"trace: \"off\"",
			},
			reporterContains:    reporterContainTestLevel,
			reporterNotContains: reporterContainStepLevel,
		},
		{
			name: "with_screenshots_and_traces",
			conf: &configpb.ProbeConf{
				Workdir:                   proto.String(tmpDir),
				SaveScreenshotsForSuccess: proto.Bool(true),
				SaveTrace:                 configpb.SaveOption_ALWAYS.Enum(),
			},
			configContains: []string{
				"screenshot: \"on\"",
				"trace: \"on\"",
			},
			reporterContains:    reporterContainTestLevel,
			reporterNotContains: reporterContainStepLevel,
		},
		{
			name: "with_retries",
			conf: &configpb.ProbeConf{
				Workdir:   proto.String(tmpDir),
				Retries:   proto.Int32(2),
				SaveTrace: configpb.SaveOption_ON_FIRST_RETRY.Enum(),
			},
			configContains: []string{
				"screenshot: \"only-on-failure\"",
				"trace: \"on-first-retry\"",
				"retries: 2",
			},
			reporterContains:    reporterContainTestLevel,
			reporterNotContains: reporterContainStepLevel,
		},
		{
			name: "with_deprecated_save_traces",
			conf: &configpb.ProbeConf{
				SaveTraces: proto.Bool(true),
			},
			configContains: []string{
				"trace: \"on\"",
			},
			reporterContains:    reporterContainTestLevel,
			reporterNotContains: reporterContainStepLevel,
		},
		{
			name: "save_trace_priority",
			conf: &configpb.ProbeConf{
				SaveTraces: proto.Bool(true),
				SaveTrace:  configpb.SaveOption_ON_FIRST_RETRY.Enum(),
			},
			configContains: []string{
				"trace: \"on-first-retry\"",
			},
			reporterContains:    reporterContainTestLevel,
			reporterNotContains: reporterContainStepLevel,
		},
		{
			name: "with_step_metrics",
			conf: &configpb.ProbeConf{
				Workdir: proto.String(tmpDir),
				TestMetricsOptions: &configpb.TestMetricsOptions{
					EnableStepMetrics: proto.Bool(true),
				},
			},
			configContains:   defaultConfigContains,
			reporterContains: append(reporterContainTestLevel, reporterContainStepLevel...),
		},
		{
			name: "disable_test_metrics",
			conf: &configpb.ProbeConf{
				Workdir: proto.String(tmpDir),
				TestMetricsOptions: &configpb.TestMetricsOptions{
					DisableTestMetrics: proto.Bool(true),
				},
			},
			configContains:      defaultConfigContains,
			reporterNotContains: append(reporterContainTestLevel, reporterContainStepLevel...),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				name:    "test_probe",
				c:       tt.conf,
				opts:    options.DefaultOptions(),
				workdir: tmpDir,
			}

			err := p.initTemplates()
			if err != nil {
				t.Fatalf("initTemplates() error = %v", err)
			}

			// Verify playwright config file
			got, err := os.ReadFile(p.playwrightConfigPath)
			if err != nil {
				t.Fatalf("Error reading playwright config: %v", err)
			}
			for _, want := range tt.configContains {
				assert.Contains(t, string(got), want, "playwright config should contain: %s", want)
			}

			// Verify reporter file
			got, err = os.ReadFile(p.reporterPath)
			if err != nil {
				t.Fatalf("Error reading playwright config: %v", err)
			}
			for _, want := range tt.reporterContains {
				assert.Contains(t, string(got), want, "reporter file should contain: %s", want)
			}
			for _, want := range tt.reporterNotContains {
				assert.NotContains(t, string(got), want, "reporter file should not contain: %s", want)
			}
		})
	}
}

func TestPlaywrightGlobalTimeoutMsec(t *testing.T) {
	tests := []struct {
		name                 string
		timeout              time.Duration
		requestsPerProbe     int
		requestsIntervalMsec int
		want                 int64
	}{
		{
			name:    "single_request",
			timeout: 10 * time.Second,
			want:    9000,
		},
		{
			name:                 "multiple_requests",
			timeout:              20 * time.Second,
			requestsPerProbe:     3,
			requestsIntervalMsec: 1000,
			want:                 16200, // (20s - (3-1)*1s) - 0.9s (buffer)
		},
		{
			name:    "large_buffer",
			timeout: 120 * time.Second,
			want:    118000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				opts: &options.Options{
					Timeout: tt.timeout,
				},
				c: &configpb.ProbeConf{
					RequestsPerProbe:     proto.Int32(int32(tt.requestsPerProbe)),
					RequestsIntervalMsec: proto.Int32(int32(tt.requestsIntervalMsec)),
				},
			}
			if got := p.playwrightGlobalTimeoutMsec(); got != tt.want {
				t.Errorf("playwrightGlobalTimeoutMsec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProbeComputeTestSpecArgs(t *testing.T) {
	tests := []struct {
		name          string
		testDir       string
		testSpec      []string
		filterInclude string
		filterExclude string
		wantArgs      []string
		wantArgsWin   []string
	}{
		{
			name:     "no_spec_no_filter",
			testDir:  "/tests",
			testSpec: nil,
			wantArgs: []string{},
		},
		{
			name:        "single_spec_relative",
			testDir:     "/tests",
			testSpec:    []string{"myspec.js"},
			wantArgs:    []string{`^.*/myspec\.js$`},
			wantArgsWin: []string{`^.*\\myspec\.js$`},
		},
		{
			name:        "single_spec_absolute",
			testDir:     "/tests",
			testSpec:    []string{"/abs/path/spec.js"},
			wantArgs:    []string{`^/abs/path/spec\.js$`},
			wantArgsWin: []string{`^\\abs\\path\\spec\.js$`},
		},
		{
			name:     "multiple_specs_mixed",
			testDir:  "/dir",
			testSpec: []string{"foo.js", "/bar/baz.js"},
			wantArgs: []string{
				`^.*/foo\.js$`,
				`^/bar/baz\.js$`,
			},
			wantArgsWin: []string{
				`^.*\\foo\.js$`,
				`^\\bar\\baz\.js$`,
			},
		},
		{
			name:     "regex_spec",
			testDir:  "/dir",
			testSpec: []string{`^foo.*\.js$`},
			wantArgs: []string{`^foo.*\.js$`},
		},
		{
			name:          "with_include_filter",
			testDir:       "/dir",
			testSpec:      []string{"foo.js"},
			filterInclude: "mytest",
			wantArgs: []string{
				"--grep=mytest",
				`^.*/foo\.js$`,
			},
			wantArgsWin: []string{
				"--grep=mytest",
				`^.*\\foo\.js$`,
			},
		},
		{
			name:          "with_exclude_filter",
			testDir:       "/dir",
			testSpec:      []string{"foo.js"},
			filterExclude: "skipme",
			wantArgs: []string{
				"--grep-invert=skipme",
				`^.*/foo\.js$`,
			},
			wantArgsWin: []string{
				"--grep-invert=skipme",
				`^.*\\foo\.js$`,
			},
		},
		{
			name:          "with_both_filters",
			testDir:       "/dir",
			testSpec:      []string{"foo.js"},
			filterInclude: "mytest",
			filterExclude: "skipme",
			wantArgs: []string{
				"--grep=mytest",
				"--grep-invert=skipme",
				`^.*/foo\.js$`,
			},
			wantArgsWin: []string{
				"--grep=mytest",
				"--grep-invert=skipme",
				`^.*\\foo\.js$`,
			},
		},
		{
			name:     "multiple_specs_with_regex",
			testDir:  "/dir",
			testSpec: []string{"foo.js", `^bar.*\.js$`},
			wantArgs: []string{
				`^.*/foo\.js$`,
				`^bar.*\.js$`,
			},
			wantArgsWin: []string{
				`^.*\\foo\.js$`,
				`^bar.*\.js$`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &configpb.ProbeConf{}
			for _, spec := range tt.testSpec {
				conf.TestSpec = append(conf.TestSpec, filepath.FromSlash(spec))
			}
			if tt.filterInclude != "" || tt.filterExclude != "" {
				conf.TestSpecFilter = &configpb.TestSpecFilter{}
				if tt.filterInclude != "" {
					conf.TestSpecFilter.Include = &tt.filterInclude
				}
				if tt.filterExclude != "" {
					conf.TestSpecFilter.Exclude = &tt.filterExclude
				}
			}
			p := &Probe{
				c:       conf,
				testDir: tt.testDir,
			}
			got := p.computeTestSpecArgs()
			if runtime.GOOS == "windows" {
				if tt.wantArgsWin == nil {
					tt.wantArgsWin = tt.wantArgs
				}
				assert.Equal(t, tt.wantArgsWin, got)
			} else {
				assert.Equal(t, tt.wantArgs, got)
			}
		})
	}
}
