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

package consul

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

type servicesLister struct {
	config *configpb.ProviderConfig
	client *client
	cache  *resourceCache
	l      *logger.Logger
}

var SupportedFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	RegexFilterKeys: []string{"name", "node", "tag", "address"},
	LabelsFilter:    true,
}

func newServicesLister(c *configpb.ProviderConfig, cl *client, l *logger.Logger) (*servicesLister, error) {
	sl := &servicesLister{
		config: c,
		client: cl,
		cache:  newResourceCache(),
		l:      l,
	}

	go sl.refreshLoop()

	return sl, nil
}

func (sl *servicesLister) refreshLoop() {
	reEvalInterval := time.Duration(sl.config.GetReEvalSec()) * time.Second

	if err := sl.refresh(); err != nil {
		sl.l.Errorf("consul.services: initial refresh failed: %v", err)
	}

	ticker := time.NewTicker(reEvalInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := sl.refresh(); err != nil {
			sl.l.Errorf("consul.services: refresh failed: %v", err)
		}
	}
}

// refresh reloads the cache from /v1/catalog/services (name → tags) and then
// fetches each service's instances.
func (sl *servicesLister) refresh() error {
	sl.l.Debugf("consul.services: refreshing service list")

	var services map[string][]string
	if err := sl.client.get("/v1/catalog/services", nil, &services); err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	sl.l.Debugf("consul.services: found %d services", len(services))

	sl.cache.clear()

	for serviceName, tags := range services {
		if sl.config.Services != nil && sl.config.Services.GetNameFilter() != "" {
			matched, err := regexp.MatchString(sl.config.Services.GetNameFilter(), serviceName)
			if err != nil {
				sl.l.Warningf("consul.services: invalid name filter regex: %v", err)
			} else if !matched {
				continue
			}
		}

		if sl.config.Services != nil && len(sl.config.Services.TagFilter) > 0 {
			if !containsAllTags(tags, sl.config.Services.TagFilter) {
				continue
			}
		}

		if err := sl.refreshService(serviceName); err != nil {
			sl.l.Warningf("consul.services: failed to refresh service %s: %v", serviceName, err)
		}
	}

	sl.l.Infof("consul.services: cached %d service instances", len(sl.cache.getAll()))
	return nil
}

// refreshService caches the instances from /v1/health/service/{serviceName}.
func (sl *servicesLister) refreshService(serviceName string) error {
	params := url.Values{}
	if sl.config.GetDatacenter() != "" {
		params.Set("dc", sl.config.GetDatacenter())
	}

	var entries []consulHealthEntry
	if err := sl.client.get(fmt.Sprintf("/v1/health/service/%s", serviceName), params, &entries); err != nil {
		return fmt.Errorf("failed to get service health: %v", err)
	}

	now := time.Now().Unix()

	for _, entry := range entries {
		healthStatus := aggregateHealthStatus(entry.Checks)

		if sl.config.Services != nil && len(sl.config.Services.HealthStatus) > 0 {
			if !containsString(sl.config.Services.HealthStatus, healthStatus) {
				continue
			}
		}

		service := entry.Service
		node := entry.Node

		// Use service address if available; fall back to node address.
		address := service.Address
		if address == "" {
			address = node.Address
		}

		key := fmt.Sprintf("%s-%s-%s-%d", serviceName, node.Node, address, service.Port)

		data := &serviceData{
			name:        serviceName,
			address:     address,
			port:        service.Port,
			tags:        service.Tags,
			meta:        service.Meta,
			nodeName:    node.Node,
			nodeMeta:    node.Meta,
			health:      healthStatus,
			lastUpdated: now,
		}

		sl.cache.set(key, data)
	}

	return nil
}

func (sl *servicesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	allFilters, err := filter.ParseFilters(req.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse filters: %v", err)
	}

	allServices := sl.cache.getAll()

	var resources []*pb.Resource

	for key, svc := range allServices {
		if nameFilter := allFilters.RegexFilters["name"]; nameFilter != nil {
			if !nameFilter.Match(svc.name, sl.l) {
				continue
			}
		}

		if nodeFilter := allFilters.RegexFilters["node"]; nodeFilter != nil {
			if !nodeFilter.Match(svc.nodeName, sl.l) {
				continue
			}
		}

		if addrFilter := allFilters.RegexFilters["address"]; addrFilter != nil {
			if !addrFilter.Match(svc.address, sl.l) {
				continue
			}
		}

		// A service matches if any of its tags matches.
		if tagFilter := allFilters.RegexFilters["tag"]; tagFilter != nil {
			matched := false
			for _, tag := range svc.tags {
				if tagFilter.Match(tag, sl.l) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		labels := make(map[string]string)

		labels["service"] = svc.name
		labels["node"] = svc.nodeName
		labels["health"] = svc.health

		if sl.config.Services == nil || sl.config.Services.GetIncludeTags() {
			for _, tag := range svc.tags {
				labels[fmt.Sprintf("tag_%s", tag)] = "true"
			}
			if len(svc.tags) > 0 {
				labels["tags"] = strings.Join(svc.tags, ",")
			}
		}

		if sl.config.Services == nil || sl.config.Services.GetIncludeMetadata() {
			for k, v := range svc.meta {
				labels[fmt.Sprintf("meta_%s", k)] = v
			}
		}

		if sl.config.Services != nil && len(sl.config.Services.NodeMetaFilter) > 0 {
			for _, metaKey := range sl.config.Services.NodeMetaFilter {
				if v, ok := svc.nodeMeta[metaKey]; ok {
					labels[fmt.Sprintf("node_meta_%s", metaKey)] = v
				}
			}
		}

		if allFilters.LabelsFilter != nil {
			if !allFilters.LabelsFilter.Match(labels, sl.l) {
				continue
			}
		}

		resource := &pb.Resource{
			Name:        proto.String(key),
			Ip:          proto.String(svc.address),
			Port:        proto.Int32(int32(svc.port)),
			Labels:      labels,
			Id:          proto.String(key),
			LastUpdated: proto.Int64(svc.lastUpdated),
		}

		resources = append(resources, resource)
	}

	sl.l.Debugf("consul.services: returning %d resources after filtering", len(resources))
	return resources, nil
}

// aggregateHealthStatus determines the overall health status from a list of
// health checks. Priority order: critical > warning > passing.
func aggregateHealthStatus(checks []consulCheck) string {
	hasCritical := false
	hasWarning := false
	hasPassing := false

	for _, check := range checks {
		switch check.Status {
		case "critical":
			hasCritical = true
		case "warning":
			hasWarning = true
		case "passing":
			hasPassing = true
		}
	}

	if hasCritical {
		return "critical"
	}
	if hasWarning {
		return "warning"
	}
	if hasPassing {
		return "passing"
	}
	return "unknown"
}

func containsAllTags(tags []string, required []string) bool {
	tagMap := make(map[string]bool)
	for _, tag := range tags {
		tagMap[tag] = true
	}

	for _, req := range required {
		if !tagMap[req] {
			return false
		}
	}

	return true
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
