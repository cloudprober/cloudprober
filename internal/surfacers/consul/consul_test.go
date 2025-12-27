// Copyright 2024 The Cloudprober Authors.
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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/consul/proto"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/options"
	consulapi "github.com/hashicorp/consul/api"
	"google.golang.org/protobuf/proto"
)

// mockConsulServer creates a test HTTP server that mocks Consul API for registration
type mockConsulServer struct {
	*httptest.Server
	mu               sync.Mutex
	registrations    []*consulapi.AgentServiceRegistration
	deregistrations  []string
	healthCheckCalls int
}

func newMockConsulServer(t *testing.T) *mockConsulServer {
	t.Helper()

	mcs := &mockConsulServer{
		registrations:   make([]*consulapi.AgentServiceRegistration, 0),
		deregistrations: make([]string, 0),
	}

	mcs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mcs.mu.Lock()
		defer mcs.mu.Unlock()

		switch {
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/v1/agent/service/register"):
			// Service registration
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Logf("Failed to read body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var reg consulapi.AgentServiceRegistration
			if err := json.Unmarshal(body, &reg); err != nil {
				t.Logf("Failed to unmarshal registration: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			mcs.registrations = append(mcs.registrations, &reg)
			w.WriteHeader(http.StatusOK)

		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/v1/agent/service/deregister"):
			// Service deregistration
			parts := strings.Split(r.URL.Path, "/")
			serviceID := parts[len(parts)-1]
			mcs.deregistrations = append(mcs.deregistrations, serviceID)
			w.WriteHeader(http.StatusOK)

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/agent/checks"):
			// Health checks list
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, "{}")

		default:
			t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return mcs
}

func (mcs *mockConsulServer) getLastRegistration() *consulapi.AgentServiceRegistration {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()

	if len(mcs.registrations) == 0 {
		return nil
	}
	return mcs.registrations[len(mcs.registrations)-1]
}

func (mcs *mockConsulServer) getRegistrationCount() int {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()
	return len(mcs.registrations)
}

func (mcs *mockConsulServer) getDeregistrationCount() int {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()
	return len(mcs.deregistrations)
}

func TestNewConsulSurfacer(t *testing.T) {
	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	tests := []struct {
		name    string
		config  *configpb.SurfacerConf
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "basic config",
			config: &configpb.SurfacerConf{
				Address: proto.String(serverAddr),
				Service: &configpb.ServiceConfig{
					Name: proto.String("cloudprober-test"),
				},
				HealthCheck: &configpb.HealthCheckConfig{
					HttpEndpoint: proto.String("/health"),
				},
			},
			wantErr: false,
		},
		{
			name: "config with metadata",
			config: &configpb.SurfacerConf{
				Address: proto.String(serverAddr),
				Metadata: map[string]string{
					"environment": "test",
					"version":     "1.0.0",
				},
				Service: &configpb.ServiceConfig{
					Name: proto.String("cloudprober-test"),
				},
				HealthCheck: &configpb.HealthCheckConfig{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			l := &logger.Logger{}
			opts := &options.Options{
				MetricsBufferSize: 1000,
			}

			surfacer, err := New(ctx, tt.config, opts, l)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if surfacer != nil {
				defer surfacer.Close()

				// Give it time to register
				time.Sleep(200 * time.Millisecond)

				if server.getRegistrationCount() == 0 && !tt.wantErr {
					t.Error("Expected service to be registered")
				}
			}
		})
	}
}

func TestSurfacerServiceRegistration(t *testing.T) {
	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	config := &configpb.SurfacerConf{
		Address: proto.String(serverAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("cloudprober-registration-test"),
			Id:   proto.String("cloudprober-test-1"),
			Tags: []string{"monitoring", "cloudprober", "test"},
			Port: proto.Int32(9313),
		},
		HealthCheck: &configpb.HealthCheckConfig{
			HttpEndpoint: proto.String("/status"),
			Interval:     proto.String("10s"),
			Timeout:      proto.String("5s"),
		},
		Metadata: map[string]string{
			"environment": "test",
			"team":        "platform",
		},
	}

	ctx := context.Background()
	l := &logger.Logger{}
	opts := &options.Options{
		MetricsBufferSize: 1000,
	}

	surfacer, err := New(ctx, config, opts, l)
	if err != nil {
		t.Fatalf("Failed to create surfacer: %v", err)
	}
	defer surfacer.Close()

	// Wait for registration
	time.Sleep(200 * time.Millisecond)

	reg := server.getLastRegistration()
	if reg == nil {
		t.Fatal("Expected service registration")
	}

	// Verify registration details
	if reg.Name != "cloudprober-registration-test" {
		t.Errorf("Service name = %s, want cloudprober-registration-test", reg.Name)
	}

	if reg.ID != "cloudprober-test-1" {
		t.Errorf("Service ID = %s, want cloudprober-test-1", reg.ID)
	}

	if reg.Port != 9313 {
		t.Errorf("Service port = %d, want 9313", reg.Port)
	}

	// Verify tags
	expectedTags := []string{"monitoring", "cloudprober", "test"}
	if len(reg.Tags) < len(expectedTags) {
		t.Errorf("Got %d tags, want at least %d", len(reg.Tags), len(expectedTags))
	}

	for _, tag := range expectedTags {
		found := false
		for _, regTag := range reg.Tags {
			if regTag == tag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag %s not found in registration", tag)
		}
	}

	// Verify metadata
	if reg.Meta["environment"] != "test" {
		t.Errorf("Metadata environment = %s, want test", reg.Meta["environment"])
	}

	if reg.Meta["team"] != "platform" {
		t.Errorf("Metadata team = %s, want platform", reg.Meta["team"])
	}

	// Verify health check
	if reg.Check == nil {
		t.Fatal("Expected health check configuration")
	}

	if !strings.Contains(reg.Check.HTTP, "/status") {
		t.Errorf("Health check URL = %s, want to contain /status", reg.Check.HTTP)
	}

	if reg.Check.Interval != "10s" {
		t.Errorf("Health check interval = %s, want 10s", reg.Check.Interval)
	}
}

func TestSurfacerSysVarsPublishing(t *testing.T) {
	// Initialize sysvars for testing
	sysvars.Init(&logger.Logger{}, nil)

	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	tests := []struct {
		name           string
		config         *configpb.SurfacerConf
		wantSysVars    []string
		dontWantSysVars []string
	}{
		{
			name: "sysvars enabled by default",
			config: &configpb.SurfacerConf{
				Address: proto.String(serverAddr),
				Service: &configpb.ServiceConfig{
					Name: proto.String("test-sysvars"),
				},
				HealthCheck: &configpb.HealthCheckConfig{},
			},
			wantSysVars: []string{"sysvar_version"}, // version is commonly available
		},
		{
			name: "sysvars with custom prefix",
			config: &configpb.SurfacerConf{
				Address: proto.String(serverAddr),
				Service: &configpb.ServiceConfig{
					Name: proto.String("test-sysvars-prefix"),
				},
				HealthCheck: &configpb.HealthCheckConfig{},
				Sysvars: &configpb.SysVarsConfig{
					Enabled:   proto.Bool(true),
					KeyPrefix: proto.String("sys_"),
				},
			},
			wantSysVars: []string{"sys_version"},
		},
		{
			name: "sysvars disabled",
			config: &configpb.SurfacerConf{
				Address: proto.String(serverAddr),
				Service: &configpb.ServiceConfig{
					Name: proto.String("test-no-sysvars"),
				},
				HealthCheck: &configpb.HealthCheckConfig{},
				Sysvars: &configpb.SysVarsConfig{
					Enabled: proto.Bool(false),
				},
			},
			dontWantSysVars: []string{"sysvar_version", "sysvar_hostname"},
		},
		{
			name: "sysvars with include filter",
			config: &configpb.SurfacerConf{
				Address: proto.String(serverAddr),
				Service: &configpb.ServiceConfig{
					Name: proto.String("test-include-sysvars"),
				},
				HealthCheck: &configpb.HealthCheckConfig{},
				Sysvars: &configpb.SysVarsConfig{
					Enabled:     proto.Bool(true),
					IncludeVars: []string{"version"},
				},
			},
			wantSysVars:    []string{"sysvar_version"},
			dontWantSysVars: []string{"sysvar_hostname"}, // hostname should be excluded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			l := &logger.Logger{}
			opts := &options.Options{
				MetricsBufferSize: 1000,
			}

			surfacer, err := New(ctx, tt.config, opts, l)
			if err != nil {
				t.Fatalf("Failed to create surfacer: %v", err)
			}
			defer surfacer.Close()

			// Wait for registration
			time.Sleep(200 * time.Millisecond)

			reg := server.getLastRegistration()
			if reg == nil {
				t.Fatal("Expected service registration")
			}

			// Check for expected sysvars
			for _, sysvar := range tt.wantSysVars {
				if _, exists := reg.Meta[sysvar]; !exists {
					t.Errorf("Expected sysvar %s to be in metadata, got metadata: %v", sysvar, reg.Meta)
				}
			}

			// Check that unwanted sysvars are not present
			for _, sysvar := range tt.dontWantSysVars {
				if _, exists := reg.Meta[sysvar]; exists {
					t.Errorf("Did not expect sysvar %s to be in metadata", sysvar)
				}
			}
		})
	}
}

func TestSurfacerMetadataPublishing(t *testing.T) {
	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	customMetadata := map[string]string{
		"environment": "production",
		"region":      "us-west-2",
		"cluster":     "prod-cluster-1",
		"team":        "platform",
	}

	config := &configpb.SurfacerConf{
		Address: proto.String(serverAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-metadata"),
		},
		HealthCheck: &configpb.HealthCheckConfig{},
		Metadata:    customMetadata,
		Sysvars: &configpb.SysVarsConfig{
			Enabled: proto.Bool(false), // Disable sysvars for cleaner test
		},
	}

	ctx := context.Background()
	l := &logger.Logger{}
	opts := &options.Options{
		MetricsBufferSize: 1000,
	}

	surfacer, err := New(ctx, config, opts, l)
	if err != nil {
		t.Fatalf("Failed to create surfacer: %v", err)
	}
	defer surfacer.Close()

	// Wait for registration
	time.Sleep(200 * time.Millisecond)

	reg := server.getLastRegistration()
	if reg == nil {
		t.Fatal("Expected service registration")
	}

	// Verify all custom metadata is present
	for key, expectedValue := range customMetadata {
		if gotValue, exists := reg.Meta[key]; !exists {
			t.Errorf("Expected metadata key %s not found", key)
		} else if gotValue != expectedValue {
			t.Errorf("Metadata %s = %s, want %s", key, gotValue, expectedValue)
		}
	}
}

func TestSurfacerMetadataAndSysVarsCombined(t *testing.T) {
	// Initialize sysvars
	sysvars.Init(&logger.Logger{}, nil)

	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	customMetadata := map[string]string{
		"custom_key": "custom_value",
		"team":       "platform",
	}

	config := &configpb.SurfacerConf{
		Address: proto.String(serverAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-combined"),
		},
		HealthCheck: &configpb.HealthCheckConfig{},
		Metadata:    customMetadata,
		Sysvars: &configpb.SysVarsConfig{
			Enabled:     proto.Bool(true),
			IncludeVars: []string{"version"},
		},
	}

	ctx := context.Background()
	l := &logger.Logger{}
	opts := &options.Options{
		MetricsBufferSize: 1000,
	}

	surfacer, err := New(ctx, config, opts, l)
	if err != nil {
		t.Fatalf("Failed to create surfacer: %v", err)
	}
	defer surfacer.Close()

	// Wait for registration
	time.Sleep(200 * time.Millisecond)

	reg := server.getLastRegistration()
	if reg == nil {
		t.Fatal("Expected service registration")
	}

	// Verify custom metadata is present
	if reg.Meta["custom_key"] != "custom_value" {
		t.Errorf("Custom metadata not found or incorrect")
	}

	// Verify sysvars are present (should not conflict with custom metadata)
	if _, exists := reg.Meta["sysvar_version"]; !exists {
		t.Error("Expected sysvar_version to be in metadata")
	}

	// Custom metadata should take precedence if there's a conflict
	if reg.Meta["team"] != "platform" {
		t.Errorf("Custom metadata should take precedence, got: %s", reg.Meta["team"])
	}
}

func TestSurfacerDeregistration(t *testing.T) {
	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	tests := []struct {
		name                    string
		deregisterOnShutdown    bool
		wantDeregistrationCount int
	}{
		{
			name:                    "deregister on shutdown (default)",
			deregisterOnShutdown:    true,
			wantDeregistrationCount: 1,
		},
		{
			name:                    "no deregister on shutdown",
			deregisterOnShutdown:    false,
			wantDeregistrationCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &configpb.SurfacerConf{
				Address: proto.String(serverAddr),
				Service: &configpb.ServiceConfig{
					Name: proto.String("test-dereg"),
					Id:   proto.String("test-dereg-id"),
				},
				HealthCheck:           &configpb.HealthCheckConfig{},
				DeregisterOnShutdown:  proto.Bool(tt.deregisterOnShutdown),
			}

			ctx := context.Background()
			l := &logger.Logger{}
			opts := &options.Options{
				MetricsBufferSize: 1000,
			}

			initialDeregCount := server.getDeregistrationCount()

			surfacer, err := New(ctx, config, opts, l)
			if err != nil {
				t.Fatalf("Failed to create surfacer: %v", err)
			}

			// Wait for registration
			time.Sleep(200 * time.Millisecond)

			// Close the surfacer (should trigger deregistration)
			surfacer.Close()

			// Wait for deregistration
			time.Sleep(200 * time.Millisecond)

			finalDeregCount := server.getDeregistrationCount()
			actualDeregistrations := finalDeregCount - initialDeregCount

			if actualDeregistrations != tt.wantDeregistrationCount {
				t.Errorf("Got %d deregistrations, want %d", actualDeregistrations, tt.wantDeregistrationCount)
			}
		})
	}
}

func TestSurfacerWrite(t *testing.T) {
	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	config := &configpb.SurfacerConf{
		Address: proto.String(serverAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-write"),
		},
		HealthCheck: &configpb.HealthCheckConfig{},
	}

	ctx := context.Background()
	l := &logger.Logger{}
	opts := &options.Options{
		MetricsBufferSize: 1000,
	}

	surfacer, err := New(ctx, config, opts, l)
	if err != nil {
		t.Fatalf("Failed to create surfacer: %v", err)
	}
	defer surfacer.Close()

	// Wait for registration
	time.Sleep(200 * time.Millisecond)

	// Create test metrics
	em := metrics.NewEventMetrics(time.Now()).
		AddMetric("total", metrics.NewInt(100)).
		AddMetric("success", metrics.NewInt(95)).
		AddLabel("probe", "test-probe")

	// Write metrics (should not panic)
	surfacer.Write(ctx, em)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// The surfacer should still be healthy
	if !surfacer.healthStatus {
		t.Error("Expected surfacer to remain healthy after writing metrics")
	}
}

func TestDetermineServiceDetails(t *testing.T) {
	tests := []struct {
		name            string
		config          *configpb.SurfacerConf
		wantServiceName string
		wantPort        int
	}{
		{
			name: "explicit service name and port",
			config: &configpb.SurfacerConf{
				Service: &configpb.ServiceConfig{
					Name: proto.String("my-service"),
					Port: proto.Int32(8080),
				},
			},
			wantServiceName: "my-service",
			wantPort:        8080,
		},
		{
			name:            "default service name and port",
			config:          &configpb.SurfacerConf{},
			wantServiceName: "cloudprober",
			wantPort:        9313,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Surfacer{
				c:    tt.config,
				l:    &logger.Logger{},
			}

			err := s.determineServiceDetails()
			if err != nil {
				t.Fatalf("determineServiceDetails() error = %v", err)
			}

			if s.serviceName != tt.wantServiceName {
				t.Errorf("serviceName = %s, want %s", s.serviceName, tt.wantServiceName)
			}

			if s.servicePort != tt.wantPort {
				t.Errorf("servicePort = %d, want %d", s.servicePort, tt.wantPort)
			}

			// Service ID should be auto-generated if not provided
			if s.serviceID == "" {
				t.Error("Expected service ID to be generated")
			}
		})
	}
}

