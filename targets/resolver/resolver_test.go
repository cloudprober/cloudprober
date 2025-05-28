// Copyright 2017-2024 The Cloudprober Authors.
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

package resolver

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// ipVersion tells if an IP address is IPv4 or IPv6.
func testIPVersion(ip net.IP) int {
	if len(ip.To4()) == net.IPv4len {
		return 4
	}
	if len(ip) == net.IPv6len {
		return 6
	}
	return 0
}

type resolveBackendWithTracking struct {
	nameToIP map[cacheRecordKey][]net.IP
	called   int
	callLog  []string
	mu       sync.Mutex
}

func (b *resolveBackendWithTracking) setNameToIP(name string, ips ...net.IP) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.nameToIP == nil {
		b.nameToIP = make(map[cacheRecordKey][]net.IP)
	}
	// Reset cache for this name
	for _, network := range []ipVersion{IP, IP4, IP6} {
		b.nameToIP[cacheRecordKey{name, network}] = nil
	}
	for _, ip := range ips {
		network := ipVersion(testIPVersion(ip))
		b.nameToIP[cacheRecordKey{name, network}] = append(b.nameToIP[cacheRecordKey{name, network}], ip)
		b.nameToIP[cacheRecordKey{name, IP}] = append(b.nameToIP[cacheRecordKey{name, IP}], ip)
	}
}

func (b *resolveBackendWithTracking) resolve(_ context.Context, network, name string) ([]net.IP, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.called++
	b.callLog = append(b.callLog, fmt.Sprintf("resolve(%s, %s)", network, name))
	if b.nameToIP == nil {
		b.nameToIP = make(map[cacheRecordKey][]net.IP)
	}
	networkT := map[string]ipVersion{"": IP, "ip": IP, "ip4": IP4, "ip6": IP6}[network]
	return b.nameToIP[cacheRecordKey{name, ipVersion(networkT)}], nil
}

func (b *resolveBackendWithTracking) calls() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.called
}

func (b *resolveBackendWithTracking) callsLog() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string{}, b.callLog...)
}

func verify(testCase string, t *testing.T, ip, expectedIP net.IP, b *resolveBackendWithTracking, expectedBackendCalls int, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: Error while resolving. Err: %v", testCase, err)
	}
	if !ip.Equal(expectedIP) {
		t.Errorf("%s: Got wrong IP address. Got: %s, Expected: %s", testCase, ip, expectedIP)
	}
	assert.Equal(t, expectedBackendCalls, b.calls(), "backend calls mismatch. Calllog: \n%v", b.callsLog())
}

// waitForChannelOrFail reads the result from the channel and fails if it
// wasn't received within the timeout.
func waitForRefreshOrFail(t *testing.T, c <-chan bool, timeout time.Duration) bool {
	select {
	case b := <-c:
		return b
	case <-time.After(timeout):
		t.Error("Channel didn't close. Stack-trace: ", debug.Stack())
		return false
	}
}

// waitForRefreshAndVerify reads the result from the channel and fails if it
// wasn't received within the timeout.
func waitForRefreshAndVerify(t *testing.T, c <-chan bool, timeout time.Duration, expectRefresh bool) {
	assert.Equal(t, expectRefresh, waitForRefreshOrFail(t, c, timeout), "refreshed")
}

