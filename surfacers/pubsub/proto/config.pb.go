// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.17.3
// source: github.com/cloudprober/cloudprober/surfacers/pubsub/proto/config.proto

package proto

import (
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

type SurfacerConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// GCP project name for pubsub. If not specified and running on GCP,
	// project is used.
	Project *string `protobuf:"bytes,1,opt,name=project" json:"project,omitempty"`
	// Pubsub topic name.
	// Default is cloudprober-{hostname}
	TopicName *string `protobuf:"bytes,2,opt,name=topic_name,json=topicName" json:"topic_name,omitempty"`
	// Compress data before writing to pubsub.
	CompressionEnabled *bool `protobuf:"varint,4,opt,name=compression_enabled,json=compressionEnabled,def=0" json:"compression_enabled,omitempty"`
}

// Default values for SurfacerConf fields.
const (
	Default_SurfacerConf_CompressionEnabled = bool(false)
)

func (x *SurfacerConf) Reset() {
	*x = SurfacerConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SurfacerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerConf) ProtoMessage() {}

func (x *SurfacerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SurfacerConf.ProtoReflect.Descriptor instead.
func (*SurfacerConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *SurfacerConf) GetProject() string {
	if x != nil && x.Project != nil {
		return *x.Project
	}
	return ""
}

func (x *SurfacerConf) GetTopicName() string {
	if x != nil && x.TopicName != nil {
		return *x.TopicName
	}
	return ""
}

func (x *SurfacerConf) GetCompressionEnabled() bool {
	if x != nil && x.CompressionEnabled != nil {
		return *x.CompressionEnabled
	}
	return Default_SurfacerConf_CompressionEnabled
}

var File_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDesc = []byte{
	0x0a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x70,
	0x75, 0x62, 0x73, 0x75, 0x62, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x70,
	0x75, 0x62, 0x73, 0x75, 0x62, 0x22, 0x7f, 0x0a, 0x0c, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65,
	0x72, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12,
	0x1d, 0x0a, 0x0a, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x36,
	0x0a, 0x13, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x65, 0x6e,
	0x61, 0x62, 0x6c, 0x65, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x3a, 0x05, 0x66, 0x61, 0x6c,
	0x73, 0x65, 0x52, 0x12, 0x63, 0x6f, 0x6d, 0x70, 0x72, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x45,
	0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64, 0x42, 0x3b, 0x5a, 0x39, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x70, 0x75, 0x62, 0x73, 0x75, 0x62, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_goTypes = []interface{}{
	(*SurfacerConf)(nil), // 0: cloudprober.surfacer.pubsub.SurfacerConf
}
var file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SurfacerConf); i {
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
			RawDescriptor: file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_pubsub_proto_config_proto_depIdxs = nil
}
