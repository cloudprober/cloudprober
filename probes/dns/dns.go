// Copyright 2017-2024 The Cloudprober Authors.
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
Package dns implements a DNS prober. It sends UDP DNS queries to a list of
targets and reports statistics on queries sent, queries received, and latency
experienced.

This prober uses the DNS library in /third_party/golang/dns/dns to construct,
send, and receive DNS messages. Every message is sent on a different UDP port.
Queries to each target are sent in parallel.
*/
package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/dns/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/miekg/dns"
)

const defaultPort = 53

// Client provides a DNS client interface for required functionality.
// This makes it possible to mock.
type Client interface {
	ExchangeContext(context.Context, *dns.Msg, string) (*dns.Msg, time.Duration, error)
	setTimeout(time.Duration)
	setSourceIP(net.IP)
	setDNSProto(configpb.DNSProto)
}

// ClientImpl is a concrete DNS client that can be instantiated.
type clientImpl struct {
	dns.Client
}

// setTimeout allows write-access to the underlying Timeout variable.
func (c *clientImpl) setTimeout(d time.Duration) {
	c.Timeout = d
}

// setSourceIP allows write-access to the underlying ReadTimeout variable.
func (c *clientImpl) setSourceIP(ip net.IP) {
	c.Dialer = &net.Dialer{
		LocalAddr: &net.UDPAddr{IP: ip},
	}
}

// setDNSProto sets the DNS transport protocol to use.
func (c *clientImpl) setDNSProto(proto configpb.DNSProto) {
	switch proto {
	case configpb.DNSProto_UDP:
		c.Net = "udp"
	case configpb.DNSProto_TCP:
		c.Net = "tcp"
	case configpb.DNSProto_TCP_TLS:
		c.Net = "tcp-tls"
	}
}

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	// book-keeping params
	targets    []endpoint.Endpoint
	queryType  uint16
	queryClass uint16
	fqdn       string
	client     Client
}

// probeRunResult captures the results of a single probe run. The way we work with
// stats makes sure that probeRunResult and its fields are not accessed concurrently
// (see documentation with statsKeeper below). That's the reason we use metrics.Int
// types instead of metrics.AtomicInt.
type probeRunResult struct {
	total             metrics.Int
	success           metrics.Int
	latency           metrics.LatencyValue
	timeouts          metrics.Int
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
		AddMetric(opts.LatencyMetricName, prr.latency.Clone()).
		AddMetric("timeouts", &prr.timeouts).
		AddLabel("ptype", "dns")

	if prr.validationFailure != nil {
		em.AddMetric("validation_failure", prr.validationFailure)
	}

	return em
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return errors.New("no dns config")
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

	totalDuration := time.Duration(p.c.GetRequestsIntervalMsec()*p.c.GetRequestsPerProbe())*time.Millisecond + p.opts.Timeout
	if totalDuration > p.opts.Interval {
		return fmt.Errorf("invalid config - executing all requests will take "+
			"longer than the probe interval, i.e. "+
			"requests_per_probe*requests_interval_msec + timeout (%s) > interval (%s)",
			totalDuration, p.opts.Interval)
	}

	p.targets = p.opts.Targets.ListEndpoints()

	queryType := p.c.GetQueryType()
	if queryType == configpb.QueryType_NONE || int32(queryType) >= int32(dns.TypeReserved) {
		return fmt.Errorf("dns_probe(%v): invalid query type %v", name, queryType)
	}
	p.queryType = uint16(queryType)
	p.queryClass = uint16(p.c.GetQueryClass())
	p.fqdn = dns.Fqdn(p.c.GetResolvedDomain())

	// I believe the client is safe for concurrent use by multiple goroutines
	// (although the documentation doesn't explicitly say so). It uses locks
	// internally and the underlying net.Conn declares that multiple goroutines
	// may invoke methods on a net.Conn simultaneously.
	p.client = new(clientImpl)
	if p.opts.SourceIP != nil {
		p.client.setSourceIP(p.opts.SourceIP)
	}

	// We need it even with context because DNS client uses the lower of this
	// timeout which is 2s by default, and context timeout, so if context
	// timeout is 5s, DNS query will timeout in 2s even though context timeout
	// is 5s.
	p.client.setTimeout(p.opts.Timeout)
	// Set DNS Protocol to use
	p.client.setDNSProto(p.c.GetDnsProto())

	return nil
}

// Return true if the underlying error indicates a dns.Client timeout.
// In our case, we're using the ReadTimeout- time until response is read.
func isClientTimeout(err error) bool {
	e, ok := err.(*net.OpError)
	return ok && e != nil && e.Timeout()
}

