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
	"fmt"
	"testing"

	k8sconfigpb "github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto"
	rdspb "github.com/cloudprober/cloudprober/internal/rds/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func Test_parseConfig(t *testing.T) {
	tests := []struct {
		cfg       string
		wantPC    *k8sconfigpb.ProviderConfig
		wantName  string
		wantValue string
	}{
		{
			cfg: `services:""`,
			wantPC: &k8sconfigpb.ProviderConfig{
				Namespace: proto.String(""),
				Services:  &k8sconfigpb.Services{},
				ReEvalSec: proto.Int32(30),
			},
			wantName: "services",
		},
		{
			cfg: `ingresses:""`,
			wantPC: &k8sconfigpb.ProviderConfig{
				Namespace: proto.String(""),
				Ingresses: &k8sconfigpb.Ingresses{},
				ReEvalSec: proto.Int32(30),
			},
			wantName: "ingresses",
		},
		{
			cfg: `pods:""`,
			wantPC: &k8sconfigpb.ProviderConfig{
				Namespace: proto.String(""),
				Pods:      &k8sconfigpb.Pods{},
				ReEvalSec: proto.Int32(30),
			},
			wantName: "pods",
		},
		{
			cfg: `namespace:"dev"
			      endpoints:".*-service"
				  labelSelector:["k8s-app","role!=canary"]`,
			wantPC: &k8sconfigpb.ProviderConfig{
				Namespace:     proto.String("dev"),
				LabelSelector: []string{"k8s-app", "role!=canary"},
				Endpoints:     &k8sconfigpb.Endpoints{},
				ReEvalSec:     proto.Int32(30),
			},
			wantName:  "endpoints",
			wantValue: ".*-service",
		},
	}
	for _, tt := range tests {
		t.Run(tt.cfg, func(t *testing.T) {
			pb := &targetspb.K8STargets{}
			err := prototext.Unmarshal([]byte(tt.cfg), pb)
			assert.NoError(t, err, "unmarshaling targets config")

			pc, name, value := parseConfig(pb)
			assert.Equal(t, tt.wantPC, pc, "Provider Config")
			assert.Equal(t, tt.wantName, name, "Resource name")
			assert.Equal(t, tt.wantValue, value, "Resource value")
		})
	}
}

func Test_rdsRequest(t *testing.T) {
	tests := []struct {
		resources  string
		nameF      string
		portFilter string
		want       *rdspb.ListResourcesRequest
	}{
		{
			resources: "test-resources",
			want: &rdspb.ListResourcesRequest{
				Provider:     proto.String("k8s"),
				ResourcePath: proto.String("test-resources"),
			},
		},
		{
			resources: "test-resources",
			nameF:     "service", // Match just service
			want: &rdspb.ListResourcesRequest{
				Provider:     proto.String("k8s"),
				ResourcePath: proto.String("test-resources"),
				Filter:       []*rdspb.Filter{{Key: proto.String("name"), Value: proto.String("^service$")}},
			},
		},
		{
			resources: "test-resources",
			nameF:     "^.*-service",
			want: &rdspb.ListResourcesRequest{
				Provider:     proto.String("k8s"),
				ResourcePath: proto.String("test-resources"),
				Filter:       []*rdspb.Filter{{Key: proto.String("name"), Value: proto.String("^.*-service$")}},
			},
		},
		{
			resources:  "test-resources",
			nameF:      ".*-service",
			portFilter: ".*dns.*",
			want: &rdspb.ListResourcesRequest{
				Provider:     proto.String("k8s"),
				ResourcePath: proto.String("test-resources"),
				Filter: []*rdspb.Filter{
					{Key: proto.String("name"), Value: proto.String("^.*-service$")},
					{Key: proto.String("port"), Value: proto.String(".*dns.*")},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("name:%s,port:%s", tt.nameF, tt.portFilter), func(t *testing.T) {
			assert.Equal(t, tt.want, rdsRequest(tt.resources, tt.nameF, tt.portFilter))
		})
	}
}
