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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cpb "github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto"
	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// testK8sClient returns a kubeapi client that talks to a test server running
// the given handler.
func testK8sClient(t *testing.T, handler http.HandlerFunc) *client {
	t.Helper()

	ts := httptest.NewTLSServer(handler)
	t.Cleanup(ts.Close)

	return &client{
		cfg:     &cpb.ProviderConfig{},
		apiHost: strings.TrimPrefix(ts.URL, "https://"),
		httpC:   ts.Client(),
	}
}

// TestListerAPIPaths verifies that each lister asks the API server for the
// right resource, in both the all-namespaces and single-namespace cases. It
// goes through the real constructors on purpose: an api prefix or kind wired
// up wrong is invisible to every other test, and in production it just returns
// no targets.
func TestListerAPIPaths(t *testing.T) {
	listers := []struct {
		kind    string
		start   func(ns string, kc *client)
		wantAll string
		wantNS  string
	}{
		{
			kind:    "pods",
			start:   func(ns string, kc *client) { newPodsLister(ns, time.Hour, kc, nil) },
			wantAll: "/api/v1/pods",
			wantNS:  "/api/v1/namespaces/test-ns/pods",
		},
		{
			kind:    "endpoints",
			start:   func(ns string, kc *client) { newEndpointsLister(ns, time.Hour, kc, nil) },
			wantAll: "/api/v1/endpoints",
			wantNS:  "/api/v1/namespaces/test-ns/endpoints",
		},
		{
			kind:    "services",
			start:   func(ns string, kc *client) { newServicesLister(ns, time.Hour, kc, nil) },
			wantAll: "/api/v1/services",
			wantNS:  "/api/v1/namespaces/test-ns/services",
		},
		{
			kind:    "ingresses",
			start:   func(ns string, kc *client) { newIngressesLister(ns, time.Hour, kc, nil) },
			wantAll: "/apis/networking.k8s.io/v1/ingresses",
			wantNS:  "/apis/networking.k8s.io/v1/namespaces/test-ns/ingresses",
		},
	}

	for _, lister := range listers {
		for _, test := range []struct {
			desc, ns, want string
		}{
			{"all-namespaces", "", lister.wantAll},
			{"single-namespace", "test-ns", lister.wantNS},
		} {
			t.Run(lister.kind+"/"+test.desc, func(t *testing.T) {
				paths := make(chan string, 1)
				kc := testK8sClient(t, func(w http.ResponseWriter, r *http.Request) {
					select {
					case paths <- r.URL.Path:
					default:
					}
					w.Write([]byte(`{"items":[]}`))
				})

				lister.start(test.ns, kc)

				select {
				case got := <-paths:
					assert.Equal(t, test.want, got, "API server path")
				case <-time.After(10 * time.Second):
					t.Fatal("timed out waiting for the lister to call the API server")
				}
			})
		}
	}
}

