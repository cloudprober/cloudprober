// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.1
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/internal/servers/proto/config.proto

package proto

import (
	proto3 "github.com/cloudprober/cloudprober/internal/servers/external/proto"
	proto2 "github.com/cloudprober/cloudprober/internal/servers/grpc/proto"
	proto "github.com/cloudprober/cloudprober/internal/servers/http/proto"
	proto1 "github.com/cloudprober/cloudprober/internal/servers/udp/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ServerDef_Type int32

const (
	ServerDef_HTTP     ServerDef_Type = 0
	ServerDef_UDP      ServerDef_Type = 1
	ServerDef_GRPC     ServerDef_Type = 2
	ServerDef_EXTERNAL ServerDef_Type = 3
)

// Enum value maps for ServerDef_Type.
var (
	ServerDef_Type_name = map[int32]string{
		0: "HTTP",
		1: "UDP",
		2: "GRPC",
		3: "EXTERNAL",
	}
	ServerDef_Type_value = map[string]int32{
		"HTTP":     0,
		"UDP":      1,
		"GRPC":     2,
		"EXTERNAL": 3,
	}
)

func (x ServerDef_Type) Enum() *ServerDef_Type {
	p := new(ServerDef_Type)
	*p = x
	return p
}

func (x ServerDef_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ServerDef_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_enumTypes[0].Descriptor()
}

func (ServerDef_Type) Type() protoreflect.EnumType {
	return &file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_enumTypes[0]
}

func (x ServerDef_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ServerDef_Type) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ServerDef_Type(num)
	return nil
}

// Deprecated: Use ServerDef_Type.Descriptor instead.
func (ServerDef_Type) EnumDescriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescGZIP(), []int{0, 0}
}

