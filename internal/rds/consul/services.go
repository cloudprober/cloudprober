// Services lister for Consul service discovery.
package consul

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"
	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/protobuf/proto"
)

// servicesLister lists Consul services.
type servicesLister struct {
	config *configpb.ProviderConfig
	client *consulapi.Client
	cache  *resourceCache
	l      *logger.Logger
}

// SupportedFilters defines the filters supported by the services lister.
var SupportedFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	// Supported regex filters:
	// - name: service name
	// - node: node name
	// - tag: service tag (matches if ANY tag matches)
	RegexFilterKeys: []string{"name", "node", "tag", "address"},
	LabelsFilter:    true,
}

// newServicesLister creates a new services lister.
func newServicesLister(c *configpb.ProviderConfig, client *consulapi.Client, l *logger.Logger) (*servicesLister, error) {
	sl := &servicesLister{
		config: c,
		client: client,
		cache:  newResourceCache(),
		l:      l,
	}

	// Start background refresh goroutine
	go sl.refreshLoop()

	return sl, nil
}

// refreshLoop periodically refreshes the service list from Consul.
func (sl *servicesLister) refreshLoop() {
	reEvalInterval := time.Duration(sl.config.GetReEvalSec()) * time.Second

	// Initial refresh
	if err := sl.refresh(); err != nil {
		sl.l.Errorf("consul.services: initial refresh failed: %v", err)
	}

	// Periodic refresh with ticker
	ticker := time.NewTicker(reEvalInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := sl.refresh(); err != nil {
			sl.l.Errorf("consul.services: refresh failed: %v", err)
		}
	}
}

// refresh fetches the latest services from Consul and updates the cache.
func (sl *servicesLister) refresh() error {
	sl.l.Debugf("consul.services: refreshing service list")

	// Get list of all services
	services, _, err := sl.client.Catalog().Services(nil)
	if err != nil {
		return fmt.Errorf("failed to list services: %v", err)
	}

	sl.l.Debugf("consul.services: found %d services", len(services))

	// Clear old cache
	sl.cache.clear()

	// For each service, get detailed information
	for serviceName, tags := range services {
		// Apply name filter if configured
		if sl.config.Services != nil && sl.config.Services.GetNameFilter() != "" {
			matched, err := regexp.MatchString(sl.config.Services.GetNameFilter(), serviceName)
			if err != nil {
				sl.l.Warningf("consul.services: invalid name filter regex: %v", err)
			} else if !matched {
				continue
			}
		}

		// Apply tag filter if configured
		if sl.config.Services != nil && len(sl.config.Services.TagFilter) > 0 {
			if !containsAllTags(tags, sl.config.Services.TagFilter) {
				continue
			}
		}

		// Get service instances
		if err := sl.refreshService(serviceName); err != nil {
			sl.l.Warningf("consul.services: failed to refresh service %s: %v", serviceName, err)
		}
	}

	sl.l.Infof("consul.services: cached %d service instances", len(sl.cache.getAll()))
	return nil
}

// refreshService fetches detailed information for a specific service.
func (sl *servicesLister) refreshService(serviceName string) error {
	// Query options for health filtering
	queryOpts := &consulapi.QueryOptions{}
	if sl.config.GetDatacenter() != "" {
		queryOpts.Datacenter = sl.config.GetDatacenter()
	}

	// Get service health with all instances
	entries, _, err := sl.client.Health().Service(serviceName, "", false, queryOpts)
	if err != nil {
		return fmt.Errorf("failed to get service health: %v", err)
	}

	now := time.Now().Unix()

	for _, entry := range entries {
		// Determine health status
		healthStatus := aggregateHealthStatus(entry.Checks)

		// Apply health status filter if configured
		if sl.config.Services != nil && len(sl.config.Services.HealthStatus) > 0 {
			if !containsString(sl.config.Services.HealthStatus, healthStatus) {
				continue
			}
		}

		// Build service data
		service := entry.Service
		node := entry.Node

		// Determine the address to use
		address := service.Address
		if address == "" {
			address = node.Address
		}

		// Create unique key for this service instance
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

// listResources implements the lister interface for services.
func (sl *servicesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	// Parse filters
	allFilters, err := filter.ParseFilters(req.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse filters: %v", err)
	}

	// Get all cached services
	allServices := sl.cache.getAll()

	var resources []*pb.Resource

	for key, svc := range allServices {
		// Apply name filter
		if nameFilter := allFilters.RegexFilters["name"]; nameFilter != nil {
			if !nameFilter.Match(svc.name, sl.l) {
				continue
			}
		}

		// Apply node filter
		if nodeFilter := allFilters.RegexFilters["node"]; nodeFilter != nil {
			if !nodeFilter.Match(svc.nodeName, sl.l) {
				continue
			}
		}

		// Apply address filter
		if addrFilter := allFilters.RegexFilters["address"]; addrFilter != nil {
			if !addrFilter.Match(svc.address, sl.l) {
				continue
			}
		}

		// Apply tag filter (matches if ANY tag matches the regex)
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

		// Build labels map
		labels := make(map[string]string)

		// Add service name as label
		labels["service"] = svc.name
		labels["node"] = svc.nodeName
		labels["health"] = svc.health

		// Add tags as labels if configured
		if sl.config.Services == nil || sl.config.Services.GetIncludeTags() {
			for _, tag := range svc.tags {
				labels[fmt.Sprintf("tag_%s", tag)] = "true"
			}
			if len(svc.tags) > 0 {
				labels["tags"] = strings.Join(svc.tags, ",")
			}
		}

		// Add service metadata as labels if configured
		if sl.config.Services == nil || sl.config.Services.GetIncludeMetadata() {
			for k, v := range svc.meta {
				labels[fmt.Sprintf("meta_%s", k)] = v
			}
		}

		// Add node metadata if configured
		if sl.config.Services != nil && len(sl.config.Services.NodeMetaFilter) > 0 {
			for _, metaKey := range sl.config.Services.NodeMetaFilter {
				if v, ok := svc.nodeMeta[metaKey]; ok {
					labels[fmt.Sprintf("node_meta_%s", metaKey)] = v
				}
			}
		}

		// Apply labels filter
		if allFilters.LabelsFilter != nil {
			if !allFilters.LabelsFilter.Match(labels, sl.l) {
				continue
			}
		}

		// Create resource
		resource := &pb.Resource{
			Name:        proto.String(svc.name),
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

// aggregateHealthStatus determines the overall health status from a list of health checks.
func aggregateHealthStatus(checks consulapi.HealthChecks) string {
	// Priority: critical > warning > passing
	hasCritical := false
	hasWarning := false
	hasPassing := false

	for _, check := range checks {
		switch check.Status {
		case consulapi.HealthCritical:
			hasCritical = true
		case consulapi.HealthWarning:
			hasWarning = true
		case consulapi.HealthPassing:
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

// containsAllTags checks if all required tags are present in the tag list.
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

// containsString checks if a slice contains a specific string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