// validateResponse checks status code and answer section for correctness and
// returns true if the response is valid. In case of validation failures, it
// also updates the result structure.
func (p *Probe) validateResponse(resp *dns.Msg, target string, result *probeRunResult) bool {
	if resp == nil || resp.Rcode != dns.RcodeSuccess {
		p.l.Warningf("Target(%s): error in response %v", target, resp)
		return false
	}

	// Validate number of answers in response.
	// TODO: Move this logic to validators.
	minAnswers := p.c.GetMinAnswers()
	if minAnswers > 0 && uint32(len(resp.Answer)) < minAnswers {
		p.l.Warningf("Target(%s): too few answers - got %d want %d.\n\tAnswerBlock: %v",
			target, len(resp.Answer), minAnswers, resp.Answer)
		return false
	}

	if p.opts.Validators != nil {
		answers := []string{}
		for _, rr := range resp.Answer {
			if rr != nil {
				answers = append(answers, rr.String())
			}
		}
		respBytes := []byte(strings.Join(answers, "\n"))

		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{ResponseBody: respBytes}, result.validationFailure, p.l)
		if len(failedValidations) > 0 {
			p.l.Debugf("Target(%s): validators %v failed. Resp: %v", target, failedValidations, answers)
			return false
		}
	}

	return true
}

func (p *Probe) doDNSRequest(ctx context.Context, target string, result *probeRunResult, resultMu *sync.Mutex) {
	// Generate a new question for each probe so transaction IDs aren't repeated.
	msg := new(dns.Msg)
	msg.SetQuestion(p.fqdn, p.queryType)
	msg.Question[0].Qclass = p.queryClass

	resp, latency, err := p.client.ExchangeContext(ctx, msg, target)

	if resultMu != nil {
		resultMu.Lock()
		defer resultMu.Unlock()
	}

	if err != nil {
		if isClientTimeout(err) {
			p.l.Warningf("Target(%s): client.Exchange: Timeout error: %v", target, err)
			result.timeouts.Inc()
		} else {
			p.l.Warningf("Target(%s): client.Exchange: %v", target, err)
		}
	} else if p.validateResponse(resp, target, result) {
		result.success.Inc()
		result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	}
}

func (p *Probe) runProbe(ctx context.Context, target endpoint.Endpoint, res sched.ProbeResult) {
	// Convert interface to struct type
	result := res.(*probeRunResult)

	port := defaultPort
	if target.Port != 0 {
		port = target.Port
	}
	result.total.IncBy(int64(p.c.GetRequestsPerProbe()))

	ipLabel := ""
	fullTarget := net.JoinHostPort(target.Name, strconv.Itoa(port))

	resolveFirst := false
	if p.c.ResolveFirst != nil {
		resolveFirst = p.c.GetResolveFirst()
	} else {
		resolveFirst = target.IP != nil
	}
	if resolveFirst {
		ip, err := target.Resolve(p.opts.IPVersion, p.opts.Targets)
		if err != nil {
			p.l.Warningf("Target(%s): Resolve error: %v", target.Name, err)
			return
		}
		ipLabel = ip.String()
		fullTarget = net.JoinHostPort(ip.String(), strconv.Itoa(port))
	}

	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target, ipLabel, port)
	}

	if p.c.GetRequestsPerProbe() == 1 {
		p.doDNSRequest(ctx, fullTarget, result, nil)
		return
	}

	// For multiple requests per probe, we launch a separate goroutine for each
	// DNS request. We use a mutex to protect access to per-target result object
	// in doDNSRequest. Note that result object is not accessed concurrently
	// anywhere else -- export of metrics happens when probe is not running.
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < int(p.c.GetRequestsPerProbe()); i++ {
		wg.Add(1)
		go func(reqNum int, result *probeRunResult) {
			defer wg.Done()

			time.Sleep(time.Duration(reqNum*int(p.c.GetRequestsIntervalMsec())) * time.Millisecond)
			p.doDNSRequest(ctx, fullTarget, result, &resultMu)
		}(i, result)
	}
	p.l.Debug("Waiting for DNS requests to finish")
	wg.Wait()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	s := &sched.Scheduler{
		ProbeName:         p.name,
		DataChan:          dataChan,
		Opts:              p.opts,
		NewResult:         func(_ *endpoint.Endpoint) sched.ProbeResult { return p.newResult() },
		RunProbeForTarget: p.runProbe,
	}
	s.UpdateTargetsAndStartProbes(ctx)
}
