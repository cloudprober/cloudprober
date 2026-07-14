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

	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/payload"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/grpcext/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Probe delegates probe runs to a sidecar gRPC service.
type Probe struct {
	name   string
	opts   *options.Options
	c      *configpb.ProbeConf
	l      *logger.Logger
	conn   *grpc.ClientConn
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
	// runs still count as probe failures (total moves, success doesn't) so
	// that alerting on total/success keeps working during a sidecar outage;
	// internal_errors separates "sidecar broken" from "target down", and the
	// failure reason is logged.
	internalErrors    int64
	latency           metrics.LatencyValue
	validationFailure *metrics.Map[int64]
	payloadMetrics    []*metrics.EventMetrics
}

func (p *Probe) newResult() sched.ProbeResult {
	r := &probeResult{}
	if p.opts.LatencyDist != nil {
		r.latency = p.opts.LatencyDist.CloneDist()
	} else {
		r.latency = metrics.NewFloat(0)
	}
	if len(p.opts.Validators) > 0 {
		r.validationFailure = validators.ValidationFailureMap(p.opts.Validators)
	}
	return r
}

func (r *probeResult) Metrics(ts time.Time, _ int64, opts *options.Options) []*metrics.EventMetrics {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(r.total)).
		AddMetric("success", metrics.NewInt(r.success)).
		AddMetric(opts.LatencyMetricName, r.latency.Clone()).
		AddMetric("internal_errors", metrics.NewInt(r.internalErrors)).
		AddLabel("ptype", "external_grpc")
	if r.validationFailure != nil {
		em.AddMetric("validation_failure", r.validationFailure)
	}
	ems := append([]*metrics.EventMetrics{em}, r.payloadMetrics...)
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
	p.conn = conn
	p.client = configpb.NewProberClient(conn)

	p.payloadParser, err = payload.NewParser(p.c.GetOutputMetricsOptions(), p.l)
	if err != nil {
		conn.Close()
		return fmt.Errorf("payload parser init: %v", err)
	}

	if err := p.validateConfigWithSidecar(); err != nil {
		conn.Close()
		return err
	}
	return nil
}

// validateConfigWithSidecar asks the sidecar to check probe_type and config
// at init time, so misconfigurations (typo'd probe type, undecodable config)
// fail at startup instead of surfacing as internal errors every cycle. An
// unreachable sidecar is not an init error — it may simply not be up yet —
// and neither is a sidecar that doesn't implement the RPC.
func (p *Probe) validateConfigWithSidecar() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.client.ValidateConfig(ctx, &configpb.ValidateConfigRequest{
		ProbeType: p.c.GetProbeType(),
		Config:    []byte(p.c.GetConfig()),
	})
	switch status.Code(err) {
	case codes.OK, codes.Unimplemented:
		return nil
	case codes.NotFound, codes.InvalidArgument:
		return fmt.Errorf("external_grpc probe: sidecar %s rejected config: %v", p.c.GetServer(), err)
	default:
		p.l.Warningf("Could not validate config with sidecar %s (%v); proceeding, config errors will surface at probe time", p.c.GetServer(), err)
		return nil
	}
}

// isInternalRPCError classifies a failed Probe RPC using both the error and
// the connection's state read immediately after the RPC returned.
//
// DeadlineExceeded/Canceled normally mean the probe ran out of time while
// the sidecar's handler was running — a plain target failure, since
// reaching a running handler at all requires an already-healthy (Ready)
// transport. But the same codes also fire when the deadline was consumed by
// connection establishment instead: gRPC dials lazily, and a peer that
// accepts the TCP connection but never completes the gRPC handshake (a
// hung sidecar process, a misbehaving proxy) leaves the RPC blocked with no
// fast failure — it just rides out to our deadline. If the transport still
// isn't Ready by the time the RPC gives up, the deadline was spent on the
// connection, not the handler, so this is sidecar/infra, not target.
//
// A connection that's already given up (TransientFailure) doesn't need this
// treatment: gRPC's default fail-fast behavior returns Unavailable for that
// case immediately, without waiting out the deadline, and Unavailable is
// already unambiguous (falls through to the `return true` below).
//
// Situation                          RPC code            Post-RPC state  internal_errors?
// ---------------------------------  ------------------  --------------  ----------------
// Handler hung (shared deadline)     DeadlineExceeded    Ready           no (target fail)
// Hung handshake / mid-connect       DeadlineExceeded    not Ready       yes
// Blackhole / connect still spinning DeadlineExceeded    Connecting      yes
// Closed port / missing UDS          Unavailable         (any)           yes
// TransientFailure fail-fast         Unavailable         (any)           yes
// Any other non-OK status            *                   (any)           yes
func isInternalRPCError(err error, connState connectivity.State) bool {
	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.Canceled:
		return connState != connectivity.Ready
	}
	return true
}

