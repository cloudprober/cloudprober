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

// Package script implements a Starlark script probe type. The probe runs a
// user-supplied Starlark function once per resolved target each interval.
// Wall time of the call becomes latency. Clean return is success; any
// unhandled error (including assertion failures) is failure.
package script

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/script/proto"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name    string
	opts    *options.Options
	c       *configpb.ProbeConf
	l       *logger.Logger
	runtime *Runtime
}

type probeResult struct {
	total, success int64
	latency        metrics.LatencyValue
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
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(r.total)).
		AddMetric("success", metrics.NewInt(r.success)).
		AddMetric(opts.LatencyMetricName, r.latency.Clone()).
		AddLabel("ptype", "script")
	return []*metrics.EventMetrics{em}
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not a script probe config")
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

	rt, err := NewRuntime(name, source, entryPoint, p.l)
	if err != nil {
		return fmt.Errorf("script compile error: %v", err)
	}
	p.runtime = rt
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
	target, result := runReq.Target, runReq.Result.(*probeResult)
	l := p.l.WithAttributes(slog.String("target", target.Name))

	result.total++

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target, "", 0)
	}

	start := time.Now()
	err := p.runtime.Run(ctx, target)
	latency := time.Since(start)

	if err != nil {
		l.Error(err.Error())
		runReq.LastRun.Set(false, 0, err)
		return
	}
	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	runReq.LastRun.Set(true, latency, nil)
}

// RunOnce runs the probe just once.
func (p *Probe) RunOnce(ctx context.Context) []*singlerun.ProbeRunResult {
	p.l.Info("Running script probe once.")
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
