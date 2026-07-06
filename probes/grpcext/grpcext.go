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

// Package grpcext implements the EXTERNAL_GRPC probe type. It delegates each
// per-target probe run to a sidecar gRPC service implementing the
// cloudprober.probes.grpcext.Prober service (proto/service.proto).
// Cloudprober keeps scheduling, targets, metrics, and surfacers; the sidecar
// runs the actual probe logic — typically heavyweight integrations that need
// real drivers (databases, message queues, etc.) — in its own process, on its
// own release cadence.
package grpcext

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/payload"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/grpcext/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Probe delegates probe runs to a sidecar gRPC service.
type Probe struct {
	name   string
	opts   *options.Options
	c      *configpb.ProbeConf
	l      *logger.Logger
	client configpb.ProberClient

	// payloadParser parses additional metrics returned by the sidecar
	// (payload-format lines) into EventMetrics. Same surface as external,
	// http, and starlark probes use for custom metrics.
	payloadParser *payload.Parser
}

type probeResult struct {
	total, success int64
	// internalErrors counts runs that failed because of sidecar/infra
	// problems (sidecar unreachable, sidecar-reported internal_error). Such
	// runs are not counted in total/success at all: "sidecar down" must not
	// look like "target down".
	internalErrors int64
	latency        metrics.LatencyValue
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
			AddMetric("internal_errors", metrics.NewInt(r.internalErrors)).
			AddLabel("ptype", "external_grpc"),
	}
	ems = append(ems, r.payloadMetrics...)
	r.payloadMetrics = nil
	return ems
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not an external_grpc probe config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c

	if p.c.GetServer() == "" {
		return errors.New("external_grpc probe: server must be specified")
	}
	if p.c.GetProbeType() == "" {
		return errors.New("external_grpc probe: probe_type must be specified")
	}

	// Sidecars are expected to be co-located (unix domain socket or
	// localhost). TODO(manugarg): Add mTLS support for TCP servers.
	conn, err := grpc.NewClient(p.c.GetServer(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("error creating gRPC client for %s: %v", p.c.GetServer(), err)
	}
	p.client = configpb.NewProberClient(conn)

	p.payloadParser, err = payload.NewParser(p.c.GetOutputMetricsOptions(), p.l)
	if err != nil {
		return fmt.Errorf("payload parser init: %v", err)
	}
	return nil
}

func (p *Probe) runProbe(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
	if runReq.Result == nil {
		runReq.Result = p.newResult()
	}
	target, result := runReq.Target, runReq.Result.(*probeResult)

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target, "", 0)
	}

	req := &configpb.ProbeRequest{
		ProbeType: p.c.GetProbeType(),
		Target: &configpb.Target{
			Name:   target.Name,
			Labels: target.Labels,
			Port:   int32(target.Port),
		},
		Config:  []byte(p.c.GetConfig()),
		Timeout: durationpb.New(p.opts.Timeout),
	}
	if target.IP != nil {
		req.Target.Ip = target.IP.String()
	}
	// state_handle is the wire image of the in-process TargetState: an opaque
	// per-target session handle that the sidecar mints and we simply echo
	// back each cycle.
	if h, ok := runReq.TargetState.([]byte); ok {
		req.StateHandle = h
	}

	resp, err := p.client.Probe(ctx, req)
	if err != nil {
		result.internalErrors++
		p.l.Warningf("target(%s): error talking to sidecar %s: %v", target.Name, p.c.GetServer(), err)
		runReq.LastRun.Set(false, 0, err)
		return
	}

	if len(resp.GetStateHandle()) > 0 {
		runReq.TargetState = resp.GetStateHandle()
	}

	if resp.GetInternalError() {
		result.internalErrors++
		err := fmt.Errorf("sidecar internal error: %s", resp.GetError())
		p.l.Warningf("target(%s): %v", target.Name, err)
		runReq.LastRun.Set(false, 0, err)
		return
	}

	if len(resp.GetPayload()) > 0 {
		text := strings.Join(resp.GetPayload(), "\n")
		for _, em := range p.payloadParser.PayloadMetrics(&payload.Input{Text: []byte(text)}, target.Dst()) {
			result.payloadMetrics = append(result.payloadMetrics, em.AddLabel("ptype", "external_grpc"))
		}
	}

	result.total++
	latency := resp.GetLatency().AsDuration()
	if !resp.GetSuccess() {
		p.l.Infof("target(%s): probe failed: %s", target.Name, resp.GetError())
		runReq.LastRun.Set(false, latency, errors.New(resp.GetError()))
		return
	}
	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	runReq.LastRun.Set(true, latency, nil)
}

// RunOnce runs the probe just once.
func (p *Probe) RunOnce(ctx context.Context) []*singlerun.ProbeRunResult {
	p.l.Info("Running external_grpc probe once.")
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
