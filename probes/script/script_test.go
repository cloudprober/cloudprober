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

package script

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/script/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

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

// loginCartServer returns a test server that:
//   - POST /login with JSON {user,pass} → {"token": "tok"} and the chosen status
//   - GET /cart with Authorization: Bearer tok → 200, else 401
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

func TestScriptProbe_Success(t *testing.T) {
	srv := loginCartServer(t, http.StatusOK)
	defer srv.Close()

	host := hostFromServer(t, srv)
	opts := newOpts(t, host, checkoutScript)

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

func TestScriptProbe_LoginFails(t *testing.T) {
	srv := loginCartServer(t, http.StatusServiceUnavailable)
	defer srv.Close()

	host := hostFromServer(t, srv)
	opts := newOpts(t, host, checkoutScript)

	p := &Probe{}
	if err := p.Init("script-fail", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	results := p.RunOnce(context.Background())
	assert.Equal(t, 1, len(results))
	r := results[0]
	assert.False(t, r.Success, "expected failure")
	assert.NotNil(t, r.Error)
}

func TestScriptProbe_InvalidSource(t *testing.T) {
	opts := newOpts(t, "example.com", "this is not valid starlark @#$")
	p := &Probe{}
	err := p.Init("script-bad", opts)
	assert.Error(t, err, "expected Init to fail on invalid source")
}

func TestScriptProbe_MissingEntryPoint(t *testing.T) {
	opts := newOpts(t, "example.com", "x = 1\n")
	p := &Probe{}
	err := p.Init("script-no-entry", opts)
	assert.Error(t, err, "expected Init to fail when entry point missing")
}
