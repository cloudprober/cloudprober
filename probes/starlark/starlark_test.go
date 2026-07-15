// Copyright 2026 The Cloudprober Authors.
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

package starlark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tlsconfigpb "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/starlark/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// ---------------------------------------------------------------------------
// Test helpers

// loginCartServer mimics a tiny token-auth API.
//
//	POST /login   {user,pass}    -> {"token":"tok"} with the chosen status
//	GET  /cart    Bearer tok     -> 200 {"items":[]}, else 401
func loginCartServer(t *testing.T, loginStatus int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["user"] != "u" || body["pass"] != "p" {
			http.Error(w, "bad creds", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(loginStatus)
		_, _ = w.Write([]byte(`{"token": "tok"}`))
	})
	mux.HandleFunc("/cart", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": []}`))
	})
	return httptest.NewServer(mux)
}

func newOpts(t *testing.T, target string, source string) *options.Options {
	t.Helper()
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets(target)
	opts.Timeout = 5 * time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{
		Source: proto.String(source),
	}
	return opts
}

func hostFromServer(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	return u.Host
}

const checkoutScript = `
def probe(target):
    base = "http://%s:%d" % (target.name, target.port)
    r = http.post(
        url = base + "/login",
        json = {"user": "u", "pass": "p"},
    )
    assert.http_status(r, 200)
    token = r.json()["token"]

    r = http.get(
        url = base + "/cart",
        headers = {"Authorization": "Bearer " + token},
    )
    assert.http_status(r, 200)
`

// ---------------------------------------------------------------------------
// End-to-end success / failure paths

func TestStarlarkProbe_Success(t *testing.T) {
	srv := loginCartServer(t, http.StatusOK)
	defer srv.Close()

	opts := newOpts(t, hostFromServer(t, srv), checkoutScript)
	p := &Probe{}
	if err := p.Init("script-success", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	results := p.RunOnce(context.Background())
	assert.Equal(t, 1, len(results))
	r := results[0]
	assert.True(t, r.Success, "expected success, error=%v", r.Error)
	assert.Nil(t, r.Error)
	assert.True(t, r.Latency > 0)
}

func TestStarlarkProbe_AssertionFailure(t *testing.T) {
	srv := loginCartServer(t, http.StatusServiceUnavailable)
	defer srv.Close()

	opts := newOpts(t, hostFromServer(t, srv), checkoutScript)
	p := &Probe{}
	if err := p.Init("script-fail", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	results := p.RunOnce(context.Background())
	assert.Equal(t, 1, len(results))
	r := results[0]
	assert.False(t, r.Success)
	assert.NotNil(t, r.Error)
	assert.Contains(t, r.Error.Error(), "expected 200")
}

// TestStarlarkProbe_OuterTimeoutCancelsInFlight pins the bug we fixed where the
// http builtin used http.DefaultClient with no deadline — a slow server hung
// the probe forever instead of failing on opts.Timeout.
func TestStarlarkProbe_OuterTimeoutCancelsInFlight(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request is cancelled. Without context plumbing
		// this would never return and the test would hang.
		<-r.Context().Done()
	}))
	defer srv.Close()

	source := `
def probe(target):
    r = http.get(url = "http://%s:%d/" % (target.name, target.port))
    assert.http_status(r, 200)
`
	opts := newOpts(t, hostFromServer(t, srv), source)
	opts.Timeout = 250 * time.Millisecond

	p := &Probe{}
	if err := p.Init("script-timeout", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	start := time.Now()
	results := p.RunOnce(context.Background())
	elapsed := time.Since(start)

	assert.Equal(t, 1, len(results))
	assert.False(t, results[0].Success)
	assert.NotNil(t, results[0].Error)
	// Should fail close to the 250ms timeout, not hang.
	assert.Less(t, elapsed, 2*time.Second, "probe didn't honor outer timeout")
}

