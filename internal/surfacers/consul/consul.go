// Package consul implements a surfacer that registers the cloudprober instance with Consul.
// It publishes the instance's existence, health status, and metadata to Consul's service catalog.
package consul

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/consul/proto"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/options"
	consulapi "github.com/hashicorp/consul/api"
)

// Surfacer implements the Surfacer interface for Consul service registration.
type Surfacer struct {
	c      *configpb.SurfacerConf
	opts   *options.Options
	client *consulapi.Client
	l      *logger.Logger

	// Service registration details
	serviceID   string
	serviceName string
	servicePort int
	serviceAddr string

	// Health check server
	healthServer *http.Server
	healthStatus bool

	// Metrics channel
	writeChan chan *metrics.EventMetrics

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new Consul surfacer.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*Surfacer, error) {
	if config == nil {
		return nil, fmt.Errorf("consul surfacer config is nil")
	}

	client, err := newConsulClient(config, l)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %v", err)
	}

	// Create context for the surfacer
	surfacerCtx, cancel := context.WithCancel(ctx)

	s := &Surfacer{
		c:            config,
		opts:         opts,
		client:       client,
		l:            l,
		healthStatus: true, // Start healthy
		writeChan:    make(chan *metrics.EventMetrics, opts.MetricsBufferSize),
		ctx:          surfacerCtx,
		cancel:       cancel,
	}

	// Determine service details
	if err := s.determineServiceDetails(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to determine service details: %v", err)
	}

	// Start health check HTTP server
	if err := s.startHealthCheckServer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start health check server: %v", err)
	}

	// Register service with Consul
	if err := s.registerService(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register service: %v", err)
	}

	// Start background goroutine to process metrics
	go s.processMetrics(surfacerCtx)

	l.Infof("consul.surfacer: registered service %s (ID: %s) at %s:%d",
		s.serviceName, s.serviceID, s.serviceAddr, s.servicePort)

	return s, nil
}

// Write implements the Surfacer interface.
// It receives metrics and can use them to update health status.
func (s *Surfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.writeChan <- em:
	case <-ctx.Done():
		return
	default:
		s.l.Warningf("consul.surfacer: write channel full, dropping metric")
	}
}

// processMetrics processes incoming metrics in the background.
// This allows the surfacer to monitor probe health and update status accordingly.
func (s *Surfacer) processMetrics(ctx context.Context) {
	for {
		select {
		case em := <-s.writeChan:
			s.handleMetric(em)
		case <-ctx.Done():
			s.l.Infof("consul.surfacer: shutting down metrics processor")
			s.deregisterService()
			return
		}
	}
}

// handleMetric processes a single metric.
// This can be extended to update health status based on probe failures.
func (s *Surfacer) handleMetric(em *metrics.EventMetrics) {
	// Check for probe failures
	// If too many probes are failing, we could mark the instance as unhealthy
	if successMetric := em.Metric("success"); successMetric != nil {
		// This is a simplified example
		// In production, you might want more sophisticated logic
		s.l.Debugf("consul.surfacer: received metric from probe %s", em.Label("probe"))
	}
}

// determineServiceDetails determines the service name, ID, port, and address.
func (s *Surfacer) determineServiceDetails() error {
	// Service name
	s.serviceName = "cloudprober"
	if s.c.Service != nil && s.c.Service.GetName() != "" {
		s.serviceName = s.c.Service.GetName()
	}

	// Service address
	s.serviceAddr = ""
	if s.c.Service != nil && s.c.Service.GetAddress() != "" {
		s.serviceAddr = s.c.Service.GetAddress()
	}
	if s.serviceAddr == "" {
		// Try to auto-detect address
		hostname, err := os.Hostname()
		if err != nil {
			s.l.Warningf("consul.surfacer: failed to get hostname: %v, using localhost", err)
			s.serviceAddr = "localhost"
		} else {
			s.serviceAddr = hostname
		}
	}

	// Service port
	s.servicePort = 9313 // Default Prometheus metrics port
	if s.c.Service != nil && s.c.Service.GetPort() > 0 {
		s.servicePort = int(s.c.Service.GetPort())
	}

	// Service ID
	s.serviceID = ""
	if s.c.Service != nil && s.c.Service.GetId() != "" {
		s.serviceID = s.c.Service.GetId()
	}
	if s.serviceID == "" {
		// Auto-generate ID
		s.serviceID = fmt.Sprintf("%s-%s-%d", s.serviceName, s.serviceAddr, s.servicePort)
	}

	return nil
}

