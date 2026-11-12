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
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	starlarklib "go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// ----------------------------------------------------------------------------
// dns module
//
// This builtin resolves a name as one step inside a script -- get the A/TXT
// records, branch on them, feed them into an http call. Monitoring a DNS
// *server* (wire rcodes, UDP/TCP/TLS, per-record validators) is the dns probe
// type's job, and this is deliberately not a second one: it is a thin wrapper
// around Go's net.Resolver, with no probes/dns or miekg dependency, the same
// way the http builtin stays independent of probes/http.
//
// That backend is a resolver, not a DNS client, which bounds what a result can
// say. There is no wire rcode, no TTL, no authority/additional section, and no
// way to tell NODATA ("name exists, no records of this type") from NXDOMAIN --
// net.DNSError.IsNotFound covers both. So a result reports only what it can
// stand behind: the answers it found, or none. Scripts that must assert real
// server behavior (SERVFAIL vs REFUSED, NODATA vs NXDOMAIN) want the dns probe
// type instead; that is a different capability, not a gap to fill in here.
//
// Finding nothing is a result (empty .answers), not an error. Every other
// lookup failure -- timeout, SERVFAIL, REFUSED, network error, malformed
// response -- raises.
//
// The two server modes are not interchangeable. server=None resolves the way
// the host does, honoring /etc/hosts, the search list, ndots and (where it
// applies) the cgo resolver -- that is the mode for "what does this machine
// see?". An explicit server forces the pure-Go resolver to dial that address,
// which skips /etc/hosts and the host's resolver choice; it answers "what does
// that nameserver say?" well enough for a probe step, but it is not dig.

func dnsModule() *starlarkstruct.Module {
	return &starlarkstruct.Module{
		Name: "dns",
		Members: starlarklib.StringDict{
			"resolve": starlarklib.NewBuiltin("dns.resolve", dnsResolve),
		},
	}
}

// dnsTypes are the record types net.Resolver can look up. Exotic types (SOA,
// CAA, ...) are intentionally out of scope for this net.Resolver-based builtin.
var dnsTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "MX": true,
	"NS": true, "TXT": true, "SRV": true, "PTR": true,
}

func dnsResolve(thread *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, kwargs []starlarklib.Tuple) (starlarklib.Value, error) {
	const fname = "dns.resolve"
	var name, rtype string
	var serverArg, timeoutArg starlarklib.Value
	rtype = "A"
	if err := starlarklib.UnpackArgs(fname, args, kwargs,
		"name", &name,
		"type?", &rtype,
		"server??", &serverArg,
		"timeout??", &timeoutArg,
	); err != nil {
		return nil, err
	}

	rtype = strings.ToUpper(rtype)
	if !dnsTypes[rtype] {
		return nil, fmt.Errorf("%s: unsupported record type %q (want one of A, AAAA, CNAME, MX, NS, TXT, SRV, PTR)", fname, rtype)
	}

	server, err := optionalString(serverArg, fname+": server")
	if err != nil {
		return nil, err
	}
	timeout, hasTimeout, err := optionalDurationSeconds(timeoutArg, fname+": timeout")
	if err != nil {
		return nil, err
	}

	ctx := ctxFromThread(thread)
	if hasTimeout {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	resolver := &net.Resolver{}
	if server != nil {
		addr := normalizeDNSServer(*server)
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		}
	}

	start := time.Now()
	answers, err := dnsLookup(ctx, resolver, rtype, name)
	latency := time.Since(start)

	if err != nil {
		// IsNotFound covers NXDOMAIN and NODATA alike; both mean "found
		// nothing", which is a result with no answers. Anything else raises.
		var dnsErr *net.DNSError
		if !errors.As(err, &dnsErr) || !dnsErr.IsNotFound {
			return nil, fmt.Errorf("%s: %v", fname, err)
		}
		answers = nil
	}

	return &dnsResult{name: name, rtype: rtype, answers: answers, latency: latency}, nil
}