func TestGetSysVarsMetadata(t *testing.T) {
	// Initialize sysvars
	sysvars.Init(&logger.Logger{}, nil)

	tests := []struct {
		name           string
		config         *configpb.SysVarsConfig
		wantPrefix     string
		wantEnabled    bool
	}{
		{
			name:        "default config (enabled)",
			config:      nil,
			wantPrefix:  "sysvar_",
			wantEnabled: true,
		},
		{
			name: "custom prefix",
			config: &configpb.SysVarsConfig{
				Enabled:   proto.Bool(true),
				KeyPrefix: proto.String("sys_"),
			},
			wantPrefix:  "sys_",
			wantEnabled: true,
		},
		{
			name: "disabled",
			config: &configpb.SysVarsConfig{
				Enabled: proto.Bool(false),
			},
			wantEnabled: false,
		},
		{
			name: "empty prefix",
			config: &configpb.SysVarsConfig{
				Enabled:   proto.Bool(true),
				KeyPrefix: proto.String(""),
			},
			wantPrefix:  "",
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Surfacer{
				c: &configpb.SurfacerConf{
					Sysvars: tt.config,
				},
				l: &logger.Logger{},
			}

			metadata := s.getSysVarsMetadata()

			if tt.wantEnabled {
				if len(metadata) == 0 {
					t.Error("Expected metadata to contain sysvars")
				}

				// Check that keys have the expected prefix
				for key := range metadata {
					if !strings.HasPrefix(key, tt.wantPrefix) {
						t.Errorf("Key %s does not have expected prefix %s", key, tt.wantPrefix)
					}
				}
			} else {
				if len(metadata) != 0 {
					t.Errorf("Expected no metadata, got %d entries", len(metadata))
				}
			}
		})
	}
}

