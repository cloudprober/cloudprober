// This protobuf defines a new cloudprober probe type.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/examples/extensions/myprober/myprobe/myprobe.proto

package myprobe

import (
	proto "github.com/cloudprober/cloudprober/probes/proto"
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

// Redis operation
type ProbeConf_Op int32

const (
	ProbeConf_GET    ProbeConf_Op = 0
	ProbeConf_SET    ProbeConf_Op = 1
	ProbeConf_DELETE ProbeConf_Op = 2
)

// Enum value maps for ProbeConf_Op.
var (
	ProbeConf_Op_name = map[int32]string{
		0: "GET",
		1: "SET",
		2: "DELETE",
	}
	ProbeConf_Op_value = map[string]int32{
		"GET":    0,
		"SET":    1,
		"DELETE": 2,
	}
)

func (x ProbeConf_Op) Enum() *ProbeConf_Op {
	p := new(ProbeConf_Op)
	*p = x
	return p
}

func (x ProbeConf_Op) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ProbeConf_Op) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_enumTypes[0].Descriptor()
}

func (ProbeConf_Op) Type() protoreflect.EnumType {
	return &file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_enumTypes[0]
}

func (x ProbeConf_Op) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ProbeConf_Op) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ProbeConf_Op(num)
	return nil
}

// Deprecated: Use ProbeConf_Op.Descriptor instead.
func (ProbeConf_Op) EnumDescriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescGZIP(), []int{0, 0}
}

type ProbeConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Op *ProbeConf_Op `protobuf:"varint,1,req,name=op,enum=myprober.ProbeConf_Op" json:"op,omitempty"`
	// Key and value for the redis operation
	Key   *string `protobuf:"bytes,2,req,name=key" json:"key,omitempty"`
	Value *string `protobuf:"bytes,3,opt,name=value" json:"value,omitempty"`
}

func (x *ProbeConf) Reset() {
	*x = ProbeConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProbeConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProbeConf) ProtoMessage() {}

func (x *ProbeConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProbeConf.ProtoReflect.Descriptor instead.
func (*ProbeConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescGZIP(), []int{0}
}

func (x *ProbeConf) GetOp() ProbeConf_Op {
	if x != nil && x.Op != nil {
		return *x.Op
	}
	return ProbeConf_GET
}

func (x *ProbeConf) GetKey() string {
	if x != nil && x.Key != nil {
		return *x.Key
	}
	return ""
}

func (x *ProbeConf) GetValue() string {
	if x != nil && x.Value != nil {
		return *x.Value
	}
	return ""
}

var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*proto.ProbeDef)(nil),
		ExtensionType: (*ProbeConf)(nil),
		Field:         200,
		Name:          "myprober.redis_probe",
		Tag:           "bytes,200,opt,name=redis_probe",
		Filename:      "github.com/cloudprober/cloudprober/examples/extensions/myprober/myprobe/myprobe.proto",
	},
}

// Extension fields to proto.ProbeDef.
var (
	// optional myprober.ProbeConf redis_probe = 200;
	E_RedisProbe = &file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_extTypes[0]
)

var File_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDesc = []byte{
	0x0a, 0x55, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x2f, 0x65, 0x78,
	0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2f, 0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2f, 0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x2f, 0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x1a, 0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x7f, 0x0a, 0x09, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x26, 0x0a, 0x02,
	0x6f, 0x70, 0x18, 0x01, 0x20, 0x02, 0x28, 0x0e, 0x32, 0x16, 0x2e, 0x6d, 0x79, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2e, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x4f, 0x70,
	0x52, 0x02, 0x6f, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x02, 0x28,
	0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0x22, 0x0a, 0x02,
	0x4f, 0x70, 0x12, 0x07, 0x0a, 0x03, 0x47, 0x45, 0x54, 0x10, 0x00, 0x12, 0x07, 0x0a, 0x03, 0x53,
	0x45, 0x54, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x44, 0x45, 0x4c, 0x45, 0x54, 0x45, 0x10, 0x02,
	0x3a, 0x53, 0x0a, 0x0b, 0x72, 0x65, 0x64, 0x69, 0x73, 0x5f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x12,
	0x1c, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x73, 0x2e, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x44, 0x65, 0x66, 0x18, 0xc8, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e,
	0x50, 0x72, 0x6f, 0x62, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x52, 0x0a, 0x72, 0x65, 0x64, 0x69, 0x73,
	0x50, 0x72, 0x6f, 0x62, 0x65, 0x42, 0x49, 0x5a, 0x47, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x65, 0x78, 0x61, 0x6d,
	0x70, 0x6c, 0x65, 0x73, 0x2f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x2f,
	0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x6d, 0x79, 0x70, 0x72, 0x6f, 0x62, 0x65,
}

var (
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescData = file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_goTypes = []interface{}{
	(ProbeConf_Op)(0),      // 0: myprober.ProbeConf.Op
	(*ProbeConf)(nil),      // 1: myprober.ProbeConf
	(*proto.ProbeDef)(nil), // 2: cloudprober.probes.ProbeDef
}
var file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_depIdxs = []int32{
	0, // 0: myprober.ProbeConf.op:type_name -> myprober.ProbeConf.Op
	2, // 1: myprober.redis_probe:extendee -> cloudprober.probes.ProbeDef
	1, // 2: myprober.redis_probe:type_name -> myprober.ProbeConf
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	2, // [2:3] is the sub-list for extension type_name
	1, // [1:2] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_init()
}
func file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_init() {
	if File_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProbeConf); i {
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
			RawDescriptor: file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 1,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_depIdxs,
		EnumInfos:         file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_enumTypes,
		MessageInfos:      file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_msgTypes,
		ExtensionInfos:    file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_extTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto = out.File
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_examples_extensions_myprober_myprobe_myprobe_proto_depIdxs = nil
}