func TestResolveWithMaxAge(t *testing.T) {
	b := &resolveBackendWithTracking{}
	r := New(WithResolveFunc(b.resolve))

	testHost := "hostA"
	expectedIP := net.ParseIP("1.2.3.4")
	b.setNameToIP(testHost, expectedIP)

	// Resolve a host, there is no cache, a backend call should be made
	expectedBackendCalls := 1
	refreshed := make(chan bool, 2)
	ip, err := r.resolveWithMaxAge(testHost, 4, 60*time.Second, refreshed)
	verify("first-run-no-cache", t, ip, expectedIP, b, expectedBackendCalls, err)
	// First Resolve calls refresh twice. Once for init (which succeeds), and
	// then again for refreshing, which is not needed. Hence the results are true
	// and then false.
	waitForRefreshAndVerify(t, refreshed, time.Second, true)
	waitForRefreshAndVerify(t, refreshed, time.Second, false)

	// Resolve same host again, it should come from cache, no backend call
	newExpectedIP := net.ParseIP("1.2.3.6")
	b.setNameToIP(testHost, newExpectedIP)
	ip, err = r.resolveWithMaxAge(testHost, 4, 60*time.Second, refreshed)
	verify("second-run-from-cache", t, ip, expectedIP, b, expectedBackendCalls, err)
	waitForRefreshAndVerify(t, refreshed, time.Second, false)

	// Resolve same host again with maxAge=0, it will issue an asynchronous (hence no increment
	// in expectedBackenddCalls) backend call
	ip, err = r.resolveWithMaxAge(testHost, 4, 0*time.Second, refreshed)
	verify("third-run-expire-cache", t, ip, expectedIP, b, expectedBackendCalls, err)
	waitForRefreshAndVerify(t, refreshed, time.Second, true)

	// Now that refresh has happened, we should see a new IP.
	expectedIP = newExpectedIP
	expectedBackendCalls++
	ip, err = r.resolveWithMaxAge(testHost, 4, 60*time.Second, refreshed)
	verify("fourth-run-new-result", t, ip, expectedIP, b, expectedBackendCalls, err)
	waitForRefreshAndVerify(t, refreshed, time.Second, false)
}

// TestResolveErr tests the behavior of the resolver when backend returns an
// error. This test uses the refreshed channel and waits on it to make the
// resolver behavior deterministic.
func TestResolveErr(t *testing.T) {
	cnt := 0
	r := New(WithResolveFunc(func(_ context.Context, network, name string) ([]net.IP, error) {
		cnt++
		if cnt == 2 || cnt == 3 {
			return nil, fmt.Errorf("time to return error, cnt: %d", cnt)
		}
		return []net.IP{net.ParseIP("0.0.0.0")}, nil
	}))
	refreshedCh := make(chan bool, 2)
	// cnt=0; returning 0.0.0.0.
	ip, err := r.resolveWithMaxAge("testHost", 4, 60*time.Second, refreshedCh)
	if err != nil {
		t.Errorf("Got unexpected error: %v", err)
	}
	assert.Equal(t, net.ParseIP("0.0.0.0"), ip)
	// First Resolve calls refresh twice. Once for init (which succeeds), and
	// then again for refreshing, which is not needed. Hence the results are
	// true and then false.
	waitForRefreshAndVerify(t, refreshedCh, time.Second, true)
	waitForRefreshAndVerify(t, refreshedCh, time.Second, false)

	// cnt=1, returning 0.0.0.0, but will trigger another refresh due to maxAge
	time.Sleep(10 * time.Millisecond)
	ip, err = r.resolveWithMaxAge("testHost", 4, 10*time.Millisecond, refreshedCh)
	if err != nil {
		t.Errorf("Got unexpected error: %v", err)
	}
	// refresh triggers because of maxAge < time since last update.
	waitForRefreshAndVerify(t, refreshedCh, time.Second, true)
	assert.Equal(t, net.ParseIP("0.0.0.0"), ip)

	// cnt=2, last backend refresh failed and cache record contains an error,
	// but maxTTL is 60 seconds, so we'll not get an error.
	// refresh will still be triggered on this call because of error.
	r.maxCacheAge = 60 * time.Second
	ip, err = r.resolveWithMaxAge("testHost", 4, 60*time.Second, refreshedCh)
	if err != nil {
		t.Errorf("Unexpected error: %v, while maxTTL is 60s", err)
	}
	waitForRefreshAndVerify(t, refreshedCh, time.Second, true)
	assert.Equal(t, net.ParseIP("0.0.0.0"), ip)

	// cnt=3, last backend refresh failed again, CR still has error. This time
	// we get error because we reduce maxTTL to 10 milliseconds.
	// refresh will still be triggered on this call because of error.
	r.maxCacheAge = 10 * time.Millisecond
	ip, err = r.resolveWithMaxAge("testHost", 4, 60*time.Second, refreshedCh)
	if err == nil {
		t.Error("Expected error for maxTTL=10ms, but got none")
	}
	waitForRefreshAndVerify(t, refreshedCh, time.Second, true)
	assert.Equal(t, net.ParseIP("0.0.0.0"), ip)

	// No errors after the last refresh.
	// No refresh triggered this time because of maxAge.
	ip, err = r.resolveWithMaxAge("testHost", 4, 10*time.Second, refreshedCh)
	if err != nil {
		t.Errorf("Got unexpected error: %v", err)
	}
	waitForRefreshAndVerify(t, refreshedCh, time.Second, false)
	assert.Equal(t, net.ParseIP("0.0.0.0"), ip)
}

