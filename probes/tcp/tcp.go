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

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/tcp/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/cloudprober/cloudprober/validators"
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// book-keeping params
	network string
	dialer  *net.Dialer // Keeps some dialing related config
}

type probeResult struct {
	total, success    int64
	latency           metrics.Value
	validationFailure *metrics.Map
}

func (result *probeResult) Metrics(ts time.Time, opts *options.Options) *metrics.EventMetrics {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric(opts.LatencyMetricName, result.latency).
		AddLabel("ptype", "tcp")

	if result.validationFailure != nil {
		em.AddMetric("validation_failure", result.validationFailure)
	}

	return em
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not http config")
	}
	p.name = name
	p.opts = opts
	if p.l = opts.Logger; p.l == nil {
		p.l = &logger.Logger{}
	}
	p.c = c

	p.network = "tcp"
	if p.opts.IPVersion != 0 {
		p.network += strconv.Itoa(p.opts.IPVersion)
	}

	// Create a dialer for our use.
	p.dialer = &net.Dialer{
		Timeout:   p.opts.Timeout,
		KeepAlive: 30 * time.Second, // TCP keep-alive
	}
	if p.opts.SourceIP != nil {
		p.dialer.LocalAddr = &net.TCPAddr{
			IP: p.opts.SourceIP,
		}
	}

	return nil
}

func (p *Probe) runProbe(ctx context.Context, target endpoint.Endpoint, res sched.ProbeResult) {
	ctx, cancelCtx := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancelCtx()

	// Convert interface to struct type
	result := res.(*probeResult)

	addr := target.Name
	if p.c.GetResolveFirst() {
		ip, err := p.opts.Targets.Resolve(addr, p.opts.IPVersion)
		if err != nil {
			p.l.Error("target: ", addr, ", resolve error: ", err.Error())
			return
		}
		addr = ip.String()
	}

	port := int(p.c.GetPort())
	if port == 0 {
		port = target.Port
	}

	start := time.Now()
	conn, err := p.dialer.DialContext(ctx, p.network, net.JoinHostPort(addr, strconv.Itoa(port)))
	latency := time.Since(start)
	if conn != nil {
		defer conn.Close()
	}

	result.total++

	if err != nil {
		p.l.Warning("Target:", target.Name, ", doTCP: ", err.Error())
		return
	}

	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
}

func (p *Probe) newResult() sched.ProbeResult {
	result := &probeResult{}

	if p.opts.Validators != nil {
		result.validationFailure = validators.ValidationFailureMap(p.opts.Validators)
	}

	if p.opts.LatencyDist != nil {
		result.latency = p.opts.LatencyDist.Clone()
	} else {
		result.latency = metrics.NewFloat(0)
	}

	return result
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	s := &sched.Scheduler{
		ProbeName:              p.name,
		DataChan:               dataChan,
		Opts:                   p.opts,
		NewResult:              p.newResult,
		RunProbeForTarget:      p.runProbe,
		IntervalBetweenTargets: time.Duration(p.c.GetIntervalBetweenTargetsMsec()) * time.Millisecond,
	}
	s.UpdateTargetsAndStartProbes(ctx)
}
