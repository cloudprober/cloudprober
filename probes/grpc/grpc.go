// Copyright 2020-2023 The Cloudprober Authors.
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
Package grpc implements a gRPC probe.

This probes a cloudprober gRPC server and reports success rate, latency, and
validation failures.
*/
package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/cloudprober/cloudprober/common/iputils"
	"github.com/cloudprober/cloudprober/internal/oauth"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/internal/tlsconfig"
	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/grpc/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/probes/probeutils"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"google.golang.org/protobuf/proto"

	pb "github.com/cloudprober/cloudprober/internal/servers/grpc/proto"
	spb "github.com/cloudprober/cloudprober/internal/servers/grpc/proto"
	"github.com/fullstorydev/grpcurl"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/alts"
	"google.golang.org/grpc/credentials/insecure"
	grpcoauth "google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/resolver"

	// Import grpclb module so it can be used by name for DirectPath connections.
	_ "google.golang.org/grpc/balancer/grpclb"
)

const loadBalancingPolicy = `{"loadBalancingConfig":[{"grpclb":{"childPolicy":[{"pick_first":{}}]}}]}`

// TargetsUpdateInterval controls frequency of target updates.
var (
	TargetsUpdateInterval = 1 * time.Minute
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	src  string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// Number of connections as configured.
	numConns int

	dialOpts []grpc.DialOption
	creds    credentials.TransportCredentials
	descSrc  grpcurl.DescriptorSource

	// Targets and cancellation function for each target.
	targets     []endpoint.Endpoint
	cancelFuncs map[string]context.CancelFunc
	targetsMu   sync.Mutex

	// Results by target.
	results map[string]*probeRunResult

	// This is used only for testing.
	healthCheckFunc func() (*grpc_health_v1.HealthCheckResponse, error)
}

// probeRunResult captures the metrics for a single target. Multiple threads
// can update metrics at the same time and the main thread periodically
// outputs the values in this struct.
type probeRunResult struct {
	sync.Mutex
	target            string
	total             metrics.Int
	success           metrics.Int
	latency           metrics.LatencyValue
	connectErrors     metrics.Int
	validationFailure *metrics.Map[int64]
}

