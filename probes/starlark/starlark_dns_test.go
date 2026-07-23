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
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	starlarklib "go.starlark.net/starlark"
)

// testDNSServer starts an in-process UDP DNS server with canned answers and
// returns its host:port. The dns.resolve builtin, given server=<this>, routes
// Go's pure-Go resolver here, so record logic is exercised without real DNS.
func testDNSServer(t *testing.T) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		q := r.Question[0]
		switch q.Name {
		case "good.example.":
			if q.Qtype == dns.TypeA {
				rr, _ := dns.NewRR("good.example. 300 IN A 10.0.0.5")
				m.Answer = append(m.Answer, rr)
			}
		case "text.example.":
			if q.Qtype == dns.TypeTXT {
				rr, _ := dns.NewRR(`text.example. 300 IN TXT "hello world"`)
				m.Answer = append(m.Answer, rr)
			}
		case "missing.example.":
			m.SetRcode(r, dns.RcodeNameError) // NXDOMAIN
		}
		_ = w.WriteMsg(m)
	})

	srv := &dns.Server{PacketConn: pc, Handler: handler}
	go func() { _ = srv.ActivateAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown() })
	return pc.LocalAddr().String()
}

// runDNSScript runs a one-probe script and returns the single result. The
// script's server= address is filled from the in-process test DNS server.
func runDNSScript(t *testing.T, script string) *dnsRunResult {
	t.Helper()
	server := testDNSServer(t)
	source := fmt.Sprintf(script, server)
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	require.NoError(t, p.Init("dns-test", opts))
	results := p.RunOnce(context.Background())
	require.Len(t, results, 1)
	return &dnsRunResult{success: results[0].Success, err: results[0].Error}
}

type dnsRunResult struct {
	success bool
	err     error
}

// The scripts below check their expectations inline via fail(), so a
// successful probe run means the DnsResult had the expected shape. (The assert
// module has no general assert.true yet -- that's future work.)

func TestDNS_ARecord(t *testing.T) {
	r := runDNSScript(t, `
def probe(target):
    r = dns.resolve("good.example.", server="%s")
    if r.rcode != "NOERROR": fail("rcode=" + r.rcode)
    if "10.0.0.5" not in r.answers: fail("answers=%%s" %% r.answers)
    if r.name != "good.example.": fail("name=" + r.name)
    if r.type != "A": fail("type=" + r.type)
    if r.latency() <= 0: fail("latency not positive")
`)
	assert.True(t, r.success, "expected success, err=%v", r.err)
}

func TestDNS_TypeLowercaseNormalized(t *testing.T) {
	r := runDNSScript(t, `
def probe(target):
    r = dns.resolve("text.example.", type="txt", server="%s")
    if r.type != "TXT": fail("type=" + r.type)
    if "hello world" not in r.answers: fail("answers=%%s" %% r.answers)
`)
	assert.True(t, r.success, "expected success, err=%v", r.err)
}

func TestDNS_NXDOMAIN_IsResultNotError(t *testing.T) {
	r := runDNSScript(t, `
def probe(target):
    r = dns.resolve("missing.example.", server="%s")
    if r.rcode != "NXDOMAIN": fail("rcode=" + r.rcode)
    if len(r.answers) != 0: fail("answers=%%s" %% r.answers)
`)
	assert.True(t, r.success, "expected success, err=%v", r.err)
}

func TestDNS_UnsupportedType(t *testing.T) {
	r := runDNSScript(t, `
def probe(target):
    dns.resolve("good.example.", type="SOA", server="%s")
`)
	assert.False(t, r.success)
	require.Error(t, r.err)
	assert.Contains(t, r.err.Error(), "unsupported record type")
}

// TestLatencyMethod covers the shared latency() unit selector directly (dns and
// http Response both use it), so the float conversions are pinned without
// depending on any real measured duration.
func TestLatencyMethod(t *testing.T) {
	fn := latencyMethod("Test", 1500*time.Millisecond)
	thread := &starlarklib.Thread{}

	got, err := starlarklib.Call(thread, fn, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, starlarklib.Float(1.5), got)

	got, err = starlarklib.Call(thread, fn, nil, []starlarklib.Tuple{{starlarklib.String("unit"), starlarklib.String("ms")}})
	require.NoError(t, err)
	assert.Equal(t, starlarklib.Float(1500), got)

	_, err = starlarklib.Call(thread, fn, nil, []starlarklib.Tuple{{starlarklib.String("unit"), starlarklib.String("us")}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown unit")
}
