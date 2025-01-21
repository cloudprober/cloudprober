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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/testutils"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"
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
	ctx := req.Context()
	trace := httptrace.ContextClientTrace(ctx)
	if trace != nil {
		if trace.DNSDone != nil {
			time.Sleep(time.Millisecond)
			trace.DNSDone(httptrace.DNSDoneInfo{})
		}
		if trace.ConnectDone != nil {
			time.Sleep(time.Millisecond)
			trace.ConnectDone("", "", nil)
		}
		if trace.TLSHandshakeDone != nil && req.URL.Scheme == "https" {
			time.Sleep(time.Millisecond)
			trace.TLSHandshakeDone(tls.ConnectionState{}, nil)
		}
		if trace.WroteRequest != nil {
			time.Sleep(time.Millisecond)
			trace.WroteRequest(httptrace.WroteRequestInfo{})
		}
		if trace.GotFirstResponseByte != nil {
			time.Sleep(time.Millisecond)
			trace.GotFirstResponseByte()
		}
	}

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

	b, err := io.ReadAll(req.Body)
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

	runReq := &sched.RunProbeForTargetRequest{
		Target: endpoint.Endpoint{Name: "test.com"},
	}
	p.runProbe(context.Background(), runReq)

	return runReq.Result.(*probeResult), nil
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
	tests := []struct {
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

	tests := []struct {
		input *configpb.ProbeConf
		want  string
	}{
		{&configpb.ProbeConf{}, "total: 1, success: 1"},
		{&configpb.ProbeConf{RequestsPerProbe: proto.Int32(1)}, "total: 1, success: 1"},
		{&configpb.ProbeConf{RequestsPerProbe: proto.Int32(4)}, "total: 4, success: 4"},
		{&configpb.ProbeConf{Method: mpb("GET")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("POST")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("POST"), Body: []string{testBody}}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PUT")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PUT"), Body: []string{testBody}}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("HEAD")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("DELETE")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("PATCH")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Method: mpb("OPTIONS")}, "total: 1, success: 1"},
		{&configpb.ProbeConf{Headers: []*configpb.ProbeConf_Header{{Name: &testHeaderName, Value: &testHeaderValue}}}, "total: 1, success: 1"},
	}

	for i, test := range tests[:1] {
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

func testProbeWithBody(t *testing.T, probeConf *configpb.ProbeConf, wantBody string) {
	testTarget := "test.com"

	// Can't use ExportResponseAsMetrics for large bodies since
	// maxResponseSizeForMetrics is small
	probeConf.ExportResponseAsMetrics = proto.Bool(false)

	p := &Probe{}
	err := p.Init("http_test", &options.Options{
		Targets:   targets.StaticTargets(testTarget),
		Interval:  2 * time.Second,
		ProbeConf: probeConf,
	})
	if err != nil {
		t.Errorf("Error while initializing probe: %v", err)
	}
	tt := &testTransport{}
	p.baseTransport = tt
	target := endpoint.Endpoint{Name: testTarget}

	// Probe 1st run
	req := p.httpRequestForTarget(target)
	runReq := &sched.RunProbeForTargetRequest{
		Target: target,
		TargetState: &targetState{
			req: req,
		},
	}
	p.runProbe(context.Background(), runReq)

	got := string(tt.lastRequestBody)
	if got != wantBody {
		t.Errorf("response body length: got=%d, expected=%d", len(got), len(wantBody))
	}

	// Probe 2nd run (we should get the same request body).
	p.runProbe(context.Background(), runReq)
	got = string(tt.lastRequestBody)
	if got != wantBody {
		t.Errorf("response body length: got=%d, expected=%d", len(got), len(wantBody))
	}
}

func TestProbeWithBody(t *testing.T) {
	largeBodyThreshold := bytes.MinRead // 512.

	for _, size := range []int{12, largeBodyThreshold - 1, largeBodyThreshold, largeBodyThreshold + 1, largeBodyThreshold * 2} {
		t.Run(fmt.Sprintf("size:%d", size), func(t *testing.T) {
			testBody := strings.Repeat("a", size)
			testProbeWithBody(t, &configpb.ProbeConf{Body: []string{testBody}}, testBody)
		})
	}
}

func TestProbeWithBodyFile(t *testing.T) {
	testBody := strings.Repeat("a", 1024)
	f, err := os.CreateTemp("", "test-body-file")
	if err != nil {
		t.Fatalf("Error creating temp file: %v", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(testBody); err != nil {
		t.Fatalf("Error writing to temp file: %v", err)
	}

	testProbeWithBody(t, &configpb.ProbeConf{BodyFile: proto.String(f.Name())}, testBody)
}

type testServer struct {
	addr *net.TCPAddr
	srv  *http.Server
}

func newTestServer(t *testing.T, ctx context.Context, ipVer int) (*testServer, error) {
	t.Helper()
	ts := &testServer{}
	network := map[int]string{4: "tcp4", 6: "tcp6", 0: "tcp"}[ipVer]
	addr := map[int]string{4: "127.0.0.1:0", 6: "[::1]:0", 0: "localhost:0"}[ipVer]
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	ts.addr = ln.Addr().(*net.TCPAddr)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Host, "fail-test.com") {
			time.Sleep(2 * time.Second)
		}
		if r.URL.Path == "/test-body-size" {
			size, _ := strconv.Atoi(r.URL.Query().Get("size"))
			b, err := io.ReadAll(r.Body)
			assert.Equal(t, nil, err)
			assert.Equal(t, size, len(b))
		}
		if r.URL.Path == "/redirect" {
			redirectURL := r.URL.Query().Get("url")
			assert.NotEmpty(t, redirectURL)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
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

type testProbeOpts struct {
	reqPerProbe int
	ipVer       int
	keepAlive   bool
	body        string
	method      string
	targets     []string
	url         string
}

func testMultipleTargetsMultipleRequests(t *testing.T, probeOpts *testProbeOpts) {
	if probeOpts.reqPerProbe == 0 {
		probeOpts.reqPerProbe = 1
	}

	ctx, cancelF := context.WithCancel(context.Background())
	defer cancelF()

	ts, err := newTestServer(t, ctx, probeOpts.ipVer)
	if err != nil {
		t.Errorf("Error starting test HTTP server: %v", err)
		return
	}
	t.Logf("Started test HTTP server at: %v", ts.addr)

	var eps []endpoint.Endpoint
	for _, name := range probeOpts.targets {
		ip := ts.addr.IP
		if name == "fails-to-resolve.com" {
			ip = nil
		}
		eps = append(eps, endpoint.Endpoint{
			Name: name,
			IP:   ip,
		})
	}

	method := map[string]configpb.ProbeConf_Method{
		"GET":  configpb.ProbeConf_GET,
		"POST": configpb.ProbeConf_POST,
	}[probeOpts.method]

	opts := &options.Options{
		Targets:             targets.StaticEndpoints(eps),
		Interval:            10 * time.Millisecond,
		Timeout:             9 * time.Millisecond,
		StatsExportInterval: 10 * time.Millisecond,
		ProbeConf: &configpb.ProbeConf{
			RelativeUrl:      &probeOpts.url,
			Port:             proto.Int32(int32(ts.addr.Port)),
			Method:           &method,
			RequestsPerProbe: proto.Int32(int32(probeOpts.reqPerProbe)),
			ResolveFirst:     proto.Bool(true),
			KeepAlive:        proto.Bool(probeOpts.keepAlive),
			Body:             []string{probeOpts.body},
		},
		IPVersion: probeOpts.ipVer,
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
	// just wait for 60 EventMetrics, which should happen within 200ms on a
	// well running system.
	ems, err := testutils.MetricsFromChannel(dataChan, 60, 500*time.Millisecond)
	if err != nil && len(ems) < 9 { // 3 EventMetrics for each target.
		t.Errorf("Error getting 6 eventmetrics from data channel: %v", err)
	}

	// Let probe run for about 20 times, exporting data 10 times
	cancelF()
	wg.Wait() // Verifies that probe really stopped.

	dataMap := testutils.MetricsMapByTarget(ems)
	for _, tgt := range probeOpts.targets {
		totalVals := dataMap[tgt]["total"]

		metricsCount := len(totalVals)
		t.Logf("Total metrics for %s: %d", tgt, len(totalVals))
		if metricsCount < 3 {
			t.Errorf("Too few metrics for %s: %v (less than 3)", tgt, totalVals)
			continue
		}

		// Expected values on a good system: metricsCount * reqPerProbe
		// Let's go with more conservative values and divide by 3 to take
		// slowness of CI systems into account:
		wantCounter := (metricsCount * probeOpts.reqPerProbe) / 3
		minTotal, minSuccess := wantCounter, wantCounter
		if tgt == "fails-to-resolve.com" {
			minTotal, minSuccess = 0, 0
		}
		if tgt == "fail-test.com" {
			minSuccess = 0
		}
		assert.LessOrEqualf(t, int64(minTotal), dataMap.LastValueInt64(tgt, "total"), "total for target: %s", tgt)
		assert.LessOrEqualf(t, int64(minSuccess), dataMap.LastValueInt64(tgt, "success"), "success for target: %s", tgt)

		if probeOpts.keepAlive && tgt == "test.com" {
			connEvent := dataMap.LastValueInt64(tgt, "connect_event")
			minConnEvent := int64(probeOpts.reqPerProbe * 1)
			maxConnEvent := int64(probeOpts.reqPerProbe * 2)
			if connEvent <= minConnEvent && connEvent >= maxConnEvent {
				t.Errorf("connect_event for target: %s, got: %d, want: <= %d, >= %d", tgt, connEvent, maxConnEvent, minConnEvent)
			}
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
					testMultipleTargetsMultipleRequests(t, &testProbeOpts{
						reqPerProbe: reqPerProbe,
						ipVer:       ipVer,
						keepAlive:   keepAlive,
						targets:     []string{"test.com", "fail-test.com", "fails-to-resolve.com"},
					})
				})
			}
		}
	}
}

func TestProbeWithReqBody(t *testing.T) {
	largeBodyThreshold := bytes.MinRead // 512.

	for _, size := range []int{0, 32, largeBodyThreshold + 1} {
		for _, method := range []string{"GET", "POST"} {
			for _, withRedirect := range []bool{false, true} {
				for _, keepAlive := range []bool{false, true} {
					testName := fmt.Sprintf("method:%s,bodysize:%d,keepAlive:%v,withRedirect:%v", method, size, keepAlive, withRedirect)
					t.Run(testName, func(t *testing.T) {
						u := "/test-body-size?size=" + strconv.Itoa(size)
						if withRedirect {
							u = "/redirect?url=" + url.QueryEscape(u)
						}
						testMultipleTargetsMultipleRequests(t, &testProbeOpts{
							body:      strings.Repeat("a", size),
							method:    method,
							keepAlive: keepAlive,
							targets:   []string{"test.com"},
							url:       u,
						})
					})
				}
			}
		}
	}
}

func compareNumberOfMetrics(t *testing.T, ems []*metrics.EventMetrics, targets [2]string, wantCloseRange bool) {
	t.Helper()

	m := testutils.MetricsMapByTarget(ems).Filter("success")
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
			runReq := &sched.RunProbeForTargetRequest{
				Target: testTarget,
				Result: result,
				TargetState: &targetState{
					req: req,
				},
			}
			p.runProbe(context.Background(), runReq)

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
		desc               string
		keepAlive          bool
		disableHTTP2       bool
		proxy              string
		proxyConnectHeader map[string]string
		disableCertCheck   bool
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
			desc:               "with_proxy",
			proxy:              "http://test-proxy",
			proxyConnectHeader: map[string]string{"Proxy-Connect": "test"},
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
				ProxyConnectHeader:    test.proxyConnectHeader,
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

			if test.proxyConnectHeader != nil {
				for k, v := range transport.ProxyConnectHeader {
					assert.Equal(t, v[0], test.proxyConnectHeader[k], "Proxy-connect header mismatch for key: %s", k)
				}
			}

			if test.disableCertCheck {
				assert.Equal(t, true, transport.TLSClientConfig.InsecureSkipVerify)
			}
		})
	}
}

