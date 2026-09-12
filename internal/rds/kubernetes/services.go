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
	"github.com/cloudprober/cloudprober/internal/rds/server/filter"
	"github.com/cloudprober/cloudprober/logger"
	"google.golang.org/protobuf/proto"
)

type servicesLister = resourceLister[*serviceInfo]

type loadBalancerStatus struct {
	Ingress []struct {
		IP       string
		Hostname string
	}
}

type serviceInfo struct {
	Metadata kMetadata
	Spec     struct {
		ClusterIP string
		Ports     []struct {
			Name string
			Port int
		}
	}
	Status struct {
		LoadBalancer loadBalancerStatus
	}
}

func (si *serviceInfo) metadata() kMetadata {
	return si.Metadata
}

func (si *serviceInfo) matchPorts(portFilter *filter.RegexFilter, l *logger.Logger) ([]int, map[int]string) {
	ports, portNameMap := []int{}, make(map[int]string)
	for _, port := range si.Spec.Ports {
		// For unnamed ports, use port number.
		portName := port.Name
		if portName == "" {
			portName = strconv.FormatInt(int64(port.Port), 10)
		}

		if portFilter != nil && !portFilter.Match(portName, l) {
			continue
		}
		ports = append(ports, port.Port)
		portNameMap[port.Port] = portName
	}
	return ports, portNameMap
}

// resources returns RDS resources corresponding to a service resource. Each
// service object can have multiple ports.
//
// a) If service has only 1 port or there is a port filter and only one port
// matches the port filter, we return only one RDS resource with same name as
// service name.
// b) If there are multiple ports, we create one RDS resource for each port and
// name each resource as: <service_name>_<port_name>
func (si *serviceInfo) resources(f *listFilters, l *logger.Logger) (resources []*pb.Resource) {
	if !f.matchesObject(si.Metadata, l) {
		return nil
	}

	ports, portNameMap := si.matchPorts(f.regex("port"), l)
	for _, port := range ports {
		resName := si.Metadata.Name
		if len(ports) != 1 {
			resName = fmt.Sprintf("%s_%s", si.Metadata.Name, portNameMap[port])
		}

		res := &pb.Resource{
			Name:   proto.String(resName),
			Port:   proto.Int32(int32(port)),
			Labels: si.Metadata.Labels,
		}

		if f.ipType == pb.IPConfig_PUBLIC {
			// If there is no ingress IP, skip the resource.
			if len(si.Status.LoadBalancer.Ingress) == 0 {
				continue
			}
			ingress := si.Status.LoadBalancer.Ingress[0]

			res.Ip = proto.String(ingress.IP)
			if ingress.IP == "" && ingress.Hostname != "" {
				res.Ip = proto.String(ingress.Hostname)
			}
		} else {
			res.Ip = proto.String(si.Spec.ClusterIP)
		}

		resources = append(resources, res)
	}
	return
}

func newServicesLister(namespace string, reEvalInterval time.Duration, kc *client, l *logger.Logger) *servicesLister {
	lister := &servicesLister{
		kind:       "services",
		apiPrefix:  "api/v1",
		namespace:  namespace,
		kClient:    kc,
		nameInPath: true,
		l:          l,
	}
	lister.start(reEvalInterval)
	return lister
}