// TestStarlarkProbe_StarlarkRuntimeError verifies that a Starlark-level error
// (not just an assertion) marks the probe as failed.
func TestStarlarkProbe_StarlarkRuntimeError(t *testing.T) {
	source := `
def probe(target):
    fail("boom")
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-runtime-err", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	results := p.RunOnce(context.Background())
	assert.Equal(t, 1, len(results))
	assert.False(t, results[0].Success)
	assert.Contains(t, results[0].Error.Error(), "boom")
}

// TestStarlarkProbe_MultipleRuns pins that a single Probe instance can run
// repeatedly (the compiled program is reused; globals are frozen).
func TestStarlarkProbe_MultipleRuns(t *testing.T) {
	srv := loginCartServer(t, http.StatusOK)
	defer srv.Close()

	opts := newOpts(t, hostFromServer(t, srv), checkoutScript)
	p := &Probe{}
	if err := p.Init("script-multi", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	for i := 0; i < 3; i++ {
		results := p.RunOnce(context.Background())
		assert.Equal(t, 1, len(results), "run %d", i)
		assert.True(t, results[0].Success, "run %d: %v", i, results[0].Error)
	}
}

// ---------------------------------------------------------------------------
// http builtin behavior

// TestHTTP_HeadersAndBodyReachServer verifies that headers and a string body
// passed to http.post actually arrive at the target unchanged.
// TestStarlarkProbe_TargetFields verifies the full target struct (name, port,
// ip, labels) is exposed to scripts. Uses StaticEndpoints to set fields
// directly since StaticTargets only handles host:port strings.
func TestStarlarkProbe_TargetFields(t *testing.T) {
	source := `
def probe(target):
    if target.name != "host-a":
        fail("name=%s" % target.name)
    if target.port != 8443:
        fail("port=%d" % target.port)
    if target.ip != "10.1.2.3":
        fail("ip=%s" % target.ip)
    if target.labels["env"] != "prod":
        fail("labels[env]=%s" % target.labels["env"])
    if target.labels["region"] != "us-east1":
        fail("labels[region]=%s" % target.labels["region"])
`
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticEndpoints([]endpoint.Endpoint{{
		Name:   "host-a",
		Port:   8443,
		IP:     net.ParseIP("10.1.2.3"),
		Labels: map[string]string{"env": "prod", "region": "us-east1"},
	}})
	opts.Timeout = time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{Source: proto.String(source)}

	p := &Probe{}
	if err := p.Init("script-target-fields", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.Equal(t, 1, len(results))
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
}

// TestStarlarkProbe_TargetEmptyOptionalFields verifies the empty-value
// conventions: port=0, ip="", labels={} when the endpoint has none.
func TestStarlarkProbe_TargetEmptyOptionalFields(t *testing.T) {
	source := `
def probe(target):
    if target.name != "host-b":
        fail("name=%s" % target.name)
    if target.port != 0:
        fail("port=%d" % target.port)
    if target.ip != "":
        fail("ip=%s" % target.ip)
    if len(target.labels) != 0:
        fail("labels=%s" % target.labels)
`
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticEndpoints([]endpoint.Endpoint{{Name: "host-b"}})
	opts.Timeout = time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{Source: proto.String(source)}

	p := &Probe{}
	if err := p.Init("script-target-empty", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
}

// TestStarlarkProbe_TargetLabelsFrozen verifies labels can't be mutated by
// the script. Mutating a frozen value raises a Starlark error which becomes
// a probe failure.
func TestStarlarkProbe_TargetLabelsFrozen(t *testing.T) {
	source := `
def probe(target):
    target.labels["leak"] = "x"
`
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticEndpoints([]endpoint.Endpoint{{
		Name:   "host-c",
		Labels: map[string]string{"a": "1"},
	}})
	opts.Timeout = time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{Source: proto.String(source)}

	p := &Probe{}
	if err := p.Init("script-target-frozen", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.False(t, results[0].Success)
	assert.Contains(t, results[0].Error.Error(), "frozen")
}

func TestHTTP_HeadersAndBodyReachServer(t *testing.T) {
	var (
		gotMethod string
		gotAuth   string
		gotCT     string
		gotBody   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	source := `
def probe(target):
    r = http.post(
        url = "http://%s:%d/" % (target.name, target.port),
        body = "raw-body-bytes",
        headers = {"Authorization": "Bearer xyz", "Content-Type": "text/plain"},
    )
    assert.http_status(r, 200)
`
	opts := newOpts(t, hostFromServer(t, srv), source)
	p := &Probe{}
	if err := p.Init("script-headers", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
	assert.Equal(t, "POST", gotMethod)
	assert.Equal(t, "Bearer xyz", gotAuth)
	assert.Equal(t, "text/plain", gotCT)
	assert.Equal(t, "raw-body-bytes", gotBody)
}

// TestHTTP_Verbs covers the put/patch/delete verbs: the method reaches the
// server, and body-carrying verbs (put/patch) deliver their body.
func TestHTTP_Verbs(t *testing.T) {
	for _, tc := range []struct {
		verb     string
		wantBody string
	}{
		{"put", "raw-put-body"},
		{"patch", "raw-patch-body"},
		{"delete", ""},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			var gotMethod, gotBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				b, _ := io.ReadAll(r.Body)
				gotBody = string(b)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			bodyArg := ""
			if tc.wantBody != "" {
				bodyArg = fmt.Sprintf(`body = %q,`, tc.wantBody)
			}
			source := fmt.Sprintf(`
def probe(target):
    r = http.%s(
        url = "http://%%s:%%d/" %% (target.name, target.port),
        %s
    )
    assert.http_status(r, 200)
`, tc.verb, bodyArg)
			opts := newOpts(t, hostFromServer(t, srv), source)
			p := &Probe{}
			if err := p.Init("script-verb-"+tc.verb, opts); err != nil {
				t.Fatalf("Init: %v", err)
			}

			results := p.RunOnce(context.Background())
			assert.True(t, results[0].Success, "err=%v", results[0].Error)
			assert.Equal(t, strings.ToUpper(tc.verb), gotMethod)
			assert.Equal(t, tc.wantBody, gotBody)
		})
	}
}

// TestHTTP_ResponseHeadersVisible covers reading response.headers from script.
func TestHTTP_ResponseHeadersVisible(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "hello-world")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	source := `
def probe(target):
    r = http.get(url = "http://%s:%d/" % (target.name, target.port))
    assert.http_status(r, 200)
    if r.headers["X-Custom"] != "hello-world":
        fail("missing header, got: %s" % r.headers["X-Custom"])
`
	opts := newOpts(t, hostFromServer(t, srv), source)
	p := &Probe{}
	if err := p.Init("script-resp-headers", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
}

// TestHTTP_ResponseJSONOnInvalidBody errors cleanly when the body isn't JSON.
func TestHTTP_ResponseJSONOnInvalidBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	source := `
def probe(target):
    r = http.get(url = "http://%s:%d/" % (target.name, target.port))
    r.json()
`
	opts := newOpts(t, hostFromServer(t, srv), source)
	p := &Probe{}
	if err := p.Init("script-bad-json", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	results := p.RunOnce(context.Background())
	assert.False(t, results[0].Success)
	assert.Contains(t, results[0].Error.Error(), "Response.json")
}

// TestHTTP_RuntimeOwnsClient verifies the review comment fix: each runtime
// has its own *http.Client rather than reusing http.DefaultClient.
func TestHTTP_RuntimeOwnsClient(t *testing.T) {
	rt1, err := newRuntime(context.Background(), "rt1", "def probe(t): pass\n", "probe", nil, nil, nil, nil, &logger.Logger{})
	if err != nil {
		t.Fatalf("rt1: %v", err)
	}
	rt2, err := newRuntime(context.Background(), "rt2", "def probe(t): pass\n", "probe", nil, nil, nil, nil, &logger.Logger{})
	if err != nil {
		t.Fatalf("rt2: %v", err)
	}
	assert.NotSame(t, http.DefaultClient, rt1.httpClient, "should not reuse http.DefaultClient")
	assert.NotSame(t, rt1.httpClient, rt2.httpClient, "each runtime should own its own client")
}

// TestHTTP_DistinctClientsBetweenProbes is the integration-level version of
// the previous test: two Probe instances should not see each other's
// requests, even though the script is identical. This guards against a
// future refactor that accidentally shares a client.
func TestHTTP_DistinctClientsBetweenProbes(t *testing.T) {
	var aHits, bHits atomic.Int64
	srvA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		aHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srvA.Close()
	srvB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srvB.Close()

	source := `
def probe(target):
    r = http.get(url = "http://%s:%d/" % (target.name, target.port))
    assert.http_status(r, 200)
`
	mkProbe := func(name, host string) *Probe {
		opts := newOpts(t, host, source)
		p := &Probe{}
		if err := p.Init(name, opts); err != nil {
			t.Fatalf("%s init: %v", name, err)
		}
		return p
	}
	pa := mkProbe("a", hostFromServer(t, srvA))
	pb := mkProbe("b", hostFromServer(t, srvB))

	pa.RunOnce(context.Background())
	pb.RunOnce(context.Background())

	assert.Equal(t, int64(1), aHits.Load(), "probe A should have hit srvA exactly once")
	assert.Equal(t, int64(1), bHits.Load(), "probe B should have hit srvB exactly once")
}

// ---------------------------------------------------------------------------
// assert builtin behavior

func TestAssert_HTTPStatusWrongType(t *testing.T) {
	source := `
def probe(target):
    assert.http_status("not a response", 200)
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-assert-wrong-type", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	results := p.RunOnce(context.Background())
	assert.False(t, results[0].Success)
	assert.Contains(t, results[0].Error.Error(), "must be a Response")
}