func TestProbeInitRedirects(t *testing.T) {
	p := &Probe{}
	maxRedirects := 10

	opts := &options.Options{
		Targets:  targets.StaticTargets("test.com"),
		Interval: 10 * time.Millisecond,
		ProbeConf: &configpb.ProbeConf{
			MaxRedirects: proto.Int32(int32(maxRedirects)),
		},
	}

	err := p.Init("http_test", opts)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if p.redirectFunc == nil {
		t.Errorf("expected redirectFunc to be initialized, found redirectFunc was not initialized")
	}

	req := &http.Request{}
	via := make([]*http.Request, maxRedirects)
	if err := p.redirectFunc(req, via[:maxRedirects-1]); err != nil {
		t.Errorf("expected redirectFunc to return nil, found %v", err)
	}

	if err := p.redirectFunc(req, via); !errors.Is(err, http.ErrUseLastResponse) {
		t.Errorf("expected redirectFunc to return ErrUseLastResponse, found %v", err)
	}
}

func TestProbeInitRedirectsNotSet(t *testing.T) {
	p := &Probe{}

	opts := &options.Options{
		Targets:  targets.StaticTargets("test.com"),
		Interval: 10 * time.Millisecond,
		ProbeConf: &configpb.ProbeConf{
			MaxRedirects: nil,
		},
	}

	err := p.Init("http_test", opts)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if p.redirectFunc != nil {
		t.Errorf("expected redirectFunc to be nil, found redirectFunc was initialized")
	}
}