func TestResolveIPv6(t *testing.T) {
	b := &resolveBackendWithTracking{}
	r := New(WithResolveFunc(b.resolve))

	testHost := "hostA"
	expectedIPv4 := net.ParseIP("1.2.3.4")
	expectedIPv6 := net.ParseIP("::1")
	b.setNameToIP(testHost, expectedIPv4, expectedIPv6)

	ip, err := r.Resolve(testHost, 4)
	expectedBackendCalls := 1
	verify("ipv4-address-not-as-expected", t, ip, expectedIPv4, b, expectedBackendCalls, err)

	// IPv6 address will cause a new backend call.
	ip, err = r.Resolve(testHost, 6)
	expectedBackendCalls++
	verify("ipv6-address-not-as-expected", t, ip, expectedIPv6, b, expectedBackendCalls, err)

	// No IP version specified, should return IPv4 as IPv4 gets preference.
	// This will create a new cache record, so a new backend call.
	ip, err = r.Resolve(testHost, 0)
	expectedBackendCalls++
	verify("ipv0-address-not-as-expected", t, ip, expectedIPv4, b, expectedBackendCalls, err)

	// New host, with no IPv4 address
	testHost = "hostB"
	expectedIPv6 = net.ParseIP("::2")
	b.setNameToIP(testHost, expectedIPv6)

	ip, err = r.Resolve(testHost, 4)
	expectedBackendCalls++
	if err == nil {
		t.Errorf("resolved IPv4 address for an IPv6 only host: %s", ip.String())
	}

	ip, err = r.Resolve(testHost, 6)
	expectedBackendCalls++
	verify("ipv6-address-not-as-expected", t, ip, expectedIPv6, b, expectedBackendCalls, err)

	ip, err = r.Resolve(testHost, 0)
	expectedBackendCalls++
	verify("ipv0-address-not-as-expected", t, ip, expectedIPv6, b, expectedBackendCalls, err)
}

// TestConcurrentInit tests that multiple Resolves in parallel on the same
// target all return the same answer, and cause just 1 call to resolve.
func TestConcurrentInit(t *testing.T) {
	cnt := 0
	resolveWait := make(chan bool)
	r := New(WithResolveFunc(func(_ context.Context, network, name string) ([]net.IP, error) {
		cnt++
		// The first call should be blocked on resolveWait.
		if cnt == 1 {
			<-resolveWait
			return []net.IP{net.ParseIP("0.0.0.0")}, nil
		}
		// The 2nd call should never happen.
		return nil, fmt.Errorf("resolve should be called just once, cnt: %d", cnt)
	}))

	// 5 because first resolve calls refresh twice.
	refreshed := make(chan bool, 101)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			_, err := r.resolveWithMaxAge("testHost", 4, 60*time.Second, refreshed)
			if err != nil {
				t.Logf("Err: %v\n", err)
				t.Errorf("Expected no error, got error")
			}
			wg.Done()
		}()
	}
	// Give offline update goroutines a chance.
	// If we call resolve more than once, this will make those resolves fail.
	runtime.Gosched()
	time.Sleep(1 * time.Millisecond)
	// Makes one of the resolve goroutines unblock refresh.
	resolveWait <- true
	resolvedCount := 0
	// 5 because first resolve calls refresh twice.
	for i := 0; i < 101; i++ {
		if waitForRefreshOrFail(t, refreshed, time.Second) {
			resolvedCount++
		}
	}
	if resolvedCount != 1 {
		t.Errorf("resolvedCount=%v, want 1", resolvedCount)
	}
	wg.Wait()
}

