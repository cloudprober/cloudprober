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

//go:build integration
// +build integration

package consul

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

// Integration tests require a running Consul instance
// Run these tests with: go test -tags=integration
//
// To start a local Consul for testing:
//
//	docker run -d -p 8500:8500 --name consul-test consul:latest agent -dev -ui -client=0.0.0.0
//
// To stop:
//
//	docker stop consul-test && docker rm consul-test
const (
	consulTestAddr = "localhost:8500"
	testTimeout    = 30 * time.Second
)

// testHTTPClient is a lightweight wrapper used only by integration test helpers
// to register/deregister services via the Consul agent REST API.
type testHTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// agentServiceCheck describes a single health check for service registration.
type agentServiceCheck struct {
	TTL string `json:"TTL,omitempty"`
}

// agentServiceRegistration is the request body for PUT /v1/agent/service/register.
type agentServiceRegistration struct {
	ID      string             `json:"ID"`
	Name    string             `json:"Name"`
	Tags    []string           `json:"Tags"`
	Port    int                `json:"Port"`
	Address string             `json:"Address"`
	Meta    map[string]string  `json:"Meta,omitempty"`
	Check   *agentServiceCheck `json:"Check,omitempty"`
}

// updateTTLRequest is the body for PUT /v1/agent/check/update/{checkId}.
type updateTTLRequest struct {
	Status string `json:"Status"`
	Output string `json:"Output,omitempty"`
}

// put encodes body as JSON and performs a PUT request.
func (tc *testHTTPClient) put(path string, body interface{}) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("PUT", tc.baseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for PUT %s", resp.StatusCode, path)
	}

	return nil
}