// normalizeDNSServer appends the default DNS port when the server has none.
// A port-less IPv6 address is accepted bare ("::1") or bracketed ("[::1]");
// JoinHostPort re-adds the brackets, so an existing pair has to come off first
// or we'd emit "[[::1]]:53".
func normalizeDNSServer(s string) string {
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = s[1 : len(s)-1]
	}
	return net.JoinHostPort(s, "53")
}

func dnsLookup(ctx context.Context, r *net.Resolver, rtype, name string) ([]string, error) {
	switch rtype {
	case "A", "AAAA":
		network := "ip4"
		if rtype == "AAAA" {
			network = "ip6"
		}
		ips, err := r.LookupIP(ctx, network, name)
		if err != nil {
			return nil, err
		}
		out := make([]string, len(ips))
		for i, ip := range ips {
			out[i] = ip.String()
		}
		return out, nil
	case "CNAME":
		cname, err := r.LookupCNAME(ctx, name)
		if err != nil {
			return nil, err
		}
		return []string{cname}, nil
	case "MX":
		mxs, err := r.LookupMX(ctx, name)
		if err != nil {
			return nil, err
		}
		out := make([]string, len(mxs))
		for i, mx := range mxs {
			out[i] = fmt.Sprintf("%d %s", mx.Pref, mx.Host)
		}
		return out, nil
	case "NS":
		nss, err := r.LookupNS(ctx, name)
		if err != nil {
			return nil, err
		}
		out := make([]string, len(nss))
		for i, ns := range nss {
			out[i] = ns.Host
		}
		return out, nil
	case "TXT":
		return r.LookupTXT(ctx, name)
	case "SRV":
		_, srvs, err := r.LookupSRV(ctx, "", "", name)
		if err != nil {
			return nil, err
		}
		out := make([]string, len(srvs))
		for i, srv := range srvs {
			out[i] = fmt.Sprintf("%d %d %d %s", srv.Priority, srv.Weight, srv.Port, srv.Target)
		}
		return out, nil
	case "PTR":
		return r.LookupAddr(ctx, name)
	}
	// Unreachable: rtype is validated against dnsTypes before we get here.
	return nil, fmt.Errorf("unsupported record type %q", rtype)
}

// ----------------------------------------------------------------------------
// DnsResult value

type dnsResult struct {
	name    string
	rtype   string
	answers []string
	latency time.Duration
}

var _ starlarklib.Value = (*dnsResult)(nil)
var _ starlarklib.HasAttrs = (*dnsResult)(nil)

func (r *dnsResult) String() string {
	return fmt.Sprintf("<dns_result name=%s type=%s answers=%d>", r.name, r.rtype, len(r.answers))
}
func (r *dnsResult) Type() string          { return "DnsResult" }
func (r *dnsResult) Freeze()               {}
func (r *dnsResult) Hash() (uint32, error) { return 0, fmt.Errorf("DnsResult is unhashable") }

// Truth makes "if r:" mean "found something", so a lookup that resolved to
// nothing is falsey without the script having to reach for len(r.answers).
func (r *dnsResult) Truth() starlarklib.Bool { return starlarklib.Bool(len(r.answers) > 0) }

func (r *dnsResult) Attr(name string) (starlarklib.Value, error) {
	switch name {
	case "name":
		return starlarklib.String(r.name), nil
	case "type":
		return starlarklib.String(r.rtype), nil
	case "answers":
		elems := make([]starlarklib.Value, len(r.answers))
		for i, a := range r.answers {
			elems[i] = starlarklib.String(a)
		}
		l := starlarklib.NewList(elems)
		l.Freeze()
		return l, nil
	case "latency":
		return latencyMethod("DnsResult", r.latency), nil
	}
	return nil, nil
}

func (r *dnsResult) AttrNames() []string {
	return []string{"answers", "latency", "name", "type"}
}
