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

/*
Package consul implements Consul-based targets for cloudprober.
*/
package consul

import (
	"context"
	"fmt"

	"github.com/cloudprober/cloudprober/internal/rds/client"
	client_configpb "github.com/cloudprober/cloudprober/internal/rds/client/proto"
	"github.com/cloudprober/cloudprober/internal/rds/consul"
	consul_configpb "github.com/cloudprober/cloudprober/internal/rds/consul/proto"
	rdspb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/targets/consul/proto"
	dnsRes "github.com/cloudprober/cloudprober/targets/resolver"
	"google.golang.org/protobuf/proto"
)

// New returns new Consul targets.
func New(opts *configpb.TargetsConf, res dnsRes.Resolver, l *logger.Logger) (*client.Client, error) {
	// Build Consul provider config
	providerConfig := &consul_configpb.ProviderConfig{
		Address:    opts.Address,
		Datacenter: opts.Datacenter,
		Token:      opts.Token,
		ReEvalSec:  opts.ReEvalSec,
	}

	// Determine resource type and configure accordingly
	var resourcePath string

	switch opts.Resources.(type) {
	case *configpb.TargetsConf_Services:
		resourcePath = "services"
		providerConfig.Services = &consul_configpb.ServicesConfig{
			NameFilter:    opts.GetServices(),
			TagFilter:     opts.Tags,
			HealthStatus:  opts.HealthStatus,
		}
	case *configpb.TargetsConf_HealthChecks:
		resourcePath = "health_checks"
		providerConfig.HealthChecks = &consul_configpb.HealthChecksConfig{
			ServiceName: opts.HealthChecks,
		}
	case *configpb.TargetsConf_Nodes:
		resourcePath = "nodes"
		providerConfig.Nodes = &consul_configpb.NodesConfig{}
	default:
		return nil, fmt.Errorf("no resource type specified (services, health_checks, or nodes)")
	}

	// Create Consul RDS provider
	provider, err := consul.New(providerConfig, l)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul provider: %v", err)
	}

	// Create RDS client configuration
	clientConf := &client_configpb.ClientConf{
		Request: &rdspb.ListResourcesRequest{
			ResourcePath: proto.String(resourcePath),
			Filter:       opts.GetFilter(),
		},
		ReEvalSec: proto.Int32(opts.GetReEvalSec()),
	}

	// Return RDS client wrapped around the Consul provider
	return client.New(clientConf, func(_ context.Context, req *rdspb.ListResourcesRequest) (*rdspb.ListResourcesResponse, error) {
		return provider.ListResources(req)
	}, l)
}