func (p *Probe) transportCredentials() (credentials.TransportCredentials, error) {
	if p.c.AltsConfig != nil && p.c.TlsConfig != nil {
		return nil, errors.New("only one of alts_config and tls_config can be set at a time")

	}
	altsCfg := p.c.GetAltsConfig()
	if altsCfg != nil {
		altsOpts := &alts.ClientOptions{
			TargetServiceAccounts:    altsCfg.GetTargetServiceAccount(),
			HandshakerServiceAddress: altsCfg.GetHandshakerServiceAddress(),
		}
		return alts.NewClientCreds(altsOpts), nil
	}
	if p.c.GetTlsConfig() != nil {
		tlsCfg := &tls.Config{}
		if err := tlsconfig.UpdateTLSConfig(tlsCfg, p.c.GetTlsConfig()); err != nil {
			return nil, fmt.Errorf("tls_config error: %v", err)
		}
		return credentials.NewTLS(tlsCfg), nil
	}
	if p.c.GetInsecureTransport() {
		return insecure.NewCredentials(), nil
	}

	// if no explicit transport creds configured, use system default.
	return credentials.NewClientTLSFromCert(nil, ""), nil
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return errors.New("not a gRPC probe config")
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
	p.targets = p.opts.Targets.ListEndpoints()

	p.cancelFuncs = make(map[string]context.CancelFunc)
	p.src = sysvars.Vars()["hostname"]

	transportCreds, err := p.transportCredentials()
	if err != nil {
		return fmt.Errorf("error creating transport credentials: %v", err)
	}
	p.creds = transportCreds

	// Initialize dial options.
	p.dialOpts = append(p.dialOpts, grpc.WithDefaultServiceConfig(loadBalancingPolicy))
	oauthCfg := p.c.GetOauthConfig()
	if oauthCfg != nil {
		oauthTS, err := oauth.TokenSourceFromConfig(oauthCfg, p.l)
		if err != nil {
			return err
		}
		p.dialOpts = append(p.dialOpts, grpc.WithPerRPCCredentials(grpcoauth.TokenSource{TokenSource: oauthTS}))
	}

	resolver.SetDefaultScheme("dns")

	p.numConns = int(p.c.GetNumConns())
	if p.numConns == 0 {
		p.numConns = 1
		backwardCompatMethods := []configpb.ProbeConf_MethodType{
			configpb.ProbeConf_ECHO,
			configpb.ProbeConf_READ,
			configpb.ProbeConf_WRITE,
		}
		if slices.Contains(backwardCompatMethods, p.c.GetMethod()) {
			p.numConns = 2
		}
	}

	if p.c.GetMethod() == configpb.ProbeConf_GENERIC {
		if err := p.initDescriptorSource(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Probe) updateTargetsAndStartProbes(ctx context.Context) {
	newTargets := p.opts.Targets.ListEndpoints()
	numNewTargets := len(newTargets)

	p.targetsMu.Lock()
	defer p.targetsMu.Unlock()
	if numNewTargets == 0 || numNewTargets < (len(p.targets)/2) {
		p.l.Errorf("Too few new targets, retaining old targets. New targets: %v, old count: %d", newTargets, len(p.targets))
		return
	}

	updatedTargets := make(map[string]string)
	defer func() {
		if len(updatedTargets) > 0 {
			p.l.Infof("Targets updated: %v", updatedTargets)
		}
	}()

	activeTargets := make(map[string]bool)
	// Create results structure and start probe loop for new targets.
	for _, target := range newTargets {
		key := target.Key()

		activeTargets[key] = true
		if _, ok := p.results[key]; ok {
			continue
		}

		updatedTargets[key] = "ADD"
		p.results[key] = p.newResult(key)
		probeCtx, probeCancelFunc := context.WithCancel(ctx)
		for i := 0; i < int(p.numConns); i++ {
			go p.oneTargetLoop(probeCtx, target, i, p.results[key])
		}
		p.cancelFuncs[key] = probeCancelFunc
	}

	// Stop probing for deleted targets by invoking cancelFunc.
	for key := range p.results {
		if activeTargets[key] {
			continue
		}
		p.cancelFuncs[key]()
		updatedTargets[key] = "DELETE"
		delete(p.results, key)
		delete(p.cancelFuncs, key)
	}
	p.targets = newTargets
}

// connectWithRetry attempts to connect to a target. On failure, it retries in
// an infinite loop until successful, incrementing connectErrors for every
// connection error. On success, it returns a client immediately.
// Interval between connects is controlled by connect_timeout_msec, defaulting
// to probe timeout.
func (p *Probe) connectWithRetry(ctx context.Context, target endpoint.Endpoint, result *probeRunResult, logAttrs ...slog.Attr) *grpc.ClientConn {
	addr := target.Name
	if target.IP != nil {
		if p.opts.IPVersion == 0 || iputils.IPVersion(target.IP) == p.opts.IPVersion {
			addr = target.IP.String()
		} else {
			p.l.Warningf("Target IP (%v) doesn't match probe IP version (%d), letting system resolve it", target.IP, p.opts.IPVersion)
		}
	}

	if target.Port > 0 {
		addr = net.JoinHostPort(addr, strconv.Itoa(target.Port))
	}

	connectTimeout := p.opts.Timeout
	if p.c.GetConnectTimeoutMsec() > 0 {
		connectTimeout = time.Duration(p.c.GetConnectTimeoutMsec()) * time.Millisecond
	}
	for {
		select {
		case <-ctx.Done():
			p.l.WarningAttrs("context cancelled in connect loop.", logAttrs...)
			return nil
		default:
		}
		connCtx, cancelFunc := context.WithTimeout(ctx, connectTimeout)

		if uriScheme := p.c.GetUriScheme(); uriScheme != "" {
			addr = uriScheme + addr
		}

		// Note we use grpcurl.BlockingDial which uses WithBlock dial option which is
		// discouraged by the gRPC docs:
		// https://github.com/grpc/grpc-go/blob/master/Documentation/anti-patterns.md.
		// In a traditional gRPC client, it makes sense for connections to be
		// fluid, and come and go, but for  aprober it's important that
		// connection is established before we start sending RPCs. We'll get a
		// much better error message if connection fails.
		conn, err := grpcurl.BlockingDial(connCtx, "tcp", addr, p.creds, p.dialOpts...)
		cancelFunc()

		if err != nil {
			p.l.WarningAttrs("Connect error: "+err.Error(), logAttrs...)
		} else {
			p.l.InfoAttrs("Connection established", logAttrs...)
			return conn
		}

		result.Lock()
		result.total.Inc()
		result.connectErrors.Inc()
		result.Unlock()

		// Sleep before retrying connection.
		time.Sleep(p.opts.Interval)
	}
}

func (p *Probe) healthCheckProbe(ctx context.Context, conn *grpc.ClientConn, logAttrs ...slog.Attr) (*grpc_health_v1.HealthCheckResponse, error) {
	var resp *grpc_health_v1.HealthCheckResponse
	var err error

	if p.healthCheckFunc != nil {
		resp, err = p.healthCheckFunc()
	} else {
		resp, err = grpc_health_v1.NewHealthClient(conn).
			Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: p.c.GetHealthCheckService()})
	}

	if err != nil {
		return nil, err
	}

	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		p.l.WarningAttrs("gRPC HealthCheck status: "+resp.GetStatus().String(), logAttrs...)
		if !p.c.GetHealthCheckIgnoreStatus() {
			return resp, fmt.Errorf("not serving (%s)", resp.GetStatus())
		}
	}
	return resp, nil
}

