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

package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRedirectFunc(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/1", http.StatusTemporaryRedirect)
		case "/1":
			http.Redirect(w, r, "/2", http.StatusTemporaryRedirect)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	cases := []struct {
		desc       string
		n          int
		wantStatus int
	}{
		{"zero blocks all redirects", 0, http.StatusTemporaryRedirect},
		{"one allows first redirect only", 1, http.StatusTemporaryRedirect},
		{"two follows both redirects", 2, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			c := &http.Client{CheckRedirect: CheckRedirectFunc(tc.n)}
			resp, err := c.Get(ts.URL)
			assert.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func mustReq(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	return req
}

func TestBuildProxy_EmptyProxyURL(t *testing.T) {
	fn, header, scheme, err := BuildProxy("", "", nil)
	require.NoError(t, err)
	assert.Nil(t, fn)
	assert.Nil(t, header)
	assert.Empty(t, scheme)
}

func TestBuildProxy_NoProxyOrHeaderWithoutURL(t *testing.T) {
	_, _, _, err := BuildProxy("", "localhost", nil)
	require.Error(t, err)

	_, _, _, err = BuildProxy("", "", map[string]string{"X": "y"})
	require.Error(t, err)
}

// Empty no_proxy must restore the pre-helper default: every request is
// proxied, including loopback/localhost targets.
func TestBuildProxy_EmptyNoProxy_ProxiesLoopback(t *testing.T) {
	fn, _, scheme, err := BuildProxy("http://proxy.example.com:3128", "", nil)
	require.NoError(t, err)
	require.NotNil(t, fn)
	assert.Equal(t, "http", scheme)

	got, err := fn(mustReq(t, "http://127.0.0.1:9999/"))
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "proxy.example.com:3128", got.Host)
}

func TestBuildProxy_NoProxyBypassesMatchingHost(t *testing.T) {
	fn, _, _, err := BuildProxy("http://proxy.example.com:3128", "bypass.example.com", nil)
	require.NoError(t, err)
	require.NotNil(t, fn)

	bypassed, err := fn(mustReq(t, "http://bypass.example.com/"))
	require.NoError(t, err)
	assert.Nil(t, bypassed, "no_proxy host should bypass the proxy")

	proxied, err := fn(mustReq(t, "http://other.example.com/"))
	require.NoError(t, err)
	require.NotNil(t, proxied)
	assert.Equal(t, "proxy.example.com:3128", proxied.Host)
}

func TestBuildProxy_InvalidURL_CredentialSafe(t *testing.T) {
	_, _, _, err := BuildProxy("http://user:secretpass@proxy.example.com:not-a-port/", "", nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secretpass")
}

func TestBuildProxy_RejectsUnsupportedScheme(t *testing.T) {
	tests := map[string]string{
		"unsupported scheme": "ftp://proxy.internal:3128",
		"relative":           "proxy.internal:3128",
		"missing host":       "http:///path",
	}
	for desc, proxyURL := range tests {
		t.Run(desc, func(t *testing.T) {
			_, _, _, err := BuildProxy(proxyURL, "", nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "proxy_url")
		})
	}
}

func TestBuildProxy_SchemeLowercased(t *testing.T) {
	tests := map[string]string{
		"http lowercase":   "http://proxy.example.com",
		"HTTP mixed case":  "HTTP://proxy.example.com",
		"https lowercase":  "https://proxy.example.com",
		"HTTPS mixed case": "HTTPS://proxy.example.com",
	}
	want := map[string]string{
		"http lowercase":   "http",
		"HTTP mixed case":  "http",
		"https lowercase":  "https",
		"HTTPS mixed case": "https",
	}
	for desc, proxyURL := range tests {
		t.Run(desc, func(t *testing.T) {
			_, _, scheme, err := BuildProxy(proxyURL, "", nil)
			require.NoError(t, err)
			assert.Equal(t, want[desc], scheme)
		})
	}
}
