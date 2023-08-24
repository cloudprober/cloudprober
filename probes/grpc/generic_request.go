// Copyright 2020-2023 The Cloudprober Authors.
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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	configpb "github.com/cloudprober/cloudprober/probes/grpc/proto"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
)

func (p *Probe) initDescriptorSource() error {
	req := p.c.GetRequest()

	if req == nil {
		return errors.New("request is required for GENERIC gRPC probe")
	}

	if req.GetProtosetFile() != "" {
		if req.GetListServices() || req.GetListServiceMethods() != "" || req.GetDescribeServiceMethod() != "" {
			return fmt.Errorf("request types list_services, list_service_method, and describe_service_method are not supported for protoset descriptor source")
		}
		descSrc, err := grpcurl.DescriptorSourceFromProtoSets(req.GetProtosetFile())
		if err != nil {
			return fmt.Errorf("error parsing protoset file: %v", err)
		}
		p.descSrc = descSrc
	}

	return nil
}

type response struct {
	body string
}

func (r *response) String() string {
	if r == nil {
		return ""
	}
	return r.body
}

func (p *Probe) callServiceMethod(ctx context.Context, req *configpb.GenericRequest, conn *grpc.ClientConn) (*response, error) {
	in := strings.NewReader(req.GetBody())
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, p.descSrc, in, grpcurl.FormatOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to construct parser and formatter: %v", err)
	}

	var out bytes.Buffer
	h := &grpcurl.DefaultEventHandler{Out: &out, Formatter: formatter}

	if err := grpcurl.InvokeRPC(ctx, p.descSrc, conn, req.GetCallServiceMethod(), nil, h, rf.Next); err != nil {
		return nil, fmt.Errorf("error invoking gRPC: %v", err)
	}

	var buf bytes.Buffer
	if err := json.Compact(&buf, out.Bytes()); err != nil {
		return nil, fmt.Errorf("error compacting response JSON (%s): %v", out.String(), err)
	}
	return &response{body: buf.String()}, nil
}

func (p *Probe) genericRequest(ctx context.Context, conn *grpc.ClientConn, req *configpb.GenericRequest) (*response, error) {
	// If we didn't load protoset from a file, we'll get it everytime
	// from the server.
	if req.GetProtosetFile() == "" {
		p.descSrc = grpcurl.DescriptorSourceFromServer(ctx, grpcreflect.NewClientAuto(ctx, conn))
	}

	if req.GetListServices() {
		services, err := grpcurl.ListServices(p.descSrc)
		if err != nil {
			return nil, fmt.Errorf("error listing services: %v", err)
		}
		return &response{body: strings.Join(services, ",")}, nil
	}

	if req.GetListServiceMethods() != "" {
		methods, err := grpcurl.ListMethods(p.descSrc, req.GetListServiceMethods())
		if err != nil {
			return nil, fmt.Errorf("error listing service (%s) methods: %v", req.GetListServiceMethods(), err)
		}
		return &response{body: strings.Join(methods, ",")}, nil
	}

	if req.GetDescribeServiceMethod() != "" {
		d, err := p.descSrc.FindSymbol(req.GetDescribeServiceMethod())
		if err != nil {
			return nil, fmt.Errorf("error describing method(%s): %v", req.GetDescribeServiceMethod(), err)
		}
		return &response{body: strings.ReplaceAll(d.AsProto().String(), "  ", " ")}, nil
	}

	if req.GetCallServiceMethod() != "" {
		return p.callServiceMethod(ctx, req, conn)
	}

	return nil, fmt.Errorf("invalid request type: %v", req)
}
