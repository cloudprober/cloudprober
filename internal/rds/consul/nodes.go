// Nodes lister for Consul.
package consul

import (
	"fmt"
	"time"

	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/protobuf/proto"
)

// nodesLister lists Consul nodes.
type nodesLister struct {
	config *configpb.ProviderConfig
	client *consulapi.Client
	cache  *resourceCache
	l      *logger.Logger
}

// NodesSupportedFilters defines the filters supported by the nodes lister.
var NodesSupportedFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	// Supported regex filters:
	// - name: node name
	// - address: node address
	// - datacenter: datacenter name
	RegexFilterKeys: []string{"name", "address", "datacenter"},
	LabelsFilter:    true,
}

// newNodesLister creates a new nodes lister.
func newNodesLister(c *configpb.ProviderConfig, client *consulapi.Client, l *logger.Logger) (*nodesLister, error) {
	nl := &nodesLister{
		config: c,
		client: client,
		cache:  newResourceCache(),
		l:      l,
	}

	// Start background refresh goroutine
	go nl.refreshLoop()

	return nl, nil
}

// refreshLoop periodically refreshes the nodes from Consul.
func (nl *nodesLister) refreshLoop() {
	reEvalInterval := time.Duration(nl.config.GetReEvalSec()) * time.Second

	// Initial refresh
	if err := nl.refresh(); err != nil {
		nl.l.Errorf("consul.nodes: initial refresh failed: %v", err)
	}

	// Periodic refresh with ticker
	ticker := time.NewTicker(reEvalInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := nl.refresh(); err != nil {
			nl.l.Errorf("consul.nodes: refresh failed: %v", err)
		}
	}
}

// refresh fetches the latest nodes from Consul and updates the cache.
func (nl *nodesLister) refresh() error {
	nl.l.Debugf("consul.nodes: refreshing nodes")

	queryOpts := &consulapi.QueryOptions{}

	// Use datacenter from nodes config if specified, otherwise from provider config
	datacenter := nl.config.GetDatacenter()
	if nl.config.Nodes != nil && nl.config.Nodes.GetDatacenter() != "" {
		datacenter = nl.config.Nodes.GetDatacenter()
	}
	if datacenter != "" {
		queryOpts.Datacenter = datacenter
	}

	nodes, _, err := nl.client.Catalog().Nodes(queryOpts)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	nl.l.Debugf("consul.nodes: found %d nodes", len(nodes))

	// Clear old cache
	nl.cache.clear()

	now := time.Now().Unix()

	for _, node := range nodes {
		// Apply metadata filter if configured
		if nl.config.Nodes != nil && len(nl.config.Nodes.MetaFilter) > 0 {
			matches := true
			for key, value := range nl.config.Nodes.MetaFilter {
				if nodeValue, ok := node.Meta[key]; !ok || nodeValue != value {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		}

		// Create unique key for this node
		key := fmt.Sprintf("%s-%s", node.Datacenter, node.Node)

		data := &serviceData{
			name:        node.Node,
			address:     node.Address,
			nodeName:    node.Node,
			nodeMeta:    node.Meta,
			lastUpdated: now,
			meta: map[string]string{
				"datacenter": node.Datacenter,
				"node_id":    node.ID,
			},
		}

		nl.cache.set(key, data)
	}

	nl.l.Infof("consul.nodes: cached %d nodes", len(nl.cache.getAll()))
	return nil
}

// listResources implements the lister interface for nodes.
func (nl *nodesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	// Parse filters
	allFilters, err := filter.ParseFilters(req.GetFilter(), NodesSupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse filters: %v", err)
	}

	// Get all cached nodes
	allNodes := nl.cache.getAll()

	var resources []*pb.Resource

	for key, node := range allNodes {
		// Apply name filter
		if nameFilter := allFilters.RegexFilters["name"]; nameFilter != nil {
			if !nameFilter.Match(node.name, nl.l) {
				continue
			}
		}

		// Apply address filter
		if addressFilter := allFilters.RegexFilters["address"]; addressFilter != nil {
			if !addressFilter.Match(node.address, nl.l) {
				continue
			}
		}

		// Apply datacenter filter
		if dcFilter := allFilters.RegexFilters["datacenter"]; dcFilter != nil {
			datacenter := node.meta["datacenter"]
			if !dcFilter.Match(datacenter, nl.l) {
				continue
			}
		}

		// Build labels map
		labels := make(map[string]string)
		labels["node"] = node.name
		labels["datacenter"] = node.meta["datacenter"]
		labels["node_id"] = node.meta["node_id"]

		// Add node metadata as labels
		for k, v := range node.nodeMeta {
			labels[fmt.Sprintf("meta_%s", k)] = v
		}

		// Apply labels filter
		if allFilters.LabelsFilter != nil {
			if !allFilters.LabelsFilter.Match(labels, nl.l) {
				continue
			}
		}

		// Create resource
		resource := &pb.Resource{
			Name:        proto.String(node.name),
			Ip:          proto.String(node.address),
			Labels:      labels,
			Id:          proto.String(key),
			LastUpdated: proto.Int64(node.lastUpdated),
		}

		resources = append(resources, resource)
	}

	nl.l.Debugf("consul.nodes: returning %d resources after filtering", len(resources))
	return resources, nil
}
