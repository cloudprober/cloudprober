// Package consul implements a Consul-based resource discovery provider for Cloudprober.
// It allows discovering services, health checks, and nodes from HashiCorp Consul.
package consul

import (
	"fmt"
	"strings"
	"sync"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	consulapi "github.com/hashicorp/consul/api"
)

const (
	// DefaultProviderID is the default provider ID for Consul RDS
	DefaultProviderID = "consul"
)

// Provider implements the RDS Provider interface for Consul.
type Provider struct {
	config  *configpb.ProviderConfig
	client  *consulapi.Client
	listers map[string]lister
	l       *logger.Logger
}

// lister is an interface for listing different Consul resource types.
type lister interface {
	listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error)
}

// New creates a new Consul provider instance.
func New(c *configpb.ProviderConfig, l *logger.Logger) (*Provider, error) {
	if c == nil {
		return nil, fmt.Errorf("consul provider config is nil")
	}

	client, err := newConsulClient(c, l)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %v", err)
	}

	p := &Provider{
		config:  c,
		client:  client,
		listers: make(map[string]lister),
		l:       l,
	}

	// Initialize listers based on configuration
	if c.Services != nil {
		servicesLister, err := newServicesLister(c, client, l)
		if err != nil {
			return nil, fmt.Errorf("failed to create services lister: %v", err)
		}
		p.listers["services"] = servicesLister
		l.Infof("consul.provider: initialized services lister")
	}

	if c.HealthChecks != nil {
		healthChecksLister, err := newHealthChecksLister(c, client, l)
		if err != nil {
			return nil, fmt.Errorf("failed to create health checks lister: %v", err)
		}
		p.listers["health_checks"] = healthChecksLister
		l.Infof("consul.provider: initialized health_checks lister")
	}

	if c.Nodes != nil {
		nodesLister, err := newNodesLister(c, client, l)
		if err != nil {
			return nil, fmt.Errorf("failed to create nodes lister: %v", err)
		}
		p.listers["nodes"] = nodesLister
		l.Infof("consul.provider: initialized nodes lister")
	}

	// Default to services if no specific resource type configured
	if len(p.listers) == 0 {
		servicesLister, err := newServicesLister(c, client, l)
		if err != nil {
			return nil, fmt.Errorf("failed to create default services lister: %v", err)
		}
		p.listers["services"] = servicesLister
		l.Infof("consul.provider: initialized default services lister")
	}

	return p, nil
}

// ListResources implements the Provider interface.
func (p *Provider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	// Parse resource path: format is "resource_type/optional_params"
	// Examples:
	//   "services"
	//   "services/web"
	//   "health_checks/web"
	//   "nodes"
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)
	resType := tok[0]

	lr := p.listers[resType]
	if lr == nil {
		return nil, fmt.Errorf("unknown resource type: %s, available types: %v", resType, p.availableResourceTypes())
	}

	resources, err := lr.listResources(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %v", resType, err)
	}

	p.l.Infof("consul.provider: listed %d resources for type %s", len(resources), resType)
	return &pb.ListResourcesResponse{Resources: resources}, nil
}

// availableResourceTypes returns a list of available resource types.
func (p *Provider) availableResourceTypes() []string {
	types := make([]string, 0, len(p.listers))
	for t := range p.listers {
		types = append(types, t)
	}
	return types
}

// newConsulClient creates a new Consul API client based on the configuration.
func newConsulClient(c *configpb.ProviderConfig, l *logger.Logger) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()

	// Set address - could be from direct config or Kubernetes service discovery
	address := c.GetAddress()
	if c.KubernetesService != nil {
		// If Kubernetes service is configured, we'll resolve it
		// For now, we'll use the address as-is and document that users can use
		// Kubernetes DNS (e.g., "consul.default.svc.cluster.local:8500")
		ksvc := c.KubernetesService
		address = fmt.Sprintf("%s.%s.svc.cluster.local:%s",
			ksvc.GetServiceName(),
			ksvc.GetNamespace(),
			ksvc.GetPort())
		l.Infof("consul.client: using Kubernetes service address: %s", address)
	}
	config.Address = address

	// Set datacenter
	if c.GetDatacenter() != "" {
		config.Datacenter = c.GetDatacenter()
	}

	// Set token (check environment variable if not in config)
	if c.GetToken() != "" {
		config.Token = c.GetToken()
	}

	// Configure TLS
	if c.Tls != nil {
		tlsConfig := c.Tls
		if tlsConfig.GetCaFile() != "" {
			config.TLSConfig.CAFile = tlsConfig.GetCaFile()
		}
		if tlsConfig.GetCertFile() != "" {
			config.TLSConfig.CertFile = tlsConfig.GetCertFile()
		}
		if tlsConfig.GetKeyFile() != "" {
			config.TLSConfig.KeyFile = tlsConfig.GetKeyFile()
		}
		if tlsConfig.GetInsecureSkipVerify() {
			config.TLSConfig.InsecureSkipVerify = true
		}
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %v", err)
	}

	return client, nil
}

// serviceData holds cached service information.
type serviceData struct {
	name        string
	address     string
	port        int
	tags        []string
	meta        map[string]string
	nodeName    string
	nodeMeta    map[string]string
	health      string
	lastUpdated int64
}

// resourceCache is a thread-safe cache for resources.
type resourceCache struct {
	mu    sync.RWMutex
	cache map[string]*serviceData
}

func newResourceCache() *resourceCache {
	return &resourceCache{
		cache: make(map[string]*serviceData),
	}
}

func (rc *resourceCache) get(key string) (*serviceData, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	data, ok := rc.cache[key]
	return data, ok
}

func (rc *resourceCache) set(key string, data *serviceData) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cache[key] = data
}

func (rc *resourceCache) getAll() map[string]*serviceData {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	result := make(map[string]*serviceData, len(rc.cache))
	for k, v := range rc.cache {
		result[k] = v
	}
	return result
}

func (rc *resourceCache) clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cache = make(map[string]*serviceData)
}

func (rc *resourceCache) delete(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	delete(rc.cache, key)
}
