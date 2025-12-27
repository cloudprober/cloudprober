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

//go:build integration
// +build integration

package consul

import (
	"context"
	"strings"
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

// Integration tests require a running Consul instance
// Run these tests with: go test -tags=integration
//
// To start a local Consul for testing:
//   docker run -d -p 8500:8500 --name consul-test consul:latest agent -dev -ui -client=0.0.0.0
//
// To stop:
//   docker stop consul-test && docker rm consul-test

const (
	consulTestAddr = "localhost:8500"
)

func setupConsulClient(t *testing.T) (*consulapi.Client, func()) {
	t.Helper()

	config := consulapi.DefaultConfig()
	config.Address = consulTestAddr

	client, err := consulapi.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create Consul client: %v", err)
	}

	// Check if Consul is accessible
	_, err = client.Agent().Self()
	if err != nil {
		t.Skipf("Consul not available at %s: %v\nRun integration tests with a local Consul instance", consulTestAddr, err)
	}

	cleanup := func() {
		// Deregister all test services
		services, err := client.Agent().Services()
		if err != nil {
			t.Logf("Failed to list services for cleanup: %v", err)
			return
		}

		for id := range services {
			if strings.HasPrefix(id, "test-integration-") {
				if err := client.Agent().ServiceDeregister(id); err != nil {
					t.Logf("Failed to deregister service %s: %v", id, err)
				}
			}
		}
	}

	return client, cleanup
}