func respError(resp *configpb.ProbeResponse) string {
	if e := resp.GetError(); e != "" {
		return e
	}
	return "no error reported by sidecar"
}

// runProbe runs one probe cycle for one target via the sidecar. oneShot
// marks one-off runs (RunOnce): the sidecar is told not to retain any
// per-target session for them.
func (p *Probe) runProbe(ctx context.Context, runReq *sched.RunProbeForTargetRequest, oneShot bool) {
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
		Config:   []byte(p.c.GetConfig()),
		Timeout:  durationpb.New(p.opts.Timeout),
		Interval: durationpb.New(p.opts.Interval),
		OneShot:  oneShot,
	}
	if target.IP != nil {
		req.Target.Ip = target.IP.String()
	}
	// state_handle is the wire image of the in-process TargetState: an opaque
	// per-target session handle that the sidecar mints and we simply echo
	// back each cycle. TargetState is scheduler-owned and is the only place
	// session handles live; one-shot runs get a throwaway runReq and don't
	// carry (or produce) a handle.
	if h, ok := runReq.TargetState.([]byte); ok {
		req.StateHandle = h
	}

	// Every run counts toward total, including sidecar/infra failures —
	// alerting works off total/success deltas, so a sidecar outage must
	// still register as failed runs. internal_errors tells the two apart.
	result.total++

	resp, err := p.client.Probe(ctx, req)
	if err != nil {
		if isInternalRPCError(err, p.conn.GetState()) {
			result.internalErrors++
		}
		p.l.Warningf("target(%s): probe via sidecar %s failed: %v", target.Name, p.c.GetServer(), err)
		runReq.LastRun.Set(false, 0, err)
		return
	}

	if h := resp.GetStateHandle(); len(h) > 0 {
		runReq.TargetState = h
	} else {
		// The sidecar clears the handle to signal the previous session is
		// gone (e.g. after InvalidateSession) — drop it here too, instead of
		// re-sending a handle the sidecar has already discarded.
		runReq.TargetState = nil
	}

	// Parse payload metrics regardless of the run's outcome — sidecars may
	// attach diagnostics to failed runs too.
	if len(resp.GetPayload()) > 0 {
		text := strings.Join(resp.GetPayload(), "\n")
		for _, em := range p.payloadParser.PayloadMetrics(&payload.Input{Text: []byte(text)}, target.Dst()) {
			result.payloadMetrics = append(result.payloadMetrics, em.AddLabel("ptype", "external_grpc"))
		}
	}

	if resp.GetInternalError() {
		result.internalErrors++
		err := fmt.Errorf("sidecar internal error: %s", respError(resp))
		p.l.Warningf("target(%s): %v", target.Name, err)
		runReq.LastRun.Set(false, 0, err)
		return
	}

	latency := resp.GetLatency().AsDuration()
	if latency < 0 {
		p.l.Warningf("target(%s): sidecar reported negative latency (%v), using 0", target.Name, latency)
		latency = 0
	}

	if !resp.GetSuccess() {
		p.l.Infof("target(%s): probe failed: %s", target.Name, respError(resp))
		runReq.LastRun.Set(false, latency, errors.New(respError(resp)))
		return
	}

	if result.validationFailure != nil {
		payloadText := strings.Join(resp.GetPayload(), "\n")
		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{ResponseBody: []byte(payloadText)}, result.validationFailure, p.l)
		if len(failedValidations) > 0 {
			err := fmt.Errorf("validation failed: %v", failedValidations)
			p.l.Infof("target(%s): %v", target.Name, err)
			runReq.LastRun.Set(false, latency, err)
			return
		}
	}

	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	runReq.LastRun.Set(true, latency, nil)
}

// RunOnce runs the probe just once, as a one-shot: the sidecar doesn't
// retain any per-target session for these runs.
func (p *Probe) RunOnce(ctx context.Context) []*singlerun.ProbeRunResult {
	p.l.Info("Running external_grpc probe once.")
	return sched.RunOnce(ctx, p.opts, func(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
		p.runProbe(ctx, runReq, true)
	})
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	s := &sched.Scheduler{
		ProbeName: p.name,
		DataChan:  dataChan,
		Opts:      p.opts,
		RunProbeForTarget: func(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
			p.runProbe(ctx, runReq, false)
		},
	}
	s.UpdateTargetsAndStartProbes(ctx)

	// UpdateTargetsAndStartProbes blocks until ctx is canceled (probe
	// removed / cloudprober shutting down); clean up the sidecar connection
	// now that nothing will use it again. Matches probes/grpc's Start.
	p.conn.Close()
}
