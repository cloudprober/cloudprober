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
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/starlark/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
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
    assert.status(r, 200)
    token = r.json()["token"]

    r = http.get(
        url = base + "/cart",
        headers = {"Authorization": "Bearer " + token},
    )
    assert.status(r, 200)
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
    assert.status(r, 200)
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
    assert.status(r, 200)
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
    assert.status(r, 200)
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

// TestHTTP_RuntimeOwnsClient verifies the review comment fix: each Runtime
// has its own *http.Client rather than reusing http.DefaultClient.
func TestHTTP_RuntimeOwnsClient(t *testing.T) {
	rt1, err := NewRuntime(context.Background(), "rt1", "def probe(t): pass\n", "probe", nil, &logger.Logger{})
	if err != nil {
		t.Fatalf("rt1: %v", err)
	}
	rt2, err := NewRuntime(context.Background(), "rt2", "def probe(t): pass\n", "probe", nil, &logger.Logger{})
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
    assert.status(r, 200)
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

func TestAssert_StatusWrongType(t *testing.T) {
	source := `
def probe(target):
    assert.status("not a response", 200)
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
assert.status(r, 200)

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
    assert.status(r, 200)
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
