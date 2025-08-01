// Copyright 2017-2022 The Cloudprober Authors.
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

// Package http implements HTTP probe type.
package http

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/common/oauth"
	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/internal/httpreq"
	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/payload"
	"github.com/cloudprober/cloudprober/metrics/singlerun"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"golang.org/x/oauth2"
)

const (
	maxResponseSizeForMetrics = 128
)

// Probe holds aggregate information about all probe runs, per-target.
type Probe struct {
	name string
	opts *options.Options
	c    *configpb.ProbeConf
	l    *logger.Logger

	baseTransport http.RoundTripper
	redirectFunc  func(req *http.Request, via []*http.Request) error

	// book-keeping params
	targets []endpoint.Endpoint
	method  string
	url     string
	oauthTS oauth2.TokenSource

	responseParser *payload.Parser

	requestBody *httpreq.RequestBody
}

type latencyDetails struct {
	dnsLatency, connectLatency, tlsLatency, reqWriteLatency, firstByteLatency metrics.LatencyValue
}

type probeResult struct {
	total, success, timeouts     int64
	connEvent                    *metrics.AtomicInt
	latency                      metrics.LatencyValue
	respCodes                    *metrics.Map[int64]
	respBodies                   *metrics.Map[int64]
	validationFailure            *metrics.Map[int64]
	latencyBreakdown             *latencyDetails
	sslEarliestExpirationSeconds int64
	payloadMetrics               []*metrics.EventMetrics
}

