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
	"os"
	"reflect"
	"testing"

	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"google.golang.org/protobuf/proto"
)

func httpRoutesListerFromDataFile(t *testing.T) *httpRoutesLister {
	t.Helper()

	httpRoutesListFile := "./testdata/httproutes.json"
	data, err := os.ReadFile(httpRoutesListFile)
	if err != nil {
		t.Fatalf("error reading test data file: %s", httpRoutesListFile)
	}
	keys, routes, err := parseHTTPRoutesJSON(data)
	if err != nil {
		t.Fatalf("Error while parsing http routes JSON data: %v", err)
	}

	return &httpRoutesLister{
		keys:  keys,
		cache: routes,
	}
}

func TestParseHTTPRoutesJSON(t *testing.T) {
	lister := httpRoutesListerFromDataFile(t)

	if len(lister.keys) != 2 {
		t.Errorf("Expected exactly two http routes, got: %d (%+v)", len(lister.keys), lister.cache)
	}

	key1 := resourceKey{"default", "rds-route"}
	key2 := resourceKey{"default", "rule-host-route"}
	if lister.cache[key1] == nil || lister.cache[key2] == nil {
		t.Errorf("Expected routes rds-route and rule-host-route, got: %+v", lister.cache)
	}
}

func TestListHTTPRouteResources(t *testing.T) {
	lister := httpRoutesListerFromDataFile(t)

	tests := []struct {
		desc      string
		filters   map[string]string
		wantNames []string
		wantFQDNs []string
		wantURLs  []string
		wantIPs   []string
	}{
		{
			desc:      "no filter",
			wantNames: []string{"rds-route_foo.bar.com__health", "rds-route_foo.bar.com__rds", "rule-host-route_bar.baz.com__api"},
			wantFQDNs: []string{"foo.bar.com", "foo.bar.com", "bar.baz.com"},
			wantURLs:  []string{"/health", "/rds", "/api"},
			wantIPs:   []string{"foo.bar.com", "foo.bar.com", "bar.baz.com"},
		},
		{
			desc:      "name filter for host regex",
			filters:   map[string]string{"name": ".*foo.bar.com.*"},
			wantNames: []string{"rds-route_foo.bar.com__health", "rds-route_foo.bar.com__rds"},
			wantFQDNs: []string{"foo.bar.com", "foo.bar.com"},
			wantURLs:  []string{"/health", "/rds"},
			wantIPs:   []string{"foo.bar.com", "foo.bar.com"},
		},
		{
			desc:      "name and label filter",
			filters:   map[string]string{"name": ".*foo.bar.com.*", "labels.relative_url": "/rds"},
			wantNames: []string{"rds-route_foo.bar.com__rds"},
			wantFQDNs: []string{"foo.bar.com"},
			wantURLs:  []string{"/rds"},
			wantIPs:   []string{"foo.bar.com"},
		},
		{
			desc:      "fqdn filter",
			filters:   map[string]string{"labels.fqdn": "bar.baz.com"},
			wantNames: []string{"rule-host-route_bar.baz.com__api"},
			wantFQDNs: []string{"bar.baz.com"},
			wantURLs:  []string{"/api"},
			wantIPs:   []string{"bar.baz.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var filters []*pb.Filter

			for k, v := range test.filters {
				filters = append(filters, &pb.Filter{
					Key:   proto.String(k),
					Value: proto.String(v),
				})
			}

			resources, err := lister.listResources(&pb.ListResourcesRequest{Filter: filters})
			if err != nil {
				t.Errorf("Error while listing resources: %v", err)
			}
			var gotNames, gotFQDNs, gotURLs, gotIPs []string
			for _, res := range resources {
				gotNames = append(gotNames, res.GetName())
				gotFQDNs = append(gotFQDNs, res.GetLabels()["fqdn"])
				gotURLs = append(gotURLs, res.GetLabels()["relative_url"])
				gotIPs = append(gotIPs, res.GetIp())
			}

			if !reflect.DeepEqual(gotNames, test.wantNames) {
				t.Errorf("gotNames: %v, wantNames: %v", gotNames, test.wantNames)
			}

			if !reflect.DeepEqual(gotFQDNs, test.wantFQDNs) {
				t.Errorf("gotFQDNs: %v, wantFQDNs: %v", gotFQDNs, test.wantFQDNs)
			}

			if !reflect.DeepEqual(gotURLs, test.wantURLs) {
				t.Errorf("gotURLs: %v, wantURLs: %v", gotURLs, test.wantURLs)
			}

			if !reflect.DeepEqual(gotIPs, test.wantIPs) {
				t.Errorf("gotIPs: %v, wantIPs: %v", gotIPs, test.wantIPs)
			}
		})
	}
}

// TestHTTPRouteNoHostnames checks that a route with no hostnames (neither
// route-level nor rule-level) produces a single resource named after the route.
func TestHTTPRouteNoHostnames(t *testing.T) {
	key := resourceKey{name: "no-host-route", namespace: "default"}
	lister := &httpRoutesLister{
		keys: []resourceKey{key},
		cache: map[resourceKey]*httpRouteInfo{
			key: {Metadata: kMetadata{Name: key.name, Namespace: key.namespace}},
		},
	}

	resources, err := lister.listResources(&pb.ListResourcesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 1 || resources[0].GetName() != "no-host-route" {
		t.Errorf("expected a single resource named no-host-route, got: %+v", resources)
	}
}