// TestHTTP_MaxRedirects exercises the per-call max_redirects kwarg on
// http.get. The server redirects /->/1->/2 then 200; the table walks the
// boundary cases including =0 (which validates the clone codepath actually
// blocks all redirects).
func TestHTTP_MaxRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/1", http.StatusTemporaryRedirect)
		case "/1":
			http.Redirect(w, r, "/2", http.StatusTemporaryRedirect)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	cases := []struct {
		name       string
		max        int
		wantStatus int
	}{
		{"zero_disables", 0, http.StatusTemporaryRedirect},
		{"one_follows_first_hop", 1, http.StatusTemporaryRedirect},
		{"two_follows_both_hops", 2, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf(`
def probe(target):
    r = http.get(url = "%s", max_redirects = %d)
    assert.http_status(r, %d)
`, srv.URL, tc.max, tc.wantStatus)
			opts := newOpts(t, hostFromServer(t, srv), source)
			p := &Probe{}
			if err := p.Init("script-max-redirects-"+tc.name, opts); err != nil {
				t.Fatalf("Init: %v", err)
			}
			results := p.RunOnce(context.Background())
			assert.True(t, results[0].Success, "probe should succeed; got error: %v", results[0].Error)
		})
	}
}

func TestHTTP_KeepAlive(t *testing.T) {
	cases := []struct {
		name          string
		callKwarg     string
		wantConnClose bool
		wantSameAddr  bool
	}{
		{"omitted_closes_connection", "", true, false},
		{"false_closes_connection", ", keep_alive = False", true, false},
		{"true_pools_connection", ", keep_alive = True", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				gotConnClose atomic.Bool
				mu           sync.Mutex
				addrs        []string
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Connection") == "close" {
					gotConnClose.Store(true)
				}
				mu.Lock()
				addrs = append(addrs, r.RemoteAddr)
				mu.Unlock()
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			source := fmt.Sprintf(`
def probe(target):
    r1 = http.get(url = "%s"%s)
    assert.http_status(r1, 200)
    r2 = http.get(url = "%s"%s)
    assert.http_status(r2, 200)
`, srv.URL, tc.callKwarg, srv.URL, tc.callKwarg)

			opts := newOpts(t, hostFromServer(t, srv), source)
			p := &Probe{}
			if err := p.Init("script-keep-alive-"+tc.name, opts); err != nil {
				t.Fatalf("Init: %v", err)
			}
			results := p.RunOnce(context.Background())
			assert.True(t, results[0].Success, "probe should succeed; got error: %v", results[0].Error)
			assert.Equal(t, tc.wantConnClose, gotConnClose.Load(), "Connection: close header")
			require.Len(t, addrs, 2, "expected exactly two requests to reach the server")
			if tc.wantSameAddr {
				assert.Equal(t, addrs[0], addrs[1], "expected pooled connection (same RemoteAddr)")
			} else {
				assert.NotEqual(t, addrs[0], addrs[1], "expected fresh connection (different RemoteAddr)")
			}
		})
	}
}

