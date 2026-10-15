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

// Package httpclient holds helpers shared by probes that build their own
// *http.Client.
package httpclient

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/http/httpproxy"
)

// CheckRedirectFunc returns a CheckRedirect that follows up to n redirects
// (n=0 disables). Uses ErrUseLastResponse so the redirect response is
// returned to the caller instead of surfacing as an error.
func CheckRedirectFunc(n int) func(*http.Request, []*http.Request) error {
	return func(_ *http.Request, via []*http.Request) error {
		if len(via) > n {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

// BuildProxy parses proxyURL and returns the Transport.Proxy func and
// ProxyConnectHeader for it, shared by every probe that exposes a proxy_url
// config field, so the same config means the same behavior regardless of
// probe type.
//
// Built on httpproxy.Config rather than http.ProxyURL: it adds no_proxy
// support, but it also means loopback/localhost targets are never proxied
// -- that bypass is hardcoded in httpproxy.Config.ProxyFunc with no way to
// override it via no_proxy. Probes moving onto this helper from a bare
// http.ProxyURL should expect that behavior change.
//
// An empty proxyURL returns a nil func, nil header, nil error; the caller
// keeps whatever default it had (e.g. http.ProxyFromEnvironment). noProxy or
// connectHeader set without proxyURL is an error, since there is nothing
// for them to attach to.
func BuildProxy(proxyURL, noProxy string, connectHeader map[string]string) (func(*http.Request) (*url.URL, error), http.Header, error) {
	if proxyURL == "" {
		if noProxy != "" || len(connectHeader) > 0 {
			return nil, nil, errors.New("no_proxy/proxy_connect_header require proxy_url to be set")
		}
		return nil, nil, nil
	}

	// Parse the proxy URL to validate it and extract the scheme and host. We
	// don't use url.ParseRequestURI because it rejects URLs with a scheme but
	// no path, which is valid for a proxy URL (e.g. "http://proxy.example.com").
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		// Unwrap *url.Error: its Error() string embeds the raw input,
		// which may contain proxy credentials (http://user:pass@host).
		if uerr, ok := errors.AsType[*url.Error](err); ok {
			err = uerr.Err
		}
		return nil, nil, fmt.Errorf("proxy_url: %v", err)
	}

	// Validate the scheme and host. httpproxy.Config.ProxyFunc will accept
	// any scheme, but we only support http, https, and socks5. The host is
	// required for a proxy URL.
	switch parsed.Scheme {
	case "http", "https", "socks5":
	default:
		return nil, nil, errors.New("proxy_url: must be an absolute URL with an http, https, or socks5 scheme and a host")
	}
	if parsed.Host == "" {
		return nil, nil, errors.New("proxy_url: must be an absolute URL with a host")
	}

	cfg := &httpproxy.Config{
		HTTPProxy:  proxyURL,
		HTTPSProxy: proxyURL,
		NoProxy:    noProxy,
	}
	fn := cfg.ProxyFunc()
	proxyFunc := func(req *http.Request) (*url.URL, error) { return fn(req.URL) }

	var header http.Header
	if len(connectHeader) > 0 {
		header = make(http.Header, len(connectHeader))
		for k, v := range connectHeader {
			header.Add(k, v)
		}
	}
	return proxyFunc, header, nil
}
