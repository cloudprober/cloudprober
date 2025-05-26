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

/*
Package browser implements a Browser probe.
*/
package browser

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/payload"
	payload_configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/cloudprober/cloudprober/probes/browser/artifacts"
	"github.com/cloudprober/cloudprober/probes/browser/artifacts/storage"
	"github.com/cloudprober/cloudprober/probes/browser/artifacts/web"
	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
	"github.com/cloudprober/cloudprober/probes/common/command"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"google.golang.org/protobuf/proto"
)

const playwrightReportDir = "_playwright_report"

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// book-keeping params
	targets              []endpoint.Endpoint
	testDir              string
	testSpecArgs         []string
	workdir              string
	outputDir            string
	playwrightDir        string
	playwrightConfigPath string
	reporterPath         string
	payloadParser        *payload.Parser
	dataChan             chan *metrics.EventMetrics
	artifactsHandler     *artifacts.ArtifactsHandler
	workdirCleanup       *artifacts.CleanupHandler

	runID   map[string]int64
	runIDMu sync.Mutex

	// startCtx is the context used to start the probe. We use this context to
	// control various background processes started by the probe.
	startCtx context.Context
}

// embed templates dir
//
//go:embed templates
var templates embed.FS

// probeRunResult captures the results of a single probe run. The way we work with
// stats makes sure that probeRunResult and its fields are not accessed concurrently
// (see documentation with statsKeeper below). That's the reason we use metrics.Int
// types instead of metrics.AtomicInt.
type probeRunResult struct {
	total             metrics.Int
	success           metrics.Int
	latency           metrics.LatencyValue
	validationFailure *metrics.Map[int64]
}

func (p *Probe) newResult() sched.ProbeResult {
	result := &probeRunResult{}

	if p.opts.Validators != nil {
		result.validationFailure = validators.ValidationFailureMap(p.opts.Validators)
	}

	if p.opts.LatencyDist != nil {
		result.latency = p.opts.LatencyDist.CloneDist()
	} else {
		result.latency = metrics.NewFloat(0)
	}

	return result
}

// Metrics converts probeRunResult into metrics.EventMetrics object
func (prr probeRunResult) Metrics(ts time.Time, _ int64, opts *options.Options) []*metrics.EventMetrics {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", &prr.total).
		AddMetric("success", &prr.success).
		AddMetric(opts.LatencyMetricName, prr.latency.Clone()).
		AddLabel("ptype", "browser")

	if prr.validationFailure != nil {
		em.AddMetric("validation_failure", prr.validationFailure)
	}

	return []*metrics.EventMetrics{em}
}

func (p *Probe) playwrightGlobalTimeoutMsec() int64 {
	timeout := p.opts.Timeout.Milliseconds()
	// For multiple requests per probe, last request's effective timeout will be less
	timeout -= int64(p.c.GetRequestsIntervalMsec() * (p.c.GetRequestsPerProbe() - 1))

	// Keep some buffer for playwright to finish up the test.
	buffer := 0.1 * float64(timeout)
	if buffer > 2000 {
		buffer = 2000
	}
	return timeout - int64(buffer)
}

func (p *Probe) initTemplateFile(templates embed.FS, fileName string, data any) (string, error) {
	fileTmpl := template.Must(template.ParseFS(templates, "*/"+fileName+".tmpl"))
	var buf bytes.Buffer
	if err := fileTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to generate file (%s) from template: %v", fileName, err)
	}
	filePath := filepath.Join(p.workdir, fileName)
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write file (%s): %v", filePath, err)
	}
	return filePath, nil
}

func (p *Probe) initTemplates() error {
	// Set up playwright config in workdir
	data := struct {
		TestDir            string
		GlobalTimeoutMsec  int64
		Screenshot         string
		Trace              string
		EnableStepMetrics  bool
		DisableTestMetrics bool
	}{
		TestDir:            p.testDirPath(),
		GlobalTimeoutMsec:  p.playwrightGlobalTimeoutMsec(),
		Screenshot:         "only-on-failure",
		Trace:              "off",
		EnableStepMetrics:  p.c.GetTestMetricsOptions().GetEnableStepMetrics(),
		DisableTestMetrics: p.c.GetTestMetricsOptions().GetDisableTestMetrics(),
	}
	if p.c.GetSaveScreenshotsForSuccess() {
		data.Screenshot = "on"
	}
	if p.c.GetSaveTraces() {
		data.Trace = "on"
	}

	configPath, err := p.initTemplateFile(templates, "playwright.config.ts", data)
	if err != nil {
		return fmt.Errorf("failed to create playwright config: %v", err)
	}
	p.playwrightConfigPath = configPath

	// Set up reporter in workdir
	reporterPath, err := p.initTemplateFile(templates, "cloudprober-reporter.ts", data)
	if err != nil {
		return fmt.Errorf("failed to create cloudprober-reporter: %v", err)
	}
	p.reporterPath = reporterPath

	return nil
}

