// Copyright 2019-2023 The Cloudprober Authors.
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
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudprober/cloudprober/common/iputils"
	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/internal/httpreq"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const relURLLabel = "relative_url"

func hostWithPort(host string, port int) string {
	if port == 0 {
		return host
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// Put square brackets around literal IPv6 hosts.
func handleIPv6(host string) string {
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if iputils.IPVersion(ip) == 6 {
		return "[" + host + "]"
	}
	return host
}

func (p *Probe) schemeForTarget(target endpoint.Endpoint) string {
	switch p.c.SchemeType.(type) {
	case *configpb.ProbeConf_Scheme_:
		return strings.ToLower(p.c.GetScheme().String())
	case *configpb.ProbeConf_Protocol:
		return strings.ToLower(p.c.GetProtocol().String())
	}

	for _, label := range []string{"__cp_scheme__"} {
		if target.Labels[label] != "" {
			return strings.ToLower(target.Labels[label])
		}
	}

	return "http"
}

func hostForTarget(target endpoint.Endpoint) string {
	for _, label := range []string{"fqdn", "__cp_host__"} {
		if target.Labels[label] != "" {
			return handleIPv6(target.Labels[label])
		}
	}

	return handleIPv6(target.Name)
}

func pathForTarget(target endpoint.Endpoint, probeURL string) string {
	if probeURL != "" {
		return probeURL
	}

	for _, label := range []string{relURLLabel, "__cp_path__"} {
		if path := target.Labels[label]; path != "" {
			if !strings.HasPrefix(path, "/") {
				return "/" + path
			}
			return path
		}
	}

	return ""
}

func (p *Probe) resolveFirst(target endpoint.Endpoint) bool {
	if p.c.ResolveFirst != nil {
		return p.c.GetResolveFirst()
	}
	return target.IP != nil
}

// setHeaders computes setHeaders for a target. Host header is computed slightly
// differently than other setHeaders.
//   - If host header is set in the probe, it overrides everything else.
//   - Otherwise we use target's host (computed elsewhere) along with port.
//
// Header values may contain @uuid@ tokens; those are kept verbatim here and
// resolved per-send by requestForSend so each request gets a fresh UUIDv4.
// Use @@ to emit a literal @.
func (p *Probe) setHeaders(req *http.Request, host string, port int) {
	var hostHeader string

	for _, h := range p.c.GetHeaders() {
		if h.GetName() == "Host" {
			hostHeader = h.GetValue()
			continue
		}
		req.Header.Set(h.GetName(), h.GetValue())
	}

	for k, v := range p.c.GetHeader() {
		if k == "Host" {
			hostHeader = v
			continue
		}
		req.Header.Set(k, v)
	}

	if hostHeader == "" {
		hostHeader = hostWithPort(host, port)
	}
	req.Host = hostHeader
}

// initDynamicHeaders records canonical names of headers whose values carry
// a substitution token. Host is excluded: req.Host is not substituted, so
// a literal '@' there is warned about and sent verbatim.
func (p *Probe) initDynamicHeaders() {
	seen := map[string]bool{}
	add := func(name, val string) {
		if !strings.Contains(val, "@") {
			return
		}
		if name == "Host" {
			p.l.Warningf("http probe %q: Host header value %q contains '@' but Host substitution is not supported; value will be sent verbatim", p.name, val)
			return
		}
		canon := http.CanonicalHeaderKey(name)
		if !seen[canon] {
			seen[canon] = true
			p.dynamicHeaderNames = append(p.dynamicHeaderNames, canon)
		}
	}
	for _, h := range p.c.GetHeaders() {
		add(h.GetName(), h.GetValue())
	}
	for k, v := range p.c.GetHeader() {
		add(k, v)
	}
	add("User-Agent", p.c.GetUserAgent())
}

// applyDynamicHeaders substitutes @uuid@ (and future tokens) in tracked
// header values. All @uuid@ occurrences within a single send resolve to
// the same UUID. req.Host is intentionally left alone.
func (p *Probe) applyDynamicHeaders(req *http.Request) {
	var subst map[string]string
	for _, name := range p.dynamicHeaderNames {
		vv := req.Header[name]
		for i, v := range vv {
			if !strings.Contains(v, "@") {
				continue
			}
			if subst == nil {
				subst = map[string]string{"uuid": uuid.NewString()}
			}
			if nv, _ := strtemplate.SubstituteLabels(v, subst); nv != v {
				vv[i] = nv
			}
		}
	}
}

// dynamicHeaderAttrs returns the resolved per-send values for tracked
// dynamic headers, for use in error-log attributes.
func (p *Probe) dynamicHeaderAttrs(req *http.Request) []slog.Attr {
	if len(p.dynamicHeaderNames) == 0 {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(p.dynamicHeaderNames))
	for _, name := range p.dynamicHeaderNames {
		if v := req.Header.Get(name); v != "" {
			attrs = append(attrs, slog.String(name, v))
		}
	}
	return attrs
}

func (p *Probe) urlHostAndIPLabel(target endpoint.Endpoint, host string) (string, string, error) {
	if !p.resolveFirst(target) {
		return host, "", nil
	}

	ip, err := target.Resolve(p.opts.IPVersion, p.opts.Targets, endpoint.WithNameOverride(host))
	if err != nil {
		return "", "", fmt.Errorf("error resolving target: %s, %v", target.Name, err)
	}

	ipStr := ip.String()

	return handleIPv6(ipStr), ipStr, nil
}

func (p *Probe) httpRequestForTarget(target endpoint.Endpoint) (*http.Request, error) {
	// Prepare HTTP.Request for Client.Do
	port := int(p.c.GetPort())
	// If port is not configured explicitly, use target's port if available.
	if port == 0 {
		port = target.Port
	}

	host := hostForTarget(target)

	urlHost, ipForLabel, err := p.urlHostAndIPLabel(target, host)
	// Make sure we update additional labels even if there is an error.
	for _, al := range p.opts.AdditionalLabels {
		al.UpdateForTarget(target, ipForLabel, port)
	}
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s://%s%s", p.schemeForTarget(target), hostWithPort(urlHost, port), pathForTarget(target, p.url))

	req, err := httpreq.NewRequest(p.method, url, p.requestBody)
	if err != nil {
		return nil, err
	}

	p.setHeaders(req, host, port)
	if p.c.GetUserAgent() != "" {
		req.Header.Set("User-Agent", p.c.GetUserAgent())
	}

	return req, nil
}

func getToken(ts oauth2.TokenSource, l *logger.Logger) (string, error) {
	tok, err := ts.Token()
	if err != nil {
		return "", err
	}
	l.Debug("Got OAuth token, len: ", strconv.FormatInt(int64(len(tok.AccessToken)), 10), ", expirationTime: ", tok.Expiry.String())

	if tok.AccessToken != "" {
		return tok.AccessToken, nil
	}

	idToken, ok := tok.Extra("id_token").(string)
	if ok {
		return idToken, nil
	}

	return "", fmt.Errorf("got unknown token: %v", tok)
}

// prepareRequest derives a per-send request from the cached one. Cloning is
// the only safe way to mutate without racing parallel sends (rpp > 1), so we
// do it once when any of the per-send mutations apply -- dynamic header
// substitution, OAuth Authorization header, or a fresh streaming body --
// and skip to a cheap WithContext copy otherwise.
func (p *Probe) prepareRequest(ctx context.Context, req *http.Request) *http.Request {
	if len(p.dynamicHeaderNames) == 0 && p.oauthTS == nil && p.requestBody.Len() == 0 {
		return req.WithContext(ctx)
	}
	req = req.Clone(ctx)

	if len(p.dynamicHeaderNames) > 0 {
		p.applyDynamicHeaders(req)
	}

	if p.oauthTS != nil {
		tok, err := getToken(p.oauthTS, p.l)
		// Note: We don't terminate the request if there is an error in getting
		// token. That is to avoid complicating the flow, and to make sure that
		// OAuth refresh failures show in probe failures.
		if err != nil {
			p.l.Error("Error getting OAuth token: ", err.Error())
			tok = "<token-missing>"
		}
		req.Header.Set("Authorization", fmt.Sprintf(p.c.GetOauthConfig().GetTokenTypeFormat(), tok))
	}

	if p.requestBody.Len() > 0 {
		req.Body = p.requestBody.Reader()
	}

	return req
}