func (p *Probe) getTransport() (*http.Transport, error) {
	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: true,
	}
	dialer := &net.Dialer{
		Timeout:   p.opts.Timeout,
		KeepAlive: 30 * time.Second, // TCP keep-alive
	}
	if p.opts.SourceIP != nil {
		dialer.LocalAddr = &net.TCPAddr{
			IP: p.opts.SourceIP,
		}
	}
	transport.DialContext = dialer.DialContext
	transport.MaxIdleConns = int(p.c.GetMaxIdleConns())
	transport.TLSHandshakeTimeout = p.opts.Timeout

	if p.c.GetProxyUrl() != "" {
		url, err := url.Parse(p.c.GetProxyUrl())
		if err != nil {
			return nil, fmt.Errorf("error parsing proxy URL (%s): %v", p.c.GetProxyUrl(), err)
		}
		transport.Proxy = http.ProxyURL(url)

		if len(p.c.GetProxyConnectHeader()) > 0 && transport.ProxyConnectHeader == nil {
			transport.ProxyConnectHeader = make(http.Header)
		}
		for k, v := range p.c.GetProxyConnectHeader() {
			transport.ProxyConnectHeader.Add(k, v)
		}
	}

	if p.c.GetDisableCertValidation() || p.c.GetTlsConfig() != nil {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}

		if p.c.GetDisableCertValidation() {
			p.l.Warning("disable_cert_validation is deprecated as of v0.10.6. Instead of this, please use \"tls_config {disable_cert_validation: true}\"")
			transport.TLSClientConfig.InsecureSkipVerify = true
		}

		if p.c.GetTlsConfig() != nil {
			if err := tlsconfig.UpdateTLSConfig(transport.TLSClientConfig, p.c.GetTlsConfig()); err != nil {
				return nil, err
			}
		}
	}

	// If HTTP keep-alives are not enabled (default), disable HTTP keep-alive in
	// transport.
	if !p.c.GetKeepAlive() {
		transport.DisableKeepAlives = true
	} else {
		// If it's been more than 2 probe intervals since connection was used, close it.
		transport.IdleConnTimeout = 2 * p.opts.Interval
		if p.c.GetRequestsPerProbe() > 1 {
			transport.MaxIdleConnsPerHost = int(p.c.GetRequestsPerProbe())
		}
	}

	if p.c.GetDisableHttp2() {
		// HTTP/2 is enabled by default if server supports it. Setting
		// TLSNextProto to an empty dict is the only way to disable it.
		// This only works if transport hasn't been previously cloned.
		// See https://github.com/cloudprober/cloudprober/issues/872
		transport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
		transport.ForceAttemptHTTP2 = false
	}

	return transport, nil
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
	if p.c == nil {
		p.c = &configpb.ProbeConf{}
	}

	totalDuration := time.Duration(p.c.GetRequestsIntervalMsec()*p.c.GetRequestsPerProbe())*time.Millisecond + p.opts.Timeout
	if totalDuration > p.opts.Interval {
		return fmt.Errorf("invalid config - executing all requests will take "+
			"longer than the probe interval, i.e. "+
			"requests_per_probe*requests_interval_msec + timeout (%s) > interval (%s)",
			totalDuration, p.opts.Interval)
	}

	p.method = p.c.GetMethod().String()

	p.url = p.c.GetRelativeUrl()
	if len(p.url) > 0 && p.url[0] != '/' {
		return fmt.Errorf("invalid relative URL: %s, must begin with '/'", p.url)
	}

	body := p.c.GetBody()
	if len(body) == 0 && p.c.GetBodyFile() != "" {
		b, err := os.ReadFile(p.c.GetBodyFile())
		if err != nil {
			return fmt.Errorf("error reading body file: %v", err)
		}
		body = []string{string(b)}
	}
	p.requestBody = httpreq.NewRequestBody(body...)

	if p.c.GetOauthConfig() != nil {
		oauthTS, err := oauth.TokenSourceFromConfig(p.c.GetOauthConfig(), p.l)
		if err != nil {
			return err
		}
		p.oauthTS = oauthTS
	}

	transport, err := p.getTransport()
	if err != nil {
		return err
	}

	p.baseTransport = transport

	if p.c.MaxRedirects != nil {
		p.redirectFunc = func(req *http.Request, via []*http.Request) error {
			if len(via) >= int(p.c.GetMaxRedirects()) {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}

	if p.c.GetResponseMetricsOptions() != nil {
		p.responseParser, err = payload.NewParser(p.c.GetResponseMetricsOptions(), p.l)
		if err != nil {
			return fmt.Errorf("error initializing response metrics parser: %v", err)
		}
	}

	p.targets = p.opts.Targets.ListEndpoints()

	return nil
}

// Return true if the underlying error indicates a http.Client timeout.
//
// Use for errors returned from http.Client methods (Get, Post).
func isClientTimeout(err error) bool {
	if uerr, ok := err.(*url.Error); ok {
		if nerr, ok := uerr.Err.(net.Error); ok && nerr.Timeout() {
			return true
		}
	}
	return false
}

func (p *Probe) addLatency(latency metrics.LatencyValue, start time.Time) {
	latency.AddFloat64(time.Since(start).Seconds() / p.opts.LatencyUnit.Seconds())
}

func (p *Probe) requestTrace(result *probeResult) *httptrace.ClientTrace {
	if result.latencyBreakdown == nil && result.connEvent == nil {
		return nil
	}

	trace := &httptrace.ClientTrace{}

	if lb := result.latencyBreakdown; lb != nil {
		var dnsStart, connectStart, tlsStart, writeStart, firstbyteStart time.Time

		if lb.dnsLatency != nil {
			trace.DNSStart = func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() }
			trace.DNSDone = func(_ httptrace.DNSDoneInfo) { p.addLatency(lb.dnsLatency, dnsStart) }
		}
		if lb.connectLatency != nil {
			trace.ConnectStart = func(_, _ string) { connectStart = time.Now() }
			trace.ConnectDone = func(_, _ string, _ error) { p.addLatency(lb.connectLatency, connectStart) }
		}
		if lb.tlsLatency != nil {
			trace.TLSHandshakeStart = func() { tlsStart = time.Now() }
			trace.TLSHandshakeDone = func(_ tls.ConnectionState, _ error) { p.addLatency(lb.tlsLatency, tlsStart) }
		}
		if lb.reqWriteLatency != nil {
			trace.WroteHeaders = func() { writeStart = time.Now() }
			trace.WroteRequest = func(_ httptrace.WroteRequestInfo) { p.addLatency(lb.reqWriteLatency, writeStart) }
		}
		if lb.firstByteLatency != nil {
			trace.GotConn = func(_ httptrace.GotConnInfo) { firstbyteStart = time.Now() }
			trace.GotFirstResponseByte = func() { p.addLatency(lb.firstByteLatency, firstbyteStart) }
		}
	}

	if result.connEvent != nil {
		oldConnectDone := trace.ConnectDone
		trace.ConnectDone = func(network, addr string, err error) {
			result.connEvent.Inc()
			if oldConnectDone != nil {
				oldConnectDone(network, addr, err)
			}
			if err != nil {
				p.l.Warning("Error establishing a new connection to: ", addr, ". Err: ", err.Error())
				return
			}
			p.l.Info("Established a new connection to: ", addr)
		}
	}

	return trace
}