// oneTargetLoop connects to and then continuously probes a single target.
func (p *Probe) oneTargetLoop(ctx context.Context, tgt endpoint.Endpoint, index int, result *probeRunResult) {
	msgPattern := fmt.Sprintf("%s,%s%s,%03d", p.src, p.c.GetUriScheme(), tgt.Name, index)
	logAttrs := []slog.Attr{
		slog.String("probeId", msgPattern),
		slog.String("request_type", p.c.GetMethod().String()),
	}

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(tgt, "", 0)
	}

	conn := p.connectWithRetry(ctx, tgt, result, logAttrs...)
	if conn == nil {
		return
	}
	defer conn.Close()

	client := spb.NewProberClient(conn)
	timeout := p.opts.Timeout
	method := p.c.GetMethod()

	msgSize := p.c.GetBlobSize()
	msg := make([]byte, msgSize)
	probeutils.PatternPayload(msg, []byte(msgPattern))
	ticker := time.NewTicker(p.opts.Interval)
	for {
		select {
		case <-ctx.Done():
			p.l.WarningAttrs("Context cancelled in request loop.", logAttrs...)
			ticker.Stop()
			return
		case <-ticker.C:
		}

		if !p.opts.IsScheduled() {
			continue
		}

		reqCtx, cancelFunc := context.WithTimeout(ctx, timeout)

		reqCtx = p.ctxWithHeaders(reqCtx)

		var delta time.Duration
		start := time.Now()

		var peer peer.Peer
		opts := []grpc.CallOption{
			grpc.WaitForReady(true),
			grpc.Peer(&peer),
		}

		var success bool
		var err error
		var r fmt.Stringer

		switch method {
		case configpb.ProbeConf_ECHO:
			r, err = client.Echo(reqCtx, &pb.EchoMessage{Blob: []byte(msg)}, opts...)
		case configpb.ProbeConf_READ:
			r, err = client.BlobRead(reqCtx, &pb.BlobReadRequest{Size: proto.Int32(msgSize)}, opts...)
		case configpb.ProbeConf_WRITE:
			r, err = client.BlobWrite(reqCtx, &pb.BlobWriteRequest{Blob: []byte(msg)}, opts...)
		case configpb.ProbeConf_HEALTH_CHECK:
			r, err = p.healthCheckProbe(reqCtx, conn, logAttrs...)
		case configpb.ProbeConf_GENERIC:
			r, err = p.genericRequest(reqCtx, conn, p.c.GetRequest())
		default:
			p.l.Criticalf("Method %v not implemented", method)
		}

		cancelFunc()

		p.l.DebugAttrs("Response: "+r.String(), logAttrs...)

		if err != nil {
			peerAddr := "unknown"
			if peer.Addr != nil {
				peerAddr = peer.Addr.String()
			}
			p.l.WarningAttrs(fmt.Sprintf("Request failed: %v. ConnState: %v", err, conn.GetState()), append(logAttrs, slog.String("peer", peerAddr))...)
		} else {
			success = true
			delta = time.Since(start)
		}

		if p.opts.Validators != nil {
			failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{ResponseBody: []byte(r.String())}, result.validationFailure, p.l)

			if len(failedValidations) > 0 {
				p.l.DebugAttrs("Some validations failed", append(logAttrs, slog.String("failed_validations", strings.Join(failedValidations, ",")))...)
				success = false
			}
		}

		result.Lock()
		result.total.Inc()
		if success {
			result.success.Inc()
		}
		result.latency.AddFloat64(delta.Seconds() / p.opts.LatencyUnit.Seconds())
		result.Unlock()
	}
}

