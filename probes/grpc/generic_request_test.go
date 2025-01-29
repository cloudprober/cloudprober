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
	"os"
	"strings"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/probes/grpc/proto"
	"github.com/cloudprober/cloudprober/probes/options"
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
		name        string
		req         *configpb.GenericRequest
		env         map[string]string
		wantResp    string
		wantInitErr bool
		wantErr     bool
	}{
		{
			name: "list_services",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_ListServices{
					ListServices: true,
				},
			},
			wantResp: strings.Join([]string{
				"cloudprober.servers.grpc.Prober",
				"grpc.reflection.v1.ServerReflection",
				"grpc.reflection.v1alpha.ServerReflection",
			}, ","),
		},
		{
			name: "list_service_methods",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_ListServiceMethods{
					ListServiceMethods: "cloudprober.servers.grpc.Prober",
				},
			},
			wantResp: strings.Join([]string{
				"cloudprober.servers.grpc.Prober.BlobRead",
				"cloudprober.servers.grpc.Prober.BlobWrite",
				"cloudprober.servers.grpc.Prober.Echo",
				"cloudprober.servers.grpc.Prober.ServerStatus",
			}, ","),
		},
		{
			name: "desc_service_method",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_DescribeServiceMethod{
					DescribeServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
			},
			wantResp: "name:\"Echo\" input_type:\".cloudprober.servers.grpc.EchoMessage\" output_type:\".cloudprober.servers.grpc.EchoMessage\" options:{}",
		},
		{
			name: "call_service_method",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_CallServiceMethod{
					CallServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
				Body: proto.String("{\"blob\": \"test\"}"),
			},
			wantResp: "{\"blob\":\"test\"}",
		},
		{
			name: "call_service_method_body_file",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_CallServiceMethod{
					CallServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
				BodyFile: proto.String("testdata/echo_request_body.txt"),
			},
			wantResp: "{\"blob\":\"test\"}",
		},
		{
			name: "call_service_method_body_file_with_env",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_CallServiceMethod{
					CallServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
				BodyFile:              proto.String("testdata/echo_request_body_env.txt"),
				BodyFileSubstituteEnv: proto.Bool(true),
			},
			env: map[string]string{
				"TESTBLOB": "testblob",
			},
			wantResp: "{\"blob\":\"testblob\"}",
		},
		{
			name: "call_service_method_body_file_error",
			req: &configpb.GenericRequest{
				RequestType: &configpb.GenericRequest_CallServiceMethod{
					CallServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
				Body:     proto.String("{\"blob\": \"test\"}"),
				BodyFile: proto.String("testdata/echo_request_body.txt"),
			},
			wantInitErr: true,
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
			if tt.env != nil {
				for k, v := range tt.env {
					defer os.Unsetenv(k)
					os.Setenv(k, v)
				}
			}
			p := &Probe{}

			opts := options.DefaultOptions()
			opts.ProbeConf = &configpb.ProbeConf{
				Method:  configpb.ProbeConf_GENERIC.Enum(),
				Request: tt.req,
			}
			if err := p.Init("test", opts); err != nil {
				if !tt.wantInitErr {
					t.Fatalf("unexpected init error: %v", err)
				}
				return
			}

			resp, err := p.genericRequest(context.Background(), conn, p.c.GetRequest())
			if (err != nil) != tt.wantErr {
				t.Errorf("Probe.genericRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, response(tt.wantResp), resp, "gRPC request: %v", p.c.GetRequest())
		})
	}
}

func TestInitDescriptorSource(t *testing.T) {
	tests := []struct {
		name    string
		request *configpb.GenericRequest
		wantErr bool
	}{
		{
			name: "list_services",
			request: &configpb.GenericRequest{
				ProtosetFile: proto.String("testdata/grpc_server.protoset"),
				RequestType: &configpb.GenericRequest_ListServices{
					ListServices: true,
				},
			},
			wantErr: true,
		},
		{
			name: "list_service_methods",
			request: &configpb.GenericRequest{
				ProtosetFile: proto.String("testdata/grpc_server.protoset"),
				RequestType: &configpb.GenericRequest_ListServiceMethods{
					ListServiceMethods: "cloudprober.servers.grpc.Prober",
				},
			},
			wantErr: true,
		},
		{
			name: "describe_service_method",
			request: &configpb.GenericRequest{
				ProtosetFile: proto.String("testdata/grpc_server.protoset"),
				RequestType: &configpb.GenericRequest_DescribeServiceMethod{
					DescribeServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
			},
			wantErr: true,
		},
		{
			name: "call_service_method",
			request: &configpb.GenericRequest{
				ProtosetFile: proto.String("testdata/grpc_server.protoset"),
				RequestType: &configpb.GenericRequest_CallServiceMethod{
					CallServiceMethod: "cloudprober.servers.grpc.Prober.Echo",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{}
			p.c = &configpb.ProbeConf{
				Request: tt.request,
			}
			if err := p.initDescriptorSource(); (err != nil) != tt.wantErr {
				t.Errorf("Probe.initDescriptorSource() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.request.GetCallServiceMethod() != "" {
				assert.NotNil(t, p.descSrc)
				desc, err := p.descSrc.FindSymbol(tt.request.GetCallServiceMethod())
				assert.NoError(t, err)
				assert.Equal(t, "cloudprober.servers.grpc", desc.GetFile().GetPackage())
			}
		})
	}
}
