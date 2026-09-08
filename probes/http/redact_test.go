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
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestRedactURL(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"no query", "/path", "/path"},
		{"relative URL", "/login?password=secret", "/login?<redacted>"},
		{"multiple params", "/a?user=admin&password=secret", "/a?<redacted>"},
		{"bare trailing ?", "/a?", "/a?"},
		// A space is legal in a query and survives url.URL.String() verbatim,
		// so anything that stops at whitespace leaks the rest of the query.
		{"space inside the query", "/a?msg=hello world&tok=secret", "/a?<redacted>"},
		{"absolute URL", "http://h/a?tok=secret", "http://h/a?<redacted>"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Probe{redactURLQueryInLogs: true}
			assert.Equal(t, test.want, p.redactURL(test.in))
		})
	}

	t.Run("redaction off is a no-op", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: false}
		assert.Equal(t, "/login?password=secret", p.redactURL("/login?password=secret"))
	})
}

func TestRedactErrMsg(t *testing.T) {
	p := &Probe{redactURLQueryInLogs: true}
	tests := []struct{ name, in, want string }{
		{
			name: "quoted URL",
			in:   `Get "http://h/a?tok=s": refused`,
			want: `Get "http://h/a?<redacted>": refused`,
		},
		{
			name: "query containing a space",
			in:   `Get "http://h/a?msg=hello world&tok=secret": refused`,
			want: `Get "http://h/a?<redacted>": refused`,
		},
		{
			name: "two quoted URLs",
			in:   `"http://a/x?s=1" and "http://b/y?s=2"`,
			want: `"http://a/x?<redacted>" and "http://b/y?<redacted>"`,
		},
		// A '?' outside a quoted token is prose, not a query. Rewriting it
		// would mangle the message for no security benefit.
		{
			name: "unquoted '?' is left alone",
			in:   "error resolving target: foo?bar, no such host",
			want: "error resolving target: foo?bar, no such host",
		},
		{"no query", `Get "http://h/a": refused`, `Get "http://h/a": refused`},
		// Errors quote plenty of things that aren't URLs. A '?' in one of
		// those is not a query, and rewriting it only destroys a diagnostic.
		{
			name: "non-URL quoted token is left alone",
			in:   `x509: certificate is valid for "a?b", not h`,
			want: `x509: certificate is valid for "a?b", not h`,
		},
		{
			name: "quoted relative URL is redacted",
			in:   `parse "/a?tok=secret": bad`,
			want: `parse "/a?<redacted>": bad`,
		},
		{"bare trailing ?", `Get "http://h/a?": refused`, `Get "http://h/a?": refused`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, p.redactErrMsg(test.in))
		})
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
	// net/url.Error embeds the full URL, query included, in its message.
	urlErr := &neturl.Error{Op: "Get", URL: rawURL, Err: fmt.Errorf("dial tcp: connection refused")}

	t.Run("nil error stays nil", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		assert.NoError(t, p.redactedErr(nil))
	})

	// With redaction off we hand back the original error untouched, so that
	// the default configuration keeps propagating the typed error.
	t.Run("redaction off returns the original error", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: false}
		got := p.redactedErr(urlErr)
		assert.Same(t, urlErr, got)
		assert.Contains(t, got.Error(), "password=secret")
	})

	t.Run("redaction on hides secret in error text", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		got := p.redactedErr(urlErr)
		assert.NotContains(t, got.Error(), "password=secret")
		assert.Contains(t, got.Error(), "<redacted>")
		// The rest of the error text is preserved.
		assert.Contains(t, got.Error(), "connection refused")
		assert.Contains(t, got.Error(), "http://example.com/login")
	})

	// net/http builds its error from the request it last made, so after a
	// redirect the URL in the error is the redirect target's -- a different
	// host, path and, crucially, a different query than the one we sent.
	t.Run("redirect to a different query is redacted", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		err := &neturl.Error{
			Op:  "Get",
			URL: "https://other.example.com/cb?code=supersecret",
			Err: fmt.Errorf("dial tcp: connection refused"),
		}
		got := p.redactedErr(err)
		assert.NotContains(t, got.Error(), "code=supersecret")
		assert.Contains(t, got.Error(), "<redacted>")
	})

	// The probe's own URL has no query at all, so there is nothing to match
	// on; the redirect target's query must still be redacted.
	t.Run("redirect adds a query where we had none", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		err := &neturl.Error{
			Op:  "Get",
			URL: "https://other.example.com/cb?code=supersecret",
			Err: fmt.Errorf("connection refused"),
		}
		got := p.redactedErr(err)
		assert.NotContains(t, got.Error(), "supersecret")
	})

	// A redirect that carries our query over and appends to it must not leak
	// the appended part.
	t.Run("redirect extends our query", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		err := &neturl.Error{
			Op:  "Get",
			URL: "https://other.example.com/cb?user=admin&token=secret",
			Err: fmt.Errorf("connection refused"),
		}
		got := p.redactedErr(err)
		assert.NotContains(t, got.Error(), "token=secret")
		assert.NotContains(t, got.Error(), "user=admin")
	})

	// A space is legal in a query and survives url.URL.String() verbatim.
	// Redacting the URL field rather than scanning the message means the
	// whole query goes, not just the part before the space.
	t.Run("query containing a space is fully redacted", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		err := &neturl.Error{
			Op:  "Get",
			URL: "http://h/a?msg=hello world&tok=secret",
			Err: fmt.Errorf("connection refused"),
		}
		got := p.redactedErr(err)
		assert.NotContains(t, got.Error(), "tok=secret")
		assert.NotContains(t, got.Error(), "hello world")
		assert.Contains(t, got.Error(), "<redacted>")
	})

	// errors.Is/As must not depend on whether redaction is enabled, so the
	// *url.Error is rebuilt rather than flattened into a bare error.
	t.Run("error type and unwrap chain are preserved", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		inner := context.DeadlineExceeded
		err := &neturl.Error{Op: "Get", URL: "http://h/a?tok=secret", Err: inner}
		got := p.redactedErr(err)

		var ue *neturl.Error
		assert.True(t, errors.As(got, &ue), "redacted error should still be a *url.Error")
		assert.True(t, errors.Is(got, context.DeadlineExceeded), "unwrap chain should survive")
		assert.Equal(t, "http://h/a?<redacted>", ue.URL)
		assert.True(t, ue.Timeout(), "Timeout() should still report true")
	})

	// url.Error renders the URL with %q, so a quote in the query arrives
	// escaped; the match must not stop at it and leave the tail exposed.
	t.Run("quote in query is fully redacted", func(t *testing.T) {
		p := &Probe{redactURLQueryInLogs: true}
		qu := mustParseURL(t, `http://example.com/a?sig=x"y`)
		err := &neturl.Error{Op: "Get", URL: qu.String(), Err: fmt.Errorf("connection refused")}
		got := p.redactedErr(err)
		assert.NotContains(t, got.Error(), "sig=x")
		assert.NotContains(t, got.Error(), `y"`)
		assert.Contains(t, got.Error(), "<redacted>")
	})
}