// setupTestConsul verifies that Consul is reachable and returns a testHTTPClient
// along with a cleanup function that deregisters any test services.
func setupTestConsul(t *testing.T) (*testHTTPClient, func()) {
	t.Helper()

	tc := &testHTTPClient{
		baseURL:    "http://" + consulTestAddr,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	// Check if Consul is accessible via GET /v1/agent/self.
	resp, err := tc.httpClient.Get(tc.baseURL + "/v1/agent/self")
	if err != nil {
		t.Skipf("Consul not available at %s: %v\nRun integration tests with a local Consul instance", consulTestAddr, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("Consul at %s returned status %d from /v1/agent/self", consulTestAddr, resp.StatusCode)
	}

	cleanup := func() {
		// List all agent services and deregister those with a "test-" prefix.
		listResp, err := tc.httpClient.Get(tc.baseURL + "/v1/agent/services")
		if err != nil {
			t.Logf("Failed to list services for cleanup: %v", err)
			return
		}
		defer listResp.Body.Close()

		var services map[string]interface{}
		if err := json.NewDecoder(listResp.Body).Decode(&services); err != nil {
			t.Logf("Failed to decode services list: %v", err)
			return
		}

		for id := range services {
			if strings.HasPrefix(id, "test-") {
				deregReq, err := http.NewRequest("PUT", tc.baseURL+"/v1/agent/service/deregister/"+id, nil)
				if err != nil {
					t.Logf("Failed to create deregister request for %s: %v", id, err)
					continue
				}
				deregResp, err := tc.httpClient.Do(deregReq)
				if err != nil {
					t.Logf("Failed to deregister test service %s: %v", id, err)
					continue
				}
				deregResp.Body.Close()
			}
		}
	}

	return tc, cleanup
}

// registerTestServices registers a standard set of test services via the
// Consul agent REST API.
func registerTestServices(t *testing.T, tc *testHTTPClient) {
	t.Helper()

	testServices := []*agentServiceRegistration{
		{
			ID:      "test-web-1",
			Name:    "test-web",
			Tags:    []string{"http", "production", "v1"},
			Port:    8080,
			Address: "10.0.1.10",
			Meta: map[string]string{
				"version":     "1.0.0",
				"environment": "prod",
			},
			Check: &agentServiceCheck{TTL: "30s"},
		},
		{
			ID:      "test-web-2",
			Name:    "test-web",
			Tags:    []string{"http", "production", "v1"},
			Port:    8080,
			Address: "10.0.1.11",
			Meta: map[string]string{
				"version":     "1.0.0",
				"environment": "prod",
			},
			Check: &agentServiceCheck{TTL: "30s"},
		},
		{
			ID:      "test-api-1",
			Name:    "test-api",
			Tags:    []string{"grpc", "production"},
			Port:    9090,
			Address: "10.0.1.20",
			Meta: map[string]string{
				"version": "2.0.0",
			},
			Check: &agentServiceCheck{TTL: "30s"},
		},
		{
			ID:      "test-db-1",
			Name:    "test-db",
			Tags:    []string{"postgres", "critical"},
			Port:    5432,
			Address: "10.0.1.30",
			Meta:    map[string]string{},
			Check:   &agentServiceCheck{TTL: "30s"},
		},
	}

	for _, svc := range testServices {
		if err := tc.put("/v1/agent/service/register", svc); err != nil {
			t.Fatalf("Failed to register service %s: %v", svc.ID, err)
		}

		// Mark the TTL check as passing.
		checkID := "service:" + svc.ID
		if err := tc.put("/v1/agent/check/update/"+checkID, updateTTLRequest{Status: "passing"}); err != nil {
			t.Logf("Failed to update TTL for %s: %v", svc.ID, err)
		}
	}

	// Allow services to propagate.
	time.Sleep(500 * time.Millisecond)
}

func TestIntegrationConsulRDSBasic(t *testing.T) {
	tc, cleanup := setupTestConsul(t)
	defer cleanup()

	registerTestServices(t, tc)

	config := &configpb.ProviderConfig{
		Address:  proto.String(consulTestAddr),
		Services: &configpb.ServicesConfig{},
	}

	l := &logger.Logger{}
	provider, err := New(config, l)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Wait for initial refresh
	time.Sleep(2 * time.Second)

	// List all services
	resp, err := provider.ListResources(&pb.ListResourcesRequest{
		ResourcePath: proto.String("services"),
	})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	// Count test services
	testServiceCount := 0
	for _, res := range resp.Resources {
		if strings.HasPrefix(res.GetName(), "test-") {
			testServiceCount++
		}
	}

	if testServiceCount < 4 {
		t.Errorf("Expected at least 4 test services, got %d", testServiceCount)
	}

	t.Logf("Successfully discovered %d test services", testServiceCount)
}

func TestIntegrationConsulRDSFiltering(t *testing.T) {
	tc, cleanup := setupTestConsul(t)
	defer cleanup()

	registerTestServices(t, tc)

	config := &configpb.ProviderConfig{
		Address: proto.String(consulTestAddr),
		Services: &configpb.ServicesConfig{
			TagFilter: []string{"production"},
		},
	}

	l := &logger.Logger{}
	provider, err := New(config, l)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Wait for initial refresh
	time.Sleep(2 * time.Second)

	tests := []struct {
		name    string
		filters []*pb.Filter
		wantMin int
		wantMax int
	}{
		{
			name:    "filter by web service name",
			filters: []*pb.Filter{{Key: proto.String("name"), Value: proto.String("test-web")}},
			wantMin: 2,
			wantMax: 2,
		},
		{
			name:    "filter by tag",
			filters: []*pb.Filter{{Key: proto.String("tag"), Value: proto.String("http")}},
			wantMin: 2,
			wantMax: 2,
		},
		{
			name:    "filter by label",
			filters: []*pb.Filter{{Key: proto.String("labels.meta_version"), Value: proto.String("1.0.0")}},
			wantMin: 2,
			wantMax: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := provider.ListResources(&pb.ListResourcesRequest{
				ResourcePath: proto.String("services"),
				Filter:       tt.filters,
			})
			if err != nil {
				t.Fatalf("ListResources() error = %v", err)
			}

			count := len(resp.Resources)
			if count < tt.wantMin || count > tt.wantMax {
				t.Errorf("Got %d resources, want between %d and %d", count, tt.wantMin, tt.wantMax)
			}

			t.Logf("Filter %s returned %d resources", tt.name, count)
		})
	}
}

func TestIntegrationConsulRDSHealthFiltering(t *testing.T) {
	tc, cleanup := setupTestConsul(t)
	defer cleanup()

	registerTestServices(t, tc)

	// Mark the db service check as critical.
	if err := tc.put("/v1/agent/check/update/service:test-db-1", updateTTLRequest{Status: "critical"}); err != nil {
		t.Fatalf("Failed to mark service as critical: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	tests := []struct {
		name          string
		healthStatus  []string
		shouldInclude string
		shouldExclude string
	}{
		{
			name:          "only passing services",
			healthStatus:  []string{"passing"},
			shouldInclude: "test-web",
			shouldExclude: "test-db",
		},
		{
			name:          "only critical services",
			healthStatus:  []string{"critical"},
			shouldInclude: "test-db",
			shouldExclude: "test-web",
		},
		{
			name:          "passing and critical services",
			healthStatus:  []string{"passing", "critical"},
			shouldInclude: "test-web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &configpb.ProviderConfig{
				Address: proto.String(consulTestAddr),
				Services: &configpb.ServicesConfig{
					HealthStatus: tt.healthStatus,
				},
			}

			l := &logger.Logger{}
			provider, err := New(config, l)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			// Wait for initial refresh
			time.Sleep(2 * time.Second)

			resp, err := provider.ListResources(&pb.ListResourcesRequest{
				ResourcePath: proto.String("services"),
			})
			if err != nil {
				t.Fatalf("ListResources() error = %v", err)
			}

			foundInclude := false
			foundExclude := false

			for _, res := range resp.Resources {
				if strings.HasPrefix(res.GetName(), tt.shouldInclude) {
					foundInclude = true
				}
				if tt.shouldExclude != "" && strings.HasPrefix(res.GetName(), tt.shouldExclude) {
					foundExclude = true
				}
			}

			if !foundInclude {
				t.Errorf("Expected to find %s in results", tt.shouldInclude)
			}

			if tt.shouldExclude != "" && foundExclude {
				t.Errorf("Did not expect to find %s in results", tt.shouldExclude)
			}
		})
	}
}