func (p *Probe) newResult(tgt string) *probeRunResult {
	var latencyValue metrics.LatencyValue
	if p.opts.LatencyDist != nil {
		latencyValue = p.opts.LatencyDist.CloneDist()
	} else {
		latencyValue = metrics.NewFloat(0)
	}

	validationFailure := validators.ValidationFailureMap(p.opts.Validators)

	return &probeRunResult{
		target:            tgt,
		latency:           latencyValue,
		validationFailure: validationFailure,
	}
}

// ctxWitHeaders attaches a list of headers to the given context
// it iterates over the headers defined in the probe configuration
func (p *Probe) ctxWithHeaders(ctx context.Context) context.Context {
	headers := p.c.GetHeaders()
	parsed := make(map[string]string, len(headers))

	// map each header to the parsed map
	for _, header := range headers {
		parsed[header.GetName()] = header.GetValue()
	}
	// create metadata from headers & attach to context
	return metadata.NewOutgoingContext(ctx, metadata.New(parsed))
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	p.results = make(map[string]*probeRunResult)
	p.updateTargetsAndStartProbes(ctx)

	ticker := time.NewTicker(p.opts.StatsExportInterval)
	defer ticker.Stop()

	targetsUpdateTicker := time.NewTicker(TargetsUpdateInterval)
	defer targetsUpdateTicker.Stop()

	for ts := range ticker.C {
		// Stop further processing and exit if context is canceled.
		// Same context is used by probe loops.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Output results.
		for _, target := range p.targets {
			result, ok := p.results[target.Key()]
			if !ok {
				continue
			}

			result.Lock()
			em := metrics.NewEventMetrics(ts).
				AddMetric("total", result.total.Clone()).
				AddMetric("success", result.success.Clone()).
				AddMetric(p.opts.LatencyMetricName, result.latency.Clone()).
				AddMetric("connecterrors", result.connectErrors.Clone()).
				AddLabel("ptype", "grpc").
				AddLabel("probe", p.name).
				AddLabel("dst", target.Dst())
			result.Unlock()

			if result.validationFailure != nil {
				em.AddMetric("validation_failure", result.validationFailure)
			}

			p.opts.RecordMetrics(target, em, dataChan)
		}

		// Finally, update targets and start new probe loops if necessary.
		// Executing this as the last step in the loop also ensures that new
		// targets have at least one cycle of probes before next output cycle.
		select {
		case <-targetsUpdateTicker.C:
			p.updateTargetsAndStartProbes(ctx)
		default:
		}
	}
}