// net/http puts the request URL in url.Error's URL field, but a redirect whose
// Location fails to parse goes in the *inner* error instead -- a different URL,
// carrying a different query. Redacting only the URL field leaks it.
func TestRedactedErrRedirectLocationInInnerError(t *testing.T) {
	// An invalid port makes url.Parse reject the Location, which is what
	// drives net/http down that path.
	loc := "https://other.example.com:notaport/cb?code=supersecret"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", loc)
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	_, err := http.Get(srv.URL + "/start?mytok=mysecret")
	if err == nil {
		t.Fatal("expected the request to fail")
	}

	p := &Probe{redactURLQueryInLogs: true}
	got := p.redactedErr(err).Error()
	assert.NotContains(t, got, "code=supersecret", "redirect Location query leaked via the inner error")
	assert.NotContains(t, got, "mytok=mysecret", "request query leaked")
	assert.Contains(t, got, "<redacted>")
}

// Rewriting the inner error is only worth it when it actually contains a URL;
// otherwise the error keeps its type and Unwrap chain.
func TestRedactedErrKeepsUnwrapWhenInnerErrIsClean(t *testing.T) {
	p := &Probe{redactURLQueryInLogs: true}
	err := &neturl.Error{Op: "Get", URL: "http://h/a?tok=secret", Err: context.DeadlineExceeded}

	got := p.redactedErr(err)
	assert.True(t, errors.Is(got, context.DeadlineExceeded), "unwrap chain should survive")
	assert.NotContains(t, got.Error(), "tok=secret")
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