func TestHTTP_TLSConfig(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	source := fmt.Sprintf(`
def probe(target):
    r = http.get(url = "%s")
    assert.http_status(r, 200)
`, srv.URL)

	t.Run("rejects_self_signed_by_default", func(t *testing.T) {
		opts := newOpts(t, hostFromServer(t, srv), source)
		p := &Probe{}
		if err := p.Init("script-tls-default", opts); err != nil {
			t.Fatalf("Init: %v", err)
		}
		results := p.RunOnce(context.Background())
		assert.False(t, results[0].Success, "default TLS config should reject self-signed cert")
		require.NotNil(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "certificate")
	})

	t.Run("accepts_with_disable_cert_validation", func(t *testing.T) {
		opts := newOpts(t, hostFromServer(t, srv), source)
		opts.ProbeConf.(*configpb.ProbeConf).TlsConfig = &tlsconfigpb.TLSConfig{
			DisableCertValidation: proto.Bool(true),
		}
		p := &Probe{}
		if err := p.Init("script-tls-skip-verify", opts); err != nil {
			t.Fatalf("Init: %v", err)
		}
		results := p.RunOnce(context.Background())
		assert.True(t, results[0].Success, "probe should succeed; got error: %v", results[0].Error)
	})
}

// TestHTTP_TLSConfigs covers the tls kwarg: named tls_configs entries are
// selected per call, and omitting the kwarg always means tls_config -- never
// "the only tls_configs entry".
func TestHTTP_TLSConfigs(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	insecure := func() map[string]*tlsconfigpb.TLSConfig {
		return map[string]*tlsconfigpb.TLSConfig{
			"insecure": {DisableCertValidation: proto.Bool(true)},
		}
	}

	t.Run("named_config_selected_by_tls_kwarg", func(t *testing.T) {
		source := fmt.Sprintf(`
def probe(target):
    r = http.get(url = "%s", tls = "insecure")
    assert.http_status(r, 200)
`, srv.URL)
		opts := newOpts(t, hostFromServer(t, srv), source)
		opts.ProbeConf.(*configpb.ProbeConf).TlsConfigs = insecure()
		p := &Probe{}
		if err := p.Init("script-tls-named", opts); err != nil {
			t.Fatalf("Init: %v", err)
		}
		results := p.RunOnce(context.Background())
		assert.True(t, results[0].Success, "probe should succeed; got error: %v", results[0].Error)
	})

	// The rule that separates tls= from oauth.token(name=""): a lone
	// tls_configs entry is not implicitly applied to calls that don't ask for
	// it, so adding an alternate config can't silently retarget the rest of
	// the script.
	t.Run("omitted_tls_uses_default_not_sole_entry", func(t *testing.T) {
		source := fmt.Sprintf(`
def probe(target):
    r = http.get(url = "%s")
    assert.http_status(r, 200)
`, srv.URL)
		opts := newOpts(t, hostFromServer(t, srv), source)
		opts.ProbeConf.(*configpb.ProbeConf).TlsConfigs = insecure()
		p := &Probe{}
		if err := p.Init("script-tls-omitted", opts); err != nil {
			t.Fatalf("Init: %v", err)
		}
		results := p.RunOnce(context.Background())
		assert.False(t, results[0].Success, "call without tls= should use tls_config, not the sole tls_configs entry")
		require.NotNil(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "certificate")
	})

	// The motivating case: a script hitting two hosts where TLS settings can't
	// be probe-wide. The default (tls_config) is strict and rejects the
	// self-signed cert; the one host that needs an exception names it.
	t.Run("default_and_named_in_one_script", func(t *testing.T) {
		strict := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer strict.Close()

		source := fmt.Sprintf(`
def probe(target):
    r = http.get(url = "%s")
    assert.http_status(r, 200)
    r = http.get(url = "%s", tls = "insecure")
    assert.http_status(r, 200)
`, strict.URL, srv.URL)
		opts := newOpts(t, hostFromServer(t, strict), source)
		opts.ProbeConf.(*configpb.ProbeConf).TlsConfigs = insecure()
		p := &Probe{}
		if err := p.Init("script-tls-mixed", opts); err != nil {
			t.Fatalf("Init: %v", err)
		}
		results := p.RunOnce(context.Background())
		assert.True(t, results[0].Success, "probe should succeed; got error: %v", results[0].Error)
	})

	t.Run("unknown_name_errors", func(t *testing.T) {
		source := fmt.Sprintf(`
def probe(target):
    http.get(url = "%s", tls = "nope")
`, srv.URL)
		opts := newOpts(t, hostFromServer(t, srv), source)
		opts.ProbeConf.(*configpb.ProbeConf).TlsConfigs = insecure()
		p := &Probe{}
		if err := p.Init("script-tls-unknown", opts); err != nil {
			t.Fatalf("Init: %v", err)
		}
		results := p.RunOnce(context.Background())
		assert.False(t, results[0].Success)
		require.NotNil(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), `no tls config named "nope" (configured: insecure)`)
	})

	t.Run("no_tls_configs_errors", func(t *testing.T) {
		source := fmt.Sprintf(`
def probe(target):
    http.get(url = "%s", tls = "insecure")
`, srv.URL)
		opts := newOpts(t, hostFromServer(t, srv), source)
		p := &Probe{}
		if err := p.Init("script-tls-none", opts); err != nil {
			t.Fatalf("Init: %v", err)
		}
		results := p.RunOnce(context.Background())
		assert.False(t, results[0].Success)
		require.NotNil(t, results[0].Error)
		assert.Contains(t, results[0].Error.Error(), "probe has no tls_configs configured")
	})
}

