// Copyright 2026 The Cloudprober Authors.
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

type httpRoutesLister = resourceLister[*httpRouteInfo]

type httpRouteMatch struct {
	Path struct {
		Type  string
		Value string
	}
}

type httpRouteRule struct {
	Matches []httpRouteMatch
}

type httpRouteInfo struct {
	Metadata kMetadata
	Spec     struct {
		Hostnames []string
		Rules     []httpRouteRule
	}
}

func (i *httpRouteInfo) metadata() kMetadata {
	return i.Metadata
}

// resources returns RDS resources corresponding to an HTTPRoute resource.
// Each route can have multiple hostnames and rules, and each rule can in turn
// have multiple path matches. We emit one RDS resource per (hostname, path)
// pair, mirroring how ingresses are expanded.
//
// Note that, unlike ingresses, an HTTPRoute does not carry a load balancer IP
// in its status. The address to connect to is the parent Gateway's address,
// which is not resolved here. We therefore use the route hostname as the
// target IP; the probe resolves it via DNS. The hostname is also exposed as
// the "fqdn" label so HTTP probes can set the correct Host header / SNI.
//
// Since the hostname is the only thing we can probe, routes without
// hostnames (which inherit them from the Gateway listener) and wildcard
// hostnames (e.g. "*.example.com") produce no resources.
//
// As with ingresses, the name and labels filters apply to the expanded
// resources rather than to the route object, because each resource carries a
// derived name and its own fqdn and relative_url labels.
func (i *httpRouteInfo) resources(f *listFilters, l *logger.Logger) (resources []*pb.Resource) {
	resName := i.Metadata.Name
	baseLabels := i.Metadata.resourceLabels()

	for _, rule := range i.Spec.Rules {
		// A rule with no matches matches all paths; treat it as "/".
		matches := rule.Matches
		if len(matches) == 0 {
			matches = []httpRouteMatch{{}}
		}

		for _, host := range i.Spec.Hostnames {
			if strings.HasPrefix(host, "*") {
				continue
			}

			for _, m := range matches {
				path := m.Path.Value
				if path == "" {
					path = "/"
				}

				nameWithPath := fmt.Sprintf("%s_%s", resName, host)
				if path != "/" {
					nameWithPath = fmt.Sprintf("%s_%s", nameWithPath, strings.Replace(path, "/", "_", -1))
				}

				// Add fqdn and url labels to the resources.
				labels := make(map[string]string, len(baseLabels)+2)
				for k, v := range baseLabels {
					labels[k] = v
				}
				if _, ok := labels["fqdn"]; !ok {
					labels["fqdn"] = host
				}
				if _, ok := labels["relative_url"]; !ok {
					labels["relative_url"] = path
				}

				if f.matches(nameWithPath, labels, l) {
					resources = append(resources, &pb.Resource{
						Name:   proto.String(nameWithPath),
						Labels: labels,
						Ip:     proto.String(host),
					})
				}
			}
		}
	}
	return
}

func newHTTPRoutesLister(namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) *httpRoutesLister {
	lister := &httpRoutesLister{
		kind:      "httproutes",
		apiPrefix: "apis/gateway.networking.k8s.io/v1",
		namespace: namespace,
		kClient:   kc,
		l:         l,
	}
	lister.start(reEvalInterval)
	return lister
}
