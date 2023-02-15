// Copyright 2023 The Cloudprober Authors.
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

package targets

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/cloudprober/cloudprober/logger"
	rdsclient "github.com/cloudprober/cloudprober/rds/client"
	rdsclientpb "github.com/cloudprober/cloudprober/rds/client/proto"
	"github.com/cloudprober/cloudprober/rds/kubernetes"
	k8sconfigpb "github.com/cloudprober/cloudprober/rds/kubernetes/proto"
	rdspb "github.com/cloudprober/cloudprober/rds/proto"
	"github.com/cloudprober/cloudprober/rds/server"
	serverconfigpb "github.com/cloudprober/cloudprober/rds/server/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/golang/protobuf/proto"
)

var global struct {
	mu      sync.RWMutex
	servers map[string]*server.Server
}

func key(namespace string, labelSelector []string, resourceType string) string {
	sort.Strings(labelSelector)
	return strings.Join([]string{namespace, strings.Join(labelSelector, ","), resourceType}, "+")
}

func initRDSServer(k string, kpc *k8sconfigpb.ProviderConfig, l *logger.Logger) (*server.Server, error) {
	global.mu.Lock()
	defer global.mu.Unlock()

	if global.servers == nil {
		global.servers = make(map[string]*server.Server)
	}

	if global.servers[k] != nil {
		return global.servers[k], nil
	}

	kc := &serverconfigpb.Provider{
		Id:     proto.String(kubernetes.DefaultProviderID),
		Config: &serverconfigpb.Provider_KubernetesConfig{KubernetesConfig: kpc},
	}

	srv, err := server.New(context.Background(), &serverconfigpb.ServerConf{Provider: []*serverconfigpb.Provider{kc}}, nil, l)
	if err != nil {
		return nil, err
	}

	global.servers[k] = srv
	return srv, nil
}

func kubernetesProviderConfig(pb *targetspb.K8STargets) (*k8sconfigpb.ProviderConfig, string) {
	pc := &k8sconfigpb.ProviderConfig{
		Namespace:     proto.String(pb.GetNamespace()),
		LabelSelector: pb.GetLabelSelector(),
		ReEvalSec:     proto.Int32(int32(pb.GetReEvalSec())),
	}

	switch pb.GetResources().(type) {
	case *targetspb.K8STargets_Endpoints:
		pc.Endpoints = &k8sconfigpb.Endpoints{}
		return pc, "endpoints"
	case *targetspb.K8STargets_Services:
		pc.Services = &k8sconfigpb.Services{}
		return pc, "services"
	case *targetspb.K8STargets_Ingresses:
		pc.Ingresses = &k8sconfigpb.Ingresses{}
		return pc, "ingresses"
	case *targetspb.K8STargets_Pods:
		pc.Pods = &k8sconfigpb.Pods{}
		return pc, "pods"
	}

	return nil, ""
}

func k8sTargets(pb *targetspb.K8STargets, l *logger.Logger) (*rdsclient.Client, error) {
	pc, resources := kubernetesProviderConfig(pb)

	if pb.GetRdsServerOptions() != nil {
		return rdsclient.New(&rdsclientpb.ClientConf{
			ServerOptions: pb.GetRdsServerOptions(),
			Request: &rdspb.ListResourcesRequest{
				Provider:     proto.String("k8s"),
				ResourcePath: &resources,
			},
			// No caching in RDS client, but server already caches.
			ReEvalSec: proto.Int32(0),
		}, nil, l)
	}

	s, err := initRDSServer(key(pb.GetNamespace(), pb.GetLabelSelector(), resources), pc, l)
	if err != nil {
		return nil, fmt.Errorf("k8s: error creating resource discovery server: %v", err)
	}
	return rdsclient.New(nil, s.ListResources, l)
}