// doHTTPRequest executes an HTTP request and updates the provided result struct.
func (p *Probe) doHTTPRequest(req *http.Request, client *http.Client, target endpoint.Endpoint, result *probeResult, resultMu *sync.Mutex) error {
	logAttrs := []slog.Attr{slog.String("target", target.Name), slog.String("url", req.URL.String())}

	req = p.prepareRequest(req)

	start := time.Now()

	if trace := p.requestTrace(result); trace != nil {
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	}

	resp, err := client.Do(req)
	latency := time.Since(start)

	if resultMu != nil {
		// Note that we take lock on result object outside of the actual request.
		resultMu.Lock()
		defer resultMu.Unlock()
	}

	result.total++

	if err != nil {
		if isClientTimeout(err) {
			result.timeouts++
		}
		p.l.WarningAttrs(err.Error(), logAttrs...)
		return err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.l.WarningAttrs(err.Error(), logAttrs...)
		return err
	}

	p.l.DebugAttrs("Response: \n"+string(respBody), logAttrs...)

	// Calling Body.Close() allows the TCP connection to be reused.
	resp.Body.Close()
	result.respCodes.IncKey(strconv.FormatInt(int64(resp.StatusCode), 10))

	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		now := time.Now()
		minExpirySeconds := resp.TLS.PeerCertificates[0].NotAfter.Sub(now).Seconds()

		for i := 1; i < len(resp.TLS.PeerCertificates); i++ {
			if resp.TLS.PeerCertificates[i].NotAfter.Sub(now).Seconds() < minExpirySeconds {
				minExpirySeconds = resp.TLS.PeerCertificates[i].NotAfter.Sub(now).Seconds()
			}
		}

		result.sslEarliestExpirationSeconds = int64(minExpirySeconds)
	}

	if p.opts.Validators != nil {
		failedValidations := validators.RunValidators(p.opts.Validators, &validators.Input{Response: resp, ResponseBody: respBody}, result.validationFailure, p.l)

		// If any validation failed, return now, leaving the success and latency
		// counters unchanged.
		if len(failedValidations) > 0 {
			msg := fmt.Sprintf("failed validations: %s", strings.Join(failedValidations, ","))
			p.l.DebugAttrs(msg, logAttrs...)
			return errors.New(msg)
		}
	}

	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	if result.respBodies != nil && len(respBody) <= maxResponseSizeForMetrics {
		result.respBodies.IncKey(string(respBody))
	}

	if p.c.GetResponseMetricsOptions() != nil {
		for _, em := range p.responseParser.PayloadMetrics(&payload.Input{Response: resp, Text: respBody}, target.Dst()) {
			result.payloadMetrics = append(result.payloadMetrics, em.AddLabel("ptype", "http"))
		}
	}

	return nil
}

func (p *Probe) parseLatencyBreakdown(baseLatencyValue metrics.LatencyValue) *latencyDetails {
	if len(p.c.GetLatencyBreakdown()) == 0 {
		return nil
	}
	lbMap := make(map[configpb.ProbeConf_LatencyBreakdown]bool)
	for _, l := range p.c.GetLatencyBreakdown() {
		lbMap[l] = true
	}

	all := lbMap[configpb.ProbeConf_ALL_STAGES]

	ld := &latencyDetails{}
	if all || lbMap[configpb.ProbeConf_DNS_LATENCY] {
		ld.dnsLatency = baseLatencyValue.Clone().(metrics.LatencyValue)
	}
	if all || lbMap[configpb.ProbeConf_CONNECT_LATENCY] {
		ld.connectLatency = baseLatencyValue.Clone().(metrics.LatencyValue)
	}
	if all || lbMap[configpb.ProbeConf_TLS_HANDSHAKE_LATENCY] {
		ld.tlsLatency = baseLatencyValue.Clone().(metrics.LatencyValue)
	}
	if all || lbMap[configpb.ProbeConf_REQ_WRITE_LATENCY] {
		ld.reqWriteLatency = baseLatencyValue.Clone().(metrics.LatencyValue)
	}
	if all || lbMap[configpb.ProbeConf_FIRST_BYTE_LATENCY] {
		ld.firstByteLatency = baseLatencyValue.Clone().(metrics.LatencyValue)
	}
	return ld
}