func TestIntegrationConsulRDSMetadataLabels(t *testing.T) {
	tc, cleanup := setupTestConsul(t)
	defer cleanup()

	registerTestServices(t, tc)

	config := &configpb.ProviderConfig{
		Address: proto.String(consulTestAddr),
		Services: &configpb.ServicesConfig{
			IncludeMetadata: proto.Bool(true),
			IncludeTags:     proto.Bool(true),
		},
	}

	l := &logger.Logger{}
	provider, err := New(config, l)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Wait for initial refresh
	time.Sleep(2 * time.Second)

	resp, err := provider.ListResources(&pb.ListResourcesRequest{
		ResourcePath: proto.String("services"),
		Filter: []*pb.Filter{
			{Key: proto.String("name"), Value: proto.String("test-web")},
		},
	})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if len(resp.Resources) == 0 {
		t.Fatal("Expected to find test-web services")
	}

	// Check first resource for expected labels
	res := resp.Resources[0]

	// These labels can be propagated to probe metrics using the standard
	// Cloudprober additional_label feature with @target.label.KEY@ syntax.
	expectedLabels := map[string]string{
		"service":          "test-web",
		"health":           "passing",
		"meta_version":     "1.0.0",
		"meta_environment": "prod",
		"tag_http":         "true",
		"tag_production":   "true",
	}

	for key, expectedValue := range expectedLabels {
		if gotValue, exists := res.Labels[key]; !exists {
			t.Errorf("Missing expected label %s (would be @target.label.%s@ in probe config)", key, key)
		} else if gotValue != expectedValue {
			t.Errorf("Label %s = %s, want %s", key, gotValue, expectedValue)
		}
	}

	t.Logf("Successfully verified metadata labels on resource %s", res.GetName())
	t.Logf("These labels can be propagated to metrics using @target.label.KEY@ syntax")
}

