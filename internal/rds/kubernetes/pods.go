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
	"time"

	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

type podsLister = resourceLister[*podInfo]

type podInfo struct {
	Metadata kMetadata
	Status   struct {
		Phase string
		PodIP string
	}
}

func (pi *podInfo) metadata() kMetadata {
	return pi.Metadata
}

// runningPod is the pods lister's parse-time filter: we cache only the pods
// that are running.
func runningPod(pi *podInfo) bool {
	return pi.Status.Phase == "Running"
}

// resources returns the RDS resource for a pod. Pods map one-to-one.
func (pi *podInfo) resources(f *listFilters, l *logger.Logger) []*pb.Resource {
	if !f.matchesObject(pi.Metadata, l) {
		return nil
	}

	return []*pb.Resource{
		{
			Name:   proto.String(pi.Metadata.Name),
			Ip:     proto.String(pi.Status.PodIP),
			Labels: pi.Metadata.Labels,
		},
	}
}

func newPodsLister(namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) *podsLister {
	pl := &podsLister{
		kind:      "pods",
		apiPrefix: "api/v1",
		namespace: namespace,
		kClient:   kc,
		keep:      runningPod,
		l:         l,
	}
	pl.start(reEvalInterval)
	return pl
}
