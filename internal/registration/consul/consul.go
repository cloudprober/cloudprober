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

// Package consul handles self-registration of cloudprober with Consul's service
// catalog. Enable and configure it via CLI flags:
//
//	--consul_registration=true
//	--consul_addr=localhost:8500
//	--consul_service_name=cloudprober
//
// Consul's HTTP REST API is used directly to avoid a dependency on the
// HashiCorp Consul Go client library. Health checks use cloudprober's
// built-in /health HTTP endpoint.
package consul

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/logger"
)

var (
	enabled     = flag.Bool("consul_registration", false, "Register cloudprober with Consul service catalog")
	consulAddr  = flag.String("consul_addr", "localhost:8500", "Consul agent HTTP API address (host:port)")
	serviceName = flag.String("consul_service_name", "cloudprober", "Consul service name")
	serviceID   = flag.String("consul_service_id", "", "Consul service ID (auto-generated from name+addr+port if empty)")
	servicePort = flag.Int("consul_service_port", 9313, "Cloudprober HTTP service port to register with Consul")
	serviceAddr = flag.String("consul_service_addr", "", "Cloudprober service address (hostname if empty)")
	serviceTags = flag.String("consul_service_tags", "cloudprober", "Comma-separated Consul service tags")
	token       = flag.String("consul_token", "", "Consul ACL token for API authentication")

	healthInterval = flag.String("consul_health_interval", "30s", "Consul health check poll interval")
	healthTimeout  = flag.String("consul_health_timeout", "5s", "Consul health check request timeout")

	// How long before Consul deregisters a service whose health check is
	// continuously failing. Consul requires a minimum of 1 minute.
	deregisterAfter      = flag.String("consul_deregister_after", "1m", "Deregister service from Consul after being critical for this long")
	deregisterOnShutdown = flag.Bool("consul_deregister_on_shutdown", true, "Deregister cloudprober from Consul on shutdown")
)

// serviceRegistration is the JSON body for PUT /v1/agent/service/register.
type serviceRegistration struct {
	ID      string            `json:"ID"`
	Name    string            `json:"Name"`
	Tags    []string          `json:"Tags,omitempty"`
	Port    int               `json:"Port"`
	Address string            `json:"Address,omitempty"`
	Meta    map[string]string `json:"Meta,omitempty"`
	Check   *healthCheck      `json:"Check,omitempty"`
}

// healthCheck is the Consul health check configuration embedded in serviceRegistration.
type healthCheck struct {
	HTTP                           string `json:"HTTP"`
	Interval                       string `json:"Interval"`
	Timeout                        string `json:"Timeout"`
	DeregisterCriticalServiceAfter string `json:"DeregisterCriticalServiceAfter,omitempty"`
}

// registration holds the active Consul registration state.
type registration struct {
	consulAddr  string
	serviceID   string
	serviceName string
	servicePort int
	serviceAddr string
	token       string
	l           *logger.Logger
}

// Start registers cloudprober with Consul if --consul_registration is set.
// It is a no-op if the flag is not set. Deregistration happens when ctx is
// cancelled (controlled by --consul_deregister_on_shutdown).
func Start(ctx context.Context, l *logger.Logger) error {
	if !*enabled {
		return nil
	}

	reg, err := newRegistration(l)
	if err != nil {
		return err
	}

	if err := reg.register(); err != nil {
		return fmt.Errorf("consul registration: %v", err)
	}

	go func() {
		<-ctx.Done()
		if *deregisterOnShutdown {
			reg.deregister()
		}
	}()

	return nil
}

// newRegistration builds a registration from CLI flags, resolving any
// auto-detected values (address, service ID).
func newRegistration(l *logger.Logger) (*registration, error) {
	addr := *serviceAddr
	if addr == "" {
		hostname, err := os.Hostname()
		if err != nil {
			l.Warningf("consul registration: failed to get hostname: %v, using localhost", err)
			addr = "localhost"
		} else {
			addr = hostname
		}
	}

	id := *serviceID
	if id == "" {
		id = fmt.Sprintf("%s-%s-%d", *serviceName, addr, *servicePort)
	}

	return &registration{
		consulAddr:  *consulAddr,
		serviceID:   id,
		serviceName: *serviceName,
		servicePort: *servicePort,
		serviceAddr: addr,
		token:       *token,
		l:           l,
	}, nil
}

// register calls Consul's service registration API.
func (reg *registration) register() error {
	// Point Consul at cloudprober's built-in /health endpoint.
	healthURL := fmt.Sprintf("http://%s:%d/health", reg.serviceAddr, reg.servicePort)

	tags := parseTags(*serviceTags)

	payload := serviceRegistration{
		ID:      reg.serviceID,
		Name:    reg.serviceName,
		Tags:    tags,
		Port:    reg.servicePort,
		Address: reg.serviceAddr,
		Check: &healthCheck{
			HTTP:                           healthURL,
			Interval:                       *healthInterval,
			Timeout:                        *healthTimeout,
			DeregisterCriticalServiceAfter: *deregisterAfter,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration payload: %v", err)
	}

	if err := reg.put("agent/service/register", body); err != nil {
		return fmt.Errorf("failed to register service %q: %v", reg.serviceID, err)
	}

	reg.l.Infof("consul registration: registered %q (ID: %s) at %s:%d",
		reg.serviceName, reg.serviceID, reg.serviceAddr, reg.servicePort)
	return nil
}

// deregister calls Consul's service deregistration API.
func (reg *registration) deregister() {
	path := fmt.Sprintf("agent/service/deregister/%s", reg.serviceID)
	if err := reg.put(path, nil); err != nil {
		reg.l.Errorf("consul registration: failed to deregister %q: %v", reg.serviceID, err)
		return
	}
	reg.l.Infof("consul registration: deregistered %q", reg.serviceID)
}

// put makes an authenticated HTTP PUT to the Consul agent API.
func (reg *registration) put(path string, body []byte) error {
	url := fmt.Sprintf("http://%s/v1/%s", reg.consulAddr, path)

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, url, bodyReader)
	if err != nil {
		return err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if reg.token != "" {
		req.Header.Set("X-Consul-Token", reg.token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("consul API returned HTTP %d for PUT %s", resp.StatusCode, url)
	}
	return nil
}

// parseTags splits a comma-separated tag string into a slice, trimming
// whitespace and skipping empty entries.
func parseTags(raw string) []string {
	var tags []string
	for _, tag := range strings.Split(raw, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
