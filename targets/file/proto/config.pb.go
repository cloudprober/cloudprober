// Configuration proto for File targets.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/targets/file/proto/config.proto

package proto

import (
	proto1 "github.com/cloudprober/cloudprober/rds/file/proto"
	proto "github.com/cloudprober/cloudprober/rds/proto"
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

type TargetsConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// File that contains resources in either textproto or json format.
	// Example in textproto format:
	//
	//	resource {
	//	  name: "switch-xx-01"
	//	  ip: "10.11.112.3"
	//	  port: 8080
	//	  labels {
	//	    key: "device_type"
	//	    value: "switch"
	//	  }
	//	}
	//
	//	resource {
	//	  name: "switch-yy-01"
	//	  ip: "10.16.110.12"
	//	  port: 8080
	//	}
	FilePath *string                       `protobuf:"bytes,1,opt,name=file_path,json=filePath" json:"file_path,omitempty"`
	Filter   []*proto.Filter               `protobuf:"bytes,2,rep,name=filter" json:"filter,omitempty"`
	Format   *proto1.ProviderConfig_Format `protobuf:"varint,3,opt,name=format,enum=cloudprober.rds.file.ProviderConfig_Format" json:"format,omitempty"`
	// If specified, file will be re-read at the given interval.
	ReEvalSec *int32 `protobuf:"varint,4,opt,name=re_eval_sec,json=reEvalSec" json:"re_eval_sec,omitempty"`
}

func (x *TargetsConf) Reset() {
	*x = TargetsConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TargetsConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TargetsConf) ProtoMessage() {}

func (x *TargetsConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TargetsConf.ProtoReflect.Descriptor instead.
func (*TargetsConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *TargetsConf) GetFilePath() string {
	if x != nil && x.FilePath != nil {
		return *x.FilePath
	}
	return ""
}

func (x *TargetsConf) GetFilter() []*proto.Filter {
	if x != nil {
		return x.Filter
	}
	return nil
}

func (x *TargetsConf) GetFormat() proto1.ProviderConfig_Format {
	if x != nil && x.Format != nil {
		return *x.Format
	}
	return proto1.ProviderConfig_Format(0)
}

func (x *TargetsConf) GetReEvalSec() int32 {
	if x != nil && x.ReEvalSec != nil {
		return *x.ReEvalSec
	}
	return 0
}

var File_github_com_cloudprober_cloudprober_targets_file_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDesc = []byte{
	0x0a, 0x42, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x66, 0x69, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x18, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x1a, 0x3e,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x72, 0x64, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x36,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x72, 0x64, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x72, 0x64, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc0, 0x01, 0x0a, 0x0b, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x1b, 0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x70,
	0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x50,
	0x61, 0x74, 0x68, 0x12, 0x2f, 0x0a, 0x06, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x18, 0x02, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x06, 0x66, 0x69,
	0x6c, 0x74, 0x65, 0x72, 0x12, 0x43, 0x0a, 0x06, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x2b, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x50, 0x72, 0x6f, 0x76,
	0x69, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x46, 0x6f, 0x72, 0x6d, 0x61,
	0x74, 0x52, 0x06, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x12, 0x1e, 0x0a, 0x0b, 0x72, 0x65, 0x5f,
	0x65, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09,
	0x72, 0x65, 0x45, 0x76, 0x61, 0x6c, 0x53, 0x65, 0x63, 0x42, 0x37, 0x5a, 0x35, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_goTypes = []interface{}{
	(*TargetsConf)(nil),               // 0: cloudprober.targets.file.TargetsConf
	(*proto.Filter)(nil),              // 1: cloudprober.rds.Filter
	(proto1.ProviderConfig_Format)(0), // 2: cloudprober.rds.file.ProviderConfig.Format
}
var file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_depIdxs = []int32{
	1, // 0: cloudprober.targets.file.TargetsConf.filter:type_name -> cloudprober.rds.Filter
	2, // 1: cloudprober.targets.file.TargetsConf.format:type_name -> cloudprober.rds.file.ProviderConfig.Format
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_targets_file_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TargetsConf); i {
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
			RawDescriptor: file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_targets_file_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_targets_file_proto_config_proto_depIdxs = nil
}
