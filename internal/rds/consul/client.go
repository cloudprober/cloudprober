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

// Package consul implements a Consul-based resource discovery provider for Cloudprober.
// This file provides a plain HTTP client for the Consul REST API, with no
// dependency on github.com/hashicorp/consul/api.
package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	"github.com/cloudprober/cloudprober/logger"
)

// client wraps a plain *http.Client to call the Consul HTTP REST API.
type client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	datacenter string
	l          *logger.Logger
}

// consulHealthEntry is the JSON structure returned by GET /v1/health/service/{service}.
type consulHealthEntry struct {
	Node    consulNode    `json:"Node"`
	Service consulService `json:"Service"`
	Checks  []consulCheck `json:"Checks"`
}

// consulNode is the Node sub-object within a health entry.
type consulNode struct {
	Node    string            `json:"Node"`
	Address string            `json:"Address"`
	Meta    map[string]string `json:"Meta"`
}

// consulService is the Service sub-object within a health entry.
type consulService struct {
	ID      string            `json:"ID"`
	Service string            `json:"Service"`
	Tags    []string          `json:"Tags"`
	Address string            `json:"Address"`
	Port    int               `json:"Port"`
	Meta    map[string]string `json:"Meta"`
}

// consulCheck represents a single Consul health check result.
// It is used both within consulHealthEntry.Checks and as the element type
// for the /v1/health/checks/{service} and /v1/health/state/any responses.
type consulCheck struct {
	Node        string `json:"Node"`
	CheckID     string `json:"CheckID"`
	Name        string `json:"Name"`
	Status      string `json:"Status"`
	Notes       string `json:"Notes"`
	Output      string `json:"Output"`
	ServiceID   string `json:"ServiceID"`
	ServiceName string `json:"ServiceName"`
}

// consulCatalogNode is the JSON structure returned by GET /v1/catalog/nodes.
type consulCatalogNode struct {
	ID         string            `json:"ID"`
	Node       string            `json:"Node"`
	Address    string            `json:"Address"`
	Datacenter string            `json:"Datacenter"`
	Meta       map[string]string `json:"Meta"`
}

// newClient creates a consul client from the provider config, falling back to
// standard Consul environment variables and sensible defaults.
func newClient(c *configpb.ProviderConfig, l *logger.Logger) (*client, error) {
	addr := resolveConsulAddr(c, l)

	useSSL := os.Getenv("CONSUL_HTTP_SSL") == "true" || os.Getenv("CONSUL_HTTP_SSL") == "1"
	scheme := "http"
	if useSSL {
		scheme = "https"
	}

	baseURL := fmt.Sprintf("%s://%s", scheme, addr)

	token := resolveToken(c.GetToken(), l)

	datacenter := c.GetDatacenter()
	if datacenter == "" {
		datacenter = os.Getenv("CONSUL_DATACENTER")
	}

	return &client{
		httpClient: &http.Client{},
		baseURL:    baseURL,
		token:      token,
		datacenter: datacenter,
		l:          l,
	}, nil
}

// resolveConsulAddr determines the Consul address using the following priority:
//  1. Kubernetes service discovery (KubernetesService config)
//  2. Explicit config address (c.Address != nil, not "auto-discover")
//  3. CONSUL_HTTP_ADDR environment variable
//  4. "auto-discover" (from config or env var) → Kubernetes DNS
//  5. Default: "localhost:8500"
func resolveConsulAddr(c *configpb.ProviderConfig, l *logger.Logger) string {
	if c.KubernetesService != nil {
		ksvc := c.KubernetesService
		address := fmt.Sprintf("%s.%s.svc.cluster.local:%s",
			ksvc.GetServiceName(),
			ksvc.GetNamespace(),
			ksvc.GetPort())
		l.Infof("consul.client: using Kubernetes service address: %s", address)
		return address
	}

	if c.Address != nil && c.GetAddress() != "" && c.GetAddress() != "auto-discover" {
		l.Infof("consul.client: using configured address: %s", c.GetAddress())
		return c.GetAddress()
	}

	envAddr := os.Getenv("CONSUL_HTTP_ADDR")

	// Handle auto-discover mode (either from config or env var).
	if c.GetAddress() == "auto-discover" || envAddr == "auto-discover" {
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
		address := fmt.Sprintf("%s.%s.svc.cluster.local:%s", serviceName, namespace, port)
		l.Infof("consul.client: auto-discover enabled, using Kubernetes DNS: %s", address)
		return address
	}

	if envAddr != "" {
		l.Infof("consul.client: using address from CONSUL_HTTP_ADDR: %s", envAddr)
		return envAddr
	}

	l.Infof("consul.client: using default address: localhost:8500")
	return "localhost:8500"
}

// resolveToken determines the ACL token using the following priority:
//  1. configToken (explicitly set in provider config)
//  2. CONSUL_HTTP_TOKEN environment variable
//  3. Contents of the file pointed to by CONSUL_HTTP_TOKEN_FILE
func resolveToken(configToken string, l *logger.Logger) string {
	if configToken != "" {
		return configToken
	}

	if token := os.Getenv("CONSUL_HTTP_TOKEN"); token != "" {
		return token
	}

	if tokenFile := os.Getenv("CONSUL_HTTP_TOKEN_FILE"); tokenFile != "" {
		data, err := os.ReadFile(tokenFile)
		if err != nil {
			l.Warningf("consul.client: failed to read token file %s: %v", tokenFile, err)
		} else {
			return strings.TrimSpace(string(data))
		}
	}

	return ""
}

// get performs a GET request against the given Consul API path, appending any
// extra query parameters. If the client has a datacenter configured and the
// caller has not already set the "dc" param, it is added automatically.
// The JSON response body is decoded into v.
func (c *client) get(path string, params url.Values, v interface{}) error {
	if params == nil {
		params = url.Values{}
	}

	// Add datacenter param only when the caller has not set one already.
	if c.datacenter != "" && params.Get("dc") == "" {
		params.Set("dc", c.datacenter)
	}

	fullURL := fmt.Sprintf("%s%s", c.baseURL, path)
	if len(params) > 0 {
		fullURL = fmt.Sprintf("%s?%s", fullURL, params.Encode())
	}

	c.l.Debugf("consul.client: GET %s", fullURL)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	if c.token != "" {
		req.Header.Set("X-Consul-Token", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, fullURL)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	return nil
}
