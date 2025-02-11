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

// Package resolver provides a caching, non-blocking DNS resolver. All requests
// for cached resources are returned immediately and if cache has expired, an
// offline goroutine is fired to update it.
package resolver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/miekg/dns"
)

// The max age and the timeout for resolving a target.
const (
	defaultMaxAge         = time.Duration(targetspb.Default_DNSOptions_TtlSec) * time.Second
	defaultResolveTimeout = time.Duration(targetspb.Default_DNSOptions_BackendTimeoutMsec) * time.Millisecond
)

type ipVersion uint8

const (
	IP  ipVersion = 0
	IP4 ipVersion = 4
	IP6 ipVersion = 6
)

func (n ipVersion) Network() string {
	return map[ipVersion]string{IP: "ip", IP4: "ip4", IP6: "ip6"}[n]
}

var defaultResolverFunc = net.DefaultResolver.LookupIP

type Resolver interface {
	Resolve(name string, ipVer int) (net.IP, error)
}

// resolverImpl provides an asynchronous caching DNS resolver.
type resolverImpl struct {
	cache          map[cacheRecordKey]*cacheRecord
	mu             sync.Mutex
	ttl            time.Duration
	maxCacheAge    time.Duration
	resolveFunc    func(context.Context, string, string) ([]net.IP, error)
	resolveTimeout time.Duration
	l              *logger.Logger

	// Backend server and networkl; these variables are used only for testing.
	backendServer  string
	backendNetwork string
}

// resolveOrTimeout is simply a context less wrapper around resolve.
func (r *resolverImpl) resolveOrTimeout(name string, version ipVersion) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.resolveTimeout)
	defer cancel()

	startTime := time.Now()
	ips, err := r.resolveFunc(ctx, version.Network(), name)
	if err != nil {
		attrs := []slog.Attr{
			slog.String("name", name),
			slog.Duration("time_elapsed", time.Since(startTime)),
			slog.Duration("timeout", r.resolveTimeout),
			slog.String("backend_server", r.backendServer),
			slog.String("backend_network", r.backendNetwork),
		}
		r.l.WarningAttrs("Resolve Error: "+err.Error(), attrs...)
	}
	return ips, err
}

// resolveWithMaxAge returns IP address for a name, issuing an update call for
// the cache record if it's older than the argument maxAge.
// refreshedCh channel is primarily used for testing. Method pushes true to
// refreshedCh channel once and if the value is refreshed, or false, if it
// doesn't need refreshing.
func (r *resolverImpl) resolveWithMaxAge(name string, ipVer int, maxAge time.Duration, refreshedCh chan<- bool) (net.IP, error) {
	cr := r.getCacheRecord(name, ipVersion(ipVer))
	cr.refreshIfRequired(r.resolveOrTimeout, maxAge, refreshedCh)
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	if cr.err != nil && cr.ip != nil && time.Since(cr.lastUpdatedAt) < r.maxCacheAge {
		r.l.Warningf("failed to resolve %s: %v, returning cached IP: %s", name, cr.err, cr.ip.String())
		return cr.ip, nil
	}

	return cr.ip, cr.err
}

// Resolve returns IP address for a name.
// Issues an update call for the cache record if it's older than defaultMaxAge.
func (r *resolverImpl) Resolve(name string, ipVer int) (net.IP, error) {
	return r.resolveWithMaxAge(name, ipVer, r.ttl, nil)
}

func ParseOverrideAddress(dnsResolverOverride string) (string, string, error) {
	// dnsResolverOverride can be in the format "network://ip:port" or "ip:port",
	// or just "ip". If network is not specified, we use Go's default.
	var network, addr string
	addrParts := strings.Split(dnsResolverOverride, "://")
	if len(addrParts) == 2 {
		network, addr = addrParts[0], addrParts[1]
	} else {
		addr = dnsResolverOverride
	}

	validNetworks := []string{"", "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6", "tls"}
	if !slices.Contains(validNetworks, network) {
		return "", "", fmt.Errorf("invalid network: %s", network)
	}

	port := "53"

	// use default dot port and change to tcp-tls for miekg/dns client
	if network == "tls" {
		network = "tcp-tls"
		port = "853"
	}

	// Check if address includes a port number. If it doesn't parse as an IP
	// address, but includes a :, then it might contain a port number
	if ip := net.ParseIP(addr); ip == nil {
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			addr, port = addr[:idx], addr[idx+1:]
		}
	}

	// Remaining part of the address should be an IP address
	if net.ParseIP(addr) == nil {
		return "", "", fmt.Errorf("invalid IP address: %s", addr)
	}

	if _, err := strconv.Atoi(port); err != nil {
		return "", "", fmt.Errorf("invalid port number: %s", port)
	}

	return network, net.JoinHostPort(addr, port), nil
}

type Option func(*resolverImpl)

func WithResolveFunc(resolveFunc func(context.Context, string, string) ([]net.IP, error)) Option {
	return func(r *resolverImpl) {
		r.resolveFunc = resolveFunc
	}
}