func TestIntegrationConsulRDSRefresh(t *testing.T) {
	tc, cleanup := setupTestConsul(t)
	defer cleanup()

	registerTestServices(t, tc)

	config := &configpb.ProviderConfig{
		Address:   proto.String(consulTestAddr),
		Services:  &configpb.ServicesConfig{},
		ReEvalSec: proto.Int32(3), // Refresh every 3 seconds
	}

	l := &logger.Logger{}
	provider, err := New(config, l)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Wait for initial refresh
	time.Sleep(2 * time.Second)

	// Get initial count
	resp1, err := provider.ListResources(&pb.ListResourcesRequest{
		ResourcePath: proto.String("services"),
		Filter: []*pb.Filter{
			{Key: proto.String("name"), Value: proto.String("test-web")},
		},
	})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	initialCount := len(resp1.Resources)
	t.Logf("Initial count: %d services", initialCount)

	// Register a new service
	newService := &agentServiceRegistration{
		ID:      "test-web-3",
		Name:    "test-web",
		Tags:    []string{"http", "production"},
		Port:    8080,
		Address: "10.0.1.12",
		Check:   &agentServiceCheck{TTL: "30s"},
	}

	if err := tc.put("/v1/agent/service/register", newService); err != nil {
		t.Fatalf("Failed to register new service: %v", err)
	}

	if err := tc.put("/v1/agent/check/update/service:test-web-3", updateTTLRequest{Status: "passing"}); err != nil {
		t.Logf("Failed to update TTL: %v", err)
	}

	// Wait for next refresh cycle
	time.Sleep(5 * time.Second)

	// Check if new service is discovered
	resp2, err := provider.ListResources(&pb.ListResourcesRequest{
		ResourcePath: proto.String("services"),
		Filter: []*pb.Filter{
			{Key: proto.String("name"), Value: proto.String("test-web")},
		},
	})
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	newCount := len(resp2.Resources)
	t.Logf("After refresh count: %d services", newCount)

	if newCount <= initialCount {
		t.Errorf("Expected service count to increase after adding new service, got %d (was %d)", newCount, initialCount)
	}

	// Cleanup the new service
	deregReq, _ := http.NewRequest("PUT", "http://"+consulTestAddr+"/v1/agent/service/deregister/test-web-3", nil)
	tc.httpClient.Do(deregReq) //nolint:errcheck
}

func TestIntegrationConsulRDSConcurrentRequests(t *testing.T) {
	tc, cleanup := setupTestConsul(t)
	defer cleanup()

	registerTestServices(t, tc)

	config := &configpb.ProviderConfig{
		Address:  proto.String(consulTestAddr),
		Services: &configpb.ServicesConfig{},
	}

	l := &logger.Logger{}
	provider, err := New(config, l)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Wait for initial refresh
	time.Sleep(2 * time.Second)

	// Make concurrent requests
	numRequests := 10
	errChan := make(chan error, numRequests)
	doneChan := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			resp, err := provider.ListResources(&pb.ListResourcesRequest{
				ResourcePath: proto.String("services"),
			})
			if err != nil {
				errChan <- fmt.Errorf("request %d failed: %v", id, err)
				return
			}
			if len(resp.Resources) == 0 {
				errChan <- fmt.Errorf("request %d returned no resources", id)
				return
			}
			doneChan <- true
		}(i)
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < numRequests; i++ {
		select {
		case err := <-errChan:
			t.Error(err)
		case <-doneChan:
			successCount++
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	if successCount != numRequests {
		t.Errorf("Only %d/%d concurrent requests succeeded", successCount, numRequests)
	}

	t.Logf("Successfully completed %d concurrent requests", successCount)
}
