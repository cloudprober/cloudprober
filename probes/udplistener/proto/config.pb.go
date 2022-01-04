// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.17.3
// source: github.com/cloudprober/cloudprober/probes/udplistener/proto/config.proto

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

// Probe response to an incoming packet: echo back or discard.
type ProbeConf_Type int32

const (
	ProbeConf_INVALID ProbeConf_Type = 0
	ProbeConf_ECHO    ProbeConf_Type = 1
	ProbeConf_DISCARD ProbeConf_Type = 2
)

// Enum value maps for ProbeConf_Type.
var (
	ProbeConf_Type_name = map[int32]string{
		0: "INVALID",
		1: "ECHO",
		2: "DISCARD",
	}
	ProbeConf_Type_value = map[string]int32{
		"INVALID": 0,
		"ECHO":    1,
		"DISCARD": 2,
	}
)

func (x ProbeConf_Type) Enum() *ProbeConf_Type {
	p := new(ProbeConf_Type)
	*p = x
	return p
}

func (x ProbeConf_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ProbeConf_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_enumTypes[0].Descriptor()
}

func (ProbeConf_Type) Type() protoreflect.EnumType {
	return &file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_enumTypes[0]
}

func (x ProbeConf_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ProbeConf_Type) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ProbeConf_Type(num)
	return nil
}

// Deprecated: Use ProbeConf_Type.Descriptor instead.
func (ProbeConf_Type) EnumDescriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescGZIP(), []int{0, 0}
}

type ProbeConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Port to listen.
	Port *int32          `protobuf:"varint,3,opt,name=port,def=32212" json:"port,omitempty"`
	Type *ProbeConf_Type `protobuf:"varint,4,opt,name=type,enum=cloudprober.probes.udplistener.ProbeConf_Type" json:"type,omitempty"`
	// Number of packets sent in a single probe.
	PacketsPerProbe *int32 `protobuf:"varint,5,opt,name=packets_per_probe,json=packetsPerProbe,def=1" json:"packets_per_probe,omitempty"`
}

// Default values for ProbeConf fields.
const (
	Default_ProbeConf_Port            = int32(32212)
	Default_ProbeConf_PacketsPerProbe = int32(1)
)

func (x *ProbeConf) Reset() {
	*x = ProbeConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProbeConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProbeConf) ProtoMessage() {}

func (x *ProbeConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_msgTypes[0]
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
	return file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *ProbeConf) GetPort() int32 {
	if x != nil && x.Port != nil {
		return *x.Port
	}
	return Default_ProbeConf_Port
}

func (x *ProbeConf) GetType() ProbeConf_Type {
	if x != nil && x.Type != nil {
		return *x.Type
	}
	return ProbeConf_INVALID
}

func (x *ProbeConf) GetPacketsPerProbe() int32 {
	if x != nil && x.PacketsPerProbe != nil {
		return *x.PacketsPerProbe
	}
	return Default_ProbeConf_PacketsPerProbe
}

var File_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDesc = []byte{
	0x0a, 0x48, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2f, 0x75, 0x64, 0x70, 0x6c,
	0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x75,
	0x64, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x22, 0xc5, 0x01, 0x0a, 0x09, 0x50,
	0x72, 0x6f, 0x62, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x19, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x05, 0x33, 0x32, 0x32, 0x31, 0x32, 0x52, 0x04, 0x70,
	0x6f, 0x72, 0x74, 0x12, 0x42, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x2e, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x75, 0x64, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e,
	0x65, 0x72, 0x2e, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x54, 0x79, 0x70,
	0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x2d, 0x0a, 0x11, 0x70, 0x61, 0x63, 0x6b, 0x65,
	0x74, 0x73, 0x5f, 0x70, 0x65, 0x72, 0x5f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x05, 0x3a, 0x01, 0x31, 0x52, 0x0f, 0x70, 0x61, 0x63, 0x6b, 0x65, 0x74, 0x73, 0x50, 0x65,
	0x72, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x22, 0x2a, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0b,
	0x0a, 0x07, 0x49, 0x4e, 0x56, 0x41, 0x4c, 0x49, 0x44, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x45,
	0x43, 0x48, 0x4f, 0x10, 0x01, 0x12, 0x0b, 0x0a, 0x07, 0x44, 0x49, 0x53, 0x43, 0x41, 0x52, 0x44,
	0x10, 0x02, 0x42, 0x3d, 0x5a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2f,
	0x75, 0x64, 0x70, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_goTypes = []interface{}{
	(ProbeConf_Type)(0), // 0: cloudprober.probes.udplistener.ProbeConf.Type
	(*ProbeConf)(nil),   // 1: cloudprober.probes.udplistener.ProbeConf
}
var file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.probes.udplistener.ProbeConf.type:type_name -> cloudprober.probes.udplistener.ProbeConf.Type
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
			RawDescriptor: file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_depIdxs,
		EnumInfos:         file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_enumTypes,
		MessageInfos:      file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_probes_udplistener_proto_config_proto_depIdxs = nil
}
