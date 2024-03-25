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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudprober/cloudprober/internal/httpreq"
	"github.com/cloudprober/cloudprober/internal/oauth"
	"github.com/cloudprober/cloudprober/internal/tlsconfig"
	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"golang.org/x/oauth2"
)

// DefaultTargetsUpdateInterval defines default frequency for target updates.
// Actual targets update interval is:
// max(DefaultTargetsUpdateInterval, probe_interval)
var DefaultTargetsUpdateInterval = 1 * time.Minute

const (
	maxResponseSizeForMetrics = 128
	targetsUpdateInterval     = 1 * time.Minute
	largeBodyThreshold        = bytes.MinRead // 512.
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

	// How often to resolve targets (in probe counts), it's the minimum of
	targetsUpdateInterval time.Duration

	// How often to export metrics (in probe counts), initialized to
	// statsExportInterval / p.opts.Interval. Metrics are exported when
	// (runCnt % statsExportFrequency) == 0
	statsExportFrequency int64

	// Cancel functions for per-target probe loop
	cancelFuncs map[string]context.CancelFunc
	waitGroup   sync.WaitGroup

	requestBody *httpreq.RequestBody
}

type latencyDetails struct {
	dnsLatency, connectLatency, tlsLatency, reqWriteLatency, firstByteLatency metrics.LatencyValue
}

type probeResult struct {
	total, success, timeouts     int64
	connEvent                    int64
	latency                      metrics.LatencyValue
	respCodes                    *metrics.Map[int64]
	respBodies                   *metrics.Map[int64]
	validationFailure            *metrics.Map[int64]
	latencyBreakdown             *latencyDetails
	sslEarliestExpirationSeconds int64
}