func TestLogSetAttr_FlowsToProbeFailureLog(t *testing.T) {
	// Closed server gives us a deterministic, immediate connection refused
	// for the failure-path subcase.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.Close()

	cases := []struct {
		name           string
		source         string
		wantSuccess    bool
		wantNotInLog   string // optional: must not appear in output
	}{
		{
			name: "script_side_log_info_carries_attr",
			source: `
def probe(target):
    log.set_attr("req_id", "abc123")
    log.info("processing")
`,
			wantSuccess: true,
		},
		{
			name: "probe_failure_log_carries_attr",
			source: fmt.Sprintf(`
def probe(target):
    log.set_attr("req_id", "abc123")
    http.get(url = "%s")
`, srv.URL),
			wantSuccess: false,
		},
		{
			name: "repeat_key_replaces",
			source: `
def probe(target):
    log.set_attr("req_id", "first")
    log.set_attr("req_id", "abc123")
    log.info("processing")
`,
			wantSuccess:  true,
			wantNotInLog: "req_id=first",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := newOpts(t, "127.0.0.1", tc.source)
			opts.Logger = logger.New(logger.WithWriter(&buf))

			p := &Probe{}
			require.NoError(t, p.Init("script-log-setattr-"+tc.name, opts))
			results := p.RunOnce(context.Background())
			assert.Equal(t, tc.wantSuccess, results[0].Success)

			output := buf.String()
			assert.Contains(t, output, "req_id=abc123", "expected attr in log output, got: %s", output)
			if tc.wantNotInLog != "" {
				assert.NotContains(t, output, tc.wantNotInLog, "stale attr value still present")
			}
		})
	}
}