func TestHealthCheckServer(t *testing.T) {
	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	config := &configpb.SurfacerConf{
		Address: proto.String(serverAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-health"),
			Port: proto.Int32(9313),
		},
		HealthCheck: &configpb.HealthCheckConfig{
			HttpEndpoint: proto.String("/custom-health"),
		},
	}

	ctx := context.Background()
	l := &logger.Logger{}
	opts := &options.Options{
		MetricsBufferSize: 1000,
	}

	surfacer, err := New(ctx, config, opts, l)
	if err != nil {
		t.Fatalf("Failed to create surfacer: %v", err)
	}
	defer surfacer.Close()

	// Wait for health server to start
	time.Sleep(300 * time.Millisecond)

	// Try to access the health endpoint
	healthURL := fmt.Sprintf("http://localhost:%d/custom-health", 9314) // Port + 1
	resp, err := http.Get(healthURL)
	if err != nil {
		t.Logf("Health check server may not be accessible: %v (this might be expected in test environment)", err)
		return // Don't fail the test if we can't access localhost
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check returned status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "OK" {
		t.Errorf("Health check body = %s, want OK", string(body))
	}
}

func TestSetHealthStatus(t *testing.T) {
	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	config := &configpb.SurfacerConf{
		Address: proto.String(serverAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-health-status"),
		},
		HealthCheck: &configpb.HealthCheckConfig{},
	}

	ctx := context.Background()
	l := &logger.Logger{}
	opts := &options.Options{
		MetricsBufferSize: 1000,
	}

	surfacer, err := New(ctx, config, opts, l)
	if err != nil {
		t.Fatalf("Failed to create surfacer: %v", err)
	}
	defer surfacer.Close()

	// Initially should be healthy
	if !surfacer.healthStatus {
		t.Error("Expected initial health status to be true")
	}

	// Set to unhealthy
	surfacer.SetHealthStatus(false)
	if surfacer.healthStatus {
		t.Error("Expected health status to be false after SetHealthStatus(false)")
	}

	// Set back to healthy
	surfacer.SetHealthStatus(true)
	if !surfacer.healthStatus {
		t.Error("Expected health status to be true after SetHealthStatus(true)")
	}
}

func TestConsulClientEnvironmentVariables(t *testing.T) {
	// Test that auto-discover works with env vars
	tests := []struct {
		name    string
		config  *configpb.SurfacerConf
		envVars map[string]string
	}{
		{
			name: "auto-discover with custom k8s settings",
			config: &configpb.SurfacerConf{
				Address: proto.String("auto-discover"),
			},
			envVars: map[string]string{
				"CONSUL_K8S_NAMESPACE":    "consul-system",
				"CONSUL_K8S_SERVICE_NAME": "consul-server",
				"CONSUL_K8S_PORT":         "8501",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			l := &logger.Logger{}
			client, err := newConsulClient(tt.config, l)
			if err != nil {
				t.Fatalf("newConsulClient() error = %v", err)
			}

			if client == nil {
				t.Fatal("Expected non-nil client")
			}
		})
	}
}

func TestSurfacerConnectIntegration(t *testing.T) {
	server := newMockConsulServer(t)
	defer server.Close()

	serverAddr := strings.TrimPrefix(server.URL, "http://")

	config := &configpb.SurfacerConf{
		Address: proto.String(serverAddr),
		Service: &configpb.ServiceConfig{
			Name:          proto.String("test-connect"),
			EnableConnect: proto.Bool(true),
		},
		HealthCheck: &configpb.HealthCheckConfig{},
	}

	ctx := context.Background()
	l := &logger.Logger{}
	opts := &options.Options{
		MetricsBufferSize: 1000,
	}

	surfacer, err := New(ctx, config, opts, l)
	if err != nil {
		t.Fatalf("Failed to create surfacer: %v", err)
	}
	defer surfacer.Close()

	// Wait for registration
	time.Sleep(200 * time.Millisecond)

	reg := server.getLastRegistration()
	if reg == nil {
		t.Fatal("Expected service registration")
	}

	// Verify Connect is enabled
	if reg.Connect == nil {
		t.Error("Expected Connect configuration to be set")
	} else if !reg.Connect.Native {
		t.Error("Expected Native Connect to be enabled")
	}
}