type ServerDef struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	Type  *ServerDef_Type        `protobuf:"varint,1,req,name=type,enum=cloudprober.servers.ServerDef_Type" json:"type,omitempty"`
	// Types that are valid to be assigned to Server:
	//
	//	*ServerDef_HttpServer
	//	*ServerDef_UdpServer
	//	*ServerDef_GrpcServer
	//	*ServerDef_ExternalServer
	Server        isServerDef_Server `protobuf_oneof:"server"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ServerDef) Reset() {
	*x = ServerDef{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerDef) ProtoMessage() {}

func (x *ServerDef) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerDef.ProtoReflect.Descriptor instead.
func (*ServerDef) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *ServerDef) GetType() ServerDef_Type {
	if x != nil && x.Type != nil {
		return *x.Type
	}
	return ServerDef_HTTP
}

func (x *ServerDef) GetServer() isServerDef_Server {
	if x != nil {
		return x.Server
	}
	return nil
}

func (x *ServerDef) GetHttpServer() *proto.ServerConf {
	if x != nil {
		if x, ok := x.Server.(*ServerDef_HttpServer); ok {
			return x.HttpServer
		}
	}
	return nil
}

func (x *ServerDef) GetUdpServer() *proto1.ServerConf {
	if x != nil {
		if x, ok := x.Server.(*ServerDef_UdpServer); ok {
			return x.UdpServer
		}
	}
	return nil
}

func (x *ServerDef) GetGrpcServer() *proto2.ServerConf {
	if x != nil {
		if x, ok := x.Server.(*ServerDef_GrpcServer); ok {
			return x.GrpcServer
		}
	}
	return nil
}

func (x *ServerDef) GetExternalServer() *proto3.ServerConf {
	if x != nil {
		if x, ok := x.Server.(*ServerDef_ExternalServer); ok {
			return x.ExternalServer
		}
	}
	return nil
}

type isServerDef_Server interface {
	isServerDef_Server()
}

type ServerDef_HttpServer struct {
	HttpServer *proto.ServerConf `protobuf:"bytes,2,opt,name=http_server,json=httpServer,oneof"`
}

type ServerDef_UdpServer struct {
	UdpServer *proto1.ServerConf `protobuf:"bytes,3,opt,name=udp_server,json=udpServer,oneof"`
}

type ServerDef_GrpcServer struct {
	GrpcServer *proto2.ServerConf `protobuf:"bytes,4,opt,name=grpc_server,json=grpcServer,oneof"`
}

type ServerDef_ExternalServer struct {
	ExternalServer *proto3.ServerConf `protobuf:"bytes,5,opt,name=external_server,json=externalServer,oneof"`
}

func (*ServerDef_HttpServer) isServerDef_Server() {}

func (*ServerDef_UdpServer) isServerDef_Server() {}

func (*ServerDef_GrpcServer) isServerDef_Server() {}

func (*ServerDef_ExternalServer) isServerDef_Server() {}

var File_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDesc = []byte{
	0x0a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x1a, 0x4b, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x73, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x4b, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2f,
	0x68, 0x74, 0x74, 0x70, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x4a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x75, 0x64, 0x70,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x4f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c,
	0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x65, 0x78, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0xae, 0x03, 0x0a, 0x09, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x44,
	0x65, 0x66, 0x12, 0x37, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x02, 0x28, 0x0e,
	0x32, 0x23, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x44, 0x65, 0x66,
	0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x47, 0x0a, 0x0b, 0x68,
	0x74, 0x74, 0x70, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x24, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0a, 0x68, 0x74, 0x74, 0x70, 0x53, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x12, 0x44, 0x0a, 0x0a, 0x75, 0x64, 0x70, 0x5f, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x75,
	0x64, 0x70, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52,
	0x09, 0x75, 0x64, 0x70, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x47, 0x0a, 0x0b, 0x67, 0x72,
	0x70, 0x63, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x24, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0a, 0x67, 0x72, 0x70, 0x63, 0x53, 0x65, 0x72,
	0x76, 0x65, 0x72, 0x12, 0x53, 0x0a, 0x0f, 0x65, 0x78, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x5f,
	0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x73, 0x2e, 0x65, 0x78, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0e, 0x65, 0x78, 0x74, 0x65, 0x72, 0x6e,
	0x61, 0x6c, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x22, 0x31, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65,
	0x12, 0x08, 0x0a, 0x04, 0x48, 0x54, 0x54, 0x50, 0x10, 0x00, 0x12, 0x07, 0x0a, 0x03, 0x55, 0x44,
	0x50, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x47, 0x52, 0x50, 0x43, 0x10, 0x02, 0x12, 0x0c, 0x0a,
	0x08, 0x45, 0x58, 0x54, 0x45, 0x52, 0x4e, 0x41, 0x4c, 0x10, 0x03, 0x42, 0x08, 0x0a, 0x06, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x42, 0x3b, 0x5a, 0x39, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_goTypes = []any{
	(ServerDef_Type)(0),       // 0: cloudprober.servers.ServerDef.Type
	(*ServerDef)(nil),         // 1: cloudprober.servers.ServerDef
	(*proto.ServerConf)(nil),  // 2: cloudprober.servers.http.ServerConf
	(*proto1.ServerConf)(nil), // 3: cloudprober.servers.udp.ServerConf
	(*proto2.ServerConf)(nil), // 4: cloudprober.servers.grpc.ServerConf
	(*proto3.ServerConf)(nil), // 5: cloudprober.servers.external.ServerConf
}
var file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.servers.ServerDef.type:type_name -> cloudprober.servers.ServerDef.Type
	2, // 1: cloudprober.servers.ServerDef.http_server:type_name -> cloudprober.servers.http.ServerConf
	3, // 2: cloudprober.servers.ServerDef.udp_server:type_name -> cloudprober.servers.udp.ServerConf
	4, // 3: cloudprober.servers.ServerDef.grpc_server:type_name -> cloudprober.servers.grpc.ServerConf
	5, // 4: cloudprober.servers.ServerDef.external_server:type_name -> cloudprober.servers.external.ServerConf
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto != nil {
		return
	}
	file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_msgTypes[0].OneofWrappers = []any{
		(*ServerDef_HttpServer)(nil),
		(*ServerDef_UdpServer)(nil),
		(*ServerDef_GrpcServer)(nil),
		(*ServerDef_ExternalServer)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_depIdxs,
		EnumInfos:         file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_enumTypes,
		MessageInfos:      file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_internal_servers_proto_config_proto_depIdxs = nil
}
