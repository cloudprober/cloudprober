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
	"net/http"
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
	for _, c := range p.clients {
		c.Transport = &testTransport{
			keepAuthHeader: keepAuthHeader,
		}
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
	req := p.httpRequestForTarget(target, nil)
	p.runProbe(context.Background(), target, req, result)

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
	p.clients = []*http.Client{{Transport: &testTransport{}}}
	target := endpoint.Endpoint{Name: testTarget}

	// Probe 1st run
	result := p.newResult()
	req := p.httpRequestForTarget(target, nil)
	p.runProbe(context.Background(), target, req, result)
	got := result.respBodies.String()
	if got != expected {
		t.Errorf("response map: got=%s, expected=%s", got, expected)
	}

	// Probe 2nd run (we should get the same request body).
	p.runProbe(context.Background(), target, req, result)
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
	p.clients = []*http.Client{{Transport: tt}}
	target := endpoint.Endpoint{Name: testTarget}

	// Probe 1st run
	result := p.newResult()
	req := p.httpRequestForTarget(target, nil)
	p.runProbe(context.Background(), target, req, result)

	got := string(tt.lastRequestBody)
	if got != testBody {
		t.Errorf("response body length: got=%d, expected=%d", len(got), len(testBody))
	}

	// Probe 2nd run (we should get the same request body).
	p.runProbe(context.Background(), target, req, result)
	got = string(tt.lastRequestBody)
	if got != testBody {
		t.Errorf("response body length: got=%d, expected=%d", len(got), len(testBody))
	}
}

func TestMultipleTargetsMultipleRequests(t *testing.T) {
	testTargets := []string{"test.com", "fail-test.com", "fails-to-resolve.com"}
	reqPerProbe := int64(3)
	opts := &options.Options{
		Targets:             targets.StaticTargets(strings.Join(testTargets, ",")),
		Interval:            10 * time.Millisecond,
		StatsExportInterval: 20 * time.Millisecond,
		ProbeConf:           &configpb.ProbeConf{RequestsPerProbe: proto.Int32(int32(reqPerProbe))},
		LogMetrics:          func(_ *metrics.EventMetrics) {},
	}

	p := &Probe{}
	err := p.Init("http_test", opts)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	patchWithTestTransport(p)

	ctx, cancelF := context.WithCancel(context.Background())
	dataChan := make(chan *metrics.EventMetrics, 100)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Start(ctx, dataChan)
	}()

	// target -> [success, total]
	wantData := map[string][2]int64{
		"test.com": {2 * reqPerProbe, 2 * reqPerProbe},

		// Test transport is configured to fail this.
		"fail-test.com": {0, 2 * reqPerProbe},

		// No probes sent because of bad target (http)
		"fails-to-resolve.com": {0, 0},
	}

	ems, err := testutils.MetricsFromChannel(dataChan, 100, time.Second)
	// We should receive at least 4 eventmetrics: 2 probe cycle x 2 targets.
	if err != nil && len(ems) < 4 {
		t.Errorf("Error getting 4 eventmetrics from data channel: %v", err)
	}

	// Following verifies that we are able to cleanly stop the probe.
	cancelF()
	wg.Wait()

	dataMap := testutils.MetricsMap(ems)
	for tgt, d := range wantData {
		wantSuccessVal, wantTotalVal := d[0], d[1]
		successVals, totalVals := dataMap["success"][tgt], dataMap["total"][tgt]

		if len(successVals) < 1 {
			t.Errorf("Success metric for %s: %v (less than 1)", tgt, successVals)
			continue
		}
		latestVal := successVals[len(successVals)-1].Metric("success").(*metrics.Int).Int64()
		if latestVal < wantSuccessVal {
			t.Errorf("Got success value for target (%s): %d, want: %d", tgt, latestVal, wantSuccessVal)
		}

		if len(totalVals) < 1 {
			t.Errorf("Total metric for %s: %v (less than 1)", tgt, totalVals)
			continue
		}
		latestVal = totalVals[len(totalVals)-1].Metric("total").(*metrics.Int).Int64()
		if latestVal < wantTotalVal {
			t.Errorf("Got total value for target (%s): %d, want: %d", tgt, latestVal, wantTotalVal)
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

	req := p.httpRequestForTarget(testTarget, nil)
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

			p.runProbe(context.Background(), testTarget, req, result)

			if result.success != wantSuccess || result.total != wantTotal {
				t.Errorf("success=%d,wanted=%d; total=%d,wanted=%d", result.success, wantSuccess, result.total, wantTotal)
			}

			for _, client := range p.clients {
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