func TestClientsForTarget(t *testing.T) {
	tests := []struct {
		name                string
		conf                *configpb.ProbeConf
		https               bool
		baseTransport       *http.Transport
		target              endpoint.Endpoint
		wantNumClients      int
		tlsConfigServerName string
	}{
		{
			name:           "default",
			conf:           &configpb.ProbeConf{},
			baseTransport:  http.DefaultTransport.(*http.Transport),
			target:         endpoint.Endpoint{Name: "cloudprober.org"},
			wantNumClients: 1,
		},
		{
			name:           "2_clients",
			conf:           &configpb.ProbeConf{RequestsPerProbe: proto.Int32(2)},
			baseTransport:  http.DefaultTransport.(*http.Transport),
			target:         endpoint.Endpoint{Name: "cloudprober.org"},
			wantNumClients: 2,
		},
		{
			name: "2_clients_https",
			conf: &configpb.ProbeConf{
				RequestsPerProbe: proto.Int32(2),
			},
			https:          true,
			baseTransport:  http.DefaultTransport.(*http.Transport),
			target:         endpoint.Endpoint{Name: "cloudprober.org"},
			wantNumClients: 2,
		},
		{
			name: "2_clients_https_server_name",
			conf: &configpb.ProbeConf{
				RequestsPerProbe: proto.Int32(2),
			},
			baseTransport: http.DefaultTransport.(*http.Transport),
			https:         true,
			target: endpoint.Endpoint{
				Name: "cloudprober.org",
				IP:   net.ParseIP("1.2.3.4"),
			},
			wantNumClients:      2,
			tlsConfigServerName: "cloudprober.org",
		},
		{
			name: "2_clients_https_server_name_from_fqdn",
			conf: &configpb.ProbeConf{
				RequestsPerProbe: proto.Int32(2),
			},
			baseTransport: http.DefaultTransport.(*http.Transport),
			target: endpoint.Endpoint{
				Name: "cloudprober.org",
				IP:   net.ParseIP("1.2.3.4"),
				Labels: map[string]string{
					"__cp_scheme__": "https", // Note: using target label here.
					"fqdn":          "manugarg.com",
				},
			},
			wantNumClients:      2,
			tlsConfigServerName: "manugarg.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.https {
				tt.conf.SchemeType = &configpb.ProbeConf_Scheme_{
					Scheme: configpb.ProbeConf_HTTPS,
				}
			}
			p := &Probe{
				baseTransport: tt.baseTransport,
				c:             tt.conf,
			}
			gotClients := p.clientsForTarget(tt.target)
			assert.Equal(t, tt.wantNumClients, len(gotClients), "number of clients is not as expected")

			for _, c := range gotClients {
				tlsConfig := c.Transport.(*http.Transport).TLSClientConfig
				assert.Equal(t, tt.tlsConfigServerName, tlsConfig.ServerName, "TLS config server name is not as expected")
			}
		})
	}
}

