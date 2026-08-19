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
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

type httpRoutesLister struct {
	c         *configpb.HTTPRoutes
	namespace string
	kClient   *client

	mu    sync.RWMutex // Mutex for names and cache
	keys  []resourceKey
	cache map[resourceKey]*httpRouteInfo
	l     *logger.Logger
}

func httpRoutesURL(ns string) string {
	if ns == "" {
		return "apis/gateway.networking.k8s.io/v1/httproutes"
	}
	return fmt.Sprintf("apis/gateway.networking.k8s.io/v1/namespaces/%s/httproutes", ns)
}

func (lister *httpRoutesLister) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	var resources []*pb.Resource

	var resName string
	tok := strings.SplitN(req.GetResourcePath(), "/", 2)
	if len(tok) == 2 {
		resName = tok[1]
	}

	allFilters, err := filter.ParseFilters(req.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	nameFilter, nsFilter, labelsFilter := allFilters.RegexFilters["name"], allFilters.RegexFilters["namespace"], allFilters.LabelsFilter

	lister.mu.RLock()
	defer lister.mu.RUnlock()

	for _, key := range lister.keys {
		if resName != "" && key.name != resName {
			continue
		}

		route := lister.cache[key]
		if nsFilter != nil && !nsFilter.Match(route.Metadata.Namespace, lister.l) {
			continue
		}

		for _, res := range route.resources() {
			if nameFilter != nil && !nameFilter.Match(res.GetName(), lister.l) {
				continue
			}
			if labelsFilter != nil && !labelsFilter.Match(res.GetLabels(), lister.l) {
				continue
			}
			resources = append(resources, res)
		}
	}

	lister.l.Debugf("kubernetes.listResources: returning %d http routes", len(resources))
	return resources, nil
}

type httpRouteMatch struct {
	Path struct {
		Type  string
		Value string
	}
}

type httpRouteRule struct {
	Hostnames []string
	Matches   []httpRouteMatch
}

type httpRouteInfo struct {
	Metadata kMetadata
	Spec     struct {
		Hostnames []string
		Rules     []httpRouteRule
	}
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
func (i *httpRouteInfo) resources() (resources []*pb.Resource) {
	resName := i.Metadata.Name
	baseLabels := i.Metadata.Labels
	routeHosts := i.Spec.Hostnames

	for _, rule := range i.Spec.Rules {
		// Rule-level hostnames override the route-level ones.
		hosts := rule.Hostnames
		if len(hosts) == 0 {
			hosts = routeHosts
		}
		if len(hosts) == 0 {
			continue
		}

		// A rule with no matches matches all paths; treat it as "/".
		matches := rule.Matches
		if len(matches) == 0 {
			matches = []httpRouteMatch{{}}
		}

		for _, host := range hosts {
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

				resources = append(resources, &pb.Resource{
					Name:   proto.String(nameWithPath),
					Labels: labels,
					Ip:     proto.String(host),
				})
			}
		}
	}

	// If no resources were generated (e.g. the route has no hostnames), emit a
	// single resource named after the route.
	if len(resources) == 0 {
		resources = append(resources, &pb.Resource{
			Name:   proto.String(resName),
			Labels: baseLabels,
		})
	}

	return
}

func parseHTTPRoutesJSON(resp []byte) (keys []resourceKey, routes map[resourceKey]*httpRouteInfo, err error) {
	var itemList struct {
		Items []*httpRouteInfo
	}

	if err = json.Unmarshal(resp, &itemList); err != nil {
		return
	}

	keys = make([]resourceKey, len(itemList.Items))
	routes = make(map[resourceKey]*httpRouteInfo)
	for i, item := range itemList.Items {
		keys[i] = resourceKey{item.Metadata.Namespace, item.Metadata.Name}
		routes[keys[i]] = item
	}

	return
}

func (lister *httpRoutesLister) expand() {
	resp, err := lister.kClient.getURL(httpRoutesURL(lister.namespace))
	if err != nil {
		lister.l.Warningf("httpRoutesLister.expand(): error while getting http routes list from API: %v", err)
	}

	keys, routes, err := parseHTTPRoutesJSON(resp)
	if err != nil {
		lister.l.Warningf("httpRoutesLister.expand(): error while parsing http routes API response (%s): %v", string(resp), err)
	}

	lister.l.Debugf("httpRoutesLister.expand(): got %d http routes", len(keys))

	lister.mu.Lock()
	defer lister.mu.Unlock()
	lister.keys = keys
	lister.cache = routes
}

func newHTTPRoutesLister(c *configpb.HTTPRoutes, namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) (*httpRoutesLister, error) {
	lister := &httpRoutesLister{
		c:         c,
		kClient:   kc,
		namespace: namespace,
		l:         l,
	}

	go func() {
		lister.expand()
		// Introduce a random delay between 0-reEvalInterval before
		// starting the refresh loop. If there are multiple cloudprober
		// gceInstances, this will make sure that each instance calls GCE
		// API at a different point of time.
		rand.Seed(time.Now().UnixNano())
		randomDelaySec := rand.Intn(int(reEvalInterval.Seconds()))
		time.Sleep(time.Duration(randomDelaySec) * time.Second)
		ticker := time.NewTicker(reEvalInterval)
		defer ticker.Stop()
		for range ticker.C {
			lister.expand()
		}
	}()

	return lister, nil
}
