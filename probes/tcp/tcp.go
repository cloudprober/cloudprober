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
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/tcp/proto"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// book-keeping params
	network     string
	dialContext func(context.Context, string, string) (net.Conn, error) // Keeps some dialing related config
}

type probeResult struct {
	total, success    int64
	latency           metrics.LatencyValue
	validationFailure *metrics.Map[int64]
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

	return result
}

func (result *probeResult) Metrics(ts time.Time, _ int64, opts *options.Options) []*metrics.EventMetrics {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric(opts.LatencyMetricName, result.latency.Clone()).
		AddLabel("ptype", "tcp")

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

	return nil
}

func (p *Probe) runProbe(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
	if runReq.Result == nil {
		runReq.Result = p.newResult()
	}

	target, result := runReq.Target, runReq.Result.(*probeResult)

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
			p.l.Error("target: ", target.Name, ", resolve error: ", err.Error())
			return
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
	conn, err := p.dialContext(ctx, p.network, addr)
	latency := time.Since(start)
	if conn != nil {
		defer conn.Close()
	}

	if p.opts.NegativeTest {
		if err == nil {
			p.l.Warning("Negative test, but connection was successful to: ", addr)
			return
		}
		result.success++
		return
	}

	if err != nil {
		p.l.Warning("Target:", target.Name, ", doTCP: ", err.Error())
		return
	}
	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
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
