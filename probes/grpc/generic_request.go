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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
)

func (p *Probe) descriptorSource() error {
	if p.c.GetRequest() == nil {
		return errors.New("request is required for GENERIC gRPC probe")
	}

	req := p.c.GetRequest()

	if req.GetProtosetFile() != "" {
		if req.GetListServices() || req.GetListServiceMethods() != "" || req.GetDescribeServiceMethod() != "" {
			return fmt.Errorf("request types list_services, list_service_method, and describe_service_method are not supported for protoset descriptor source")
		}
		descSrc, err := grpcurl.DescriptorSourceFromProtoSets(p.c.GetRequest().GetProtosetFile())
		if err != nil {
			return fmt.Errorf("error parsing protoset file: %v", err)
		}
		p.descSrc = descSrc
	}

	return nil
}

func (p *Probe) genericRequest(ctx context.Context, conn *grpc.ClientConn) {
	req := p.c.GetRequest()
	if req.GetProtosetFile() == "" {
		client := grpcreflect.NewClientAuto(ctx, conn)
		p.descSrc = grpcurl.DescriptorSourceFromServer(ctx, client)
	}

	if req.GetListServices() {
		services, err := grpcurl.ListServices(p.descSrc)
		if err != nil {
			p.l.Errorf("error listing services: %v", err)
		}
		p.l.Infof("Services: %v", services)
		return
	}

	if req.GetListServiceMethods() != "" {
		methods, err := grpcurl.ListMethods(p.descSrc, req.GetListServiceMethods())
		if err != nil {
			p.l.Errorf("error listing service (%s) methods: %v", req.GetListServiceMethods(), err)
		}
		p.l.Infof("Methods: %v", methods)
		return
	}

	if req.GetDescribeServiceMethod() != "" {
		desc, err := p.descSrc.FindSymbol(req.GetDescribeServiceMethod())
		if err != nil {
			p.l.Errorf("error describing method(%s): %v", req.GetDescribeServiceMethod(), err)
		}
		p.l.Infof("Descriptor: %v", desc)
		return
	}

	if req.GetCallServiceMethod() != "" {
		in := strings.NewReader(req.GetBody())
		rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, p.descSrc, in, grpcurl.FormatOptions{})
		if err != nil {
			p.l.Errorf("failed to construct parser and formatter: %v", err)
			return
		}

		var out bytes.Buffer
		h := &grpcurl.DefaultEventHandler{
			Out:       &out,
			Formatter: formatter,
		}
		err = grpcurl.InvokeRPC(ctx, p.descSrc, conn, req.GetCallServiceMethod(), nil, h, rf.Next)
		if err != nil {
			p.l.Errorf("error invoking gRPC: %v", err)
		}
		time.Sleep(2 * time.Second)
		p.l.Debugf("Status: %v", h.Status.Code())
		p.l.Debugf("Response: %v", out.String())
	}
}
