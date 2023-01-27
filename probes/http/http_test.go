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

package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/testutils"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// The Transport is mocked instead of the Client because Client is not an
// interface, but RoundTripper (which Transport implements) is.
type testTransport struct {
	mu              sync.Mutex
	lastRequestBody []byte
	lastAuthHeader  string
	keepAuthHeader  bool
}

func patchWithTestTransport(p *Probe) {
	keepAuthHeader := false
	if p.oauthTS != nil {
		keepAuthHeader = true
	}
	p.baseTransport = &testTransport{
		keepAuthHeader: keepAuthHeader,
	}
}

// This mocks the Body of an http.Response.
type testReadCloser struct {
	b *bytes.Buffer
}

func (trc *testReadCloser) Read(p []byte) (n int, err error) {
	return trc.b.Read(p)
}

func (trc *testReadCloser) Close() error {
	return nil
}

func (tt *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	authHeader := req.Header.Get("Authorization")
	if tt.keepAuthHeader {
		tt.mu.Lock()
		tt.lastAuthHeader = authHeader
		tt.mu.Unlock()
	}
	if strings.Contains(authHeader, "missing") {
		return nil, fmt.Errorf("auth header: %s", authHeader)
	}

	if req.URL.Host == "fail-test.com" {
		return nil, errors.New("failing for fail-target.com")
	}

	if req.Body == nil {
		return &http.Response{Body: http.NoBody}, nil
	}

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()

	tt.mu.Lock()
	tt.lastRequestBody = b
	tt.mu.Unlock()

	return &http.Response{
		Body: &testReadCloser{
			b: bytes.NewBuffer(b),
		},
	}, nil
}

func (tt *testTransport) CancelRequest(req *http.Request) {}

func testProbe(opts *options.Options) (*probeResult, error) {
	p := &Probe{}
	err := p.Init("http_test", opts)
	if err != nil {
		return nil, err
	}
	patchWithTestTransport(p)

	target := endpoint.Endpoint{Name: "test.com"}
	result := p.newResult()
	req := p.httpRequestForTarget(target)

	p.runProbe(context.Background(), target, p.clientsForTarget(target), req, result)

	return result, nil
}

func TestProbeInitError(t *testing.T) {
	opts := func(c *configpb.ProbeConf) *options.Options {
		return &options.Options{
			Targets:   targets.StaticTargets("test.com"),
			Interval:  2 * time.Second,
			Timeout:   1 * time.Second,
			ProbeConf: c,
		}
	}
	var tests = []struct {
		desc    string
		c       *configpb.ProbeConf
		opts    *options.Options
		wantErr bool
	}{
		{
			desc: "default",
			c:    &configpb.ProbeConf{},
		},
		{
			desc: "with_valid_requests_per_probe",
			c: &configpb.ProbeConf{
				RequestsPerProbe:     proto.Int32(20),
				RequestsIntervalMsec: proto.Int32(50),
			},
		},
		{
			desc: "with_invalid_requests_per_probe",
			c: &configpb.ProbeConf{
				RequestsPerProbe:     proto.Int32(20),
				RequestsIntervalMsec: proto.Int32(60),
			},
			wantErr: true,
		},
		{
			desc: "invalid_url",
			c: &configpb.ProbeConf{
				RelativeUrl: proto.String("status"),
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			p := &Probe{}
			if test.opts == nil {
				test.opts = opts(test.c)
			}
			err := p.Init("http_test", test.opts)
			if err != nil {
				if !test.wantErr {
					t.Errorf("Got unexpected error: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Error("Expected error but didn't get it.")
			}
		})
	}
}

func TestProbeVariousMethods(t *testing.T) {
	mpb := func(s string) *configpb.ProbeConf_Method {
		return configpb.ProbeConf_Method(configpb.ProbeConf_Method_value[s]).Enum()
	}

	testBody := "Test HTTP Body"
	testHeaderName, testHeaderValue := "Content-Type", "application/json"

	var tests = []struct {
		input *configpb.ProbeConf
		want  string
	}{
		{&configpb.ProbeConf{}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Protocol: configpb.ProbeConf_HTTPS.Enum()}, "total: 1, success: 1"},
		{&configpb.ProbeConf{RequestsPerProbe: proto.Int32(1)}, "total: 1, success: 1"},
		{&configpb.ProbeConf{RequestsPerProbe: proto.Int32(4)}, "total: 4, success: 4"},
		{&configpb.ProbeConf{Method: mpb("GET")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("POST")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("POST"), Body: &testBody}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PUT")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PUT"), Body: &testBody}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("HEAD")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("DELETE")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PATCH")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("OPTIONS")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Headers: []*configpb.ProbeConf_Header{{Name: &testHeaderName, Value: &testHeaderValue}}}, "total: 1, success: 1"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test_case(%d)_config(%v)", i, test.input), func(t *testing.T) {
			opts := &options.Options{
				Targets:   targets.StaticTargets("test.com"),
				Interval:  2 * time.Second,
				Timeout:   time.Second,
				ProbeConf: test.input,
			}

			result, err := testProbe(opts)
			if err != nil {
				if fmt.Sprintf("error: '%s'", err.Error()) != test.want {
					t.Errorf("Unexpected initialization error: %v", err)
				}
				return
			}

			got := fmt.Sprintf("total: %d, success: %d", result.total, result.success)
			if got != test.want {
				t.Errorf("Mismatch got '%s', want '%s'", got, test.want)
			}
		})
	}
}

