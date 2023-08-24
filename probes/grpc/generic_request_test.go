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

package grpc

import (
	"context"
	"strings"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/probes/grpc/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func TestGenericRequest(t *testing.T) {
	addr, err := globalGRPCServer(50 * time.Millisecond)
	assert.NoError(t, err, "Error starting global gRPC server")

	conn, err := grpc.DialContext(context.Background(), addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err, "Error connecting to global gRPC server")
	defer conn.Close()

	tests := []struct {
		name     string
		req      *configpb.GenericRequest
		wantResp *response
		wantErr  bool
	}{
		{
			name: "list_services",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_ListServices{
					ListServices: true,
				},
			},
			wantResp: &response{
				body: strings.Join([]string{
					"cloudprober.servers.grpc.Prober",
					"grpc.reflection.v1alpha.ServerReflection",
				}, ","),
			},
		},
		{
			name: "list_service_methods",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_ListServiceMethods{
					ListServiceMethods: "cloudprober.servers.grpc.Prober",
				},
			},
			wantResp: &response{
				body: strings.Join([]string{
					"cloudprober.servers.grpc.Prober.BlobRead",
					"cloudprober.servers.grpc.Prober.BlobWrite",
					"cloudprober.servers.grpc.Prober.Echo",
					"cloudprober.servers.grpc.Prober.ServerStatus",
				}, ","),
			},
		},
		{
			name: "desc_service_method",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_DescribeServiceMethod{
					DescribeServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
			},
			wantResp: &response{
				body: "name:\"Echo\" input_type:\".cloudprober.servers.grpc.EchoMessage\" output_type:\".cloudprober.servers.grpc.EchoMessage\" options:{}",
			},
		},
		{
			name: "call_service_method",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_CallServiceMethod{
					CallServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
				Body: proto.String("{\"blob\": \"test\"}"),
			},
			wantResp: &response{
				body: "{\"blob\":\"test\"}",
			},
		},
		{
			name: "call_service_method_error",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_CallServiceMethod{
					CallServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
				Body: proto.String("{\"blob2\": \"test\"}"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{}

			resp, err := p.genericRequest(context.Background(), conn, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Probe.genericRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.wantResp, resp)
		})
	}
}
