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
	"fmt"
	"strconv"
	"time"

	pb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

type epLister = resourceLister[*epInfo]

type epSubset struct {
	Addresses []struct {
		IP        string
		NodeName  string
		TargetRef struct {
			Kind string
			Name string
		}
	}
	Ports []struct {
		Name string
		Port int
	}
}

type epInfo struct {
	Metadata kMetadata
	Subsets  []epSubset
}

func (epi *epInfo) metadata() kMetadata {
	return epi.Metadata
}

// resources returns RDS resources corresponding to an endpoints resource. Each
// endpoints object can have multiple endpoint subsets and each subset in turn
// is composed of multiple addresses and ports. If an endpoint subset as 3
// addresses and 2 ports, there will be 6 resources corresponding to that
// subset.
func (epi *epInfo) resources(f *listFilters, l *logger.Logger) (resources []*pb.Resource) {
	baseLabels := epi.Metadata.resourceLabels()
	if !f.matches(epi.Metadata.Name, baseLabels, l) {
		return nil
	}

	portFilter := f.regex("port")
	for _, eps := range epi.Subsets {
		// There is usually one port, but there can be multiple ports, e.g. 9313
		// and 9314.
		for _, port := range eps.Ports {
			// For unnamed ports, use port number.
			portName := port.Name
			if portName == "" {
				portName = strconv.FormatInt(int64(port.Port), 10)
			}

			if portFilter != nil && !portFilter.Match(portName, l) {
				continue
			}

			for _, addr := range eps.Addresses {
				// We name the resource as <endpoints_name>_<IP>_<port>
				resName := fmt.Sprintf("%s_%s_%s", epi.Metadata.Name, addr.IP, portName)

				labels := make(map[string]string, len(baseLabels)+2)
				for k, v := range baseLabels {
					labels[k] = v
				}
				// As with namespace, the object's own labels take precedence.
				if _, ok := labels["node"]; !ok {
					labels["node"] = addr.NodeName
				}
				if _, ok := labels["pod"]; !ok && addr.TargetRef.Kind == "Pod" {
					labels["pod"] = addr.TargetRef.Name
				}

				resources = append(resources, &pb.Resource{
					Name:   proto.String(resName),
					Ip:     proto.String(addr.IP),
					Port:   proto.Int32(int32(port.Port)),
					Labels: labels,
				})
			}
		}
	}
	return
}

func newEndpointsLister(namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) *epLister {
	lister := &epLister{
		kind:      "endpoints",
		apiPrefix: "api/v1",
		namespace: namespace,
		kClient:   kc,
		l:         l,
	}
	lister.start(reEvalInterval)
	return lister
}