// Set up benchmarks. Apart from performance stats it verifies the library's behavior during concurrent
// runs. It's kind of important as we use mutexes a lot, even though never in long running path, e.g.
// actual backend resolver is called outside mutexes.
//
// Use following command to run benchmark tests:
// BMC=6 BMT=4 // 6 CPUs, 4 sec
//
//	blaze test --config=gotsan :resolver_test --test_arg=-test.bench=. \
//	  --test_arg=-test.benchtime=${BMT}s --test_arg=-test.cpu=$BMC
type resolveBackendBenchmark struct {
	delay   time.Duration // artificial delay in resolving
	callCnt int64
	t       time.Time
}

func (rb *resolveBackendBenchmark) resolve(_ context.Context, network, name string) ([]net.IP, error) {
	rb.callCnt++
	fmt.Printf("Time since initiation: %s\n", time.Since(rb.t))
	if rb.delay != 0 {
		time.Sleep(rb.delay)
	}
	return []net.IP{net.ParseIP("0.0.0.0")}, nil
}

func BenchmarkResolve(b *testing.B) {
	rb := &resolveBackendBenchmark{
		delay: 10 * time.Millisecond,
		t:     time.Now(),
	}
	r := New(WithResolveFunc(rb.resolve))
	// RunParallel executes its body in parallel, in multiple goroutines. Parallelism is controlled by
	// the test -cpu (test.cpu) flag (default is GOMAXPROCS). So if benchmarks runs N times, that N
	// is spread over these goroutines.
	//
	// Example benchmark results with cpu=6
	// BenchmarkResolve-6  3000	   1689466 ns/op
	//
	// 3000 is the total number of iterations (N) and it took on an average 1.69ms per iteration.
	// Total run time = 1.69 x 3000 = 5.07s. Since each goroutine executed 3000/6 or 500 iterations, with
	// each iteration taking 10ms because of artificial delay, each goroutine will take at least 5s, very
	// close to what benchmark found out.
	b.RunParallel(func(pb *testing.PB) {
		// Next() returns true if there are more iterations to execute.
		for pb.Next() {
			r.resolveWithMaxAge("test", 4, 500*time.Millisecond, nil)
			time.Sleep(10 * time.Millisecond)
		}
	})
	fmt.Printf("Called backend resolve %d times\n", rb.callCnt)
}

func TestParseOverrideAddress(t *testing.T) {
	tests := []struct {
		dnsResolverOverride string
		wantNetwork         string
		wantAddr            string
		wantErr             bool
	}{
		{
			dnsResolverOverride: "1.1.1.1",
			wantNetwork:         "",
			wantAddr:            "1.1.1.1:53",
		},
		{
			dnsResolverOverride: "tcp://1.1.1.1",
			wantNetwork:         "tcp",
			wantAddr:            "1.1.1.1:53",
		},
		{
			dnsResolverOverride: "tcp://1.1.1.1:413",
			wantNetwork:         "tcp",
			wantAddr:            "1.1.1.1:413",
		},
		{
			dnsResolverOverride: "udp://1.1.1.1",
			wantNetwork:         "udp",
			wantAddr:            "1.1.1.1:53",
		},
		{
			dnsResolverOverride: "tls://1.1.1.1",
			wantNetwork:         "tcp-tls",
			wantAddr:            "1.1.1.1:853",
		},
		{
			dnsResolverOverride: "tls://1.1.1.1:8443",
			wantNetwork:         "tcp-tls",
			wantAddr:            "1.1.1.1:8443",
		},
		{
			dnsResolverOverride: "udp65://1.1.1.1",
			wantErr:             true,
		},
		{
			dnsResolverOverride: "udp://1.1.1.1:4a",
			wantErr:             true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.dnsResolverOverride, func(t *testing.T) {
			network, addr, err := ParseOverrideAddress(tt.dnsResolverOverride)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOverrideAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			assert.Equal(t, tt.wantNetwork, network, "network")
			assert.Equal(t, tt.wantAddr, addr, "addr")
		})
	}
}

