// Copyright 2019 The Cloudprober Authors.
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

// Package endpoint provides the type Endpoint, to be used with the
// targets.Targets interface.
package endpoint

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/common/iputils"
	endpointpb "github.com/cloudprober/cloudprober/targets/endpoint/proto"
)

// Endpoint represents a target and associated parameters.
type Endpoint struct {
	Name        string
	Labels      map[string]string
	LastUpdated time.Time
	Port        int
	IP          net.IP
	Info        string
}

// Clone creates a deep copy of an Endpoint.
func (ep *Endpoint) Clone() *Endpoint {
	epCopy := *ep
	epCopy.Labels = make(map[string]string, len(ep.Labels))
	for k, v := range ep.Labels {
		epCopy.Labels[k] = v
	}
	return &epCopy
}

type keyOptions struct {
	ignoreLabels []string
}

type KeyOption func(*keyOptions) *keyOptions

// WithIgnoreLabels specifies a list of labels that should not be included in
// the key computation.
func WithIgnoreLabels(ignoreLabels ...string) KeyOption {
	return func(ro *keyOptions) *keyOptions {
		ro.ignoreLabels = ignoreLabels
		return ro
	}
}

// Key returns a string key that uniquely identifies that endpoint.
// Endpoint key consists of endpoint name, port and labels.
func (ep *Endpoint) Key(opts ...KeyOption) string {
	ro := &keyOptions{}
	for _, opt := range opts {
		ro = opt(ro)
	}

	labelSlice := make([]string, 0, len(ep.Labels))
	for k, v := range ep.Labels {
		if ro.ignoreLabels != nil && slices.Contains(ro.ignoreLabels, k) {
			continue
		}
		labelSlice = append(labelSlice, k+":"+v)
	}
	sort.Strings(labelSlice)

	ip := ""
	if ep.IP != nil {
		ip = ep.IP.String()
	}
	return strings.Join(append([]string{ep.Name, ip, strconv.Itoa(ep.Port)}, labelSlice...), "_")
}

// Lister should implement the ListEndpoints method.
type Lister interface {
	// ListEndpoints returns list of endpoints (name, port tupples).
	ListEndpoints() []Endpoint
}

type Resolver interface {
	// Resolve, given a target and IP Version will return the IP address for that
	// target.
	Resolve(name string, ipVer int) (net.IP, error)
}

// EndpointsFromNames is convenience function to build a list of endpoints
// from only names. It leaves the Port field in Endpoint unset and initializes
// Labels field to an empty map.
func EndpointsFromNames(names []string) []Endpoint {
	result := make([]Endpoint, len(names))
	for i, name := range names {
		result[i].Name = name
		result[i].Labels = make(map[string]string)
	}
	return result
}

// Dst return the "dst" label for the endpoint
func (ep *Endpoint) Dst() string {
	if ep.Port == 0 {
		return ep.Name
	}
	return net.JoinHostPort(ep.Name, strconv.Itoa(ep.Port))
}

type resolverOptions struct {
	nameOverride string
}

type ResolverOption func(*resolverOptions)

func WithNameOverride(nameOverride string) ResolverOption {
	return func(ro *resolverOptions) {
		ro.nameOverride = nameOverride
	}
}

// Resolve resolves endpoint to an IP address. If endpoint has an embedded IP
// address it uses that, otherwise a global reolver is used.
func (ep *Endpoint) Resolve(ipVersion int, resolver Resolver, opts ...ResolverOption) (net.IP, error) {
	ro := &resolverOptions{}
	for _, opt := range opts {
		opt(ro)
	}

	if ep.IP != nil {
		if ipVersion == 0 || iputils.IPVersion(ep.IP) == ipVersion {
			return ep.IP, nil
		}

		return nil, fmt.Errorf("no IPv%d address (IP: %s) for %s", ipVersion, ep.IP.String(), ep.Name)
	}

	name := ep.Name
	if ro.nameOverride != "" {
		name = ro.nameOverride
	}
	return resolver.Resolve(name, ipVersion)
}

// NamesFromEndpoints is convenience function to build a list of names
// from endpoints.
func NamesFromEndpoints(endpoints []Endpoint) []string {
	result := make([]string, len(endpoints))
	for i, ep := range endpoints {
		result[i] = ep.Name
	}
	return result
}

func parseURL(s string) (scheme, host, path string, port int, err error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("invalid URL: %v", err)
	}

	scheme = u.Scheme
	host = u.Hostname()
	port, _ = strconv.Atoi(u.Port())
	path = "/"

	hostPath := strings.TrimPrefix(s, scheme+"://")
	if i := strings.Index(hostPath, "/"); i != -1 {
		path = hostPath[i:]
	}
	return scheme, host, path, port, nil
}

func FromProtoMessage(endpointspb []*endpointpb.Endpoint) ([]Endpoint, error) {
	var endpoints []Endpoint
	seen := make(map[string]bool)
	timestamp := time.Now()

	for _, pb := range endpointspb {
		ep := Endpoint{
			Name:        pb.GetName(),
			Labels:      pb.GetLabels(),
			IP:          net.ParseIP(pb.GetIp()),
			Port:        int(pb.GetPort()),
			LastUpdated: timestamp,
		}

		if pb.GetUrl() != "" {
			scheme, host, path, port, err := parseURL(pb.GetUrl())
			if err != nil {
				return nil, err
			}
			if ep.Labels == nil {
				ep.Labels = make(map[string]string)
			}
			ep.Labels["__cp_scheme__"] = scheme
			ep.Labels["__cp_host__"] = host
			ep.Labels["__cp_path__"] = path

			if ep.Port == 0 {
				ep.Port = port
			}
		}
		epKey := ep.Key()
		if seen[epKey] {
			return nil, fmt.Errorf("duplicate endpoint: %s", ep.Key())
		}
		seen[epKey] = true
		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}
