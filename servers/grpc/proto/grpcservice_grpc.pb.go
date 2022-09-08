// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.5
// source: github.com/cloudprober/cloudprober/servers/grpc/proto/grpcservice.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ProberClient is the client API for Prober service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ProberClient interface {
	// Echo echoes back incoming messages.
	Echo(ctx context.Context, in *EchoMessage, opts ...grpc.CallOption) (*EchoMessage, error)
	// BlobRead returns a blob of bytes to the prober.
	BlobRead(ctx context.Context, in *BlobReadRequest, opts ...grpc.CallOption) (*BlobReadResponse, error)
	// ServerStatus returns the current server status.
	ServerStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error)
	// BlobWrite allows client to write a blob to the server.
	BlobWrite(ctx context.Context, in *BlobWriteRequest, opts ...grpc.CallOption) (*BlobWriteResponse, error)
}

type proberClient struct {
	cc grpc.ClientConnInterface
}

func NewProberClient(cc grpc.ClientConnInterface) ProberClient {
	return &proberClient{cc}
}

func (c *proberClient) Echo(ctx context.Context, in *EchoMessage, opts ...grpc.CallOption) (*EchoMessage, error) {
	out := new(EchoMessage)
	err := c.cc.Invoke(ctx, "/cloudprober.servers.grpc.Prober/Echo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *proberClient) BlobRead(ctx context.Context, in *BlobReadRequest, opts ...grpc.CallOption) (*BlobReadResponse, error) {
	out := new(BlobReadResponse)
	err := c.cc.Invoke(ctx, "/cloudprober.servers.grpc.Prober/BlobRead", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *proberClient) ServerStatus(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, "/cloudprober.servers.grpc.Prober/ServerStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *proberClient) BlobWrite(ctx context.Context, in *BlobWriteRequest, opts ...grpc.CallOption) (*BlobWriteResponse, error) {
	out := new(BlobWriteResponse)
	err := c.cc.Invoke(ctx, "/cloudprober.servers.grpc.Prober/BlobWrite", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProberServer is the server API for Prober service.
// All implementations must embed UnimplementedProberServer
// for forward compatibility
type ProberServer interface {
	// Echo echoes back incoming messages.
	Echo(context.Context, *EchoMessage) (*EchoMessage, error)
	// BlobRead returns a blob of bytes to the prober.
	BlobRead(context.Context, *BlobReadRequest) (*BlobReadResponse, error)
	// ServerStatus returns the current server status.
	ServerStatus(context.Context, *StatusRequest) (*StatusResponse, error)
	// BlobWrite allows client to write a blob to the server.
	BlobWrite(context.Context, *BlobWriteRequest) (*BlobWriteResponse, error)
	mustEmbedUnimplementedProberServer()
}

// UnimplementedProberServer must be embedded to have forward compatible implementations.
type UnimplementedProberServer struct {
}

func (UnimplementedProberServer) Echo(context.Context, *EchoMessage) (*EchoMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Echo not implemented")
}
func (UnimplementedProberServer) BlobRead(context.Context, *BlobReadRequest) (*BlobReadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BlobRead not implemented")
}
func (UnimplementedProberServer) ServerStatus(context.Context, *StatusRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ServerStatus not implemented")
}
func (UnimplementedProberServer) BlobWrite(context.Context, *BlobWriteRequest) (*BlobWriteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BlobWrite not implemented")
}
func (UnimplementedProberServer) mustEmbedUnimplementedProberServer() {}

// UnsafeProberServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ProberServer will
// result in compilation errors.
type UnsafeProberServer interface {
	mustEmbedUnimplementedProberServer()
}

func RegisterProberServer(s grpc.ServiceRegistrar, srv ProberServer) {
	s.RegisterService(&Prober_ServiceDesc, srv)
}

func _Prober_Echo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EchoMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProberServer).Echo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cloudprober.servers.grpc.Prober/Echo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProberServer).Echo(ctx, req.(*EchoMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _Prober_BlobRead_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BlobReadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProberServer).BlobRead(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cloudprober.servers.grpc.Prober/BlobRead",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProberServer).BlobRead(ctx, req.(*BlobReadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Prober_ServerStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProberServer).ServerStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cloudprober.servers.grpc.Prober/ServerStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProberServer).ServerStatus(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Prober_BlobWrite_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BlobWriteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ProberServer).BlobWrite(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cloudprober.servers.grpc.Prober/BlobWrite",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ProberServer).BlobWrite(ctx, req.(*BlobWriteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Prober_ServiceDesc is the grpc.ServiceDesc for Prober service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Prober_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cloudprober.servers.grpc.Prober",
	HandlerType: (*ProberServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Echo",
			Handler:    _Prober_Echo_Handler,
		},
		{
			MethodName: "BlobRead",
			Handler:    _Prober_BlobRead_Handler,
		},
		{
			MethodName: "ServerStatus",
			Handler:    _Prober_ServerStatus_Handler,
		},
		{
			MethodName: "BlobWrite",
			Handler:    _Prober_BlobWrite_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "github.com/cloudprober/cloudprober/servers/grpc/proto/grpcservice.proto",
}