func TestProbeWithBody(t *testing.T) {
	testBody := "TestHTTPBody"
	testTarget := "test.com"
	// Build the expected response code map
	expectedMap := metrics.NewMap("resp", metrics.NewInt(0))
	expectedMap.IncKey(testBody)
	expected := expectedMap.String()

	p := &Probe{}
	err := p.Init("http_test", &options.Options{
		Targets:  targets.StaticTargets(testTarget),
		Interval: 2 * time.Second,
		ProbeConf: &configpb.ProbeConf{
			Body:                    &testBody,
			ExportResponseAsMetrics: proto.Bool(true),
		},
	})

	if err != nil {
		t.Errorf("Error while initializing probe: %v", err)
	}
	p.baseTransport = &testTransport{}
	target := endpoint.Endpoint{Name: testTarget}

	// Probe 1st run
	result := p.newResult()
	req := p.httpRequestForTarget(target)
	p.runProbe(context.Background(), target, p.clientsForTarget(target), req, result)
	got := result.respBodies.String()
	if got != expected {
		t.Errorf("response map: got=%s, expected=%s", got, expected)
	}

	// Probe 2nd run (we should get the same request body).
	p.runProbe(context.Background(), target, p.clientsForTarget(target), req, result)
	expectedMap.IncKey(testBody)
	expected = expectedMap.String()
	got = result.respBodies.String()
	if got != expected {
		t.Errorf("response map: got=%s, expected=%s", got, expected)
	}
}

func TestProbeWithLargeBody(t *testing.T) {
	for _, size := range []int{largeBodyThreshold - 1, largeBodyThreshold, largeBodyThreshold + 1, largeBodyThreshold * 2} {
		t.Run(fmt.Sprintf("size:%d", size), func(t *testing.T) {
			testProbeWithLargeBody(t, size)
		})
	}
}

func testProbeWithLargeBody(t *testing.T, bodySize int) {
	testBody := strings.Repeat("a", bodySize)
	testTarget := "test-large-body.com"

	p := &Probe{}
	err := p.Init("http_test", &options.Options{
		Targets:  targets.StaticTargets(testTarget),
		Interval: 2 * time.Second,
		ProbeConf: &configpb.ProbeConf{
			Body: &testBody,
			// Can't use ExportResponseAsMetrics for large bodies,
			// since maxResponseSizeForMetrics is small
			ExportResponseAsMetrics: proto.Bool(false),
		},
	})

	if err != nil {
		t.Errorf("Error while initializing probe: %v", err)
	}
	tt := &testTransport{}
	p.baseTransport = tt
	target := endpoint.Endpoint{Name: testTarget}

	// Probe 1st run
	result := p.newResult()
	req := p.httpRequestForTarget(target)
	p.runProbe(context.Background(), target, p.clientsForTarget(target), req, result)

	got := string(tt.lastRequestBody)
	if got != testBody {
		t.Errorf("response body length: got=%d, expected=%d", len(got), len(testBody))
	}

	// Probe 2nd run (we should get the same request body).
	p.runProbe(context.Background(), target, p.clientsForTarget(target), req, result)
	got = string(tt.lastRequestBody)
	if got != testBody {
		t.Errorf("response body length: got=%d, expected=%d", len(got), len(testBody))
	}
}

