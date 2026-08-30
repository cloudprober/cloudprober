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
	"fmt"
	neturl "net/url"
	"testing"

	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func mustParseURL(t *testing.T, s string) *neturl.URL {
	t.Helper()
	u, err := neturl.Parse(s)
	if err != nil {
		t.Fatalf("error parsing URL %q: %v", s, err)
	}
	return u
}

func TestRawQuery(t *testing.T) {
	for _, test := range []struct{ in, want string }{
		{"", ""},
		{"/path", ""},
		{"/login?password=secret", "password=secret"},
		{"/a?user=admin&password=secret", "user=admin&password=secret"},
		{"/a?x=1?y=2", "x=1?y=2"},
	} {
		assert.Equal(t, test.want, rawQuery(test.in), "rawQuery(%q)", test.in)
	}
}

func TestRedactedURL(t *testing.T) {
	tests := []struct {
		name   string
		redact bool
		url    string
		want   string
	}{
		{
			name:   "redaction off, no query",
			redact: false,
			url:    "http://example.com/path",
			want:   "http://example.com/path",
		},
		{
			name:   "redaction off, with query -- query kept",
			redact: false,
			url:    "http://example.com/login?password=secret",
			want:   "http://example.com/login?password=secret",
		},
		{
			name:   "redaction on, no query -- unchanged",
			redact: true,
			url:    "http://example.com/path",
			want:   "http://example.com/path",
		},
		{
			name:   "redaction on, with query -- query hidden",
			redact: true,
			url:    "http://example.com/login?password=secret",
			want:   "http://example.com/login?<redacted>",
		},
		{
			name:   "redaction on, multiple params -- all hidden",
			redact: true,
			url:    "https://example.com/a?user=admin&password=secret&x=1",
			want:   "https://example.com/a?<redacted>",
		},
		{
			name:   "redaction on, fragment kept",
			redact: true,
			url:    "https://example.com/a?token=secret#frag",
			want:   "https://example.com/a?<redacted>#frag",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Probe{redactURLQueryInLogs: test.redact}
			got := p.redactedURL(mustParseURL(t, test.url))
			assert.Equal(t, test.want, got)
		})
	}
}

func TestRedactedErr(t *testing.T) {
	rawURL := "http://example.com/login?password=secret"
	u := mustParseURL(t, rawURL)

	t.Run("nil error", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		assert.Equal(t, "", p.redactedErr(u.RawQuery, nil))
	})

	t.Run("redaction off keeps secret", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: false}
		// net/url.Error embeds the full URL in its message.
		err := &neturl.Error{Op: "Get", URL: rawURL, Err: fmt.Errorf("dial tcp: connection refused")}
		got := p.redactedErr(u.RawQuery, err)
		assert.Contains(t, got, "password=secret")
	})

	t.Run("redaction on hides secret in error text", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		err := &neturl.Error{Op: "Get", URL: rawURL, Err: fmt.Errorf("dial tcp: connection refused")}
		got := p.redactedErr(u.RawQuery, err)
		assert.NotContains(t, got, "password=secret")
		assert.Contains(t, got, "<redacted>")
		// The rest of the error text is preserved.
		assert.Contains(t, got, "connection refused")
	})

	t.Run("redaction on, no query in URL is no-op", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		noQueryURL := mustParseURL(t, "http://example.com/path")
		err := fmt.Errorf("some error")
		assert.Equal(t, "some error", p.redactedErr(noQueryURL.RawQuery, err))
	})

	// net/http builds its error from the request it last made, so after a
	// redirect the scheme, host and path are the redirect target's, while the
	// query is carried over. Matching on the query alone still redacts it.
	t.Run("redirected URL is still redacted", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		err := &neturl.Error{
			Op:  "Get",
			URL: "https://other.example.com/login?password=secret",
			Err: fmt.Errorf("dial tcp: connection refused"),
		}
		got := p.redactedErr(u.RawQuery, err)
		assert.NotContains(t, got, "password=secret")
		assert.Contains(t, got, "<redacted>")
	})

	// url.Error renders the URL with %q, so a quote in the query arrives
	// escaped and doesn't match the query's plain form.
	t.Run("quote in query is still redacted", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		qu := mustParseURL(t, `http://example.com/a?sig=x"y`)
		err := &neturl.Error{Op: "Get", URL: qu.String(), Err: fmt.Errorf("connection refused")}
		got := p.redactedErr(qu.RawQuery, err)
		assert.NotContains(t, got, "sig=x")
		assert.Contains(t, got, "<redacted>")
	})
}

func TestRedactErr(t *testing.T) {
	rawURL := "http://example.com/login?password=secret"
	u := mustParseURL(t, rawURL)
	urlErr := &neturl.Error{Op: "Get", URL: rawURL, Err: fmt.Errorf("connection refused")}

	t.Run("nil error stays nil", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		assert.NoError(t, p.redactErr(u.RawQuery, nil))
	})

	// With redaction off we must hand back the original error, unwrapped, so
	// that the default configuration is bit-for-bit unchanged.
	t.Run("redaction off returns the original error", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: false}
		assert.Same(t, urlErr, p.redactErr(u.RawQuery, urlErr))
	})

	t.Run("redaction on hides secret", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		got := p.redactErr(u.RawQuery, urlErr)
		assert.NotContains(t, got.Error(), "password=secret")
		assert.Contains(t, got.Error(), "<redacted>")
	})
}

// A malformed relative_url is rejected at Init, and that error carries the
// relative URL -- including its query -- back to the caller.
func TestInitInvalidRelativeURLIsRedacted(t *testing.T) {
	for _, test := range []struct{ name, want string }{
		{"redaction off", "invalid relative URL: login?password=secret, must begin with '/'"},
		{"redaction on", "invalid relative URL: login?<redacted>, must begin with '/'"},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := &Probe{}
			err := p.Init("test-probe", &options.Options{
				ProbeConf: &configpb.ProbeConf{
					RelativeUrl:          proto.String("login?password=secret"),
					RedactUrlQueryInLogs: proto.Bool(test.name == "redaction on"),
				},
			})
			assert.EqualError(t, err, test.want)
		})
	}
}
