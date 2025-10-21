// Copyright 2020-2025 The Cloudprober Authors.
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
	"github.com/cloudprober/cloudprober/common/oauth"
	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/internal/file"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/common/sched"
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

	// Import grpclb module so it can be used by name for DirectPath connections.
	_ "google.golang.org/grpc/balancer/grpclb"
)

const connIndexLabel = "conn_index"

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
	conns    map[string]*grpc.ClientConn
	connsMu  sync.Mutex

	dialOpts []grpc.DialOption
	creds    credentials.TransportCredentials
	descSrc  grpcurl.DescriptorSource

	targets []endpoint.Endpoint

	// Results by target.
	results   map[string]*probeRunResult
	resultsMu sync.Mutex

	// This is used only for testing.
	healthCheckFunc func() (*grpc_health_v1.HealthCheckResponse, error)
}

// probeRunResult captures the metrics for a single target. Multiple threads
// can update metrics at the same time and the main thread periodically
// outputs the values in this struct.
type probeRunResult struct {
	sync.Mutex
	total             metrics.Int
	success           metrics.Int
	latency           metrics.LatencyValue
	connectErrors     metrics.Int
	validationFailure *metrics.Map[int64]
	lastRunID         int64
}

func (p *Probe) newResult(target *endpoint.Endpoint) sched.ProbeResult {
	p.resultsMu.Lock()
	defer p.resultsMu.Unlock()

	// We want one result per target, make sure to ignore the connIndexLabel
	key := target.Key(endpoint.WithIgnoreLabels(connIndexLabel))

	result, ok := p.results[key]
	if !ok {
		var latencyValue metrics.LatencyValue
		if p.opts.LatencyDist != nil {
			latencyValue = p.opts.LatencyDist.CloneDist()
		} else {
			latencyValue = metrics.NewFloat(0)
		}

		result = &probeRunResult{
			latency:           latencyValue,
			validationFailure: validators.ValidationFailureMap(p.opts.Validators),
		}

		p.results[key] = result
	}
	return result
}

func (prr *probeRunResult) Metrics(ts time.Time, runID int64, opts *options.Options) []*metrics.EventMetrics {
	prr.Lock()
	defer prr.Unlock()

	// This is sort of a de-duplication exercise. Metrics() will be called by
	// the scheduler 'numConns' times per run. We use runID check to ensure we
	// output metrics only once per run.
	if prr.lastRunID == runID {
		return nil
	}

	prr.lastRunID = runID
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", prr.total.Clone()).
		AddMetric("success", prr.success.Clone()).
		AddMetric(opts.LatencyMetricName, prr.latency.Clone()).
		AddMetric("connecterrors", prr.connectErrors.Clone()).
		AddLabel("ptype", "grpc")

	if prr.validationFailure != nil {
		em.AddMetric("validation_failure", prr.validationFailure)
	}

	return []*metrics.EventMetrics{em}
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

// listEndpoints denoramlizes the targets list by connection index before
// returning it. This is required because 'sched' schedules one probe loop per
// target and we want to have one probe loop per target per connection.
func (p *Probe) listEndpoints() []endpoint.Endpoint {
	targets := p.opts.Targets.ListEndpoints()

	if p.numConns == 1 {
		return targets
	}

	var out []endpoint.Endpoint
	// For each target, create 'numConns' clones, and add connection index as
	// a label so that they are distinguished from each other.
	for _, target := range targets {
		for i := 0; i < p.numConns; i++ {
			tgt := target.Clone()
			tgt.Labels[connIndexLabel] = strconv.Itoa(i)
			out = append(out, *tgt)
		}
	}
	return out
}

func (p *Probe) defaultServiceConfig() string {
	if p.c.GetDefaultLbConfig() != "" {
		return fmt.Sprintf(`{"loadBalancingConfig":%s}`, p.c.GetDefaultLbConfig())
	}
	if p.c.GetUriScheme() == "grpclb" {
		// We've kept it here for backward compatibility with Google's internal
		// deployment. Typically service config is provided by the resolver.
		// See: https://github.com/grpc/grpc/blob/master/doc/service_config.md
		// Here is an example of our own resolver implementation:
		// https://github.com/cloudprober/cloudprober/blob/d1f62e55672c793cac00363d6847b7b23c60da09/internal/rds/client/srvlist.go#L98
		return `{"loadBalancingConfig":[{"grpclb":{"childPolicy":[{"pick_first":{}}]}}]}`
	}
	return ""
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

	p.src = sysvars.GetVar("hostname")

	transportCreds, err := p.transportCredentials()
	if err != nil {
		return fmt.Errorf("error creating transport credentials: %v", err)
	}
	p.creds = transportCreds

	if strings.HasPrefix(p.c.GetUriScheme(), "xds") && !xdsSupported {
		return fmt.Errorf(`xds support not enabled. Compile cloudprober with "-tags grpc_xds" to add xds support`)
	}

	if defaultServiceConfig := p.defaultServiceConfig(); defaultServiceConfig != "" {
		p.dialOpts = append(p.dialOpts, grpc.WithDefaultServiceConfig(defaultServiceConfig))
	}

	oauthCfg := p.c.GetOauthConfig()
	if oauthCfg != nil {
		oauthTS, err := oauth.TokenSourceFromConfig(oauthCfg, p.l)
		if err != nil {
			return err
		}
		p.dialOpts = append(p.dialOpts, grpc.WithPerRPCCredentials(grpcoauth.TokenSource{TokenSource: oauthTS}))
	}

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
		if filePath := p.c.GetRequest().GetBodyFile(); filePath != "" {
			if p.c.GetRequest().GetBody() != "" {
				return errors.New("bad config: both body and body_file are set")
			}
			var opts []file.ReadOption
			if p.c.GetRequest().GetBodyFileSubstituteEnv() {
				opts = append(opts, file.WithEnvSubstitution())
			}
			b, err := file.ReadFile(context.Background(), filePath, opts...)
			if err != nil {
				return fmt.Errorf("error reading request body from file (%s): %v", filePath, err)
			}
			p.c.Request.Body = proto.String(string(b))
			fmt.Println("--", p.c.Request.GetBody(), "--")
		}
	}

	// Initialize maps
	p.results = make(map[string]*probeRunResult)
	p.conns = make(map[string]*grpc.ClientConn)

	return nil
}