type testServer struct {
	addr *net.TCPAddr
	srv  *http.Server
}

func newTestServer(ctx context.Context, ipVer int) (*testServer, error) {
	ts := &testServer{}
	ln, err := net.Listen(map[int]string{4: "tcp4", 6: "tcp6", 0: "tcp"}[ipVer], ":0")
	if err != nil {
		return nil, err
	}
	ts.addr = ln.Addr().(*net.TCPAddr)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Host, "fail-test.com") {
			time.Sleep(2 * time.Second)
		}
		w.Write([]byte("ok"))
	})

	ts.srv = &http.Server{
		Handler: serverMux,
	}

	// Setup a background function to close server if context is canceled.
	go func() {
		<-ctx.Done()
		ts.srv.Close()
	}()

	go func() {
		ts.srv.Serve(ln)
	}()

	return ts, nil
}

func testMultipleTargetsMultipleRequests(t *testing.T, reqPerProbe int, ipVer int, keepAlive bool) {
	ctx, cancelF := context.WithCancel(context.Background())
	defer cancelF()

	ts, err := newTestServer(ctx, ipVer)
	if err != nil {
		t.Errorf("Error starting test HTTP server: %v", err)
		return
	}
	t.Logf("Started test HTTP server at: %v", ts.addr)

	testTargets := []string{"test.com", "fail-test.com", "fails-to-resolve.com"}
	var eps []endpoint.Endpoint
	for _, name := range testTargets {
		ip := ts.addr.IP
		if name == "fails-to-resolve.com" {
			ip = nil
		}
		eps = append(eps, endpoint.Endpoint{
			Name: name,
			IP:   ip,
		})
	}

	opts := &options.Options{
		Targets:             targets.StaticEndpoints(eps),
		Interval:            10 * time.Millisecond,
		Timeout:             9 * time.Millisecond,
		StatsExportInterval: 10 * time.Millisecond,
		ProbeConf: &configpb.ProbeConf{
			Port:             proto.Int32(int32(ts.addr.Port)),
			RequestsPerProbe: proto.Int32(int32(reqPerProbe)),
			ResolveFirst:     proto.Bool(true),
			KeepAlive:        proto.Bool(keepAlive),
		},
		IPVersion:  ipVer,
		LogMetrics: func(_ *metrics.EventMetrics) {},
	}

	p := &Probe{}
	err = p.Init("http_test", opts)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	dataChan := make(chan *metrics.EventMetrics, 100)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Start(ctx, dataChan)
	}()

	// Let's wait for 500ms. Probes should run about 50 times during this
	// period. For 3 targets, we should get about 150 eventmetrics, but let's
	// just wait for 30 EventMetrics, which should happen within 100ms on a
	// well running system.
	ems, err := testutils.MetricsFromChannel(dataChan, 30, 500*time.Millisecond)
	if err != nil && len(ems) < 9 { // 3 EventMetrics for each target.
		t.Errorf("Error getting 6 eventmetrics from data channel: %v", err)
	}

	// Let probe run for about 20 times, exporting data 10 times
	cancelF()
	wg.Wait() // Verifies that probe really stopped.

	latestVal := func(ems []*metrics.EventMetrics, name string) int {
		return int(ems[len(ems)-1].Metric(name).(*metrics.Int).Int64())
	}

	dataMap := testutils.MetricsMap(ems)
	for _, tgt := range testTargets {
		successVals, totalVals := dataMap["success"][tgt], dataMap["total"][tgt]
		var connEventVals []*metrics.EventMetrics
		if keepAlive {
			connEventVals = dataMap["connect_event"][tgt]
		}

		metricsCount := len(totalVals)
		t.Logf("Total metrics for %s: %d", tgt, len(totalVals))
		if metricsCount < 3 {
			t.Errorf("Too few metrics for %s: %v (less than 3)", tgt, totalVals)
			continue
		}

		// Expected values on a good system: metricsCount * reqPerProbe
		// Let's go with more conservative values and divide by 3 to take
		// slowness of CI/CD systems into account:
		wantCounter := (metricsCount * reqPerProbe) / 3
		minTotal, minSuccess := wantCounter, wantCounter
		if tgt == "fails-to-resolve.com" {
			minTotal, minSuccess = 0, 0
		}
		if tgt == "fail-test.com" {
			minSuccess = 0
		}
		assert.LessOrEqualf(t, minTotal, latestVal(totalVals, "total"), "total for target: %s", tgt)
		assert.LessOrEqualf(t, minSuccess, latestVal(successVals, "success"), "success for target: %s", tgt)

		if keepAlive && tgt == "test.com" {
			maxConnEvent := reqPerProbe * 2
			assert.GreaterOrEqualf(t, maxConnEvent, latestVal(connEventVals, "connect_event"), "connect_event for target: %s", tgt)
		}
	}
}

