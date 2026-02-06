// Copyright 2022 The Cloudprober Authors.
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

// Package tcp implements a TCP probe type.
package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/tcp/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"google.golang.org/protobuf/proto"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// book-keeping params
	network          string
	tlsConfig        *tls.Config
	dialContext      func(context.Context, string, string) (net.Conn, error) // Keeps some dialing related config
	handshakeContext func(context.Context, net.Conn, *tls.Config) error
}

type probeResult struct {
	total, success      int64
	latency             metrics.LatencyValue
	connLatency         metrics.LatencyValue
	tlsHandshakeLatency metrics.LatencyValue
	validationFailure   *metrics.Map[int64]
}

func (p *Probe) newResult() sched.ProbeResult {
	result := &probeResult{}

	if p.opts.Validators != nil {
		result.validationFailure = validators.ValidationFailureMap(p.opts.Validators)
	}

	if p.opts.LatencyDist != nil {
		result.latency = p.opts.LatencyDist.CloneDist()
	} else {
		result.latency = metrics.NewFloat(0)
	}

	if p.c.GetTlsHandshake() {
		if p.opts.LatencyDist != nil {
			result.connLatency = p.opts.LatencyDist.CloneDist()
			result.tlsHandshakeLatency = p.opts.LatencyDist.CloneDist()
		} else {
			result.connLatency = metrics.NewFloat(0)
			result.tlsHandshakeLatency = metrics.NewFloat(0)
		}
	}

	return result
}

func (result *probeResult) Metrics(ts time.Time, _ int64, opts *options.Options) []*metrics.EventMetrics {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric(opts.LatencyMetricName, result.latency.Clone()).
		AddLabel("ptype", "tcp")

	// If TLS handshake is enabled, add conn and tls_handshake latency metrics.
	if result.tlsHandshakeLatency != nil {
		em.AddMetric("connect_latency", result.connLatency.Clone())
		em.AddMetric("tls_handshake_latency", result.tlsHandshakeLatency.Clone())
	}

	if result.validationFailure != nil {
		em.AddMetric("validation_failure", result.validationFailure)
	}

	return []*metrics.EventMetrics{em}
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	if opts.ProbeConf == nil {
		opts.ProbeConf = &configpb.ProbeConf{}
	}

	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not tcp probe config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}

	p.c = c
	if p.c == nil {
		p.c = &configpb.ProbeConf{}
	}

	p.network = "tcp"
	if p.opts.IPVersion != 0 {
		p.network += strconv.Itoa(p.opts.IPVersion)
	}

	// Create a dialer for our use.
	dialer := &net.Dialer{
		Timeout:   p.opts.Timeout,
		KeepAlive: 30 * time.Second, // TCP keep-alive
	}
	if p.opts.SourceIP != nil {
		dialer.LocalAddr = &net.TCPAddr{
			IP: p.opts.SourceIP,
		}
	}
	p.dialContext = dialer.DialContext

	if p.c.GetTlsConfig() != nil {
		if p.c.TlsHandshake == nil {
			p.c.TlsHandshake = proto.Bool(true)
		}

		// tls_handshake is explicitly set to false, return error
		if !p.c.GetTlsHandshake() {
			return fmt.Errorf("tls_config is set, but tls_handshake is false")
		}

		p.tlsConfig = &tls.Config{}
		if err := tlsconfig.UpdateTLSConfig(p.tlsConfig, p.c.GetTlsConfig()); err != nil {
			return fmt.Errorf("tls_config error: %v", err)
		}
	}

	return nil
}

func (p *Probe) connectAndHandshake(ctx context.Context, addr, targetName string, result *probeResult) error {
	start := time.Now()
	conn, err := p.dialContext(ctx, p.network, addr)
	if err != nil {
		return err
	}

	if p.c.GetTlsHandshake() {
		result.connLatency.AddFloat64(time.Since(start).Seconds() / p.opts.LatencyUnit.Seconds())
		start = time.Now()

		// ServerName is required for TLS handshake
		if p.tlsConfig == nil {
			p.tlsConfig = &tls.Config{}
		}
		if p.tlsConfig.ServerName == "" {
			p.tlsConfig.ServerName = targetName
		}

		if p.handshakeContext == nil {
			p.handshakeContext = func(ctx context.Context, nc net.Conn, tlsConfig *tls.Config) error {
				return tls.Client(nc, tlsConfig).HandshakeContext(ctx)
			}
		}
		if err := p.handshakeContext(ctx, conn, p.tlsConfig); err != nil {
			return err
		}
		result.tlsHandshakeLatency.AddFloat64(time.Since(start).Seconds() / p.opts.LatencyUnit.Seconds())
	}

	if conn != nil {
		conn.Close()
	}
	return nil
}

func (p *Probe) runProbeForTarget(ctx context.Context, target endpoint.Endpoint, result *probeResult) error {
	result.total++

	host := target.Name
	ipLabel := ""

	resolveFirst := false
	if p.c.ResolveFirst != nil {
		resolveFirst = p.c.GetResolveFirst()
	} else {
		resolveFirst = target.IP != nil
	}
	if resolveFirst {
		ip, err := target.Resolve(p.opts.IPVersion, p.opts.Targets)
		if err != nil {
			return fmt.Errorf("resolve error: %v", err)
		}
		host = ip.String()
		ipLabel = host
	}

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target, ipLabel, int(p.c.GetPort()))
	}

	port := int(p.c.GetPort())
	if port == 0 {
		port = target.Port
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))

	start := time.Now()
	err := p.connectAndHandshake(ctx, addr, target.Name, result)
	latency := time.Since(start)

	if p.opts.NegativeTest {
		if err == nil {
			return fmt.Errorf("negative test, but connection was successful to: %s", addr)
		}
		result.success++
		return nil
	}

	if err != nil {
		return err
	}
	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	return nil
}

func (p *Probe) runProbe(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
	if runReq.Result == nil {
		runReq.Result = p.newResult()
	}

	target, result := runReq.Target, runReq.Result.(*probeResult)

	if err := p.runProbeForTarget(ctx, target, result); err != nil {
		p.l.ErrorAttrs(err.Error(), slog.String("target", target.Name))
	}
}

// RunOnce runs the probe just once.
func (p *Probe) RunOnce(ctx context.Context) []*singlerun.ProbeRunResult {
	p.l.Info("Running TCP probe once.")

	var out []*singlerun.ProbeRunResult
	var outMu sync.Mutex

	var wg sync.WaitGroup
	for _, target := range p.opts.Targets.ListEndpoints() {
		wg.Add(1)
		go func(target endpoint.Endpoint) {
			defer wg.Done()

			updateOut := func(result *singlerun.ProbeRunResult) {
				outMu.Lock()
				defer outMu.Unlock()
				result.Target = target
				out = append(out, result)
			}

			result := p.newResult().(*probeResult)

			start := time.Now()
			if err := p.runProbeForTarget(ctx, target, result); err != nil {
				updateOut(&singlerun.ProbeRunResult{
					Error: fmt.Errorf("TCP request failed for target, %s: %v", target.Name, err),
				})
				return
			}

			updateOut(&singlerun.ProbeRunResult{
				Metrics: result.Metrics(time.Now(), 1, p.opts),
				Success: result.success > 0,
				Latency: time.Since(start),
			})
		}(target)
	}
	wg.Wait()

	// Sort the results by target name
	sort.Slice(out, func(i, j int) bool {
		return out[i].Target.Name < out[j].Target.Name
	})

	return out
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