func TestParseLatencyBreakdown(t *testing.T) {
	tests := []struct {
		name string
		lb   []configpb.ProbeConf_LatencyBreakdown
		base metrics.LatencyValue
		want *latencyDetails
	}{
		{
			name: "default",
			want: nil,
		},
		{
			name: "all",
			lb: []configpb.ProbeConf_LatencyBreakdown{
				configpb.ProbeConf_ALL_STAGES,
			},
			base: metrics.NewFloat(0),
			want: &latencyDetails{
				dnsLatency:       metrics.NewFloat(0),
				connectLatency:   metrics.NewFloat(0),
				tlsLatency:       metrics.NewFloat(0),
				reqWriteLatency:  metrics.NewFloat(0),
				firstByteLatency: metrics.NewFloat(0),
			},
		},
		{
			name: "dns_tls",
			lb: []configpb.ProbeConf_LatencyBreakdown{
				configpb.ProbeConf_DNS_LATENCY,
				configpb.ProbeConf_TLS_HANDSHAKE_LATENCY,
			},
			base: metrics.NewDistribution([]float64{.01, .1, .5}),
			want: &latencyDetails{
				dnsLatency: metrics.NewDistribution([]float64{.01, .1, .5}),
				tlsLatency: metrics.NewDistribution([]float64{.01, .1, .5}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				c: &configpb.ProbeConf{
					LatencyBreakdown: tt.lb,
				},
			}

			if got := p.parseLatencyBreakdown(tt.base); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Probe.parseLatencyBreakdown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProbeWithLatencyBreakdown(t *testing.T) {
	ts := time.Unix(1711090290, 0)

	tests := []struct {
		name        string
		tls         bool
		lb          []configpb.ProbeConf_LatencyBreakdown
		wantNil     []string
		wantNonNil  []string
		wantZero    string
		wantMetrics []string
	}{
		{
			name: "all",
			lb: []configpb.ProbeConf_LatencyBreakdown{
				configpb.ProbeConf_ALL_STAGES,
			},
			wantNonNil:  []string{"dns", "connect", "tls_handshake", "req_write", "first_byte"},
			wantZero:    "tls_handshake",
			wantMetrics: []string{"dns_latency", "connect_latency", "tls_handshake_latency", "req_write_latency", "first_byte_latency"},
		},
		{
			name: "dns_tls",
			tls:  true,
			lb: []configpb.ProbeConf_LatencyBreakdown{
				configpb.ProbeConf_DNS_LATENCY,
				configpb.ProbeConf_TLS_HANDSHAKE_LATENCY,
			},
			wantNonNil:  []string{"dns", "tls_handshake"},
			wantNil:     []string{"connect", "req_write", "first_byte"},
			wantMetrics: []string{"dns_latency", "tls_handshake_latency"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &configpb.ProbeConf{
				LatencyBreakdown: tt.lb,
			}
			if tt.tls {
				cfg.SchemeType = &configpb.ProbeConf_Scheme_{
					Scheme: configpb.ProbeConf_HTTPS,
				}
			}

			opts := options.DefaultOptions()
			opts.ProbeConf = cfg

			p := &Probe{}
			if err := p.Init("http_test", opts); err != nil {
				t.Errorf("Error while initializing probe: %v", err)
			}

			patchWithTestTransport(p)

			target := endpoint.Endpoint{Name: "test.com"}
			result := p.newResult()

			p.runProbe(context.Background(), &sched.RunProbeForTargetRequest{Target: target, Result: result})

			assert.NotNil(t, result.latencyBreakdown, "latencyDetails not populated")

			lb := result.latencyBreakdown
			latenciesMap := map[string]metrics.Value{
				"dns":           lb.dnsLatency,
				"connect":       lb.connectLatency,
				"tls_handshake": lb.tlsLatency,
				"req_write":     lb.reqWriteLatency,
				"first_byte":    lb.firstByteLatency,
			}

			for _, k := range tt.wantNil {
				assert.Nil(t, latenciesMap[k], fmt.Sprintf("%s: latency value not populated", k))
			}
			for _, k := range tt.wantNonNil {
				assert.NotNil(t, latenciesMap[k], fmt.Sprintf("%s: latency value not populated", k))
				if k != tt.wantZero {
					assert.Greater(t, latenciesMap[k].(metrics.NumValue).Float64(), float64(0), fmt.Sprintf("%s: latency value is not non-zero", k))
				} else {
					assert.Zero(t, latenciesMap[k].(metrics.NumValue).Float64(), fmt.Sprintf("%s: latency value is zero", k))
				}
			}

			em := result.Metrics(ts, 0, p.opts)[0]
			for _, m := range tt.wantMetrics {
				assert.NotNil(t, em.Metric(m), fmt.Sprintf("%s: metric not exported", m))
				wantValue := latenciesMap[strings.TrimSuffix(m, "_latency")]
				assert.Equal(t, wantValue.String(), em.Metric(m).String(), fmt.Sprintf("%s: metric value not as expected", m))
			}

			// 5 more metrics are exported for total, success, latency, timeouts, resp_code
			wantNumMetrics := len(tt.wantMetrics) + 5
			assert.Equal(t, wantNumMetrics, len(em.MetricsKeys()), "number of metrics exported")
		})
	}
}
