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

package kubernetes

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"
)

// resourceInfo is implemented by the Kubernetes objects that we cache:
// podInfo, epInfo, serviceInfo, and ingressInfo. It provides the two things
// the shared lister machinery needs from each object type: its metadata, and
// how it expands into RDS resources.
type resourceInfo interface {
	metadata() kMetadata

	// resources expands a Kubernetes object into the RDS resources it
	// represents: zero or more, depending on the type. Implementations are
	// responsible for applying the name and labels filters, as the two are
	// intertwined with the expansion (see listFilters.matches).
	resources(f *listFilters, l *logger.Logger) []*pb.Resource
}

// listFilters holds everything a lister needs from a ListResources request:
// the filters, and the object name if the request's resource path named one
// (e.g. "services/service-a"), which selects just that object.
type listFilters struct {
	*filter.Filters
	name   string
	ipType pb.IPConfig_IPType
}

func parseListFilters(req *pb.ListResourcesRequest) (*listFilters, error) {
	filters, err := filter.ParseFilters(req.GetFilter(), SupportedFilters.RegexFilterKeys, "")
	if err != nil {
		return nil, err
	}

	f := &listFilters{Filters: filters, ipType: req.GetIpConfig().GetIpType()}
	if tok := strings.SplitN(req.GetResourcePath(), "/", 2); len(tok) == 2 {
		f.name = tok[1]
	}
	return f, nil
}

// regex returns the regex filter for a key, nil if the request didn't set it.
func (f *listFilters) regex(key string) *filter.RegexFilter {
	return f.RegexFilters[key]
}

// matches reports whether a name and a set of labels pass the request's name
// and labels filters. Callers differ in what they pass: pods, endpoints and
// services match on the Kubernetes object's own name and labels, while
// ingresses match the resources they expand into, whose names and labels are
// derived per rule.
func (f *listFilters) matches(name string, labels map[string]string, l *logger.Logger) bool {
	if nameFilter := f.regex("name"); nameFilter != nil && !nameFilter.Match(name, l) {
		return false
	}
	if f.LabelsFilter != nil && !f.LabelsFilter.Match(labels, l) {
		return false
	}
	return true
}

// resourceLister caches one Kubernetes resource type and refreshes it from the
// API server every reEvalInterval. Everything that varies by resource type is
// either a field here or lives in the type's resources() method.
type resourceLister[T resourceInfo] struct {
	kind      string // e.g. "pods"; used in the API URL and in log messages
	apiPrefix string // e.g. "api/v1"
	namespace string // empty means all namespaces
	kClient   *client

	// keep, if set, filters objects at parse time. Only pods use it, to cache
	// only the pods that are running.
	keep func(T) bool

	mu    sync.RWMutex // protects keys and cache
	keys  []resourceKey
	cache map[resourceKey]T
	l     *logger.Logger
}

func (rl *resourceLister[T]) url() string {
	if rl.namespace == "" {
		return fmt.Sprintf("%s/%s", rl.apiPrefix, rl.kind)
	}
	return fmt.Sprintf("%s/namespaces/%s/%s", rl.apiPrefix, rl.namespace, rl.kind)
}

func (rl *resourceLister[T]) listResources(req *pb.ListResourcesRequest) ([]*pb.Resource, error) {
	f, err := parseListFilters(req)
	if err != nil {
		return nil, err
	}

	nsFilter := f.regex("namespace")

	rl.mu.RLock()
	defer rl.mu.RUnlock()

	var resources []*pb.Resource
	for _, key := range rl.keys {
		if f.name != "" && key.name != f.name {
			continue
		}

		info := rl.cache[key]
		if nsFilter != nil && !nsFilter.Match(info.metadata().Namespace, rl.l) {
			continue
		}

		resources = append(resources, info.resources(f, rl.l)...)
	}

	rl.l.Debugf("kubernetes.%s.listResources: returning %d resources", rl.kind, len(resources))
	return resources, nil
}

// parseResourceList parses an API server list response into cache keys and the
// objects they map to. Objects rejected by keep are left out of both.
func parseResourceList[T resourceInfo](resp []byte, keep func(T) bool) ([]resourceKey, map[resourceKey]T, error) {
	var itemList struct {
		Items []T
	}

	if err := json.Unmarshal(resp, &itemList); err != nil {
		return nil, nil, err
	}

	keys := make([]resourceKey, 0, len(itemList.Items))
	cache := make(map[resourceKey]T, len(itemList.Items))
	for _, item := range itemList.Items {
		if keep != nil && !keep(item) {
			continue
		}
		md := item.metadata()
		key := resourceKey{md.Namespace, md.Name}
		keys = append(keys, key)
		cache[key] = item
	}

	return keys, cache, nil
}

// expand refreshes the cache from the API server. If the refresh fails we keep
// the resources we already have: an empty cache means "these targets are gone"
// to everyone downstream, which is not what a failed API call tells us.
func (rl *resourceLister[T]) expand() {
	resp, err := rl.kClient.getURL(rl.url())
	if err != nil {
		rl.l.Warningf("kubernetes.%s: error getting %s list from API, keeping existing resources: %v", rl.kind, rl.kind, err)
		return
	}

	keys, cache, err := parseResourceList(resp, rl.keep)
	if err != nil {
		rl.l.Warningf("kubernetes.%s: error parsing %s API response (%s), keeping existing resources: %v", rl.kind, rl.kind, string(resp), err)
		return
	}

	rl.l.Debugf("kubernetes.%s: got %d %s", rl.kind, len(keys), rl.kind)

	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.keys = keys
	rl.cache = cache
}

// start does the initial expansion and then refreshes in the background.
func (rl *resourceLister[T]) start(reEvalInterval time.Duration) {
	go func() {
		rl.expand()

		// Introduce a random delay between 0-reEvalInterval before starting
		// the refresh loop. If there are multiple cloudprober instances, this
		// makes sure that each one calls the API server at a different point
		// of time.
		time.Sleep(time.Duration(rand.Int63n(int64(reEvalInterval))))

		ticker := time.NewTicker(reEvalInterval)
		defer ticker.Stop()
		for range ticker.C {
			rl.expand()
		}
	}()
}