func (p *Probe) computeTestSpecArgs() []string {
	args := []string{}

	if p.c.GetTestSpecFilter() != nil {
		if p.c.GetTestSpecFilter().GetInclude() != "" {
			args = append(args, "--grep="+p.c.GetTestSpecFilter().GetInclude())
		}
		if p.c.GetTestSpecFilter().GetExclude() != "" {
			args = append(args, "--grep-invert="+p.c.GetTestSpecFilter().GetExclude())
		}
	}

	for _, ts := range p.c.GetTestSpec() {
		// If test spec is a regex, skip modifying it.
		if strings.ContainsAny(ts, "^$*") {
			args = append(args, ts)
			continue
		}
		if !filepath.IsAbs(ts) {
			ts = filepath.Join(p.testDirPath(), ts)
		}
		args = append(args, "^"+regexp.QuoteMeta(ts)+"$")
	}

	return args
}

// testDirPath returns the test directory. If not specified, it returns the
// directory of the config file.
// Note that this function is not concurrency-safe, which is fine since it is
// called only during initialization.
func (p *Probe) testDirPath() string {
	if p.testDir != "" {
		return p.testDir
	}
	p.testDir = p.c.GetTestDir()
	if p.c.TestDir == nil {
		p.testDir = filepath.Dir(state.ConfigFilePath())
	}
	return p.testDir
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("no browser probe config: %T", opts.ProbeConf)
	}

	p.c = c
	if p.c == nil {
		p.c = &configpb.ProbeConf{}
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.runID = make(map[string]int64)

	p.testSpecArgs = p.computeTestSpecArgs()

	totalDuration := time.Duration(p.c.GetRequestsIntervalMsec()*p.c.GetRequestsPerProbe())*time.Millisecond + p.opts.Timeout
	if totalDuration > p.opts.Interval {
		return fmt.Errorf("invalid config - executing all requests will take "+
			"longer than the probe interval, i.e. "+
			"requests_per_probe*requests_interval_msec + timeout (%s) > interval (%s)",
			totalDuration, p.opts.Interval)
	}

	p.targets = p.opts.Targets.ListEndpoints()

	p.playwrightDir = p.c.GetPlaywrightDir()
	if p.playwrightDir == "" && os.Getenv("PLAYWRIGHT_DIR") != "" {
		p.playwrightDir = os.Getenv("PLAYWRIGHT_DIR")
	}
	if p.playwrightDir == "" {
		return fmt.Errorf("playwrightDir is not provided through config or PLAYWRIGHT_DIR env variable")
	}

	p.workdir = p.c.GetWorkdir()
	if p.c.GetWorkdir() == "" {
		d, err := os.MkdirTemp("", "cloudprober_"+p.name)
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %v", err)
		}
		p.workdir = d
	}
	p.outputDir = filepath.Join(p.workdir, "output")

	if !p.c.GetTestMetricsOptions().GetDisableTestMetrics() {
		omo := &payload_configpb.OutputMetricsOptions{
			// All our metrics start with "test_".
			LineAcceptRegex: proto.String(`^test_.+`),
		}
		if !p.c.GetTestMetricsOptions().GetDisableAggregation() {
			omo.MetricsKind = payload_configpb.OutputMetricsOptions_CUMULATIVE.Enum()
			omo.AggregateInCloudprober = proto.Bool(true)
		}
		payloadParser, err := payload.NewParser(omo, p.l)
		if err != nil {
			return fmt.Errorf("failed to create payload parser: %v", err)
		}
		p.payloadParser = payloadParser
	}

	if err := p.initTemplates(); err != nil {
		return fmt.Errorf("failed to initialize templates: %v", err)
	}

	// We strip unnecessary /_playwright_report/ subdirectory from the path.
	destPathFn := storage.RemovePathSegmentFn(p.outputDir, playwrightReportDir)

	ah, err := artifacts.InitArtifactsHandler(context.Background(), p.c.GetArtifactsOptions(), p.outputDir, p.opts, destPathFn, p.l)
	if err != nil {
		return fmt.Errorf("failed to initialize artifacts handler: %v", err)
	}
	p.artifactsHandler = ah

	if p.c.GetWorkdirCleanupOptions() != nil {
		ch, err := artifacts.NewCleanupHandler(p.outputDir, p.c.GetWorkdirCleanupOptions(), p.l)
		if err != nil {
			return fmt.Errorf("failed to initialize cleanup handler: %v", err)
		}
		p.workdirCleanup = ch
	}

	return nil
}

func (p *Probe) generateRunID(target endpoint.Endpoint) int64 {
	p.runIDMu.Lock()
	defer p.runIDMu.Unlock()
	key := target.Key()
	runID := p.runID[key]
	p.runID[key]++
	return runID
}

func targetEnvVars(target endpoint.Endpoint) []string {
	if target.Name == "" {
		return nil
	}
	envVars := []string{
		fmt.Sprintf("target_name=%s", target.Name),
		fmt.Sprintf("target_ip=%s", target.IP.String()),
		fmt.Sprintf("target_port=%d", target.Port),
	}
	for k, v := range target.Labels {
		envVars = append(envVars, fmt.Sprintf("target_label_%s=%s", k, v))
	}
	return envVars
}

