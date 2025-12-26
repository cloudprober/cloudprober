// Package consul implements a Consul-based resource discovery provider for Cloudprober.
// It allows discovering services, health checks, and nodes from HashiCorp Consul.
package consul

import (
	"fmt"
	"os"
	"strings"
	"sync"

	configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
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
// It respects standard Consul environment variables:
//   - CONSUL_HTTP_ADDR: HTTP API address (default: localhost:8500)
//     Special value "auto-discover" enables Kubernetes service discovery
//     Uses consul.default.svc.cluster.local:8500 by default
//     Can be customized with CONSUL_K8S_NAMESPACE, CONSUL_K8S_SERVICE_NAME, CONSUL_K8S_PORT
//   - CONSUL_HTTP_TOKEN: ACL token
//   - CONSUL_HTTP_TOKEN_FILE: Path to file containing ACL token
//   - CONSUL_HTTP_SSL: Enable HTTPS (true/false)
//   - CONSUL_HTTP_SSL_VERIFY: Verify SSL certificates (default: true)
//   - CONSUL_CACERT: Path to CA certificate
//   - CONSUL_CAPATH: Path to directory of CA certificates
//   - CONSUL_CLIENT_CERT: Path to client certificate
//   - CONSUL_CLIENT_KEY: Path to client key
//   - CONSUL_DATACENTER: Datacenter name
//   - CONSUL_NAMESPACE: Namespace (Consul Enterprise)
//   - CONSUL_TLS_SERVER_NAME: TLS server name for SNI
//   - CONSUL_HTTP_AUTH: HTTP basic auth (username:password)
//
// Auto-discovery environment variables (when CONSUL_HTTP_ADDR="auto-discover"):
//   - CONSUL_K8S_NAMESPACE: Kubernetes namespace (default: default)
//   - CONSUL_K8S_SERVICE_NAME: Kubernetes service name (default: consul)
//   - CONSUL_K8S_PORT: Kubernetes service port (default: 8500)
//
// Configuration values explicitly set in the config take precedence over environment variables.
func newConsulClient(c *configpb.ProviderConfig, l *logger.Logger) (*consulapi.Client, error) {
	// DefaultConfig reads environment variables automatically
	config := consulapi.DefaultConfig()

	// Set address - could be from direct config or Kubernetes service discovery
	// Only override if explicitly configured (don't override CONSUL_HTTP_ADDR env var)
	if c.KubernetesService != nil {
		// If Kubernetes service is configured, use it
		ksvc := c.KubernetesService
		address := fmt.Sprintf("%s.%s.svc.cluster.local:%s",
			ksvc.GetServiceName(),
			ksvc.GetNamespace(),
			ksvc.GetPort())
		config.Address = address
		l.Infof("consul.client: using Kubernetes service address: %s", address)
	} else if c.Address != nil && c.GetAddress() != "" && c.GetAddress() != "auto-discover" {
		// Only set if explicitly provided (not just the proto default) and not "auto-discover"
		config.Address = c.GetAddress()
		l.Infof("consul.client: using configured address: %s", config.Address)
	} else if config.Address == "auto-discover" || c.GetAddress() == "auto-discover" {
		// Special handling for auto-discover
		namespace := os.Getenv("CONSUL_K8S_NAMESPACE")
		if namespace == "" {
			namespace = "default"
		}
		serviceName := os.Getenv("CONSUL_K8S_SERVICE_NAME")
		if serviceName == "" {
			serviceName = "consul"
		}
		port := os.Getenv("CONSUL_K8S_PORT")
		if port == "" {
			port = "8500"
		}
		config.Address = fmt.Sprintf("%s.%s.svc.cluster.local:%s", serviceName, namespace, port)
		l.Infof("consul.client: auto-discover enabled, using Kubernetes DNS: %s", config.Address)
	} else {
		// Using address from environment variable or DefaultConfig default
		l.Infof("consul.client: using address from environment or default: %s", config.Address)
	}

	// Set datacenter - only override if explicitly configured
	if c.GetDatacenter() != "" {
		config.Datacenter = c.GetDatacenter()
		l.Infof("consul.client: using configured datacenter: %s", config.Datacenter)
	}

	// Set token - only override if explicitly configured
	// Note: DefaultConfig() reads CONSUL_HTTP_TOKEN and CONSUL_HTTP_TOKEN_FILE automatically
	if c.GetToken() != "" {
		config.Token = c.GetToken()
		l.Infof("consul.client: using configured ACL token (length: %d)", len(config.Token))
	}

	// Configure TLS - only override if explicitly configured
	// Note: DefaultConfig() reads CONSUL_CACERT, CONSUL_CLIENT_CERT, CONSUL_CLIENT_KEY,
	// CONSUL_HTTP_SSL, CONSUL_HTTP_SSL_VERIFY, and CONSUL_TLS_SERVER_NAME
	if c.Tls != nil {
		if tlsConfig := c.Tls; tlsConfig != nil {
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
		l.Infof("consul.client: using configured TLS settings")
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