// TestLogSetAttr_PreservesInsertionOrder pins that two log.set_attr calls
// with different keys produce attrs in the order they were set (not
// alphabetical, not random map-iteration order).
func TestLogSetAttr_PreservesInsertionOrder(t *testing.T) {
	source := `
def probe(target):
    log.set_attr("zeta", "1")
    log.set_attr("alpha", "2")
    log.set_attr("middle", "3")
    log.info("ordered")
`
	var buf bytes.Buffer
	opts := newOpts(t, "127.0.0.1", source)
	opts.Logger = logger.New(logger.WithWriter(&buf))

	p := &Probe{}
	require.NoError(t, p.Init("script-log-setattr-order", opts))
	results := p.RunOnce(context.Background())
	require.True(t, results[0].Success, "probe should succeed; got error: %v", results[0].Error)

	output := buf.String()
	iZeta := strings.Index(output, "zeta=1")
	iAlpha := strings.Index(output, "alpha=2")
	iMiddle := strings.Index(output, "middle=3")
	require.NotEqual(t, -1, iZeta, "zeta not in log output: %s", output)
	require.NotEqual(t, -1, iAlpha, "alpha not in log output: %s", output)
	require.NotEqual(t, -1, iMiddle, "middle not in log output: %s", output)
	assert.Less(t, iZeta, iAlpha, "zeta should appear before alpha (set first)")
	assert.Less(t, iAlpha, iMiddle, "alpha should appear before middle (set second)")
}

// ---------------------------------------------------------------------------
// Init / config validation

func TestInit_InvalidSource(t *testing.T) {
	opts := newOpts(t, "example.com", "this is not valid starlark @#$")
	p := &Probe{}
	err := p.Init("script-bad-source", opts)
	assert.Error(t, err)
}

func TestInit_TopLevelHTTPHasRuntimeClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	source := fmt.Sprintf(`
r = http.get(url = "%s")
assert.http_status(r, 200)

def probe(target):
    pass
`, srv.URL)
	opts := newOpts(t, hostFromServer(t, srv), source)

	p := &Probe{}
	assert.NoError(t, p.Init("script-load-http", opts))
}

func TestInit_MissingEntryPoint(t *testing.T) {
	opts := newOpts(t, "example.com", "x = 1\n")
	p := &Probe{}
	err := p.Init("script-no-entry", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInit_EntryPointNotFunction(t *testing.T) {
	opts := newOpts(t, "example.com", "probe = 42\n")
	p := &Probe{}
	err := p.Init("script-not-fn", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a function")
}

func TestInit_EntryPointBadArity(t *testing.T) {
	cases := []struct{ name, source string }{
		{"zero args", "def probe(): pass\n"},
		{"two args", "def probe(a, b): pass\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := newOpts(t, "example.com", tc.source)
			p := &Probe{}
			err := p.Init("script-arity", opts)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "exactly one argument")
		})
	}
}

func TestInit_SourceFileLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.star")
	if err := os.WriteFile(path, []byte("def probe(target):\n    pass\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("example.com")
	opts.Timeout = time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{SourceFile: proto.String(path)}

	p := &Probe{}
	if err := p.Init("script-from-file", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success)
}

func TestInit_SourceFileMissing(t *testing.T) {
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("example.com")
	opts.Timeout = time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{SourceFile: proto.String("/nonexistent/path.star")}

	p := &Probe{}
	err := p.Init("script-missing-file", opts)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "source_file"))
}

func TestInit_SourceMutuallyExclusive(t *testing.T) {
	cases := []struct {
		name string
		c    *configpb.ProbeConf
	}{
		{"both set", &configpb.ProbeConf{Source: proto.String("def probe(t): pass"), SourceFile: proto.String("/x")}},
		{"neither set", &configpb.ProbeConf{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := options.DefaultOptions()
			opts.Targets = targets.StaticTargets("example.com")
			opts.Timeout = time.Second
			opts.Logger = &logger.Logger{}
			opts.LatencyUnit = time.Millisecond
			opts.ProbeConf = tc.c
			p := &Probe{}
			err := p.Init("script-source-xor", opts)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "exactly one")
		})
	}
}

