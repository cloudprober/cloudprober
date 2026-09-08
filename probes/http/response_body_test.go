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

package http

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

// failingBody fails every read, the way a truncated or reset response does,
// and records whether it was closed.
type failingBody struct {
	mu     sync.Mutex
	closed bool
}

func (b *failingBody) Read([]byte) (int, error) {
	return 0, errors.New("unexpected EOF")
}

func (b *failingBody) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

func (b *failingBody) wasClosed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}

type failingBodyTransport struct{ body *failingBody }

func (t *failingBodyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: t.body, Header: http.Header{}}, nil
}

// An unclosed body holds its connection out of the idle pool, so a server that
// keeps truncating responses would leak one connection per probe.
func TestResponseBodyClosedOnReadError(t *testing.T) {
	p := &Probe{}
	err := p.Init("close_test", &options.Options{
		Targets:   targets.StaticTargets("test.com"),
		Interval:  2 * time.Second,
		Timeout:   time.Second,
		ProbeConf: &configpb.ProbeConf{},
	})
	assert.NoError(t, err)

	body := &failingBody{}
	req, err := http.NewRequest("GET", "http://test.com/", nil)
	assert.NoError(t, err)

	gotErr := p.doHTTPRequest(req, &http.Client{Transport: &failingBodyTransport{body: body}},
		endpoint.Endpoint{Name: "test.com"}, p.newResult(), nil)

	assert.Error(t, gotErr, "a failed body read should return an error")
	assert.True(t, body.wasClosed(), "response body must be closed on the read-error path")
}