func TestMultipleTargetsMultipleRequests(t *testing.T) {
	for _, ipVer := range []int{0, 4, 6} {
		// Disable windows IPv6 tests.
		if ipVer == 6 && os.Getenv("DISABLE_IPV6_TESTS") == "yes" {
			return
		}
		for _, reqPerProbe := range []int{1, 3} {
			for _, keepAlive := range []bool{false, true} {
				t.Run(fmt.Sprintf("ip_ver=%d,req_per_probe=%d,keepAlive=%v", ipVer, reqPerProbe, keepAlive), func(t *testing.T) {
					testMultipleTargetsMultipleRequests(t, reqPerProbe, ipVer, keepAlive)
				})
			}
		}
	}
}

func compareNumberOfMetrics(t *testing.T, ems []*metrics.EventMetrics, targets [2]string, wantCloseRange bool) {
	t.Helper()

	m := testutils.MetricsMap(ems)["success"]
	num1 := len(m[targets[0]])
	num2 := len(m[targets[1]])

	diff := num1 - num2
	threshold := num1 / 2
	notCloseRange := diff < -(threshold) || diff > threshold

	if notCloseRange && wantCloseRange {
		t.Errorf("Number of metrics for two targets are not within a close range (%d, %d)", num1, num2)
	}
	if !notCloseRange && !wantCloseRange {
		t.Errorf("Number of metrics for two targets are within a close range (%d, %d)", num1, num2)
	}
}

func TestUpdateTargetsAndStartProbes(t *testing.T) {
	testTargets := [2]string{"test1.com", "test2.com"}
	reqPerProbe := int64(3)
	opts := &options.Options{
		Targets:             targets.StaticTargets(fmt.Sprintf("%s,%s", testTargets[0], testTargets[1])),
		Interval:            10 * time.Millisecond,
		StatsExportInterval: 20 * time.Millisecond,
		ProbeConf:           &configpb.ProbeConf{RequestsPerProbe: proto.Int32(int32(reqPerProbe))},
		LogMetrics:          func(_ *metrics.EventMetrics) {},
	}
	p := &Probe{}
	p.Init("http_test", opts)
	patchWithTestTransport(p)

	dataChan := make(chan *metrics.EventMetrics, 100)

	ctx, cancelF := context.WithCancel(context.Background())
	p.updateTargetsAndStartProbes(ctx, dataChan)
	if len(p.cancelFuncs) != 2 {
		t.Errorf("len(p.cancelFunc)=%d, want=2", len(p.cancelFuncs))
	}
	ems, _ := testutils.MetricsFromChannel(dataChan, 100, time.Second)
	compareNumberOfMetrics(t, ems, testTargets, true)

	// Updates targets to just one target. This should cause one probe loop to
	// exit. We should get only one data stream after that.
	opts.Targets = targets.StaticTargets(testTargets[0])
	p.updateTargetsAndStartProbes(ctx, dataChan)
	if len(p.cancelFuncs) != 1 {
		t.Errorf("len(p.cancelFunc)=%d, want=1", len(p.cancelFuncs))
	}
	ems, _ = testutils.MetricsFromChannel(dataChan, 100, time.Second)
	compareNumberOfMetrics(t, ems, testTargets, false)

	cancelF()
	p.wait()
}

