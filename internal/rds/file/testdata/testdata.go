// Copyright 2021-2024 The Cloudprober Authors.
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

package testdata

import (
	"net"

	rdspb "github.com/cloudprober/cloudprober/internal/rds/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"google.golang.org/protobuf/proto"
)

var ExpectedResources = []*rdspb.Resource{
	{
		Name: proto.String("switch-xx-1"),
		Port: proto.Int32(8080),
		Ip:   proto.String("10.1.1.1"),
		Labels: map[string]string{
			"device_type": "switch",
			"cluster":     "xx",
		},
	},
	{
		Name: proto.String("switch-xx-2"),
		Port: proto.Int32(8081),
		Ip:   proto.String("10.1.1.2"),
		Labels: map[string]string{
			"cluster": "xx",
		},
	},
	{
		Name: proto.String("switch-yy-1"),
		Port: proto.Int32(8080),
		Ip:   proto.String("10.1.2.1"),
	},
	{
		Name: proto.String("switch-zz-1"),
		Port: proto.Int32(8080),
		Ip:   proto.String("::aaa:1"),
	},
	{
		Name: proto.String("web-1"),
		Ip:   proto.String("web-1"),
		Port: proto.Int32(80),
		Labels: map[string]string{
			"__cp_host__":   "cloudprober.org",
			"__cp_path__":   "/",
			"__cp_scheme__": "https",
		},
	},
}

func ExpectedEndpoints() []endpoint.Endpoint {
	var result []endpoint.Endpoint
	for _, resource := range ExpectedResources {
		result = append(result, endpoint.Endpoint{
			Name:   *resource.Name,
			Port:   int(*resource.Port),
			IP:     net.ParseIP(resource.GetIp()),
			Labels: resource.Labels,
		})
	}
	return result
}

func ExpectedIPs() map[string]string {
	result := make(map[string]string)
	for _, resource := range ExpectedResources {
		if resource.Ip != nil {
			result[*resource.Name] = resource.GetIp()
		}
	}
	return result
}