func (p *Probe) connectionString(target endpoint.Endpoint) string {
	addr := target.Name

	if target.IP != nil {
		if p.opts.IPVersion == 0 || iputils.IPVersion(target.IP) == p.opts.IPVersion {
			addr = target.IP.String()
		} else {
			p.l.Warningf("Target IP (%v) doesn't match probe IP version (%d), letting system resolve it", target.IP, p.opts.IPVersion)
		}
	}

	port := 443 // default
	if target.Port != 0 {
		port = target.Port
	}
	if p.c.GetPort() != 0 {
		port = int(p.c.GetPort())
	}

	addr = net.JoinHostPort(addr, strconv.Itoa(port))

	if uriScheme := p.c.GetUriScheme(); uriScheme != "" {
		addr = uriScheme + addr
	}

	return addr
}

// connect attempts to connect to a target.
func (p *Probe) connect(ctx context.Context, target endpoint.Endpoint) (*grpc.ClientConn, error) {

	if p.c.GetConnectTimeoutMsec() > 0 {
		connctx, cancelFunc := context.WithTimeout(ctx, time.Duration(p.c.GetConnectTimeoutMsec())*time.Millisecond)
		defer cancelFunc()
		ctx = connctx
	}

	// Note we use grpcurl.BlockingDial which uses WithBlock dial option which is
	// discouraged by the gRPC docs:
	// https://github.com/grpc/grpc-go/blob/master/Documentation/anti-patterns.md.
	// In a traditional gRPC client, it makes sense for connections to be
	// fluid, and come and go, but for  aprober it's important that
	// connection is established before we start sending RPCs. We'll get a
	// much better error message if connection fails.
	return grpcurl.BlockingDial(ctx, "tcp", p.connectionString(target), p.creds, p.dialOpts...)
}

func (p *Probe) getConn(ctx context.Context, target endpoint.Endpoint, targetKey string, l *logger.Logger) (*grpc.ClientConn, error) {
	if p.c.GetDisableReuseConn() {
		return p.connect(ctx, target)
	}
	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	if conn := p.conns[targetKey]; conn != nil {
		return conn, nil
	}

	conn, err := p.connect(ctx, target)
	if err != nil {
		l.Warning("Connect error: " + err.Error())
		return nil, err
	}
	l.Info("Connection established")
	p.conns[targetKey] = conn
	return conn, nil
}

