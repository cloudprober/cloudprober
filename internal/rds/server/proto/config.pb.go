// Configuration proto for ResourceDiscovery (rds) server.
// Example config:
//
// provider {
//   id: "gcp"
//   gcp_config {
//     project_id: 'test-project-id'
//     gce_instances {}
//     forwarding_rules {}
//   }
// }

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/internal/rds/server/proto/config.proto

package proto

import (
	proto "github.com/cloudprober/cloudprober/internal/rds/file/proto"
	proto1 "github.com/cloudprober/cloudprober/internal/rds/gcp/proto"
	proto2 "github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ServerConf struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// List of providers that server supports.
	Provider      []*Provider `protobuf:"bytes,1,rep,name=provider" json:"provider,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ServerConf) Reset() {
	*x = ServerConf{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConf) ProtoMessage() {}

func (x *ServerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerConf.ProtoReflect.Descriptor instead.
func (*ServerConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *ServerConf) GetProvider() []*Provider {
	if x != nil {
		return x.Provider
	}
	return nil
}

type Provider struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Provider identifier, e.g. "gcp". Server routes incoming requests to various
	// providers based on this id.
	Id *string `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	// Types that are valid to be assigned to Config:
	//
	//	*Provider_FileConfig
	//	*Provider_GcpConfig
	//	*Provider_KubernetesConfig
	Config        isProvider_Config `protobuf_oneof:"config"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Provider) Reset() {
	*x = Provider{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Provider) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Provider) ProtoMessage() {}

func (x *Provider) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Provider.ProtoReflect.Descriptor instead.
func (*Provider) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *Provider) GetId() string {
	if x != nil && x.Id != nil {
		return *x.Id
	}
	return ""
}

func (x *Provider) GetConfig() isProvider_Config {
	if x != nil {
		return x.Config
	}
	return nil
}

func (x *Provider) GetFileConfig() *proto.ProviderConfig {
	if x != nil {
		if x, ok := x.Config.(*Provider_FileConfig); ok {
			return x.FileConfig
		}
	}
	return nil
}

func (x *Provider) GetGcpConfig() *proto1.ProviderConfig {
	if x != nil {
		if x, ok := x.Config.(*Provider_GcpConfig); ok {
			return x.GcpConfig
		}
	}
	return nil
}

func (x *Provider) GetKubernetesConfig() *proto2.ProviderConfig {
	if x != nil {
		if x, ok := x.Config.(*Provider_KubernetesConfig); ok {
			return x.KubernetesConfig
		}
	}
	return nil
}

type isProvider_Config interface {
	isProvider_Config()
}

type Provider_FileConfig struct {
	FileConfig *proto.ProviderConfig `protobuf:"bytes,4,opt,name=file_config,json=fileConfig,oneof"`
}

type Provider_GcpConfig struct {
	GcpConfig *proto1.ProviderConfig `protobuf:"bytes,2,opt,name=gcp_config,json=gcpConfig,oneof"`
}

type Provider_KubernetesConfig struct {
	KubernetesConfig *proto2.ProviderConfig `protobuf:"bytes,3,opt,name=kubernetes_config,json=kubernetesConfig,oneof"`
}

func (*Provider_FileConfig) isProvider_Config() {}

func (*Provider_GcpConfig) isProvider_Config() {}

func (*Provider_KubernetesConfig) isProvider_Config() {}

var File_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto protoreflect.FileDescriptor

const file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDesc = "" +
	"\n" +
	"Igithub.com/cloudprober/cloudprober/internal/rds/server/proto/config.proto\x12\x0fcloudprober.rds\x1aGgithub.com/cloudprober/cloudprober/internal/rds/file/proto/config.proto\x1aFgithub.com/cloudprober/cloudprober/internal/rds/gcp/proto/config.proto\x1aMgithub.com/cloudprober/cloudprober/internal/rds/kubernetes/proto/config.proto\"C\n" +
	"\n" +
	"ServerConf\x125\n" +
	"\bprovider\x18\x01 \x03(\v2\x19.cloudprober.rds.ProviderR\bprovider\"\x8e\x02\n" +
	"\bProvider\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12G\n" +
	"\vfile_config\x18\x04 \x01(\v2$.cloudprober.rds.file.ProviderConfigH\x00R\n" +
	"fileConfig\x12D\n" +
	"\n" +
	"gcp_config\x18\x02 \x01(\v2#.cloudprober.rds.gcp.ProviderConfigH\x00R\tgcpConfig\x12Y\n" +
	"\x11kubernetes_config\x18\x03 \x01(\v2*.cloudprober.rds.kubernetes.ProviderConfigH\x00R\x10kubernetesConfigB\b\n" +
	"\x06configB>Z<github.com/cloudprober/cloudprober/internal/rds/server/proto"

var (
	file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_goTypes = []any{
	(*ServerConf)(nil),            // 0: cloudprober.rds.ServerConf
	(*Provider)(nil),              // 1: cloudprober.rds.Provider
	(*proto.ProviderConfig)(nil),  // 2: cloudprober.rds.file.ProviderConfig
	(*proto1.ProviderConfig)(nil), // 3: cloudprober.rds.gcp.ProviderConfig
	(*proto2.ProviderConfig)(nil), // 4: cloudprober.rds.kubernetes.ProviderConfig
}
var file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_depIdxs = []int32{
	1, // 0: cloudprober.rds.ServerConf.provider:type_name -> cloudprober.rds.Provider
	2, // 1: cloudprober.rds.Provider.file_config:type_name -> cloudprober.rds.file.ProviderConfig
	3, // 2: cloudprober.rds.Provider.gcp_config:type_name -> cloudprober.rds.gcp.ProviderConfig
	4, // 3: cloudprober.rds.Provider.kubernetes_config:type_name -> cloudprober.rds.kubernetes.ProviderConfig
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto != nil {
		return
	}
	file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_msgTypes[1].OneofWrappers = []any{
		(*Provider_FileConfig)(nil),
		(*Provider_GcpConfig)(nil),
		(*Provider_KubernetesConfig)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_internal_rds_server_proto_config_proto_depIdxs = nil
}
