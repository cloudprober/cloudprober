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
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"text/template"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/payload"
	configpb "github.com/cloudprober/cloudprober/probes/browser/proto"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	testSpec string
	// book-keeping params
	targets              []endpoint.Endpoint
	workdir              string
	playwrightConfigPath string
	reporterPath         string
	payloadParser        *payload.Parser
	dataChan             chan *metrics.EventMetrics

	runID   map[string]int64
	runIDMu sync.Mutex
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
func (prr probeRunResult) Metrics(ts time.Time, _ int64, opts *options.Options) *metrics.EventMetrics {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", &prr.total).
		AddMetric("success", &prr.success).
		AddMetric(opts.LatencyMetricName, prr.latency.Clone())

	if prr.validationFailure != nil {
		em.AddMetric("validation_failure", prr.validationFailure)
	}

	return em
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

	totalDuration := time.Duration(p.c.GetRequestsIntervalMsec()*p.c.GetRequestsPerProbe())*time.Millisecond + p.opts.Timeout
	if totalDuration > p.opts.Interval {
		return fmt.Errorf("invalid config - executing all requests will take "+
			"longer than the probe interval, i.e. "+
			"requests_per_probe*requests_interval_msec + timeout (%s) > interval (%s)",
			totalDuration, p.opts.Interval)
	}

	p.targets = p.opts.Targets.ListEndpoints()

	p.workdir = p.c.GetWorkdir()
	if p.c.GetWorkdir() == "" {
		d, err := os.MkdirTemp("", "cloudprober_"+p.name)
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %v", err)
		}
		p.workdir = d
	}

	payloadParser, err := payload.NewParser(p.c.GetOutputMetricsOptions(), p.l)
	if err != nil {
		return fmt.Errorf("failed to create payload parser: %v", err)
	}
	p.payloadParser = payloadParser

	// Set up test spec
	p.testSpec = p.c.GetTestSpec()
	if p.testSpec == "" {
		if p.c.GetTestSpecFile() == "" {
			return fmt.Errorf("test spec is not provided")
		}
		b, err := os.ReadFile(p.c.GetTestSpecFile())
		if err != nil {
			return fmt.Errorf("failed to read test spec from file (%s): %v", p.c.GetTestSpecFile(), err)
		}
		p.testSpec = string(b)
	}
	testSpecPath := filepath.Join(p.workdir, "test.spec.ts")
	if err := os.WriteFile(testSpecPath, []byte(p.testSpec), 0644); err != nil {
		return fmt.Errorf("failed to write test spec to workdir (%s): %v", p.workdir, err)
	}

	// Set up playwright config in workdir
	data := struct {
		Screenshot string
		Trace      string
	}{
		Screenshot: "only-on-failure",
		Trace:      "off",
	}
	if p.c.GetEnableScreenshotsForSuccess() {
		data.Screenshot = "on"
	}
	if p.c.GetEnableTraces() {
		data.Trace = "on"
	}

	configTmpl := template.Must(template.ParseFS(templates, "*/playwright.config.ts.tmpl"))
	var buf bytes.Buffer
	if err := configTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to generate playwright config: %v", err)
	}
	configPath := filepath.Join(p.workdir, "playwright.config.ts")
	if err := os.WriteFile(configPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write playwright config: %v", err)
	}
	p.playwrightConfigPath = configPath

	// Set up reporter in workdir
	reporter, err := templates.ReadFile("templates/cloudprober-reporter.ts")
	if err != nil {
		panic(fmt.Sprintf("failed to read reporter from the templates: %v", err))
	}
	p.reporterPath = filepath.Join(p.workdir, "cloudprober-reporter.ts")
	if err := os.WriteFile(p.reporterPath, reporter, 0644); err != nil {
		return fmt.Errorf("failed to write reporter to workdir: %v", err)
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

func (p *Probe) runPWTest(ctx context.Context, target endpoint.Endpoint, result *probeRunResult, resultMu *sync.Mutex) {
	start := time.Now()
	runID := p.generateRunID(target)

	outputDir := filepath.Join(p.workdir, target.Name, fmt.Sprintf("%d_%d", runID, time.Now().Unix()))

	command := []string{
		"npx",
		"playwright",
		"test",
		"--config=" + p.playwrightConfigPath,
		"--output=" + outputDir,
		"--reporter=" + p.reporterPath,
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = p.c.GetPlaywrightDir()
	cmd.Env = append(os.Environ(), fmt.Sprintf("NODE_PATH=%s", filepath.Join(p.c.GetPlaywrightDir(), "node_modules")))

	p.l.Infof("Target(%s): running command %v, in dir %s", target.Name, cmd.String(), cmd.Dir)

	out, err := cmd.Output()
	p.l.Debugf("Target(%s): command output: %s", target.Name, string(out))
	if err != nil {
		attrs := []slog.Attr{slog.String("target", target.Name), slog.String("err", err.Error()), slog.String("stdout", string(out))}
		if exitErr, ok := err.(*exec.ExitError); ok {
			attrs = append(attrs, slog.String("stderr", string(exitErr.Stderr)))
		}
		p.l.WarningAttrs("failed to run playwright test", attrs...)
		return
	}
	latency := time.Since(start)

	if resultMu != nil {
		resultMu.Lock()
		defer resultMu.Unlock()
	}

	result.success.Inc()
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())

	for _, em := range p.payloadParser.PayloadMetrics(&payload.Input{Text: out}, target.Dst()) {
		em.AddLabel("ptype", "browser").AddLabel("probe", p.name).AddLabel("dst", target.Dst()).AddLabel("run_id", strconv.FormatInt(runID, 10))
		p.opts.RecordMetrics(target, em, p.dataChan, options.WithNoAlert())
	}
}

func (p *Probe) runProbe(ctx context.Context, target endpoint.Endpoint, res sched.ProbeResult) {
	// Convert interface to struct type
	result := res.(*probeRunResult)

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
	p.dataChan = dataChan

	s := &sched.Scheduler{
		ProbeName:         p.name,
		DataChan:          dataChan,
		Opts:              p.opts,
		NewResult:         func(_ *endpoint.Endpoint) sched.ProbeResult { return p.newResult() },
		RunProbeForTarget: p.runProbe,
	}
	s.UpdateTargetsAndStartProbes(ctx)
}