// startHealthCheckServer starts an HTTP server for Consul health checks.
func (s *Surfacer) startHealthCheckServer() error {
	endpoint := "/status"
	if s.c.HealthCheck != nil && s.c.HealthCheck.GetHttpEndpoint() != "" {
		endpoint = s.c.HealthCheck.GetHttpEndpoint()
	}

	mux := http.NewServeMux()
	mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		if s.healthStatus {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Unhealthy"))
		}
	})

	// Bind to a different port for health checks (servicePort + 1)
	// This avoids conflicts with the main Cloudprober server
	healthPort := s.servicePort + 1
	addr := fmt.Sprintf(":%d", healthPort)

	s.healthServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in background
	go func() {
		s.l.Infof("consul.surfacer: starting health check server on %s%s", addr, endpoint)
		if err := s.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.l.Errorf("consul.surfacer: health check server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// getSysVarsMetadata retrieves and filters system variables based on configuration.
func (s *Surfacer) getSysVarsMetadata() map[string]string {
	// Default configuration if not specified
	cfg := s.c.GetSysvars()
	if cfg == nil {
		cfg = &configpb.SysVarsConfig{}
	}

	// If sysvars are disabled, return empty map
	if !cfg.GetEnabled() {
		return make(map[string]string)
	}

	// Get all sysvars
	allVars := sysvars.Vars()
	if len(allVars) == 0 {
		s.l.Warning("consul.surfacer: sysvars.Vars() returned empty map - sysvars may not be initialized")
		return make(map[string]string)
	}

	// Apply filtering
	result := make(map[string]string)
	includeVars := cfg.GetIncludeVars()
	excludeVars := cfg.GetExcludeVars()
	prefix := cfg.GetKeyPrefix()

	// Build exclude set for efficient lookup
	excludeSet := make(map[string]bool)
	for _, v := range excludeVars {
		excludeSet[v] = true
	}

	// If include_vars is specified, only include those
	if len(includeVars) > 0 {
		for _, varName := range includeVars {
			if excludeSet[varName] {
				continue // Skip if in exclude list
			}
			if val, ok := allVars[varName]; ok {
				result[prefix+varName] = val
			}
		}
	} else {
		// Include all vars except those in exclude list
		for varName, val := range allVars {
			if excludeSet[varName] {
				continue
			}
			result[prefix+varName] = val
		}
	}

	return result
}

// registerService registers the cloudprober instance with Consul.
func (s *Surfacer) registerService() error {
	// Build service tags
	tags := []string{"cloudprober"}
	if s.c.Service != nil {
		tags = append(tags, s.c.Service.Tags...)
	}

	// Build metadata - start with user-configured metadata
	meta := make(map[string]string)
	for k, v := range s.c.Metadata {
		meta[k] = v
	}

	// Add system variables (sysvars)
	// This is enabled by default and includes: version, hostname, start_timestamp, cloud metadata, etc.
	sysVarsMetadata := s.getSysVarsMetadata()
	for k, v := range sysVarsMetadata {
		// User-configured metadata takes precedence
		if _, exists := meta[k]; !exists {
			meta[k] = v
		}
	}

	// Log metadata being published
	s.l.Infof("consul.surfacer: publishing %d metadata keys (%d from sysvars)", len(meta), len(sysVarsMetadata))

	// Build health check
	healthCheckPort := s.servicePort + 1
	healthCheckURL := fmt.Sprintf("http://%s:%d%s",
		s.serviceAddr,
		healthCheckPort,
		s.c.HealthCheck.GetHttpEndpoint())

	check := &consulapi.AgentServiceCheck{
		HTTP:                           healthCheckURL,
		Interval:                       s.c.HealthCheck.GetInterval(),
		Timeout:                        s.c.HealthCheck.GetTimeout(),
		DeregisterCriticalServiceAfter: s.c.HealthCheck.GetDeregisterCriticalServiceAfter(),
		TLSSkipVerify:                  s.c.HealthCheck.GetTlsSkipVerify(),
	}

	if s.c.HealthCheck != nil && s.c.HealthCheck.GetNotes() != "" {
		check.Notes = s.c.HealthCheck.GetNotes()
	}

	// Build service registration
	registration := &consulapi.AgentServiceRegistration{
		ID:      s.serviceID,
		Name:    s.serviceName,
		Tags:    tags,
		Port:    s.servicePort,
		Address: s.serviceAddr,
		Meta:    meta,
		Check:   check,
	}

	// Enable Connect if configured
	if s.c.Service != nil && s.c.Service.GetEnableConnect() {
		registration.Connect = &consulapi.AgentServiceConnect{
			Native: true,
		}
	}

	// Register with Consul
	if err := s.client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register service: %v", err)
	}

	s.l.Infof("consul.surfacer: successfully registered service with Consul")
	return nil
}

// deregisterService deregisters the service from Consul.
func (s *Surfacer) deregisterService() {
	if !s.c.GetDeregisterOnShutdown() {
		s.l.Infof("consul.surfacer: skipping deregistration (deregister_on_shutdown=false)")
		return
	}

	s.l.Infof("consul.surfacer: deregistering service %s", s.serviceID)
	if err := s.client.Agent().ServiceDeregister(s.serviceID); err != nil {
		s.l.Errorf("consul.surfacer: failed to deregister service: %v", err)
	}

	// Shutdown health check server
	if s.healthServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.healthServer.Shutdown(ctx); err != nil {
			s.l.Errorf("consul.surfacer: failed to shutdown health server: %v", err)
		}
	}
}

// newConsulClient creates a new Consul API client based on the configuration.
func newConsulClient(c *configpb.SurfacerConf, l *logger.Logger) (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()

	// Set address - could be from direct config or Kubernetes service discovery
	address := c.GetAddress()
	if c.KubernetesService != nil {
		// If Kubernetes service is configured, we'll resolve it
		ksvc := c.KubernetesService
		address = fmt.Sprintf("%s.%s.svc.cluster.local:%s",
			ksvc.GetServiceName(),
			ksvc.GetNamespace(),
			ksvc.GetPort())
		l.Infof("consul.surfacer: using Kubernetes service address: %s", address)
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

// SetHealthStatus allows external control of the health status.
func (s *Surfacer) SetHealthStatus(healthy bool) {
	s.healthStatus = healthy
	s.l.Infof("consul.surfacer: health status set to %v", healthy)
}

// Close cleans up the surfacer resources.
func (s *Surfacer) Close() error {
	s.cancel()
	return nil
}