func createFakeDNSServer(t *testing.T, dataA, dataAAAA map[string]string, useTCP bool, pause time.Duration, callCount *int, callCountMu *sync.Mutex) *dns.Server {
	t.Helper()

	dnsServer := &dns.Server{}
	if useTCP {
		ln, err := net.Listen("tcp", "localhost:")
		if err != nil {
			t.Fatalf("Error creating tcp listener: %v", err)
		}
		dnsServer.Listener = ln
	} else {
		packetConn, err := net.ListenPacket("udp", "localhost:")
		if err != nil {
			t.Fatalf("Error creating packet conn: %v", err)
		}
		dnsServer.PacketConn = packetConn
	}

	dnsServer.Handler = dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		start := time.Now()
		t.Logf("FakeDNSServer: (request id=%d) Received request", r.Id)

		callCountMu.Lock()
		(*callCount)++
		callCountMu.Unlock()

		m := new(dns.Msg)
		m.SetReply(r)

		var answers []dns.RR
		for _, q := range r.Question {
			switch q.Qtype {
			case dns.TypeA:
				if ip := dataA[q.Name]; ip != "" {
					answers = append(answers, &dns.A{
						Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
						A:   net.ParseIP(ip),
					})
				}
			case dns.TypeAAAA:
				if ip := dataAAAA[q.Name]; ip != "" {
					answers = append(answers, &dns.AAAA{
						Hdr:  dns.RR_Header{Name: q.Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 60},
						AAAA: net.ParseIP(ip),
					})
				}
			}
		}
		if len(answers) == 0 {
			m.SetRcode(r, dns.RcodeNameError)
		}
		m.Answer = answers
		t.Logf("FakeDNSServer: (request id=%d) Ready with response after: %v", r.Id, time.Since(start))
		// Pause to simulate a real DNS server.
		time.Sleep(pause)
		w.WriteMsg(m)
		t.Logf("FakeDNSServer: (request id=%d) Sent response after: %v", r.Id, time.Since(start))
	})
	go dnsServer.ActivateAndServe()
	return dnsServer
}

func TestNewWithOverrideResolver(t *testing.T) {
	dataA := map[string]string{
		"hostA.example.com.": "1.2.3.4",
	}
	dataAAAA := map[string]string{
		"hostA.example.com.": "2001:db8::1",
	}

	callCount := 0
	callCountMu := sync.Mutex{}
	defaultPause := 2 * time.Millisecond

	dnsServer := createFakeDNSServer(t, dataA, dataAAAA, false, defaultPause, &callCount, &callCountMu)
	udpAdrr := dnsServer.PacketConn.LocalAddr().String()
	t.Logf("DNS UDP server started at: %s", udpAdrr)
	defer dnsServer.Shutdown()

	dnsTCPServer := createFakeDNSServer(t, dataA, dataAAAA, true, defaultPause, &callCount, &callCountMu)
	tcpAddr := dnsTCPServer.Listener.Addr().String()
	t.Logf("DNS TCP server started at: %s", tcpAddr)
	defer dnsTCPServer.Shutdown()

	dnsTCPServerTimeout := createFakeDNSServer(t, dataA, dataAAAA, true, 100*time.Millisecond, &callCount, &callCountMu)
	tcpAddrTimeout := dnsTCPServerTimeout.Listener.Addr().String()
	t.Logf("DNS TCP server with timeout started at: %s", tcpAddrTimeout)
	defer dnsTCPServerTimeout.Shutdown()

	tests := []struct {
		name                string
		dnsResolverOverride string
		resolveTimeout      time.Duration
		runCount            int
		wantMinCount        int
		wantIP              string
		wantIP6             string
		wantParseErr        bool
		wantResolveErr      bool
	}{
		{
			name:                "valid-udp",
			dnsResolverOverride: udpAdrr,
			runCount:            10,
			wantMinCount:        10,
			wantIP:              "1.2.3.4",
			wantIP6:             "2001:db8::1",
		},
		{
			name:                "valid-tcp",
			dnsResolverOverride: "tcp://" + tcpAddr,
			runCount:            10,
			wantMinCount:        20,
			wantIP:              "1.2.3.4",
			wantIP6:             "2001:db8::1",
		},
		{
			name:                "timeout",
			dnsResolverOverride: "tcp://" + tcpAddrTimeout,
			resolveTimeout:      20 * time.Millisecond,
			runCount:            10,
			wantMinCount:        30,
			wantResolveErr:      true,
			wantIP:              "<nil>",
			wantIP6:             "<nil>",
		},
		{
			name:                "error",
			dnsResolverOverride: "mars://" + tcpAddr,
			wantParseErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network, address, err := ParseOverrideAddress(tt.dnsResolverOverride)
			if (err != nil) != tt.wantParseErr {
				t.Errorf("ParseOverrideAddress() error = %v, wantErr %v", err, tt.wantParseErr)
				return
			}
			if err != nil {
				return
			}

			if tt.resolveTimeout == 0 {
				tt.resolveTimeout = 100 * defaultPause
			}
			r := New(WithTTL(0), WithDNSServer(network, address), WithResolveTimeout(tt.resolveTimeout))

			if runtime.GOOS == "windows" {
				tt.runCount = tt.runCount / 5
			}
			for i := 0; i < tt.runCount; i++ {
				t.Run(fmt.Sprintf("run-%d", i), func(t *testing.T) {
					got, err := r.Resolve("hostA.example.com.", 4)
					if (err != nil) != tt.wantResolveErr {
						t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantParseErr)
						return
					}
					assert.Equal(t, tt.wantIP, got.String(), "ip")

					got, err = r.Resolve("hostA.example.com.", 6)
					if (err != nil) != tt.wantResolveErr {
						t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantParseErr)
						return
					}
					assert.Equal(t, tt.wantIP6, got.String(), "ip6")
				})
				time.Sleep(defaultPause + 3*time.Millisecond)
			}
			callCountMu.Lock()
			defer callCountMu.Unlock()
			if runtime.GOOS == "windows" {
				tt.wantMinCount = tt.wantMinCount / 5
			}
			assert.GreaterOrEqual(t, callCount, tt.wantMinCount, "callCount")
		})
	}
}

