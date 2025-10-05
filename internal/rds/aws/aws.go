// Copyright 2017-2023 The Cloudprober Authors.
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
Package aws implements a AWS (Amazon Web Services) resources provider for
ResourceDiscovery server.

See ResourceTypes variable for the list of supported resource types.

AWS provider is configured through a protobuf based config file
(proto/config.proto). Example config:

	{
		aws_instances {}
	}
*/

package aws

import (
	"fmt"

	configpb "github.com/cloudprober/cloudprober/internal/rds/aws/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	serverconfigpb "github.com/cloudprober/cloudprober/internal/rds/server/proto"
	"github.com/cloudprober/cloudprober/logger"

	"google.golang.org/protobuf/proto"
)

// DefaultProviderID is the povider id to use for this provider if a provider
// id is not configured explicitly.
const DefaultProviderID = "aws"

// ResourceTypes declares resource types supported by the AWS provider.
var ResourceTypes = struct {
	EC2Instances, ElastiCacheClusters, ElastiCacheReplicationGroups, RDSClusters, RDSInstances string
}{
	"ec2_instances",
	"elasticache_clusters",
	"elasticache_replicationgroups",
	"rds_clusters",
	"rds_instances",
}

type lister interface {
	listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error)
}

// Provider implements a AWS provider for a ResourceDiscovery server.
type Provider struct {
	listers map[string]lister
}

// ListResources returns the list of resources based on the given request.
func (p *Provider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	lr := p.listers[req.GetResourcePath()]
	if lr == nil {
		return nil, fmt.Errorf("unknown resource type: %s", req.GetResourcePath())
	}

	resources, err := lr.listResources(req)
	return &pb.ListResourcesResponse{Resources: resources}, err
}

func initAWSProject(c *configpb.ProviderConfig, l *logger.Logger) (map[string]lister, error) {
	resourceLister := make(map[string]lister)

	// TODO when resources are added

	return resourceLister, nil
}

// New creates a AWS provider for RDS server, based on the provided config.
func New(c *configpb.ProviderConfig, l *logger.Logger) (*Provider, error) {
	resourceLister, err := initAWSProject(c, l)
	if err != nil {
		return nil, err
	}

	return &Provider{
		listers: resourceLister,
	}, nil
}

// DefaultProviderConfig is a convenience function that builds and returns a
// basic AWS provider config based on the given parameters.
func DefaultProviderConfig(resTypes map[string]string, reEvalSec int) *serverconfigpb.Provider {
	c := &configpb.ProviderConfig{}

	for k := range resTypes {
		switch k {
		case ResourceTypes.EC2Instances:
			c.Ec2Instances = &configpb.EC2Instances{
				ReEvalSec: proto.Int32(int32(reEvalSec)),
			}

		case ResourceTypes.ElastiCacheClusters:
			c.ElasticacheClusters = &configpb.ElastiCacheClusters{
				ReEvalSec: proto.Int32(int32(reEvalSec)),
			}

		case ResourceTypes.ElastiCacheReplicationGroups:
			c.ElasticacheReplicationgroups = &configpb.ElastiCacheReplicationGroups{
				ReEvalSec: proto.Int32(int32(reEvalSec)),
			}

		case ResourceTypes.RDSInstances:
			c.RdsInstances = &configpb.RDSInstances{
				ReEvalSec: proto.Int32(int32(reEvalSec)),
			}

		case ResourceTypes.RDSClusters:
			c.RdsClusters = &configpb.RDSClusters{
				ReEvalSec: proto.Int32(int32(reEvalSec)),
			}
		}

	}

	return &serverconfigpb.Provider{
		Id:     proto.String(DefaultProviderID),
		Config: &serverconfigpb.Provider_AwsConfig{AwsConfig: c},
	}
}