func (p *Probe) newResult() *probeResult {
	result := &probeResult{
		respCodes:                    metrics.NewMap("code"),
		sslEarliestExpirationSeconds: -1,
	}

	if p.c.GetKeepAlive() {
		result.connEvent = metrics.NewAtomicInt(0)
	}

	if p.opts.Validators != nil {
		result.validationFailure = validators.ValidationFailureMap(p.opts.Validators)
	}

	if p.opts.LatencyDist != nil {
		result.latency = p.opts.LatencyDist.CloneDist()
	} else {
		result.latency = metrics.NewFloat(0)
	}

	result.latencyBreakdown = p.parseLatencyBreakdown(result.latency)

	if p.c.GetExportResponseAsMetrics() {
		result.respBodies = metrics.NewMap("resp")
	}

	return result
}

func (result *probeResult) Metrics(ts time.Time, runID int64, opts *options.Options) []*metrics.EventMetrics {
	var ems []*metrics.EventMetrics

	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric(opts.LatencyMetricName, result.latency.Clone()).
		AddMetric("timeouts", metrics.NewInt(result.timeouts)).
		AddMetric("resp-code", result.respCodes.Clone())

	if result.respBodies != nil {
		em.AddMetric("resp-body", result.respBodies.Clone())
	}

	if result.connEvent != nil {
		em.AddMetric("connect_event", result.connEvent.Clone())
	}

	if result.validationFailure != nil {
		em.AddMetric("validation_failure", result.validationFailure)
	}

	if result.latencyBreakdown != nil {
		if dl := result.latencyBreakdown.dnsLatency; dl != nil {
			em.AddMetric("dns_latency", dl.Clone())
		}
		if cl := result.latencyBreakdown.connectLatency; cl != nil {
			em.AddMetric("connect_latency", cl.Clone())
		}
		if tl := result.latencyBreakdown.tlsLatency; tl != nil {
			em.AddMetric("tls_handshake_latency", tl.Clone())
		}
		if rwl := result.latencyBreakdown.reqWriteLatency; rwl != nil {
			em.AddMetric("req_write_latency", rwl.Clone())
		}
		if fbl := result.latencyBreakdown.firstByteLatency; fbl != nil {
			em.AddMetric("first_byte_latency", fbl.Clone())
		}
	}

	em.AddLabel("ptype", "http") // Other labels are added by scheduler.
	ems = append(ems, em)

	// SSL earliest cert expiry is exported in an independent EM as it's a
	// GAUGE metrics.
	if result.sslEarliestExpirationSeconds >= 0 {
		em := metrics.NewEventMetrics(ts).
			AddMetric("ssl_earliest_cert_expiry_sec", metrics.NewInt(result.sslEarliestExpirationSeconds))
		em.Kind = metrics.GAUGE
		em.SetNotForAlerting()       // Helps with quick discard in alert path.
		em.AddLabel("ptype", "http") // Other labels are added by scheduler.
		ems = append(ems, em)
	}

	// Append any payload metrics and reset.
	ems = append(ems, result.payloadMetrics...)
	result.payloadMetrics = nil

	return ems
}

func (p *Probe) httpClient(target endpoint.Endpoint) *http.Client {
	// We check for http.Transport because tests use a custom
	// RoundTripper implementation.
	if ht, ok := p.baseTransport.(*http.Transport); ok {
		t := ht.Clone()

		// If we're resolving target first, url.Host will be an IP address.
		// In that case, we need to set ServerName in TLSClientConfig to
		// the actual hostname.
		if p.schemeForTarget(target) == "https" && p.resolveFirst(target) {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = &tls.Config{}
			}
			if t.TLSClientConfig.ServerName == "" {
				t.TLSClientConfig.ServerName = hostForTarget(target)
			}
		}

		return &http.Client{Transport: t}
	}
	return &http.Client{Transport: p.baseTransport}
}

