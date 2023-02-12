package targets

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

func initRDSServer(namespace string, labelSelector []string, resourceType string, reEvalSec int, l *logger.Logger) (*server.Server, error) {
	global.mu.Lock()
	defer global.mu.Unlock()

	if global.servers == nil {
		global.servers = make(map[string]*server.Server)
	}

	k := key(namespace, labelSelector, resourceType)
	if global.servers[k] != nil {
		return global.servers[k], nil
	}

	kc := kubernetes.DefaultProviderConfig(namespace, labelSelector, resourceType, reEvalSec)
	srv, err := server.New(context.Background(), &serverconfigpb.ServerConf{Provider: []*serverconfigpb.Provider{kc}}, nil, l)
	if err != nil {
		return nil, err
	}

	global.servers[k] = srv
	return srv, nil
}

func k8sTargets(pb *targetspb.K8STargets, l *logger.Logger) (*rdsclient.Client, error) {
	var resources string

	switch pb.GetResources().(type) {
	case *targetspb.K8STargets_Endpoints:
		resources = "endpoints"
	case *targetspb.K8STargets_Services:
		resources = "services"
	case *targetspb.K8STargets_Ingresses:
		resources = "ingresses"
	case *targetspb.K8STargets_Pods:
		resources = "pods"
	}

	if pb.GetRdsServerOptions() != nil {
		return rdsclient.New(&rdsclientpb.ClientConf{
			ServerOptions: pb.GetRdsServerOptions(),
			Request: &rdspb.ListResourcesRequest{
				Provider:     proto.String("k8s"),
				ResourcePath: &resources,
				IpConfig:     pb.GetIpConfig(),
			},
		}, nil, l)
	}

	s, err := initRDSServer(pb.GetNamespace(), pb.GetLabelSelector(), resources, int(pb.GetReEvalSec()), l)
	if err != nil {
		return nil, fmt.Errorf("k8s: error creating resource discovery server: %v", err)
	}
	return rdsclient.New(nil, s.ListResources, l)
}
