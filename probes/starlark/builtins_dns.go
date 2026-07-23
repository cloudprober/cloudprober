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
// Thin wrapper around Go's net.Resolver (deliberately not a reuse of
// probes/dns, mirroring how the http builtin stays independent of probes/http).
// server=None uses the system resolver config; an explicit server routes the
// pure-Go resolver at that address. A response with an NXDOMAIN/NODATA rcode is
// a result (empty .answers), not an error -- only transport failures raise.

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
		"server?", &serverArg,
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

	rcode := "NOERROR"
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			rcode, answers = "NXDOMAIN", nil
		} else {
			return nil, fmt.Errorf("%s: %v", fname, err)
		}
	}

	return &dnsResult{name: name, rtype: rtype, answers: answers, rcode: rcode, latency: latency}, nil
}

// normalizeDNSServer appends the default DNS port when the server has none.
func normalizeDNSServer(s string) string {
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
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
	rcode   string
	latency time.Duration
}

var _ starlarklib.Value = (*dnsResult)(nil)
var _ starlarklib.HasAttrs = (*dnsResult)(nil)

func (r *dnsResult) String() string {
	return fmt.Sprintf("<dns_result name=%s type=%s rcode=%s answers=%d>", r.name, r.rtype, r.rcode, len(r.answers))
}
func (r *dnsResult) Type() string            { return "DnsResult" }
func (r *dnsResult) Freeze()                 {}
func (r *dnsResult) Truth() starlarklib.Bool { return starlarklib.Bool(r.rcode == "NOERROR") }
func (r *dnsResult) Hash() (uint32, error)   { return 0, fmt.Errorf("DnsResult is unhashable") }

func (r *dnsResult) Attr(name string) (starlarklib.Value, error) {
	switch name {
	case "name":
		return starlarklib.String(r.name), nil
	case "type":
		return starlarklib.String(r.rtype), nil
	case "rcode":
		return starlarklib.String(r.rcode), nil
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
	return []string{"answers", "latency", "name", "rcode", "type"}
}