func TestIntegrationSurfacerBasicRegistration(t *testing.T) {
	client, cleanup := setupConsulClient(t)
	defer cleanup()

	serviceID := "test-integration-basic"
	config := &configpb.SurfacerConf{
		Address: proto.String(consulTestAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-integration-cloudprober"),
			Id:   proto.String(serviceID),
			Tags: []string{"test", "integration"},
			Port: proto.Int32(19313),
		},
		HealthCheck: &configpb.HealthCheckConfig{
			HttpEndpoint: proto.String("/health"),
			Interval:     proto.String("30s"),
		},
		Sysvars: &configpb.SysVarsConfig{
			Enabled: proto.Bool(false), // Disable for simpler test
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
	time.Sleep(1 * time.Second)

	// Verify service is registered
	services, err := client.Agent().Services()
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	service, exists := services[serviceID]
	if !exists {
		t.Fatalf("Service %s not found in Consul", serviceID)
	}

	if service.Service != "test-integration-cloudprober" {
		t.Errorf("Service name = %s, want test-integration-cloudprober", service.Service)
	}

	if service.Port != 19313 {
		t.Errorf("Service port = %d, want 19313", service.Port)
	}

	// Verify tags
	expectedTags := []string{"test", "integration"}
	for _, tag := range expectedTags {
		found := false
		for _, serviceTag := range service.Tags {
			if serviceTag == tag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag %s not found", tag)
		}
	}

	t.Logf("Successfully registered and verified service %s", serviceID)
}

func TestIntegrationSurfacerMetadataPublishing(t *testing.T) {
	client, cleanup := setupConsulClient(t)
	defer cleanup()

	serviceID := "test-integration-metadata"
	metadata := map[string]string{
		"environment": "integration-test",
		"region":      "test-region",
		"version":     "1.2.3",
		"team":        "platform",
	}

	config := &configpb.SurfacerConf{
		Address: proto.String(consulTestAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-metadata"),
			Id:   proto.String(serviceID),
		},
		HealthCheck: &configpb.HealthCheckConfig{},
		Metadata:    metadata,
		Sysvars: &configpb.SysVarsConfig{
			Enabled: proto.Bool(false),
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
	time.Sleep(1 * time.Second)

	// Verify metadata is published
	services, err := client.Agent().Services()
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	service, exists := services[serviceID]
	if !exists {
		t.Fatalf("Service %s not found in Consul", serviceID)
	}

	// Check all metadata
	for key, expectedValue := range metadata {
		if gotValue, ok := service.Meta[key]; !ok {
			t.Errorf("Metadata key %s not found", key)
		} else if gotValue != expectedValue {
			t.Errorf("Metadata %s = %s, want %s", key, gotValue, expectedValue)
		}
	}

	t.Logf("Successfully verified %d metadata entries", len(metadata))
}

func TestIntegrationSurfacerSysVarsPublishing(t *testing.T) {
	// Initialize sysvars
	sysvars.Init(&logger.Logger{}, nil, nil, nil, nil)

	client, cleanup := setupConsulClient(t)
	defer cleanup()

	serviceID := "test-integration-sysvars"
	config := &configpb.SurfacerConf{
		Address: proto.String(consulTestAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-sysvars"),
			Id:   proto.String(serviceID),
		},
		HealthCheck: &configpb.HealthCheckConfig{},
		Sysvars: &configpb.SysVarsConfig{
			Enabled:   proto.Bool(true),
			KeyPrefix: proto.String("sysvar_"),
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
	time.Sleep(1 * time.Second)

	// Verify sysvars are published
	services, err := client.Agent().Services()
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	service, exists := services[serviceID]
	if !exists {
		t.Fatalf("Service %s not found in Consul", serviceID)
	}

	// Check for sysvar metadata (should have sysvar_ prefix)
	sysvarCount := 0
	for key := range service.Meta {
		if strings.HasPrefix(key, "sysvar_") {
			sysvarCount++
			t.Logf("Found sysvar: %s = %s", key, service.Meta[key])
		}
	}

	if sysvarCount == 0 {
		t.Error("Expected at least one sysvar to be published")
	}

	// Specifically check for version sysvar (commonly available)
	if _, exists := service.Meta["sysvar_version"]; !exists {
		t.Error("Expected sysvar_version to be published")
	}

	t.Logf("Successfully verified %d sysvars published", sysvarCount)
}

func TestIntegrationSurfacerSysVarsFiltering(t *testing.T) {
	// Initialize sysvars
	sysvars.Init(&logger.Logger{}, nil, nil, nil, nil)

	client, cleanup := setupConsulClient(t)
	defer cleanup()

	tests := []struct {
		name           string
		config         *configpb.SysVarsConfig
		wantVars       []string
		dontWantVars   []string
	}{
		{
			name: "include only version",
			config: &configpb.SysVarsConfig{
				Enabled:     proto.Bool(true),
				IncludeVars: []string{"version"},
				KeyPrefix:   proto.String("sys_"),
			},
			wantVars:     []string{"sys_version"},
			dontWantVars: []string{"sys_hostname"},
		},
		{
			name: "exclude version",
			config: &configpb.SysVarsConfig{
				Enabled:     proto.Bool(true),
				ExcludeVars: []string{"version"},
				KeyPrefix:   proto.String("sysvar_"),
			},
			dontWantVars: []string{"sysvar_version"},
		},
		{
			name: "custom prefix",
			config: &configpb.SysVarsConfig{
				Enabled:     proto.Bool(true),
				IncludeVars: []string{"version"},
				KeyPrefix:   proto.String("custom_"),
			},
			wantVars:     []string{"custom_version"},
			dontWantVars: []string{"sysvar_version", "sys_version"},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceID := fmt.Sprintf("test-integration-sysvars-filter-%d", i)
			config := &configpb.SurfacerConf{
				Address: proto.String(consulTestAddr),
				Service: &configpb.ServiceConfig{
					Name: proto.String("test-sysvars-filter"),
					Id:   proto.String(serviceID),
				},
				HealthCheck: &configpb.HealthCheckConfig{},
				Sysvars:     tt.config,
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
			time.Sleep(1 * time.Second)

			// Get service
			services, err := client.Agent().Services()
			if err != nil {
				t.Fatalf("Failed to get services: %v", err)
			}

			service, exists := services[serviceID]
			if !exists {
				t.Fatalf("Service %s not found", serviceID)
			}

			// Check wanted vars
			for _, wantVar := range tt.wantVars {
				if _, exists := service.Meta[wantVar]; !exists {
					t.Errorf("Expected var %s not found in metadata", wantVar)
				}
			}

			// Check unwanted vars
			for _, dontWantVar := range tt.dontWantVars {
				if _, exists := service.Meta[dontWantVar]; exists {
					t.Errorf("Did not expect var %s in metadata", dontWantVar)
				}
			}
		})
	}
}

func TestIntegrationSurfacerDeregistration(t *testing.T) {
	client, cleanup := setupConsulClient(t)
	defer cleanup()

	serviceID := "test-integration-dereg"
	config := &configpb.SurfacerConf{
		Address: proto.String(consulTestAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-dereg"),
			Id:   proto.String(serviceID),
		},
		HealthCheck:          &configpb.HealthCheckConfig{},
		DeregisterOnShutdown: proto.Bool(true),
		Sysvars: &configpb.SysVarsConfig{
			Enabled: proto.Bool(false),
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

	// Wait for registration
	time.Sleep(1 * time.Second)

	// Verify service is registered
	services, err := client.Agent().Services()
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	if _, exists := services[serviceID]; !exists {
		t.Fatalf("Service %s not registered", serviceID)
	}

	// Close surfacer (should deregister)
	surfacer.Close()

	// Wait for deregistration
	time.Sleep(1 * time.Second)

	// Verify service is deregistered
	services, err = client.Agent().Services()
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	if _, exists := services[serviceID]; exists {
		t.Errorf("Service %s should have been deregistered", serviceID)
	}

	t.Logf("Successfully verified deregistration of service %s", serviceID)
}

func TestIntegrationSurfacerWriteMetrics(t *testing.T) {
	client, cleanup := setupConsulClient(t)
	defer cleanup()

	serviceID := "test-integration-write"
	config := &configpb.SurfacerConf{
		Address: proto.String(consulTestAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-write"),
			Id:   proto.String(serviceID),
		},
		HealthCheck: &configpb.HealthCheckConfig{},
		Sysvars: &configpb.SysVarsConfig{
			Enabled: proto.Bool(false),
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
	time.Sleep(1 * time.Second)

	// Write metrics
	for i := 0; i < 10; i++ {
		em := metrics.NewEventMetrics(time.Now()).
			AddMetric("total", metrics.NewInt(100+int64(i))).
			AddMetric("success", metrics.NewInt(95+int64(i))).
			AddLabel("probe", "test-probe")

		surfacer.Write(ctx, em)
		time.Sleep(100 * time.Millisecond)
	}

	// Verify service still exists
	services, err := client.Agent().Services()
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	if _, exists := services[serviceID]; !exists {
		t.Error("Service should still be registered after writing metrics")
	}

	t.Log("Successfully wrote metrics without errors")
}

func TestIntegrationSurfacerMetadataPrecedence(t *testing.T) {
	// Initialize sysvars
	sysvars.Init(&logger.Logger{}, nil, nil, nil, nil)

	client, cleanup := setupConsulClient(t)
	defer cleanup()

	// Custom metadata that might conflict with sysvars
	serviceID := "test-integration-precedence"
	customMetadata := map[string]string{
		"version": "custom-version-override",
		"custom":  "custom-value",
	}

	config := &configpb.SurfacerConf{
		Address: proto.String(consulTestAddr),
		Service: &configpb.ServiceConfig{
			Name: proto.String("test-precedence"),
			Id:   proto.String(serviceID),
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
	time.Sleep(1 * time.Second)

	// Verify metadata precedence
	services, err := client.Agent().Services()
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	service, exists := services[serviceID]
	if !exists {
		t.Fatalf("Service %s not found", serviceID)
	}

	// Custom metadata should take precedence over sysvars
	if service.Meta["version"] != "custom-version-override" {
		t.Errorf("Custom metadata should take precedence, got version = %s", service.Meta["version"])
	}

	// Custom metadata should exist
	if service.Meta["custom"] != "custom-value" {
		t.Error("Custom metadata should exist")
	}

	// Sysvar version (with prefix) should also exist
	if _, exists := service.Meta["sysvar_version"]; !exists {
		t.Error("Expected sysvar_version to exist alongside custom version")
	}

	t.Log("Successfully verified metadata precedence")
}
