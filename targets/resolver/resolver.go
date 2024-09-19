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
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
)

// The max age and the timeout for resolving a target.
const (
	defaultMaxAge         = 5 * time.Minute
	defaultResolveTimeout = 30 * time.Second
)

var defaultResolverFunc = func(ctx context.Context, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}

type cacheRecord struct {
	ip4              net.IP
	ip6              net.IP
	lastUpdatedAt    time.Time
	err              error
	mu               sync.RWMutex
	updateInProgress bool
	callInit         sync.Once
}

type Resolver interface {
	Resolve(name string, ipVer int) (net.IP, error)
}

// resolverImpl provides an asynchronous caching DNS resolver.
type resolverImpl struct {
	cache          map[string]*cacheRecord
	mu             sync.Mutex
	ttl            time.Duration
	maxCacheAge    time.Duration
	resolve        func(context.Context, string) ([]net.IP, error) // used for testing
	resolveTimeout time.Duration
	l              *logger.Logger

	// Backend server and networkl; these variables are used only for testing.
	backendServer  string
	backendNetwork string
}

// ipVersion tells if an IP address is IPv4 or IPv6.
func ipVersion(ip net.IP) int {
	if len(ip.To4()) == net.IPv4len {
		return 4
	}
	if len(ip) == net.IPv6len {
		return 6
	}
	return 0
}

// resolveOrTimeout is simply a context less wrapper around resolve.
func (r *resolverImpl) resolveOrTimeout(name string) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.resolveTimeout)
	defer cancel()

	return r.resolve(ctx, name)
}

// Resolve returns IP address for a name.
// Issues an update call for the cache record if it's older than defaultMaxAge.
func (r *resolverImpl) Resolve(name string, ipVer int) (net.IP, error) {
	return r.resolveWithMaxAge(name, ipVer, r.ttl, nil)
}

// getCacheRecord returns the cache record for the target.
// It must be kept light, as it blocks the main mutex of the map.
func (r *resolverImpl) getCacheRecord(name string) *cacheRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	cr := r.cache[name]
	// This will happen only once for a given name.
	if cr == nil {
		cr = &cacheRecord{
			err: errors.New("cache record not initialized yet"),
		}
		r.cache[name] = cr
	}
	return cr
}

// resolveWithMaxAge returns IP address for a name, issuing an update call for
// the cache record if it's older than the argument maxAge.
// refreshedCh channel is primarily used for testing. Method pushes true to
// refreshedCh channel once and if the value is refreshed, or false, if it
// doesn't need refreshing.
func (r *resolverImpl) resolveWithMaxAge(name string, ipVer int, maxAge time.Duration, refreshedCh chan<- bool) (net.IP, error) {
	cr := r.getCacheRecord(name)
	cr.refreshIfRequired(name, r.resolveOrTimeout, maxAge, refreshedCh)
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	var ip net.IP

	switch ipVer {
	case 0:
		if cr.ip4 != nil {
			ip = cr.ip4
		} else if cr.ip6 != nil {
			ip = cr.ip6
		}
	case 4:
		ip = cr.ip4
	case 6:
		ip = cr.ip6
	default:
		return nil, fmt.Errorf("unknown IP version: %d", ipVer)
	}

	if ip == nil && cr.err == nil {
		return nil, fmt.Errorf("found no IP%d IP for %s", ipVer, name)
	}

	if cr.err != nil && ip != nil && time.Since(cr.lastUpdatedAt) < r.maxCacheAge {
		r.l.Warningf("failed to resolve %s: %v, returning cached IP: %s", name, cr.err, ip.String())
		return ip, nil
	}
	return ip, cr.err
}

