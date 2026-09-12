// Copyright 2019 The Cloudprober Authors.
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
Package kubernetes implements a kubernetes resources provider for
ResourceDiscovery server.

See:

	ResourceTypes variable for the list of supported resource types.
	SupportedFilters variable for the list of supported filters.

Kubernetes provider is configured through a protobuf based config file
(proto/config.proto). Example config:

	{
		pods {}
	}
*/
package kubernetes

import (
	"fmt"
	"slices"
	"strings"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
)

// DefaultProviderID is the povider id to use for this provider if a provider
// id is not configured explicitly.
const DefaultProviderID = "k8s"

// ResourceTypes declares resource types supported by the Kubernetes provider.
var ResourceTypes = struct {
	Pods, Endpoints, Services, Ingresses string
}{
	"pods",
	"endpoints",
	"services",
	"ingresses",
}

/*
SupportedFilters defines filters supported by this provider.

	 Example filters:
	 filter {
		 key: "name"
		 value: "cloudprober.*"
	 }
	 filter {
		 key: "namespace"
		 value: "teamx.*"
	 }
	 filter {
		 key: "port"
		 value: "http.*"
	 }
	 filter {
		 key: "labels.app"
		 value: "service-a"
	 }
*/
var SupportedFilters = struct {
	RegexFilterKeys []string
	LabelsFilter    bool
}{
	// Note: the port filter applies only to endpoints and services.
	[]string{"name", "namespace", "port"},
	true,
}

type lister interface {
	listResources(*pb.ListResourcesRequest) ([]*pb.Resource, error)
}

// Provider implements a Kubernetes (K8s) provider for use with a
// ResourceDiscovery server.
type Provider struct {
	listers map[string]lister
}

// kMetadata represents metadata for all Kubernetes resources.
type kMetadata struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

type resourceKey struct {
	namespace, name string
}

func fixRegexFiltersInRequest(req *pb.ListResourcesRequest) *pb.ListResourcesRequest {
	for _, filter := range req.GetFilter() {
		if slices.Contains(SupportedFilters.RegexFilterKeys, filter.GetKey()) {
			v := filter.GetValue()
			if v[0] != '^' {
				v = "^" + v
			}
			if v[len(v)-1] != '$' {
				v = v + "$"
			}
			filter.Value = &v
		}
	}
	return req
}

// ListResources returns the list of resources from the cache.
func (p *Provider) ListResources(req *pb.ListResourcesRequest) (*pb.ListResourcesResponse, error) {
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)

	resType := tok[0]

	lr := p.listers[resType]
	if lr == nil {
		return nil, fmt.Errorf("kubernetes: unsupported resource type: %s", resType)
	}

	resources, err := lr.listResources(fixRegexFiltersInRequest(req))
	return &pb.ListResourcesResponse{Resources: resources}, err
}

// New creates a Kubernetes (k8s) provider for RDS server, based on the
// provided config.
func New(c *configpb.ProviderConfig, l *logger.Logger) (*Provider, error) {
	// Guard the refresh loop's ticker, which panics on a non-positive
	// interval. Note that an unset re_eval_sec defaults to 60, so this
	// catches only an explicit zero or a negative value.
	if c.GetReEvalSec() <= 0 {
		return nil, fmt.Errorf("kubernetes: re_eval_sec (%d) must be positive", c.GetReEvalSec())
	}

	client, err := newClient(c, l)
	if err != nil {
		return nil, fmt.Errorf("error while creating the kubernetes client: %v", err)
	}

	p := &Provider{
		listers: make(map[string]lister),
	}

	reEvalInterval := time.Duration(c.GetReEvalSec()) * time.Second

	// Enable a lister for each resource type that is configured.
	if c.GetPods() != nil {
		p.listers[ResourceTypes.Pods] = newPodsLister(c.GetNamespace(), reEvalInterval, client, l)
	}
	if c.GetEndpoints() != nil {
		p.listers[ResourceTypes.Endpoints] = newEndpointsLister(c.GetNamespace(), reEvalInterval, client, l)
	}
	if c.GetServices() != nil {
		p.listers[ResourceTypes.Services] = newServicesLister(c.GetNamespace(), reEvalInterval, client, l)
	}
	if c.GetIngresses() != nil {
		p.listers[ResourceTypes.Ingresses] = newIngressesLister(c.GetNamespace(), reEvalInterval, client, l)
	}

	return p, nil
}
