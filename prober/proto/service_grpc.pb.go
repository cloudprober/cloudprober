// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.5
// source: github.com/cloudprober/cloudprober/prober/proto/service.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Cloudprober_AddProbe_FullMethodName         = "/cloudprober.Cloudprober/AddProbe"
	Cloudprober_RemoveProbe_FullMethodName      = "/cloudprober.Cloudprober/RemoveProbe"
	Cloudprober_ListProbes_FullMethodName       = "/cloudprober.Cloudprober/ListProbes"
	Cloudprober_SaveProbesConfig_FullMethodName = "/cloudprober.Cloudprober/SaveProbesConfig"
)

// CloudproberClient is the client API for Cloudprober service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CloudproberClient interface {
	// AddProbe adds a probe to cloudprober. Error is returned if probe is already
	// defined or there is an error during initialization of the probe.
	AddProbe(ctx context.Context, in *AddProbeRequest, opts ...grpc.CallOption) (*AddProbeResponse, error)
	// RemoveProbe stops the probe and removes it from the in-memory database.
	RemoveProbe(ctx context.Context, in *RemoveProbeRequest, opts ...grpc.CallOption) (*RemoveProbeResponse, error)
	// ListProbes lists active probes.
	ListProbes(ctx context.Context, in *ListProbesRequest, opts ...grpc.CallOption) (*ListProbesResponse, error)
	SaveProbesConfig(ctx context.Context, in *SaveProbesConfigRequest, opts ...grpc.CallOption) (*SaveProbesConfigResponse, error)
}

type cloudproberClient struct {
	cc grpc.ClientConnInterface
}

func NewCloudproberClient(cc grpc.ClientConnInterface) CloudproberClient {
	return &cloudproberClient{cc}
}

func (c *cloudproberClient) AddProbe(ctx context.Context, in *AddProbeRequest, opts ...grpc.CallOption) (*AddProbeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AddProbeResponse)
	err := c.cc.Invoke(ctx, Cloudprober_AddProbe_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cloudproberClient) RemoveProbe(ctx context.Context, in *RemoveProbeRequest, opts ...grpc.CallOption) (*RemoveProbeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RemoveProbeResponse)
	err := c.cc.Invoke(ctx, Cloudprober_RemoveProbe_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cloudproberClient) ListProbes(ctx context.Context, in *ListProbesRequest, opts ...grpc.CallOption) (*ListProbesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListProbesResponse)
	err := c.cc.Invoke(ctx, Cloudprober_ListProbes_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *cloudproberClient) SaveProbesConfig(ctx context.Context, in *SaveProbesConfigRequest, opts ...grpc.CallOption) (*SaveProbesConfigResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SaveProbesConfigResponse)
	err := c.cc.Invoke(ctx, Cloudprober_SaveProbesConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CloudproberServer is the server API for Cloudprober service.
// All implementations must embed UnimplementedCloudproberServer
// for forward compatibility.
type CloudproberServer interface {
	// AddProbe adds a probe to cloudprober. Error is returned if probe is already
	// defined or there is an error during initialization of the probe.
	AddProbe(context.Context, *AddProbeRequest) (*AddProbeResponse, error)
	// RemoveProbe stops the probe and removes it from the in-memory database.
	RemoveProbe(context.Context, *RemoveProbeRequest) (*RemoveProbeResponse, error)
	// ListProbes lists active probes.
	ListProbes(context.Context, *ListProbesRequest) (*ListProbesResponse, error)
	SaveProbesConfig(context.Context, *SaveProbesConfigRequest) (*SaveProbesConfigResponse, error)
	mustEmbedUnimplementedCloudproberServer()
}

// UnimplementedCloudproberServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedCloudproberServer struct{}

func (UnimplementedCloudproberServer) AddProbe(context.Context, *AddProbeRequest) (*AddProbeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddProbe not implemented")
}
func (UnimplementedCloudproberServer) RemoveProbe(context.Context, *RemoveProbeRequest) (*RemoveProbeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveProbe not implemented")
}
func (UnimplementedCloudproberServer) ListProbes(context.Context, *ListProbesRequest) (*ListProbesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListProbes not implemented")
}
func (UnimplementedCloudproberServer) SaveProbesConfig(context.Context, *SaveProbesConfigRequest) (*SaveProbesConfigResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SaveProbesConfig not implemented")
}
func (UnimplementedCloudproberServer) mustEmbedUnimplementedCloudproberServer() {}
func (UnimplementedCloudproberServer) testEmbeddedByValue()                     {}

// UnsafeCloudproberServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CloudproberServer will
// result in compilation errors.
type UnsafeCloudproberServer interface {
	mustEmbedUnimplementedCloudproberServer()
}

func RegisterCloudproberServer(s grpc.ServiceRegistrar, srv CloudproberServer) {
	// If the following call pancis, it indicates UnimplementedCloudproberServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Cloudprober_ServiceDesc, srv)
}

func _Cloudprober_AddProbe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddProbeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudproberServer).AddProbe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Cloudprober_AddProbe_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudproberServer).AddProbe(ctx, req.(*AddProbeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cloudprober_RemoveProbe_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveProbeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudproberServer).RemoveProbe(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Cloudprober_RemoveProbe_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudproberServer).RemoveProbe(ctx, req.(*RemoveProbeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cloudprober_ListProbes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListProbesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudproberServer).ListProbes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Cloudprober_ListProbes_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudproberServer).ListProbes(ctx, req.(*ListProbesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Cloudprober_SaveProbesConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SaveProbesConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudproberServer).SaveProbesConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Cloudprober_SaveProbesConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudproberServer).SaveProbesConfig(ctx, req.(*SaveProbesConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Cloudprober_ServiceDesc is the grpc.ServiceDesc for Cloudprober service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Cloudprober_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cloudprober.Cloudprober",
	HandlerType: (*CloudproberServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddProbe",
			Handler:    _Cloudprober_AddProbe_Handler,
		},
		{
			MethodName: "RemoveProbe",
			Handler:    _Cloudprober_RemoveProbe_Handler,
		},
		{
			MethodName: "ListProbes",
			Handler:    _Cloudprober_ListProbes_Handler,
		},
		{
			MethodName: "SaveProbesConfig",
			Handler:    _Cloudprober_SaveProbesConfig_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "github.com/cloudprober/cloudprober/prober/proto/service.proto",
}
