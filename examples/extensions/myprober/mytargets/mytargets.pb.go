// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/examples/extensions/myprober/mytargets/mytargets.proto

package mytargets

import (
	proto "github.com/cloudprober/cloudprober/targets/proto"
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

type MyTargetsConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Hostname *string `protobuf:"bytes,1,opt,name=hostname" json:"hostname,omitempty"`
	Port     *int32  `protobuf:"varint,2,opt,name=port" json:"port,omitempty"`
}

func (x *MyTargetsConf) Reset() {
	*x = MyTargetsConf{}
	mi := &file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MyTargetsConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MyTargetsConf) ProtoMessage() {}

func (x *MyTargetsConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MyTargetsConf.ProtoReflect.Descriptor instead.
func (*MyTargetsConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDescGZIP(), []int{0}
}

func (x *MyTargetsConf) GetHostname() string {
	if x != nil && x.Hostname != nil {
		return *x.Hostname
	}
	return ""
}

func (x *MyTargetsConf) GetPort() int32 {
	if x != nil && x.Port != nil {
		return *x.Port
	}
	return 0
}

var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*proto.TargetsDef)(nil),
		ExtensionType: (*MyTargetsConf)(nil),
		Field:         200,
		Name:          "myprober.mytargets",
		Tag:           "bytes,200,opt,name=mytargets",
		Filename:      "github.com/cloudprober/cloudprober/examples/extensions/myprober/mytargets/mytargets.proto",
	},
}

// Extension fields to proto.TargetsDef.
var (
	// optional myprober.MyTargetsConf mytargets = 200;
	E_Mytargets = &file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_extTypes[0]
)

var File_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDesc = []byte{
	0x0a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2f, 0x65, 0x78,
	0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2f, 0x6d, 0x79, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x6d, 0x79, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x6d, 0x79, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x1a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3f, 0x0a, 0x0d, 0x4d, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x1a, 0x0a, 0x08, 0x68, 0x6f, 0x73, 0x74, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x68, 0x6f, 0x73, 0x74, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x3a, 0x57, 0x0a, 0x09, 0x6d, 0x79, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x12, 0x1f, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x73, 0x44, 0x65, 0x66, 0x18, 0xc8, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x6d, 0x79,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x4d, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
	0x43, 0x6f, 0x6e, 0x66, 0x52, 0x09, 0x6d, 0x79, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x42,
	0x4b, 0x5a, 0x49, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2f, 0x65,
	0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x6d, 0x79, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
}

var (
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDescData = file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_goTypes = []any{
	(*MyTargetsConf)(nil),    // 0: myprober.MyTargetsConf
	(*proto.TargetsDef)(nil), // 1: cloudprober.targets.TargetsDef
}
var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_depIdxs = []int32{
	1, // 0: myprober.mytargets:extendee -> cloudprober.targets.TargetsDef
	0, // 1: myprober.mytargets:type_name -> myprober.MyTargetsConf
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	1, // [1:2] is the sub-list for extension type_name
	0, // [0:1] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_init()
}
func file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_init() {
	if File_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 1,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_msgTypes,
		ExtensionInfos:    file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_extTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto = out.File
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_mytargets_mytargets_proto_depIdxs = nil
}