func TestInit_WrongConfigType(t *testing.T) {
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("example.com")
	opts.Timeout = time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = struct{}{} // not a *script.ProbeConf

	p := &Probe{}
	err := p.Init("script-bad-conf", opts)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// vars builtin

func TestVars_GetReturnsConfiguredValues(t *testing.T) {
	source := `
def probe(target):
    if vars.get("USER") != "alice":
        fail("USER=%s" % vars.get("USER"))
    if vars.get("PASS") != "s3cret":
        fail("PASS=%s" % vars.get("PASS"))
`
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("example.com")
	opts.Timeout = time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{
		Source: proto.String(source),
		Vars:   map[string]string{"USER": "alice", "PASS": "s3cret"},
	}

	p := &Probe{}
	if err := p.Init("script-vars-get", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
}

func TestVars_GetMissingKey(t *testing.T) {
	source := `
def probe(target):
    if vars.get("MISSING") != None:
        fail("expected None for missing key, got %r" % vars.get("MISSING"))
    if vars.get("MISSING", "fallback") != "fallback":
        fail("expected fallback, got %r" % vars.get("MISSING", "fallback"))
`
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("example.com")
	opts.Timeout = time.Second
	opts.Logger = &logger.Logger{}
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{Source: proto.String(source)}

	p := &Probe{}
	if err := p.Init("script-vars-missing", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
}

func TestVars_FlowsIntoHTTPRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer s3cret" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	source := `
def probe(target):
    r = http.get(
        url = "http://%s:%d/" % (target.name, target.port),
        headers = {"Authorization": "Bearer " + vars.get("TOKEN")},
    )
    assert.http_status(r, 200)
`
	opts := newOpts(t, hostFromServer(t, srv), source)
	opts.ProbeConf.(*configpb.ProbeConf).Vars = map[string]string{"TOKEN": "s3cret"}

	p := &Probe{}
	if err := p.Init("script-vars-http", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
}

// ---------------------------------------------------------------------------
// log builtin

// newLoggingOpts builds an options that captures log output to buf, so tests
// can assert that log.* calls actually reach the cloudprober logger.
func newLoggingOpts(t *testing.T, source string, buf *bytes.Buffer) *options.Options {
	t.Helper()
	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("example.com")
	opts.Timeout = time.Second
	opts.Logger = logger.New(logger.WithWriter(buf))
	opts.LatencyUnit = time.Millisecond
	opts.ProbeConf = &configpb.ProbeConf{Source: proto.String(source)}
	return opts
}

func TestLog_RoutesThroughLogger(t *testing.T) {
	source := `
def probe(target):
    log.info("info-line")
    log.warn(msg="warn-line")  # Test msg kwarg
    log.error("error-line")
`
	var buf bytes.Buffer
	opts := newLoggingOpts(t, source, &buf)
	p := &Probe{}
	if err := p.Init("script-log", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)

	out := buf.String()
	assert.Contains(t, out, "info-line")
	assert.Contains(t, out, "warn-line")
	assert.Contains(t, out, "error-line")
}

// TestLog_CarriesTargetAttribute pins that log.* lines emitted from a script
// inherit the per-target attribute attached by runProbe. If we ever regress
// to passing the probe-level logger into runtime.Run instead of the
// per-target one, this test catches it.
func TestLog_CarriesTargetAttribute(t *testing.T) {
	source := `
def probe(target):
    log.info("hello-from-script")
`
	var buf bytes.Buffer
	opts := newLoggingOpts(t, source, &buf)
	opts.Targets = targets.StaticTargets("specific-host.example.com")
	p := &Probe{}
	if err := p.Init("script-log-target", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
	assert.Contains(t, buf.String(), "specific-host.example.com")
}

// ---------------------------------------------------------------------------
// state builtin

// runProbeWith drives p.runProbe with a stable runReq, mirroring the
// scheduler's per-target goroutine. Tests that need cross-run persistence
// can't use Probe.RunOnce — it builds fresh runReqs (and thus fresh state
// buckets) per call.
func runProbeWith(t *testing.T, p *Probe, runReq *sched.RunProbeForTargetRequest) {
	t.Helper()
	if runReq.LastRun == nil {
		runReq.LastRun = &sched.LastRunResult{}
	}
	p.runProbe(context.Background(), runReq)
}

func TestState_PersistsAcrossRuns(t *testing.T) {
	source := `
def probe(target):
    n = state.get("count", 0)
    state.set("count", n + 1)
    if n + 1 == 3:
        log.info("hit-three")
`
	var buf bytes.Buffer
	opts := newLoggingOpts(t, source, &buf)
	p := &Probe{}
	if err := p.Init("script-state-persist", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	runReq := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: "example.com"}}
	for i := 0; i < 3; i++ {
		runProbeWith(t, p, runReq)
		assert.NoError(t, runReq.LastRun.Error, "run %d", i)
	}
	assert.Contains(t, buf.String(), "hit-three", "third run should see count==3")
}

func TestState_DefaultForMissing(t *testing.T) {
	source := `
def probe(target):
    if state.get("missing") != None:
        fail("expected None for missing key")
    if state.get("missing", "fallback") != "fallback":
        fail("expected fallback")
    if state.get("missing", 42) != 42:
        fail("expected 42")
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-state-default", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
}

// TestState_IsolatedPerTarget interleaves two targets — the second A run
// reads back A's owner, which would fail if B's write had clobbered it.
func TestState_IsolatedPerTarget(t *testing.T) {
	source := `
def probe(target):
    seen = state.get("owner")
    if seen != None and seen != target.name:
        fail("bucket leak: target %s saw owner=%s" % (target.name, seen))
    state.set("owner", target.name)
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-state-isolation", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	reqA := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: "host-a"}}
	reqB := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: "host-b"}}
	runProbeWith(t, p, reqA)
	runProbeWith(t, p, reqB)
	runProbeWith(t, p, reqA)
	assert.NoError(t, reqA.LastRun.Error)
	assert.NoError(t, reqB.LastRun.Error)
}

func TestState_DeepCopyOnGet(t *testing.T) {
	source := `
def probe(target):
    if state.get("items") == None:
        state.set("items", [1, 2, 3])
        return
    items = state.get("items")
    items.append(99)
    again = state.get("items")
    if len(again) != 3:
        fail("expected stored list unchanged, got %s" % again)
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-state-deepcopy", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	runReq := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: "example.com"}}
	for i := 0; i < 2; i++ {
		runProbeWith(t, p, runReq)
		assert.NoError(t, runReq.LastRun.Error, "run %d", i)
	}
}

// TestState_FreshBucketAcrossRunOnce pins that persistence is a
// scheduler-loop property, not a probe-instance one — RunOnce builds a fresh
// runReq each call, so state from a prior RunOnce is gone.
func TestState_FreshBucketAcrossRunOnce(t *testing.T) {
	source := `
def probe(target):
    n = state.get("count", 0)
    if n != 0:
        fail("expected fresh bucket on RunOnce, got count=%d" % n)
    state.set("count", n + 1)
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-state-runonce-fresh", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	for i := 0; i < 3; i++ {
		results := p.RunOnce(context.Background())
		assert.True(t, results[0].Success, "run %d: %v", i, results[0].Error)
	}
}

func TestState_RejectsUnsupportedValue(t *testing.T) {
	source := `
def helper():
    return 1

def probe(target):
    state.set("fn", helper)
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-state-bad-value", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	results := p.RunOnce(context.Background())
	assert.False(t, results[0].Success)
	assert.Contains(t, results[0].Error.Error(), "state.set")
}

// TestState_TupleRoundTrip pins that a Tuple stored via state.set comes back
// as a Tuple, not a List — preserving immutability and hashability. Covers
// top-level tuples and tuples nested in lists.
func TestState_TupleRoundTrip(t *testing.T) {
	source := `
def probe(target):
    if state.get("pair") == None:
        state.set("pair", ("a", 2))
        state.set("nested", [("x", 1), ("y", 2)])
        return
    pair = state.get("pair")
    if type(pair) != "tuple":
        fail("expected tuple, got %s" % type(pair))
    if pair != ("a", 2):
        fail("expected (\"a\", 2), got %s" % str(pair))
    # Hashability check — a list would fail here.
    {pair: "ok"}
    nested = state.get("nested")
    if type(nested[0]) != "tuple":
        fail("expected nested[0] to be tuple, got %s" % type(nested[0]))
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-state-tuple-rt", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	runReq := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: "example.com"}}
	for i := 0; i < 2; i++ {
		runProbeWith(t, p, runReq)
		assert.NoError(t, runReq.LastRun.Error, "run %d", i)
	}
}

// TestState_MaxKeysCap exercises the 1024-key bucket cap. Keys above the
// limit fail with a clear error; replacing an existing key still works.
//
// The 1024 in the scripts mirrors stateMaxKeys in builtins.go. They can't
// share the constant directly — one's Go, the other's a Starlark literal.
func TestState_MaxKeysCap(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		wantErr  string // empty == expect success
	}{
		{
			name: "overflow",
			source: `
def probe(target):
    for i in range(1024):
        state.set("k%d" % i, i)
    state.set("overflow", 1)  # 1025th distinct key
`,
			wantErr: "max keys",
		},
		{
			name: "replacement",
			source: `
def probe(target):
    for i in range(1024):
        state.set("k%d" % i, i)
    state.set("k0", 999)  # replacing existing key — must not trip cap
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := newOpts(t, "example.com", tc.source)
			p := &Probe{}
			if err := p.Init("script-state-cap-"+tc.name, opts); err != nil {
				t.Fatalf("Init: %v", err)
			}
			results := p.RunOnce(context.Background())
			if tc.wantErr != "" {
				assert.False(t, results[0].Success)
				assert.Contains(t, results[0].Error.Error(), tc.wantErr)
			} else {
				assert.True(t, results[0].Success, "err=%v", results[0].Error)
			}
		})
	}
}

// TestState_AtModuleLevel pins that top-level state.{get,set} doesn't panic
// — the ExecFile thread carries a scratch bucket that's discarded after load.
func TestState_AtModuleLevel(t *testing.T) {
	source := `
state.set("loaded", True)
_loaded = state.get("loaded", False)

def probe(target):
    pass
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	err := p.Init("script-state-toplevel", opts)
	assert.NoError(t, err)
}

// TestLog_AtModuleLevel checks that log.info called from top-level (i.e.
// during newRuntime, before any probe() call) doesn't panic. The thread
// constructed for ExecFile carries the probe-level logger as a fallback.
func TestLog_AtModuleLevel(t *testing.T) {
	source := `
log.info("module-load-line")

def probe(target):
    pass
`
	var buf bytes.Buffer
	opts := newLoggingOpts(t, source, &buf)
	p := &Probe{}
	if err := p.Init("script-log-toplevel", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	assert.Contains(t, buf.String(), "module-load-line")
	results := p.RunOnce(context.Background())
	assert.True(t, results[0].Success, "err=%v", results[0].Error)
}