type tokenSource struct {
	tok string
	err error
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: ts.tok}, ts.err
}

func TestRunProbeWithOAuth(t *testing.T) {
	p := &Probe{
		l: &logger.Logger{},
	}

	ts := &tokenSource{}
	p.oauthTS = ts

	testTarget := endpoint.Endpoint{Name: "test.com"}
	reqPerProbe := int64(3)
	opts := options.DefaultOptions()
	opts.ProbeConf = &configpb.ProbeConf{RequestsPerProbe: proto.Int32(int32(reqPerProbe))}

	if err := p.Init("http_test", opts); err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	patchWithTestTransport(p)

	req := p.httpRequestForTarget(testTarget)
	result := p.newResult()

	var wantSuccess, wantTotal int64
	for _, tok := range []string{"tok-1", "tok-2", ""} {
		wantTotal += reqPerProbe
		wantHeader := "Bearer <token-missing>"
		if tok != "" {
			wantSuccess += reqPerProbe
			wantHeader = "Bearer " + tok
		}

		t.Run("tok: "+tok, func(t *testing.T) {
			ts.tok = tok
			if tok == "" {
				ts.err = errors.New("bad token")
			} else {
				ts.err = nil
			}

			clients := p.clientsForTarget(testTarget)
			p.runProbe(context.Background(), testTarget, clients, req, result)

			if result.success != wantSuccess || result.total != wantTotal {
				t.Errorf("success=%d,wanted=%d; total=%d,wanted=%d", result.success, wantSuccess, result.total, wantTotal)
			}

			for _, client := range clients {
				tt := client.Transport.(*testTransport)
				if tt.lastAuthHeader != wantHeader {
					t.Errorf("Auth header: %s, wanted: %s", tt.lastAuthHeader, wantHeader)
				}
			}
		})
	}
}

func TestGetTransport(t *testing.T) {
	opts := options.DefaultOptions()
	p := &Probe{opts: opts}

	tests := []struct {
		desc             string
		keepAlive        bool
		disableHTTP2     bool
		proxy            string
		disableCertCheck bool
	}{
		{
			desc: "default transport",
		},
		{
			desc:         "disable_http2",
			disableHTTP2: true,
		},
		{
			desc:      "disable_keepalive",
			keepAlive: true,
		},
		{
			desc:  "with_proxy",
			proxy: "http://test-proxy",
		},
		{
			desc:             "disable_cert_check",
			disableCertCheck: true,
		},
		{
			desc:             "all_custom",
			disableHTTP2:     true,
			keepAlive:        true,
			proxy:            "http://test-proxy",
			disableCertCheck: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			p.c = &configpb.ProbeConf{
				DisableHttp2:          &test.disableHTTP2,
				KeepAlive:             &test.keepAlive,
				RequestsPerProbe:      proto.Int32(10),
				ProxyUrl:              &test.proxy,
				DisableCertValidation: &test.disableCertCheck,
			}

			transport, err := p.getTransport()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			assert.Equal(t, opts.Timeout, transport.TLSHandshakeTimeout)
			assert.Equal(t, !test.keepAlive, transport.DisableKeepAlives)
			assert.Equal(t, !test.disableHTTP2, transport.ForceAttemptHTTP2)

			if test.disableHTTP2 {
				assert.NotNil(t, transport.TLSNextProto)
				assert.Empty(t, transport.TLSNextProto)
			} else {
				assert.Nil(t, transport.TLSNextProto)
			}

			if test.keepAlive {
				assert.Equal(t, 10, transport.MaxIdleConnsPerHost)
				assert.Equal(t, 2*opts.Interval, transport.IdleConnTimeout)
			}

			if test.proxy != "" {
				proxy, _ := transport.Proxy(nil)
				assert.Equal(t, test.proxy, proxy.String())
			}

			if test.disableCertCheck {
				assert.Equal(t, true, transport.TLSClientConfig.InsecureSkipVerify)
			}
		})
	}
}