func (p *Probe) getTransport() (*http.Transport, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
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

	p.requestBody = httpreq.NewRequestBody(p.c.GetBody()...)

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

	p.statsExportFrequency = p.opts.StatsExportInterval.Nanoseconds() / p.opts.Interval.Nanoseconds()
	if p.statsExportFrequency == 0 {
		p.statsExportFrequency = 1
	}

	p.targets = p.opts.Targets.ListEndpoints()
	p.cancelFuncs = make(map[string]context.CancelFunc, len(p.targets))

	p.targetsUpdateInterval = DefaultTargetsUpdateInterval
	// There is no point refreshing targets before probe interval.
	if p.targetsUpdateInterval < p.opts.Interval {
		p.targetsUpdateInterval = p.opts.Interval
	}
	p.l.Infof("Targets update interval: %v", p.targetsUpdateInterval)

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

// httpRequest executes an HTTP request and updates the provided result struct.
func (p *Probe) doHTTPRequest(req *http.Request, client *http.Client, targetName string, result *probeResult, resultMu *sync.Mutex) {
	req = p.prepareRequest(req)

	start := time.Now()

	trace := &httptrace.ClientTrace{}

	if lb := result.latencyBreakdown; lb != nil {
		if lb.dnsLatency != nil {
			trace.DNSDone = func(_ httptrace.DNSDoneInfo) { p.addLatency(lb.dnsLatency, start) }
		}
		if lb.connectLatency != nil {
			trace.ConnectDone = func(_, _ string, _ error) { p.addLatency(lb.connectLatency, start) }
		}
		if lb.tlsLatency != nil {
			trace.TLSHandshakeDone = func(_ tls.ConnectionState, _ error) { p.addLatency(lb.tlsLatency, start) }
		}
		if lb.reqWriteLatency != nil {
			trace.WroteRequest = func(_ httptrace.WroteRequestInfo) { p.addLatency(lb.reqWriteLatency, start) }
		}
		if lb.firstByteLatency != nil {
			trace.GotFirstResponseByte = func() { p.addLatency(lb.firstByteLatency, start) }
		}
	}

	var connEvent atomic.Int32

	if p.c.GetKeepAlive() {
		oldConnectDone := trace.ConnectDone
		trace.ConnectDone = func(network, addr string, err error) {
			connEvent.Add(1)
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

	if trace != nil {
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
	result.connEvent += int64(connEvent.Load())

	if err != nil {
		if isClientTimeout(err) {
			p.l.WarningAttrs(err.Error(), slog.String("target", targetName), slog.String("url", req.URL.String()))
			result.timeouts++
			return
		}
		p.l.WarningAttrs(err.Error(), slog.String("target", targetName), slog.String("url", req.URL.String()))
		return
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		p.l.WarningAttrs(err.Error(), slog.String("target", targetName), slog.String("url", req.URL.String()))
		return
	}

	p.l.Debug("Target:", targetName, ", URL:", req.URL.String(), ", response: ", string(respBody))

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
			p.l.Debug("Target:", targetName, ", URL:", req.URL.String(), ", http.doHTTPRequest: failed validations: ", strings.Join(failedValidations, ","))
			return
		}
	}

	result.success++
	result.latency.AddFloat64(latency.Seconds() / p.opts.LatencyUnit.Seconds())
	if result.respBodies != nil && len(respBody) <= maxResponseSizeForMetrics {
		result.respBodies.IncKey(string(respBody))
	}
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

func (p *Probe) runProbe(ctx context.Context, target endpoint.Endpoint, clients []*http.Client, req *http.Request, result *probeResult) {
	reqCtx, cancelReqCtx := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancelReqCtx()

	if p.c.GetRequestsPerProbe() == 1 {
		p.doHTTPRequest(req.WithContext(reqCtx), clients[0], target.Name, result, nil)
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
		go func(req *http.Request, numReq int, targetName string, result *probeResult) {
			defer wg.Done()

			time.Sleep(time.Duration(numReq*int(p.c.GetRequestsIntervalMsec())) * time.Millisecond)
			p.doHTTPRequest(req.WithContext(reqCtx), clients[numReq], targetName, result, &resultMu)
		}(req, numReq, target.Name, result)
	}
	wg.Wait()
}

func (p *Probe) newResult() *probeResult {
	result := &probeResult{
		respCodes:                    metrics.NewMap("code"),
		sslEarliestExpirationSeconds: -1,
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

func (p *Probe) exportMetrics(ts time.Time, result *probeResult, target endpoint.Endpoint, dataChan chan *metrics.EventMetrics) {
	em := metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(result.total)).
		AddMetric("success", metrics.NewInt(result.success)).
		AddMetric(p.opts.LatencyMetricName, result.latency.Clone()).
		AddMetric("timeouts", metrics.NewInt(result.timeouts)).
		AddMetric("resp-code", result.respCodes.Clone())

	if result.respBodies != nil {
		em.AddMetric("resp-body", result.respBodies.Clone())
	}

	if p.c.GetKeepAlive() {
		em.AddMetric("connect_event", metrics.NewInt(result.connEvent))
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

	em.AddLabel("ptype", "http").AddLabel("probe", p.name).AddLabel("dst", target.Name)
	p.opts.RecordMetrics(target, em, dataChan)

	// SSL earliest cert expiry is exported in an independent EM as it's a
	// GAUGE metrics.
	if result.sslEarliestExpirationSeconds >= 0 {
		em := metrics.NewEventMetrics(ts).
			AddMetric("ssl_earliest_cert_expiry_sec", metrics.NewInt(result.sslEarliestExpirationSeconds))
		em.Kind = metrics.GAUGE
		em.AddLabel("ptype", "http").AddLabel("probe", p.name).AddLabel("dst", target.Name)
		p.opts.RecordMetrics(target, em, dataChan, options.WithNoAlert())
	}
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

			clients[i] = &http.Client{Transport: t}
		} else {
			clients[i] = &http.Client{Transport: p.baseTransport}
		}

		clients[i].CheckRedirect = p.redirectFunc
	}
	return clients
}

func (p *Probe) startForTarget(ctx context.Context, target endpoint.Endpoint, dataChan chan *metrics.EventMetrics) {
	p.l.Debug("Starting probing for the target ", target.Name)

	// We use this counter to decide when to export stats.
	var runCnt int64

	result := p.newResult()
	req := p.httpRequestForTarget(target)
	ticker := time.NewTicker(p.opts.Interval)
	defer ticker.Stop()

	clients := p.clientsForTarget(target)
	for ts := time.Now(); true; ts = <-ticker.C {
		// Don't run another probe if context is canceled already.
		if ctxDone(ctx) {
			return
		}

		if !p.opts.IsScheduled() {
			continue
		}

		// If request is nil (most likely because target resolving failed or it
		// was an invalid target), skip this probe cycle. Note that request
		// creation gets retried at a regular interval (stats export interval).
		if req != nil {
			p.runProbe(ctx, target, clients, req, result)
		} else {
			result.total += int64(p.c.GetRequestsPerProbe())
		}

		// Export stats if it's the time to do so.
		runCnt++
		if (runCnt % p.statsExportFrequency) == 0 {
			p.exportMetrics(ts, result, target, dataChan)

			// If we are resolving first, this is also a good time to recreate HTTP
			// request in case target's IP has changed.
			if p.c.GetResolveFirst() {
				req = p.httpRequestForTarget(target)
			}
		}
	}
}

func (p *Probe) gapBetweenTargets() time.Duration {
	interTargetGap := time.Duration(p.c.GetIntervalBetweenTargetsMsec()) * time.Millisecond

	// If not configured by user, determine based on probe interval and number of
	// targets.
	if interTargetGap == 0 && len(p.targets) != 0 {
		// Use 1/10th of the probe interval to spread out target groroutines.
		interTargetGap = p.opts.Interval / time.Duration(10*len(p.targets))
	}

	return interTargetGap
}

// updateTargetsAndStartProbes refreshes targets and starts probe loop for
// new targets and cancels probe loops for targets that are no longer active.
// Note that this function is not concurrency safe. It is never called
// concurrently by Start().
func (p *Probe) updateTargetsAndStartProbes(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	p.targets = p.opts.Targets.ListEndpoints()

	p.l.Debugf("Probe(%s) got %d targets", p.name, len(p.targets))

	// updatedTargets is used only for logging.
	updatedTargets := make(map[string]string)
	defer func() {
		if len(updatedTargets) > 0 {
			p.l.Infof("Probe(%s) targets updated: %v", p.name, updatedTargets)
		}
	}()

	activeTargets := make(map[string]endpoint.Endpoint)
	for _, target := range p.targets {
		key := target.Key()
		activeTargets[key] = target
	}

	// Stop probing for deleted targets by invoking cancelFunc.
	for targetKey, cancelF := range p.cancelFuncs {
		if _, ok := activeTargets[targetKey]; ok {
			continue
		}
		cancelF()
		updatedTargets[targetKey] = "DELETE"
		delete(p.cancelFuncs, targetKey)
	}

	gapBetweenTargets := p.gapBetweenTargets()
	var startWaitTime time.Duration

	// Start probe loop for new targets.
	for key, target := range activeTargets {
		// This target is already initialized.
		if _, ok := p.cancelFuncs[key]; ok {
			continue
		}
		updatedTargets[key] = "ADD"

		probeCtx, cancelF := context.WithCancel(ctx)
		p.waitGroup.Add(1)

		go func(target endpoint.Endpoint, waitTime time.Duration) {
			defer p.waitGroup.Done()

			// To evenly spread out target probes, wait for a randomized
			// duration before starting the target go-routine.
			if waitTime > 0 {
				// For random padding using 1/10th of the gap.
				jitterMaxUsec := gapBetweenTargets.Microseconds() / 10
				// Make sure we don't pass 0 to rand.Int63n.
				if jitterMaxUsec <= 0 {
					jitterMaxUsec = 1
				}
				time.Sleep(waitTime + time.Duration(rand.Int63n(jitterMaxUsec))*time.Microsecond)
			}

			p.startForTarget(probeCtx, target, dataChan)
		}(target, startWaitTime)

		startWaitTime += gapBetweenTargets

		p.cancelFuncs[key] = cancelF
	}
}

func ctxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// wait waits for child go-routines (one per target) to clean up.
func (p *Probe) wait() {
	p.waitGroup.Wait()
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	defer p.wait()

	p.updateTargetsAndStartProbes(ctx, dataChan)

	// Do more frequent listing of targets until we get a non-zero list of
	// targets.
	initialRefreshInterval := p.opts.Interval
	// Don't wait too long if p.opts.Interval is large.
	if initialRefreshInterval > time.Second {
		initialRefreshInterval = time.Second
	}

	for {
		if ctxDone(ctx) {
			return
		}
		if len(p.targets) != 0 {
			break
		}
		p.updateTargetsAndStartProbes(ctx, dataChan)
		time.Sleep(initialRefreshInterval)
	}

	targetsUpdateTicker := time.NewTicker(p.targetsUpdateInterval)
	defer targetsUpdateTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-targetsUpdateTicker.C:
			p.updateTargetsAndStartProbes(ctx, dataChan)
		}
	}
}