func (p *Probe) outputDirPath(target endpoint.Endpoint, ts time.Time) string {
	outputDirPath := []string{p.outputDir, ts.Format("2006-01-02"), strconv.FormatInt(ts.UnixMilli(), 10)}
	if target.Name != "" {
		outputDirPath = append(outputDirPath, target.Name)
	}
	return filepath.Join(outputDirPath...)
}

func (p *Probe) prepareCommand(target endpoint.Endpoint, ts time.Time) (*command.Command, string) {
	runID := p.generateRunID(target)

	outputDir := p.outputDirPath(target, ts)
	reportDir := filepath.Join(outputDir, playwrightReportDir)

	envVars := []string{
		fmt.Sprintf("NODE_PATH=%s", filepath.Join(p.playwrightDir, "node_modules")),
		fmt.Sprintf("PLAYWRIGHT_HTML_REPORT=%s", reportDir),
		"PLAYWRIGHT_HTML_OPEN=never",
	}
	for k, v := range p.c.GetEnvVar() {
		if v == "" {
			v = "1" // default to a truthy value
		}
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}
	envVars = append(envVars, targetEnvVars(target)...)

	cmdLine := []string{
		p.c.GetNpxPath(),
		"playwright",
		"test",
		"--config=" + p.playwrightConfigPath,
		"--output=" + filepath.Join(outputDir, "results"),
		"--reporter=html," + p.reporterPath,
	}
	cmdLine = append(cmdLine, p.testSpecArgs...)

	p.l.Infof("Running command line: %v", cmdLine)
	cmd := &command.Command{
		CmdLine: cmdLine,
		WorkDir: p.playwrightDir,
		EnvVars: envVars,
	}
	cmd.ProcessStreamingOutput = func(line []byte) {
		if p.payloadParser == nil {
			return
		}
		for _, em := range p.payloadParser.PayloadMetrics(&payload.Input{Text: line}, target.Dst()) {
			em.AddLabel("ptype", "browser").AddLabel("probe", p.name).AddLabel("dst", target.Dst())
			if p.c.GetTestMetricsOptions().GetDisableAggregation() {
				em.AddLabel("run_id", fmt.Sprintf("%d", runID))
			}
			p.opts.RecordMetrics(target, em, p.dataChan)
		}
	}

	return cmd, reportDir
}

func (p *Probe) runPWTest(ctx context.Context, target endpoint.Endpoint, result *probeRunResult, resultMu *sync.Mutex) {
	startTime := time.Now()

	cmd, reportDir := p.prepareCommand(target, startTime)
	_, err := cmd.Execute(ctx, p.l)

	if err != nil {
		p.l.Errorf("error running playwright test: %v", err)
		if err := os.WriteFile(filepath.Join(reportDir, web.FailureMarkerFile), []byte("1"), 0644); err != nil {
			p.l.Errorf("error writing failure marker in %s: %v", reportDir, err)
		}
	}

	// We use startCtx here to make sure artifactsHandler keeps running (if
	// required) even after this probe run.
	p.artifactsHandler.Handle(p.startCtx, reportDir)

	if err != nil {
		return
	}

	if resultMu != nil {
		resultMu.Lock()
		defer resultMu.Unlock()
	}

	result.success.Inc()
	result.latency.AddFloat64(time.Since(startTime).Seconds() / p.opts.LatencyUnit.Seconds())
}

func (p *Probe) runProbe(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
	if runReq.Result == nil {
		runReq.Result = p.newResult()
	}
	target, result := runReq.Target, runReq.Result.(*probeRunResult)

	port := target.Port
	result.total.IncBy(int64(p.c.GetRequestsPerProbe()))

	ipLabel := ""

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target, ipLabel, port)
	}

	if p.c.GetRequestsPerProbe() == 1 {
		p.runPWTest(ctx, target, result, nil)
		return
	}

	// For multiple requests per probe, we launch a separate goroutine for each
	// DNS request. We use a mutex to protect access to per-target result object
	// in doPWRequest. Note that result object is not accessed concurrently
	// anywhere else -- export of metrics happens when probe is not running.
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < int(p.c.GetRequestsPerProbe()); i++ {
		wg.Add(1)
		go func(reqNum int, result *probeRunResult) {
			defer wg.Done()

			time.Sleep(time.Duration(reqNum*int(p.c.GetRequestsIntervalMsec())) * time.Millisecond)
			p.runPWTest(ctx, target, result, &resultMu)
		}(i, result)
	}
	p.l.Debug("Waiting for Browser requests to finish")
	wg.Wait()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	p.startCtx = ctx

	if p.workdirCleanup != nil {
		go p.workdirCleanup.Start(p.startCtx)
	}
	if p.artifactsHandler != nil {
		// Starts cleanup handlers in goroutines.
		p.artifactsHandler.StartCleanup(p.startCtx)
	}

	p.dataChan = dataChan

	s := &sched.Scheduler{
		ProbeName:         p.name,
		DataChan:          dataChan,
		Opts:              p.opts,
		RunProbeForTarget: p.runProbe,
	}
	s.UpdateTargetsAndStartProbes(ctx)
}
