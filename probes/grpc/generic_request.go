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
	"google.golang.org/grpc/codes"
)

func (p *Probe) initDescriptorSource() error {
	req := p.c.GetRequest()

	if req == nil {
		return errors.New("request is required for GENERIC gRPC probe")
	}

	if req.GetProtosetFile() != "" {
		if req.GetCallServiceMethod() == "" {
			return fmt.Errorf("only call_service_method request type is supported for protoset descriptor source")
		}

		descSrc, err := grpcurl.DescriptorSourceFromProtoSets(req.GetProtosetFile())
		if err != nil {
			return fmt.Errorf("error parsing protoset file: %v", err)
		}

		if _, err = descSrc.FindSymbol(req.GetCallServiceMethod()); err != nil {
			return fmt.Errorf("error finding symbol (%s) in protoset file: %v", req.GetCallServiceMethod(), err)
		}
		p.descSrc = descSrc
	}

	return nil
}

type response string

func (r response) String() string {
	return string(r)
}

func (p *Probe) callServiceMethod(ctx context.Context, req *configpb.GenericRequest, descSrc grpcurl.DescriptorSource, conn *grpc.ClientConn) (response, error) {
	in := strings.NewReader(req.GetBody())
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, descSrc, in, grpcurl.FormatOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to construct parser and formatter: %v", err)
	}

	var out bytes.Buffer
	h := &grpcurl.DefaultEventHandler{Out: &out, Formatter: formatter}

	if err := grpcurl.InvokeRPC(ctx, descSrc, conn, req.GetCallServiceMethod(), nil, h, rf.Next); err != nil {
		return "", fmt.Errorf("error invoking gRPC: %v", err)
	}

	if h.Status.Code() != codes.OK {
		return "", fmt.Errorf("gRPC call failed: %s", h.Status.Message())
	}

	respBytes := out.Bytes()
	if len(respBytes) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	if err := json.Compact(&buf, respBytes); err != nil {
		return "", fmt.Errorf("error compacting response JSON (%s): %v", out.String(), err)
	}
	return response(buf.String()), nil
}

func (p *Probe) genericRequest(ctx context.Context, conn *grpc.ClientConn, req *configpb.GenericRequest) (response, error) {
	// If we didn't load protoset from a file, we'll get it everytime
	// from the server.
	descSrc := p.descSrc
	if descSrc == nil {
		descSrc = grpcurl.DescriptorSourceFromServer(ctx, grpcreflect.NewClientAuto(ctx, conn))
	}

	switch req.RequestType.(type) {
	case *configpb.GenericRequest_ListServices:
		services, err := grpcurl.ListServices(descSrc)
		if err != nil {
			return "", fmt.Errorf("error listing services: %v", err)
		}
		return response(strings.Join(services, ",")), nil
	case *configpb.GenericRequest_ListServiceMethods:
		methods, err := grpcurl.ListMethods(descSrc, req.GetListServiceMethods())
		if err != nil {
			return "", fmt.Errorf("error listing service (%s) methods: %v", req.GetListServiceMethods(), err)
		}
		return response(strings.Join(methods, ",")), nil
	case *configpb.GenericRequest_DescribeServiceMethod:
		d, err := descSrc.FindSymbol(req.GetDescribeServiceMethod())
		if err != nil {
			return "", fmt.Errorf("error describing method(%s): %v", req.GetDescribeServiceMethod(), err)
		}
		return response(strings.ReplaceAll(d.AsProto().String(), "  ", " ")), nil
	case *configpb.GenericRequest_CallServiceMethod:
		return p.callServiceMethod(ctx, req, descSrc, conn)
	}

	return "", fmt.Errorf("invalid request type: %v", req)
}