func WithResolveTimeout(resolveTimeout time.Duration) Option {
	return func(r *resolverImpl) {
		r.resolveTimeout = resolveTimeout
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(r *resolverImpl) {
		r.ttl = ttl
	}
}

func WithMaxTTL(ttl time.Duration) Option {
	return func(r *resolverImpl) {
		r.maxCacheAge = ttl
	}
}

func WithDNSServer(serverNetworkOverride, serverAddressOverride string) Option {
	return func(r *resolverImpl) {
		r.backendServer = serverAddressOverride
		r.backendNetwork = serverNetworkOverride

		r.resolveFunc = func(ctx context.Context, network, host string) ([]net.IP, error) {
			fqdn := dns.Fqdn(host)
			dnsClient := &dns.Client{
				Net: r.backendNetwork,
			}
			// miekg/dns client doesn't honor the context timeout if it's
			// higher than the client's Timeout which by default is 2s.
			if deadline, ok := ctx.Deadline(); ok {
				dnsClient.Timeout = time.Until(deadline)
			}
			conn, err := dnsClient.DialContext(ctx, r.backendServer)
			if err != nil {
				return nil, err
			}
			defer conn.Close()

			// This immitates the behavior of net.DefaultResolver.LookupIP,
			// which queries both A and AAAA records for a given host if network
			// is set to "ip" (no IP version is given).
			qTypes := []uint16{dns.TypeA, dns.TypeAAAA}
			if network == IP4.Network() {
				qTypes = []uint16{dns.TypeA}
			} else if network == IP6.Network() {
				qTypes = []uint16{dns.TypeAAAA}
			}

			// Following is more efficient than LookupIP, as we return as soon as we
			// find an IP address.
			var ip net.IP
			for _, qType := range qTypes {
				msg := new(dns.Msg).SetQuestion(fqdn, qType)
				resp, _, err := dnsClient.ExchangeWithConnContext(ctx, msg, conn)
				if err != nil {
					return nil, err
				}
				for _, ans := range resp.Answer {
					switch qType {
					case dns.TypeA:
						if a, ok := ans.(*dns.A); ok {
							ip = a.A
						}
					case dns.TypeAAAA:
						if aaaa, ok := ans.(*dns.AAAA); ok {
							ip = aaaa.AAAA
						}
					}
					if ip != nil {
						return []net.IP{ip}, nil
					}
				}
			}
			return nil, fmt.Errorf("no IPs found for %s", host)
		}
	}
}

func WithLogger(l *logger.Logger) Option {
	return func(r *resolverImpl) {
		r.l = l
	}
}

func GetResolverOptions(targetsDef *targetspb.TargetsDef, l *logger.Logger) ([]Option, error) {
	dopts := targetsDef.GetDnsOptions()

	server := dopts.GetServer()
	if targetsDef.GetDnsServer() != "" {
		l.Warningf("dns_server is now deprecated. please use dns_options.server instead")

		if server != "" {
			return nil, fmt.Errorf("dns_server and dns_options.server are mutually exclusive")
		} else {
			server = targetsDef.GetDnsServer()
		}
	}

	var opts []Option

	ttlSec := dopts.GetTtlSec()
	if ttlSec != int32(defaultMaxAge.Seconds()) {
		opts = append(opts, WithTTL(time.Duration(ttlSec)*time.Second))
	}

	maxCacheAge := dopts.GetMaxCacheAgeSec()
	if maxCacheAge != 0 {
		if maxCacheAge < ttlSec {
			return nil, fmt.Errorf("max_cache_age (%d) must be >= ttl_sec (%d)", maxCacheAge, ttlSec)
		}
		opts = append(opts, WithMaxTTL(time.Duration(maxCacheAge)*time.Second))
	}

	if server != "" {
		l.Infof("Overriding default resolver with: %s", server)
		network, address, err := ParseOverrideAddress(server)
		if err != nil {
			return nil, err
		}
		opts = append(opts, WithDNSServer(network, address))
	}

	if dopts.GetBackendTimeoutMsec() != int32(defaultResolveTimeout.Milliseconds()) {
		opts = append(opts, WithResolveTimeout(time.Duration(dopts.GetBackendTimeoutMsec())*time.Millisecond))
	}

	return opts, nil
}

// New returns a new Resolver.
func New(opts ...Option) *resolverImpl {
	r := &resolverImpl{
		cache:          make(map[cacheRecordKey]*cacheRecord),
		resolveFunc:    defaultResolverFunc,
		ttl:            defaultMaxAge,
		resolveTimeout: defaultResolveTimeout,
		l:              logger.NewWithAttrs(slog.String("component", "global-resolver")),
	}

	for _, opt := range opts {
		opt(r)
	}

	// maxTTL cannot be less than ttl
	if r.maxCacheAge < r.ttl {
		r.maxCacheAge = r.ttl
	}

	// Make sure that resolveTimeout is never set to 0.
	if r.resolveTimeout == 0 {
		r.resolveTimeout = defaultResolveTimeout
	}

	return r
}
