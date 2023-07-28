// Provides all configuration necesary to list targets for a cloudprober probe.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/targets/lameduck/proto/config.proto

package proto

import (
	proto "github.com/cloudprober/cloudprober/rds/client/proto"
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

type Options struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// How often to check for lame-ducked targets
	ReEvalSec *int32 `protobuf:"varint,1,opt,name=re_eval_sec,json=reEvalSec,def=10" json:"re_eval_sec,omitempty"`
	// Runtime config project. If running on GCE, this defaults to the project
	// containing the VM.
	RuntimeconfigProject *string `protobuf:"bytes,2,opt,name=runtimeconfig_project,json=runtimeconfigProject" json:"runtimeconfig_project,omitempty"`
	// Lame duck targets runtime config name. An operator will create a variable
	// here to mark a target as lame-ducked.
	RuntimeconfigName *string `protobuf:"bytes,3,opt,name=runtimeconfig_name,json=runtimeconfigName,def=lame-duck-targets" json:"runtimeconfig_name,omitempty"`
	// Lame duck targets pubsub topic name. An operator will create a message
	// here to mark a target as lame-ducked.
	PubsubTopic *string `protobuf:"bytes,7,opt,name=pubsub_topic,json=pubsubTopic" json:"pubsub_topic,omitempty"`
	// Lame duck expiration time. We ignore variables (targets) that have been
	// updated more than these many seconds ago. This is a safety mechanism for
	// failing to cleanup. Also, the idea is that if a target has actually
	// disappeared, automatic targets expansion will take care of that some time
	// during this expiration period.
	ExpirationSec *int32 `protobuf:"varint,4,opt,name=expiration_sec,json=expirationSec,def=300" json:"expiration_sec,omitempty"`
	// Use an RDS client to get lame-duck-targets.
	// This option is always true now and will be removed after v0.10.7.
	//
	// Deprecated: Marked as deprecated in github.com/cloudprober/cloudprober/targets/lameduck/proto/config.proto.
	UseRds *bool `protobuf:"varint,5,opt,name=use_rds,json=useRds" json:"use_rds,omitempty"`
	// RDS server options, for example:
	//
	//	rds_server_options {
	//	  server_address: "rds-server.xyz:9314"
	//	  oauth_config: {
	//	    ...
	//	  }
	RdsServerOptions *proto.ClientConf_ServerOptions `protobuf:"bytes,6,opt,name=rds_server_options,json=rdsServerOptions" json:"rds_server_options,omitempty"`
}

// Default values for Options fields.
const (
	Default_Options_ReEvalSec         = int32(10)
	Default_Options_RuntimeconfigName = string("lame-duck-targets")
	Default_Options_ExpirationSec     = int32(300)
)

func (x *Options) Reset() {
	*x = Options{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Options) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Options) ProtoMessage() {}

func (x *Options) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Options.ProtoReflect.Descriptor instead.
func (*Options) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *Options) GetReEvalSec() int32 {
	if x != nil && x.ReEvalSec != nil {
		return *x.ReEvalSec
	}
	return Default_Options_ReEvalSec
}

func (x *Options) GetRuntimeconfigProject() string {
	if x != nil && x.RuntimeconfigProject != nil {
		return *x.RuntimeconfigProject
	}
	return ""
}

func (x *Options) GetRuntimeconfigName() string {
	if x != nil && x.RuntimeconfigName != nil {
		return *x.RuntimeconfigName
	}
	return Default_Options_RuntimeconfigName
}

func (x *Options) GetPubsubTopic() string {
	if x != nil && x.PubsubTopic != nil {
		return *x.PubsubTopic
	}
	return ""
}

func (x *Options) GetExpirationSec() int32 {
	if x != nil && x.ExpirationSec != nil {
		return *x.ExpirationSec
	}
	return Default_Options_ExpirationSec
}

// Deprecated: Marked as deprecated in github.com/cloudprober/cloudprober/targets/lameduck/proto/config.proto.
func (x *Options) GetUseRds() bool {
	if x != nil && x.UseRds != nil {
		return *x.UseRds
	}
	return false
}

func (x *Options) GetRdsServerOptions() *proto.ClientConf_ServerOptions {
	if x != nil {
		return x.RdsServerOptions
	}
	return nil
}

var File_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDesc = []byte{
	0x0a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x6c, 0x61, 0x6d,
	0x65, 0x64, 0x75, 0x63, 0x6b, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1c, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x6c, 0x61,
	0x6d, 0x65, 0x64, 0x75, 0x63, 0x6b, 0x1a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x72, 0x64, 0x73, 0x2f, 0x63,
	0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xe9, 0x02, 0x0a, 0x07, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x12, 0x22, 0x0a, 0x0b, 0x72, 0x65, 0x5f, 0x65, 0x76, 0x61, 0x6c, 0x5f,
	0x73, 0x65, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x02, 0x31, 0x30, 0x52, 0x09, 0x72,
	0x65, 0x45, 0x76, 0x61, 0x6c, 0x53, 0x65, 0x63, 0x12, 0x33, 0x0a, 0x15, 0x72, 0x75, 0x6e, 0x74,
	0x69, 0x6d, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63,
	0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x40, 0x0a,
	0x12, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x11, 0x6c, 0x61, 0x6d, 0x65, 0x2d,
	0x64, 0x75, 0x63, 0x6b, 0x2d, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x52, 0x11, 0x72, 0x75,
	0x6e, 0x74, 0x69, 0x6d, 0x65, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x4e, 0x61, 0x6d, 0x65, 0x12,
	0x21, 0x0a, 0x0c, 0x70, 0x75, 0x62, 0x73, 0x75, 0x62, 0x5f, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x18,
	0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x70, 0x75, 0x62, 0x73, 0x75, 0x62, 0x54, 0x6f, 0x70,
	0x69, 0x63, 0x12, 0x2a, 0x0a, 0x0e, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x5f, 0x73, 0x65, 0x63, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x03, 0x33, 0x30, 0x30, 0x52,
	0x0d, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x63, 0x12, 0x1b,
	0x0a, 0x07, 0x75, 0x73, 0x65, 0x5f, 0x72, 0x64, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x42,
	0x02, 0x18, 0x01, 0x52, 0x06, 0x75, 0x73, 0x65, 0x52, 0x64, 0x73, 0x12, 0x57, 0x0a, 0x12, 0x72,
	0x64, 0x73, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x52, 0x10, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x42, 0x3b, 0x5a, 0x39, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x2f, 0x6c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63, 0x6b, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_goTypes = []interface{}{
	(*Options)(nil),                        // 0: cloudprober.targets.lameduck.Options
	(*proto.ClientConf_ServerOptions)(nil), // 1: cloudprober.rds.ClientConf.ServerOptions
}
var file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_depIdxs = []int32{
	1, // 0: cloudprober.targets.lameduck.Options.rds_server_options:type_name -> cloudprober.rds.ClientConf.ServerOptions
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Options); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_targets_lameduck_proto_config_proto_depIdxs = nil
}
