// Copyright 2025 The Cloudprober Authors.
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

// Health checks lister for Consul.
package consul

import (
	"fmt"
	"net/url"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

// healthChecksLister lists Consul health checks.
type healthChecksLister struct {
	config *configpb.ProviderConfig
	client *client
	cache  *resourceCache
	l      *logger.Logger
}

// HealthCheckSupportedFilters defines the filters supported by the health checks lister.
var HealthCheckSupportedFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	// Supported regex filters:
	// - name: check name
	// - service: service name
	// - node: node name
	// - status: check status
	RegexFilterKeys: []string{"name", "service", "node", "status"},
	LabelsFilter:    true,
}

// newHealthChecksLister creates a new health checks lister.
func newHealthChecksLister(c *configpb.ProviderConfig, cl *client, l *logger.Logger) (*healthChecksLister, error) {
	hcl := &healthChecksLister{
		config: c,
		client: cl,
		cache:  newResourceCache(),
		l:      l,
	}

	// Start background refresh goroutine
	go hcl.refreshLoop()

	return hcl, nil
}

// refreshLoop periodically refreshes the health checks from Consul.
func (hcl *healthChecksLister) refreshLoop() {
	reEvalInterval := time.Duration(hcl.config.GetReEvalSec()) * time.Second

	// Initial refresh
	if err := hcl.refresh(); err != nil {
		hcl.l.Errorf("consul.health_checks: initial refresh failed: %v", err)
	}

	// Periodic refresh with ticker
	ticker := time.NewTicker(reEvalInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := hcl.refresh(); err != nil {
			hcl.l.Errorf("consul.health_checks: refresh failed: %v", err)
		}
	}
}

// refresh fetches the latest health checks from Consul and updates the cache.
// If a service name is configured:
//
//	GET /v1/health/checks/{service} → []consulCheck
//
// Otherwise:
//
//	GET /v1/health/state/any → []consulCheck
func (hcl *healthChecksLister) refresh() error {
	hcl.l.Debugf("consul.health_checks: refreshing health checks")

	params := url.Values{}
	if hcl.config.GetDatacenter() != "" {
		params.Set("dc", hcl.config.GetDatacenter())
	}

	var checks []consulCheck
	var err error

	// If service name is specified, get checks for that service
	if hcl.config.HealthChecks != nil && hcl.config.HealthChecks.GetServiceName() != "" {
		serviceName := hcl.config.HealthChecks.GetServiceName()
		err = hcl.client.get(fmt.Sprintf("/v1/health/checks/%s", serviceName), params, &checks)
		if err != nil {
			return fmt.Errorf("failed to list health checks for service %s: %v", serviceName, err)
		}
	} else {
		// Get all health checks via the state endpoint
		err = hcl.client.get("/v1/health/state/any", params, &checks)
		if err != nil {
			return fmt.Errorf("failed to list health checks: %v", err)
		}
	}

	hcl.l.Debugf("consul.health_checks: found %d health checks", len(checks))

	// Clear old cache
	hcl.cache.clear()

	now := time.Now().Unix()

	for _, check := range checks {
		// Apply status filter if configured
		if hcl.config.HealthChecks != nil && len(hcl.config.HealthChecks.StatusFilter) > 0 {
			if !containsString(hcl.config.HealthChecks.StatusFilter, check.Status) {
				continue
			}
		}

		// Create unique key for this health check
		key := fmt.Sprintf("%s-%s-%s", check.Node, check.ServiceName, check.CheckID)

		// Note: Health checks don't have IP/port directly, we use node name
		data := &serviceData{
			name:        check.Name,
			address:     check.Node, // Use node name as address
			nodeName:    check.Node,
			health:      check.Status,
			lastUpdated: now,
			meta: map[string]string{
				"check_id":     check.CheckID,
				"service_name": check.ServiceName,
				"status":       check.Status,
				"notes":        check.Notes,
				"output":       check.Output,
			},
		}

		hcl.cache.set(key, data)
	}

	hcl.l.Infof("consul.health_checks: cached %d health checks", len(hcl.cache.getAll()))
	return nil
}

// listResources implements the lister interface for health checks.
func (hcl *healthChecksLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	// Parse filters
	allFilters, err := filter.ParseFilters(req.GetFilter(), HealthCheckSupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse filters: %v", err)
	}

	// Get all cached health checks
	allChecks := hcl.cache.getAll()

	var resources []*pb.Resource

	for key, check := range allChecks {
		// Apply name filter
		if nameFilter := allFilters.RegexFilters["name"]; nameFilter != nil {
			if !nameFilter.Match(check.name, hcl.l) {
				continue
			}
		}

		// Apply service filter
		if serviceFilter := allFilters.RegexFilters["service"]; serviceFilter != nil {
			serviceName := check.meta["service_name"]
			if !serviceFilter.Match(serviceName, hcl.l) {
				continue
			}
		}

		// Apply node filter
		if nodeFilter := allFilters.RegexFilters["node"]; nodeFilter != nil {
			if !nodeFilter.Match(check.nodeName, hcl.l) {
				continue
			}
		}

		// Apply status filter
		if statusFilter := allFilters.RegexFilters["status"]; statusFilter != nil {
			if !statusFilter.Match(check.health, hcl.l) {
				continue
			}
		}

		// Build labels map
		labels := make(map[string]string)
		labels["check_name"] = check.name
		labels["node"] = check.nodeName
		labels["status"] = check.health

		// Add metadata as labels
		for k, v := range check.meta {
			labels[k] = v
		}

		// Apply labels filter
		if allFilters.LabelsFilter != nil {
			if !allFilters.LabelsFilter.Match(labels, hcl.l) {
				continue
			}
		}

		// Create resource
		resource := &pb.Resource{
			Name:        proto.String(key),
			Ip:          proto.String(check.address),
			Labels:      labels,
			Id:          proto.String(key),
			LastUpdated: proto.Int64(check.lastUpdated),
		}

		resources = append(resources, resource)
	}

	hcl.l.Debugf("consul.health_checks: returning %d resources after filtering", len(resources))
	return resources, nil
}