func TestNew(t *testing.T) {
	b := &resolveBackendWithTracking{}
	tests := []struct {
		name            string
		ttl             time.Duration
		maxTTL          time.Duration
		resolveFunc     func(context.Context, string, string) ([]net.IP, error)
		wantTTL         time.Duration
		wantMaxTTL      time.Duration
		wantResolveFunc func(context.Context, string, string) ([]net.IP, error)
		wantTimeout     time.Duration
	}{
		{
			name:        "default",
			wantTTL:     defaultMaxAge,
			wantMaxTTL:  defaultMaxAge,
			wantTimeout: defaultResolveTimeout,
		},
		{
			name:        "set ttl",
			ttl:         600 * time.Second,
			wantTTL:     600 * time.Second,
			wantMaxTTL:  600 * time.Second,
			wantTimeout: defaultResolveTimeout,
		},
		{
			name:        "max ttl < ttl",
			ttl:         600 * time.Second,
			maxTTL:      200 * time.Second,
			wantTTL:     600 * time.Second,
			wantMaxTTL:  600 * time.Second,
			wantTimeout: defaultResolveTimeout,
		},
		{
			name:        "max ttl > ttl",
			ttl:         600 * time.Second,
			maxTTL:      3600 * time.Second,
			wantTTL:     600 * time.Second,
			wantMaxTTL:  3600 * time.Second,
			wantTimeout: defaultResolveTimeout,
		},
		{
			name:        "timeout 10",
			ttl:         600 * time.Second,
			maxTTL:      3600 * time.Second,
			wantTTL:     600 * time.Second,
			wantMaxTTL:  3600 * time.Second,
			wantTimeout: 10 * time.Second,
		},
		{
			name:            "set resolve func",
			resolveFunc:     b.resolve,
			wantResolveFunc: b.resolve,
			wantTTL:         defaultMaxAge,
			wantMaxTTL:      defaultMaxAge,
			wantTimeout:     defaultResolveTimeout,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.ttl != 0 {
				opts = append(opts, WithTTL(tt.ttl))
			}
			if tt.maxTTL != 0 {
				opts = append(opts, WithMaxTTL(tt.maxTTL))
			}

			r := New(opts...)
			assert.Equal(t, tt.wantTTL, r.ttl, "ttl")
			assert.Equal(t, tt.wantMaxTTL, r.maxCacheAge, "maxTTL")
		})
	}
}

