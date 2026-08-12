// Copyright 2017-2025 The Cloudprober Authors.
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

	"github.com/stretchr/testify/assert"
)

func mustParseURL(t *testing.T, s string) *neturl.URL {
	t.Helper()
	u, err := neturl.Parse(s)
	if err != nil {
		t.Fatalf("error parsing URL %q: %v", s, err)
	}
	return u
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
		assert.Equal(t, "", p.redactedErr(u, nil))
	})

	t.Run("redaction off keeps secret", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: false}
		// net/url.Error embeds the full URL in its message.
		err := &neturl.Error{Op: "Get", URL: rawURL, Err: fmt.Errorf("dial tcp: connection refused")}
		got := p.redactedErr(u, err)
		assert.Contains(t, got, "password=secret")
	})

	t.Run("redaction on hides secret in error text", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		err := &neturl.Error{Op: "Get", URL: rawURL, Err: fmt.Errorf("dial tcp: connection refused")}
		got := p.redactedErr(u, err)
		assert.NotContains(t, got, "password=secret")
		assert.Contains(t, got, "<redacted>")
		// The rest of the error text is preserved.
		assert.Contains(t, got, "connection refused")
	})

	t.Run("redaction on, no query in URL is no-op", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		noQueryURL := mustParseURL(t, "http://example.com/path")
		err := fmt.Errorf("some error")
		assert.Equal(t, "some error", p.redactedErr(noQueryURL, err))
	})
}