// refresh refreshes the cacheRecord by making a call to the provided "resolve" function.
func (cr *cacheRecord) refresh(name string, resolve func(string) ([]net.IP, error), refreshed chan<- bool) {
	// Note that we call backend's resolve outside of the mutex locks and take the lock again
	// to update the cache record once we have the results from the backend.
	ips, err := resolve(name)

	cr.mu.Lock()
	defer cr.mu.Unlock()
	if refreshed != nil {
		refreshed <- true
	}
	cr.err = err
	cr.updateInProgress = false
	// If we have an error, we don't update the cache record so that callers
	// can use cached IP addresses if they want.
	if err != nil {
		return
	}
	cr.lastUpdatedAt = time.Now()
	cr.ip4 = nil
	cr.ip6 = nil
	for _, ip := range ips {
		switch ipVersion(ip) {
		case 4:
			cr.ip4 = ip
		case 6:
			cr.ip6 = ip
		}
	}
}

func (cr *cacheRecord) shouldUpdateNow(maxAge time.Duration) bool {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return !cr.updateInProgress && (time.Since(cr.lastUpdatedAt) >= maxAge || cr.err != nil)
}

// refreshIfRequired does most of the work. Overall goal is to minimize the
// lock period of the cache record. To that end, if the cache record needs
// updating, we do that with the mutex unlocked.
//
// If cache record is new, blocks until it's resolved for the first time.
// If cache record needs updating, kicks off refresh asynchronously.
// If cache record is already being updated or fresh enough, returns immediately.
func (cr *cacheRecord) refreshIfRequired(name string, resolve func(string) ([]net.IP, error), maxAge time.Duration, refreshedCh chan<- bool) {
	cr.callInit.Do(func() { cr.refresh(name, resolve, refreshedCh) })

	// Cache record is old and no update in progress, issue a request to update.
	if cr.shouldUpdateNow(maxAge) {
		cr.mu.Lock()
		cr.updateInProgress = true
		cr.mu.Unlock()
		go cr.refresh(name, resolve, refreshedCh)
	} else if refreshedCh != nil {
		refreshedCh <- false
	}
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

	validNetworks := []string{"", "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6"}
	if !slices.Contains(validNetworks, network) {
		return "", "", fmt.Errorf("invalid network: %s", network)
	}

	port := "53"
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

func WithResolveFunc(resolveFunc func(context.Context, string) ([]net.IP, error)) Option {
	return func(r *resolverImpl) {
		r.resolve = resolveFunc
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
	netResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			if serverNetworkOverride != "" {
				network = serverNetworkOverride
			}
			// Note: we ignore the address in the argument
			return d.DialContext(ctx, network, serverAddressOverride)
		},
	}

	return func(r *resolverImpl) {
		r.backendServer = serverAddressOverride
		r.backendNetwork = serverNetworkOverride
		r.resolve = func(ctx context.Context, host string) ([]net.IP, error) {
			return netResolver.LookupIP(ctx, "ip", host)
		}
	}
}

func GetResolverOptions(targetsDef *targetspb.TargetsDef, l *logger.Logger) ([]Option, error) {
	dopts := targetsDef.GetDnsOptions()

	server, ttlSec, maxCacheAge, timeout := dopts.GetServer(), dopts.GetTtlSec(), dopts.GetMaxCacheAgeSec(), dopts.GetBackendTimeoutMsec()
	if targetsDef.GetDnsServer() != "" {
		l.Warningf("dns_server is now deprecated. please use dns_options.server instead")

		if server != "" {
			return nil, fmt.Errorf("dns_server and dns_options.server are mutually exclusive")
		} else {
			server = targetsDef.GetDnsServer()
		}
	}

	var opts []Option

	if ttlSec != targetspb.Default_DNSOptions_TtlSec {
		opts = append(opts, WithTTL(time.Duration(ttlSec)*time.Second))
	}

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

	if timeout != targetspb.Default_DNSOptions_BackendTimeoutMsec {
		opts = append(opts, WithResolveTimeout(time.Duration(timeout)*time.Millisecond))
	}

	return opts, nil
}

// New returns a new Resolver.
func New(opts ...Option) *resolverImpl {
	r := &resolverImpl{
		cache:          make(map[string]*cacheRecord),
		resolve:        defaultResolverFunc,
		ttl:            defaultMaxAge,
		resolveTimeout: defaultResolveTimeout,
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