func (p *Probe) healthCheckProbe(ctx context.Context, conn *grpc.ClientConn, l *logger.Logger) (*grpc_health_v1.HealthCheckResponse, error) {
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
		l.Warning("gRPC HealthCheck status: " + resp.GetStatus().String())
		if !p.c.GetHealthCheckIgnoreStatus() {
			return resp, fmt.Errorf("not serving (%s)", resp.GetStatus())
		}
	}
	return resp, nil
}

type targetState struct {
	targetKey string
}

// runProbeForTargetAndConnIndex runs a single probe for a target + connection index.
func (p *Probe) runProbeForTargetAndConnIndex(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
	if runReq.TargetState == nil {
		runReq.TargetState = &targetState{targetKey: runReq.Target.Key()}
	}
	tgtState := runReq.TargetState.(*targetState)

	msgPattern := fmt.Sprintf("%s,%s%s,connIndex:%s", p.src, p.c.GetUriScheme(), runReq.Target.Name, runReq.Target.Labels[connIndexLabel])
	l := p.l.WithAttributes(
		slog.String("probeId", msgPattern),
		slog.String("request_type", p.c.GetMethod().String()),
	)

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(runReq.Target, "", 0)
	}

	if runReq.Result == nil {
		runReq.Result = p.newResult(&runReq.Target)
	}
	result := runReq.Result.(*probeRunResult)

	// On connection failure, this is where probe will end.
	conn, err := p.getConn(ctx, runReq.Target, tgtState.targetKey, l)
	if err != nil {
		result.Lock()
		result.total.Inc()
		result.connectErrors.Inc()
		result.Unlock()
		return
	}
	if p.c.GetDisableReuseConn() {
		defer conn.Close()
	}

	client := spb.NewProberClient(conn)
	reqCtx := p.ctxWithHeaders(ctx)

	var delta time.Duration
	start := time.Now()

	var peer peer.Peer
	opts := []grpc.CallOption{
		grpc.WaitForReady(true),
		grpc.Peer(&peer),
	}

	var success bool
	var r fmt.Stringer

	getPaylod := func() []byte {
		msg := make([]byte, p.c.GetBlobSize())
		probeutils.PatternPayload(msg, []byte(msgPattern))
		return msg
	}

	switch p.c.GetMethod() {
	case configpb.ProbeConf_ECHO:
		r, err = client.Echo(reqCtx, &pb.EchoMessage{Blob: []byte(getPaylod())}, opts...)
	case configpb.ProbeConf_READ:
		r, err = client.BlobRead(reqCtx, &pb.BlobReadRequest{Size: proto.Int32(p.c.GetBlobSize())}, opts...)
	case configpb.ProbeConf_WRITE:
		r, err = client.BlobWrite(reqCtx, &pb.BlobWriteRequest{Blob: []byte(getPaylod())}, opts...)
	case configpb.ProbeConf_HEALTH_CHECK:
		r, err = p.healthCheckProbe(reqCtx, conn, l)
	case configpb.ProbeConf_GENERIC:
		r, err = p.genericRequest(reqCtx, conn, p.c.GetRequest())
	default:
		p.l.Criticalf("Method %v not implemented", p.c.GetMethod())
	}

	l.Debug("Response: " + r.String())

	if err != nil {
		peerAddr := "unknown"
		if peer.Addr != nil {
			peerAddr = peer.Addr.String()
		}
		p.l.WarningAttrs(fmt.Sprintf("Request failed: %v. ConnState: %v", err, conn.GetState()), slog.String("peer", peerAddr))
	} else {
		success = true
		delta = time.Since(start)
	}

	if success && p.opts.Validators != nil {
		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{ResponseBody: []byte(r.String())}, result.validationFailure, p.l)

		if len(failedValidations) > 0 {
			l.Error("failed validations: ", strings.Join(failedValidations, ","))
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
	s := &sched.Scheduler{
		ProbeName:         p.name,
		DataChan:          dataChan,
		Opts:              p.opts,
		ListEndpoints:     p.listEndpoints,
		RunProbeForTarget: p.runProbeForTargetAndConnIndex,
	}

	s.UpdateTargetsAndStartProbes(ctx)

	// We'll come here when context is cancelled, clean up connections.
	for _, conn := range p.conns {
		if conn != nil {
			conn.Close()
		}
	}
}