// TestExpandOnError verifies that a failed refresh keeps the resources we
// already have. Replacing them with an empty list would tell everything
// downstream that these targets are gone, which is not what a failed API call
// means: an emptied cache reaches the RDS client as a successful response with
// zero resources, not as an error.
func TestExpandOnError(t *testing.T) {
	const onePod = `{"items":[{"metadata":{"name":"pod-b","namespace":"ns"},"status":{"phase":"Running","podIP":"10.0.0.2"}}]}`

	tests := []struct {
		desc      string
		handler   http.HandlerFunc
		wantNames []string
	}{
		{
			desc:      "API error keeps existing resources",
			handler:   func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", http.StatusInternalServerError) },
			wantNames: []string{"pod-a"},
		},
		{
			desc:      "unparseable response keeps existing resources",
			handler:   func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("not json")) },
			wantNames: []string{"pod-a"},
		},
		{
			desc:      "empty response clears resources",
			handler:   func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"items":[]}`)) },
			wantNames: nil,
		},
		{
			desc:      "good response replaces resources",
			handler:   func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(onePod)) },
			wantNames: []string{"pod-b"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			existing := resourceKey{"ns", "pod-a"}
			pl := &podsLister{
				kind:      "pods",
				apiPrefix: "api/v1",
				keep:      runningPod,
				kClient:   testK8sClient(t, test.handler),
				keys:      []resourceKey{existing},
				cache: map[resourceKey]*podInfo{
					existing: testPodInfo("pod-a", "ns", "10.0.0.1", map[string]string{}),
				},
			}

			pl.expand()

			var gotNames []string
			for _, key := range pl.keys {
				gotNames = append(gotNames, key.name)
			}
			assert.Equal(t, test.wantNames, gotNames, "cached resource names")
			assert.Len(t, pl.cache, len(test.wantNames), "cache size")
		})
	}
}

// waitForCache waits for a lister's first refresh to land.
func waitForCache[T resourceInfo](t *testing.T, rl *resourceLister[T]) {
	t.Helper()

	assert.Eventually(t, func() bool {
		rl.mu.RLock()
		defer rl.mu.RUnlock()
		return rl.cache != nil
	}, 10*time.Second, 5*time.Millisecond, "lister did not populate its cache")
}

func listRequest(resourcePath string) *pb.ListResourcesRequest {
	return &pb.ListResourcesRequest{ResourcePath: proto.String(resourcePath)}
}

const (
	podsFixture = `{"items":[
		{"metadata":{"name":"pod-a","namespace":"ns"},"status":{"phase":"Running","podIP":"10.0.0.1"}},
		{"metadata":{"name":"pod-b","namespace":"ns"},"status":{"phase":"Running","podIP":"10.0.0.2"}},
		{"metadata":{"name":"pod-pending","namespace":"ns"},"status":{"phase":"Pending","podIP":"10.0.0.3"}}
	]}`

	servicesFixture = `{"items":[
		{"metadata":{"name":"svc-a","namespace":"ns"},"spec":{"clusterIP":"10.1.1.1","ports":[{"port":80}]}},
		{"metadata":{"name":"svc-b","namespace":"ns"},"spec":{"clusterIP":"10.1.1.2","ports":[{"port":80}]}}
	]}`
)

func serveFixture(t *testing.T, body string) *client {
	return testK8sClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
}

func resourceNames(resources []*pb.Resource) []string {
	var names []string
	for _, res := range resources {
		names = append(names, res.GetName())
	}
	return names
}

// TestNameInPath pins the deliberate difference in how the listers treat a
// name in the resource path: services, endpoints and ingresses select a single
// object with it, pods ignore it and return everything. It builds the listers
// through their real constructors so that the wiring is covered too.
func TestNameInPath(t *testing.T) {
	pl := newPodsLister("", time.Hour, serveFixture(t, podsFixture), nil)
	waitForCache(t, pl)

	got, err := pl.listResources(listRequest("pods/pod-a"))
	assert.NoError(t, err)
	assert.Equal(t, []string{"pod-a", "pod-b"}, resourceNames(got), "pods ignore a name in the resource path")

	sl := newServicesLister("", time.Hour, serveFixture(t, servicesFixture), nil)
	waitForCache(t, sl)

	got, err = sl.listResources(listRequest("services/svc-a"))
	assert.NoError(t, err)
	assert.Equal(t, []string{"svc-a"}, resourceNames(got), "services select a single object by resource path")

	got, err = sl.listResources(listRequest("services"))
	assert.NoError(t, err)
	assert.Equal(t, []string{"svc-a", "svc-b"}, resourceNames(got), "no name in path returns everything")
}

// TestPodsListerCachesOnlyRunningPods covers the pods lister's parse-time
// filter, including its wiring in newPodsLister.
func TestPodsListerCachesOnlyRunningPods(t *testing.T) {
	pl := newPodsLister("", time.Hour, serveFixture(t, podsFixture), nil)
	waitForCache(t, pl)

	got, err := pl.listResources(listRequest("pods"))
	assert.NoError(t, err)
	assert.Equal(t, []string{"pod-a", "pod-b"}, resourceNames(got), "pods that are not running should not be cached")
}
