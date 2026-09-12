// Copyright 2020 The Cloudprober Authors.
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

package kubernetes

import (
	"fmt"
	"strings"
	"time"

	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

type ingressesLister = resourceLister[*ingressInfo]

type ingressRule struct {
	Host string
	HTTP struct {
		Paths []struct {
			Path string
		}
	}
}

type ingressInfo struct {
	Metadata kMetadata
	Spec     struct {
		Rules []ingressRule
	}
	Status struct {
		LoadBalancer loadBalancerStatus
	}
}

func (i *ingressInfo) metadata() kMetadata {
	return i.Metadata
}

// resources returns RDS resources corresponding to an ingress resource.
//
// Unlike the other resource types, the name and labels filters are applied to
// the expanded resources rather than to the ingress object: an ingress expands
// into one resource per rule path, each with a derived name and its own fqdn
// and relative_url labels, and filtering on the object would make those
// unselectable.
func (i *ingressInfo) resources(f *listFilters, l *logger.Logger) (resources []*pb.Resource) {
	resName := i.Metadata.Name
	baseLabels := i.Metadata.Labels

	// Note that for ingress we don't check the type of the IP in the request.
	// That is mainly because ingresses typically have only one ingress
	// controller and hence one IP address. Also, the difference of private vs
	// public IP doesn't really exist.
	var ip string
	if len(i.Status.LoadBalancer.Ingress) > 0 {
		ii := i.Status.LoadBalancer.Ingress[0]
		ip = ii.IP
		if ip == "" && ii.Hostname != "" {
			ip = ii.Hostname
		}
	}

	appendIfMatches := func(res *pb.Resource) {
		if f.matchesResource(res, l) {
			resources = append(resources, res)
		}
	}

	if len(i.Spec.Rules) == 0 {
		appendIfMatches(&pb.Resource{
			Name:   proto.String(resName),
			Labels: baseLabels,
			Ip:     proto.String(ip),
		})
		return
	}

	for _, rule := range i.Spec.Rules {
		nameWithHost := fmt.Sprintf("%s_%s", resName, rule.Host)

		for _, p := range rule.HTTP.Paths {
			nameWithPath := nameWithHost
			if p.Path != "/" {
				nameWithPath = fmt.Sprintf("%s_%s", nameWithHost, strings.Replace(p.Path, "/", "_", -1))
			}

			// Add fqdn and url labels to the resources.
			labels := make(map[string]string, len(baseLabels)+2)
			for k, v := range baseLabels {
				labels[k] = v
			}
			if _, ok := labels["fqdn"]; !ok {
				labels["fqdn"] = rule.Host
			}
			if _, ok := labels["relative_url"]; !ok {
				labels["relative_url"] = p.Path
			}

			appendIfMatches(&pb.Resource{
				Name:   proto.String(nameWithPath),
				Labels: labels,
				Ip:     proto.String(ip),
			})
		}
	}

	return
}

func newIngressesLister(namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) *ingressesLister {
	lister := &ingressesLister{
		kind:       "ingresses",
		apiPrefix:  "apis/networking.k8s.io/v1",
		namespace:  namespace,
		kClient:    kc,
		nameInPath: true,
		l:          l,
	}
	lister.start(reEvalInterval)
	return lister
}