// Returns clients for a target. We use a different HTTP client (transport) for
// each request within a probe cycle. For example, if you configure
// requests_per_probe as 100, we'll create and use 100 HTTP clients. This
// is to provide a more deterministic way to create multiple connections to
// a single target (while still using keep_alive to avoid the cost of TCP
// connection setup in the probing path). This behavior is desirable if you
// want to hit as many backends as possible, behind a single VIP.
func (p *Probe) clientsForTarget(target endpoint.Endpoint) []*http.Client {
	clients := make([]*http.Client, p.c.GetRequestsPerProbe())
	for i := range clients {
		clients[i] = p.httpClient(target)
	}
	return clients
}

type targetState struct {
	req     *http.Request
	clients []*http.Client
	runCnt  int64
}

func (p *Probe) runProbe(ctx context.Context, runReq *sched.RunProbeForTargetRequest) {
	if runReq.Result == nil {
		runReq.Result = p.newResult()
	}

	// We cache the HTTP requests, clients, and runCnt in the target state.
	if runReq.TargetState == nil {
		runReq.TargetState = &targetState{}
	}

	target, result, tgtState := runReq.Target, runReq.Result.(*probeResult), runReq.TargetState.(*targetState)

	tgtState.runCnt++
	if tgtState.clients == nil {
		tgtState.clients = p.clientsForTarget(runReq.Target)
	}

	// Update request if it is nil, or if we are resolving first and we are at
	// the stats export interval. This is to make sure that we are using the
	// correct IP address for the target.
	if tgtState.req == nil || (p.c.GetResolveFirst() && tgtState.runCnt%p.opts.StatsExportFrequency() == 0) {
		req, err := p.httpRequestForTarget(runReq.Target)
		if err != nil {
			p.l.Error("Error creating HTTP request for target: ", target.Name, ", err: ", err.Error())
			result.total += int64(p.c.GetRequestsPerProbe())
			return
		}
		tgtState.req = req
	}

	if p.c.GetRequestsPerProbe() == 1 {
		// Ignore the error returned by doHTTPRequest, as it's already logged.
		_ = p.doHTTPRequest(tgtState.req.WithContext(ctx), tgtState.clients[0], target, result, nil)
		return
	}

	// For multiple requests per probe, we launch a separate goroutine for each
	// HTTP request. We use a mutex to protect access to per-target result object
	// in doHTTPRequest. Note that result object is not accessed concurrently
	// anywhere else -- export of metrics happens when probe is not running.
	var resultMu sync.Mutex

	wg := sync.WaitGroup{}
	for numReq := 0; numReq < int(p.c.GetRequestsPerProbe()); numReq++ {
		wg.Add(1)
		go func(req *http.Request, numReq int, target endpoint.Endpoint, result *probeResult) {
			defer wg.Done()

			time.Sleep(time.Duration(numReq*int(p.c.GetRequestsIntervalMsec())) * time.Millisecond)
			// Ignore the error returned by doHTTPRequest, as it's already logged.
			_ = p.doHTTPRequest(req.WithContext(ctx), tgtState.clients[numReq], target, result, &resultMu)
		}(tgtState.req, numReq, target, result)
	}
	wg.Wait()
}

// RunOnce runs the probe just once.
func (p *Probe) RunOnce(ctx context.Context) []*singlerun.ProbeRunResult {
	p.l.Info("Running HTTP probe once.")

	if p.c.GetRequestsPerProbe() > 1 {
		p.l.Warningf("requests_per_probe > 1 is not supported in single run mode, ignoring it.")
	}

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

			result := p.newResult()

			req, err := p.httpRequestForTarget(target)
			if err != nil {
				updateOut(&singlerun.ProbeRunResult{
					Error: fmt.Errorf("error creating HTTP request for target: %s, err: %v", target.Name, err),
				})
				return
			}

			start := time.Now()
			if err := p.doHTTPRequest(req.WithContext(ctx), p.httpClient(target), target, result, nil); err != nil {
				updateOut(&singlerun.ProbeRunResult{
					Error: fmt.Errorf("error doing HTTP request for target: %s, err: %v", target.Name, err),
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