func TestGetResolverOptions(t *testing.T) {
	tests := []struct {
		name         string
		dnsServer    string
		dnsOptions   *targetspb.DNSOptions
		wantResolver *resolverImpl
		wantErr      bool
	}{
		{
			name:       "no options",
			dnsOptions: &targetspb.DNSOptions{},
			wantResolver: &resolverImpl{
				ttl:            defaultMaxAge,
				maxCacheAge:    defaultMaxAge,
				resolveTimeout: defaultResolveTimeout,
			},
		},
		{
			name: "dns TTL",
			dnsOptions: &targetspb.DNSOptions{
				TtlSec: proto.Int32(600),
			},
			wantResolver: &resolverImpl{
				ttl:            600 * time.Second,
				maxCacheAge:    600 * time.Second,
				resolveTimeout: defaultResolveTimeout,
			},
		},
		{
			name: "dns TTL & max TTL",
			dnsOptions: &targetspb.DNSOptions{
				TtlSec:         proto.Int32(600),
				MaxCacheAgeSec: proto.Int32(1200),
			},
			wantResolver: &resolverImpl{
				ttl:            600 * time.Second,
				maxCacheAge:    1200 * time.Second,
				resolveTimeout: defaultResolveTimeout,
			},
		},
		{
			name: "0 timeout fixed",
			dnsOptions: &targetspb.DNSOptions{
				BackendTimeoutMsec: proto.Int32(0),
			},
			wantResolver: &resolverImpl{
				ttl:            defaultMaxAge,
				maxCacheAge:    defaultMaxAge,
				resolveTimeout: defaultResolveTimeout,
			},
		},
		{
			name: "dns TTL & max TTL & dns server",
			dnsOptions: &targetspb.DNSOptions{
				TtlSec:         proto.Int32(600),
				MaxCacheAgeSec: proto.Int32(1200),
				Server:         proto.String("8.8.8.8"),
			},
			wantResolver: &resolverImpl{
				ttl:            600 * time.Second,
				maxCacheAge:    1200 * time.Second,
				resolveTimeout: defaultResolveTimeout,
				backendServer:  "8.8.8.8:53",
			},
		},
		{
			name: "dns TTL & max TTL & dns server & resolve timeout",
			dnsOptions: &targetspb.DNSOptions{
				TtlSec:             proto.Int32(600),
				MaxCacheAgeSec:     proto.Int32(1200),
				Server:             proto.String("8.8.8.8"),
				BackendTimeoutMsec: proto.Int32(5000),
			},
			wantResolver: &resolverImpl{
				ttl:            600 * time.Second,
				maxCacheAge:    1200 * time.Second,
				resolveTimeout: 5 * time.Second,
				backendServer:  "8.8.8.8:53",
			},
		},
		{
			name:      "server-address-from-targets-proto",
			dnsServer: "tcp://1.1.1.1",
			wantResolver: &resolverImpl{
				ttl:            defaultMaxAge,
				maxCacheAge:    defaultMaxAge,
				resolveTimeout: defaultResolveTimeout,
				backendServer:  "1.1.1.1:53",
				backendNetwork: "tcp",
			},
		},
		{
			name:      "error-server-address-multiple",
			dnsServer: "udp://1.1.1.1",
			dnsOptions: &targetspb.DNSOptions{
				Server: proto.String("udp://1.1.1.1"),
			},
			wantErr: true,
		},
		{
			name: "error-server-address-bad",
			dnsOptions: &targetspb.DNSOptions{
				Server: proto.String("mars://1.1.1.1"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetsDef := &targetspb.TargetsDef{
				DnsServer:  &tt.dnsServer,
				DnsOptions: tt.dnsOptions,
			}
			got, err := GetResolverOptions(targetsDef, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("getResolverOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			res := New(got...)
			assert.Equal(t, tt.wantResolver.ttl, res.ttl, "ttl")
			assert.Equal(t, tt.wantResolver.maxCacheAge, res.maxCacheAge, "maxCacheAge")
			assert.Equal(t, tt.wantResolver.resolveTimeout, res.resolveTimeout, "resolveTimeout")
			assert.Equal(t, tt.wantResolver.backendServer, res.backendServer, "backendServer")
			assert.Equal(t, tt.wantResolver.backendNetwork, res.backendNetwork, "backendNetwork")
		})
	}
}
