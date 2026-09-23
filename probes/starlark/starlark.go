// Copyright 2026 The Cloudprober Authors.
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

// Package starlark implements a Starlark probe type. The probe runs a
// user-supplied Starlark function once per resolved target each interval.
// Wall time of the call becomes latency. Clean return is success; any
// unhandled error (including assertion failures) is failure.
package starlark

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/payload"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/starlark/proto"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name    string
	opts    *options.Options
	c       *configpb.ProbeConf
	l       *logger.Logger
	runtime *runtime

	// payloadParser parses the per-run buffer of print_metric lines into
	// EventMetrics. Configured via the output_metrics_options proto field;
	// shared across targets (aggregation is keyed by target inside the
	// parser). Same surface as external + http probes use for custom
	// metrics.
	payloadParser *payload.Parser
}

type probeResult struct {
	total, success int64
	latency        metrics.LatencyValue
	// payloadMetrics accumulates EMs produced from print_metric lines until
	// the scheduler exports them via Metrics(). Mirrors http probe's
	// payloadMetrics field.
	payloadMetrics []*metrics.EventMetrics
}

func (p *Probe) newResult() sched.ProbeResult {
	r := &probeResult{}
	if p.opts.LatencyDist != nil {
		r.latency = p.opts.LatencyDist.CloneDist()
	} else {
		r.latency = metrics.NewFloat(0)
	}
	return r
}

func (r *probeResult) Metrics(ts time.Time, _ int64, opts *options.Options) []*metrics.EventMetrics {
	ems := []*metrics.EventMetrics{
		metrics.NewEventMetrics(ts).
			AddMetric("total", metrics.NewInt(r.total)).
			AddMetric("success", metrics.NewInt(r.success)).
			AddMetric(opts.LatencyMetricName, r.latency.Clone()).
			AddLabel("ptype", "starlark"),
	}
	ems = append(ems, r.payloadMetrics...)
	r.payloadMetrics = nil
	return ems
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not a starlark probe config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c

	source, err := loadSource(p.c)
	if err != nil {
		return err
	}

	entryPoint := p.c.GetEntryPoint()
	if entryPoint == "" {
		entryPoint = "probe"
	}

	var tlsCfgs map[string]*tls.Config
	if len(p.c.GetTlsConfigs()) > 0 {
		tlsCfgs = make(map[string]*tls.Config, len(p.c.GetTlsConfigs()))
		for cfgName, c := range p.c.GetTlsConfigs() {
			// An empty name is unreachable: tls="" is an error, and an omitted
			// tls uses system defaults without consulting this map. Reject it
			// so a config that can never be selected fails loudly at Init.
			if cfgName == "" {
				return fmt.Errorf("starlark tls_configs: empty config name; every entry must have a non-empty key")
			}
			tc, err := tlsconfig.FromProto(c)
			if err != nil {
				return fmt.Errorf("starlark tls_configs[%q]: %v", cfgName, err)
			}
			tlsCfgs[cfgName] = tc
		}
	}

	oauthID, err := newOAuthIdentities(p.c.GetOauthConfigs(), p.l)
	if err != nil {
		return err
	}

	// Timeout to compile the starlark script. This is a generous timeout
	// to avoid accidental infinite loops in the script.
	loadTimeout := 30 * time.Second
	loadCtx, cancel := context.WithTimeout(context.Background(), loadTimeout)
	defer cancel()

	rt, err := newRuntime(loadCtx, &runtimeOpts{
		name:       name,
		source:     source,
		entryPoint: entryPoint,
		vars:       p.c.GetVars(),
		tlsCfgs:    tlsCfgs,
		oauth:      oauthID,
		l:          p.l,
	})
	if err != nil {
		return fmt.Errorf("starlark compile error: %v", err)
	}
	p.runtime = rt

	// payload.NewParser tolerates a nil opts (zero-value defaults: GAUGE
	// kind, no aggregation, no distributions, no JSON / header metrics),
	// so print_metric works out of the box for simple counters / gauges.
	// Users wanting CUMULATIVE / dist buckets / aggregation set
	// output_metrics_options in proto.
	p.payloadParser, err = payload.NewParser(p.c.GetOutputMetricsOptions(), p.l)
	if err != nil {
		return fmt.Errorf("payload parser init: %v", err)
	}
	return nil
}

func loadSource(c *configpb.ProbeConf) (string, error) {
	src, file := c.GetSource(), c.GetSourceFile()
	if (src == "") == (file == "") {
		return "", fmt.Errorf("exactly one of source or source_file must be set")
	}
	if src != "" {
		return src, nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("reading source_file %q: %v", file, err)
	}
	return string(b), nil
}

func (p *Probe) runProbe(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
	if runReq.Result == nil {
		runReq.Result = p.newResult()
	}
	if runReq.TargetState == nil {
		runReq.TargetState = newStateBucket()
	}
	target, result, bucket := runReq.Target, runReq.Result.(*probeResult), runReq.TargetState.(*stateBucket)
	l := p.l.WithAttributes(slog.String("target", target.Name))

	result.total++

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target, "", 0)
	}

	runCtx, cancel := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancel()
	// Stream print_metric output through the parser as the script runs.
	// Metrics emitted before a later script error survive — the parser
	// has already produced their EMs and they're already on the result.
	// Malformed lines log a warning inside the parser; we don't fail the
	// run for them, matching external + http probe behavior.
	dst := target.Dst()
	emit := metricEmitFn(func(line string) {
		for _, em := range p.payloadParser.PayloadMetrics(&payload.Input{Text: []byte(line)}, dst) {
			result.payloadMetrics = append(result.payloadMetrics, em.AddLabel("ptype", "starlark"))
		}
	})
	start := time.Now()
	finalL, err := p.runtime.Run(runCtx, target, l, bucket, emit)
	latency := time.Since(start)

	if err != nil {
		finalL.Error(err.Error())
		runReq.LastRun.Set(false, 0, err)
		return
	}
	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	runReq.LastRun.Set(true, latency, nil)
}

// RunOnce runs the probe just once.
func (p *Probe) RunOnce(ctx context.Context) []*singlerun.ProbeRunResult {
	p.l.Info("Running starlark probe once.")
	return sched.RunOnce(ctx, p.opts, p.runProbe)
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	s := &sched.Scheduler{
		ProbeName:         p.name,
		DataChan:          dataChan,
		Opts:              p.opts,
		RunProbeForTarget: p.runProbe,
	}
	s.UpdateTargetsAndStartProbes(ctx)
}
