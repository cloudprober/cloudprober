// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.3.0
// source: github.com/cloudprober/cloudprober/validators/proto/config.proto

package proto

import (
	proto "github.com/cloudprober/cloudprober/validators/http/proto"
	proto1 "github.com/cloudprober/cloudprober/validators/integrity/proto"
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

type Validator struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name *string `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	// Types that are assignable to Type:
	//	*Validator_HttpValidator
	//	*Validator_IntegrityValidator
	//	*Validator_Regex
	Type isValidator_Type `protobuf_oneof:"type"`
}

func (x *Validator) Reset() {
	*x = Validator{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_validators_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Validator) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Validator) ProtoMessage() {}

func (x *Validator) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_validators_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Validator.ProtoReflect.Descriptor instead.
func (*Validator) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *Validator) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (m *Validator) GetType() isValidator_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *Validator) GetHttpValidator() *proto.Validator {
	if x, ok := x.GetType().(*Validator_HttpValidator); ok {
		return x.HttpValidator
	}
	return nil
}

func (x *Validator) GetIntegrityValidator() *proto1.Validator {
	if x, ok := x.GetType().(*Validator_IntegrityValidator); ok {
		return x.IntegrityValidator
	}
	return nil
}

func (x *Validator) GetRegex() string {
	if x, ok := x.GetType().(*Validator_Regex); ok {
		return x.Regex
	}
	return ""
}

type isValidator_Type interface {
	isValidator_Type()
}

type Validator_HttpValidator struct {
	HttpValidator *proto.Validator `protobuf:"bytes,2,opt,name=http_validator,json=httpValidator,oneof"`
}

type Validator_IntegrityValidator struct {
	// Data integrity validator
	IntegrityValidator *proto1.Validator `protobuf:"bytes,3,opt,name=integrity_validator,json=integrityValidator,oneof"`
}

type Validator_Regex struct {
	// Regex validator
	Regex string `protobuf:"bytes,4,opt,name=regex,oneof"`
}

func (*Validator_HttpValidator) isValidator_Type() {}

func (*Validator_IntegrityValidator) isValidator_Type() {}

func (*Validator_Regex) isValidator_Type() {}

var File_github_com_cloudprober_cloudprober_validators_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDesc = []byte{
	0x0a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x73, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x16, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e,
	0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x73, 0x1a, 0x45, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x4a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x73,
	0x2f, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x69, 0x74, 0x79, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xf0, 0x01,
	0x0a, 0x09, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12,
	0x4f, 0x0a, 0x0e, 0x68, 0x74, 0x74, 0x70, 0x5f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f,
	0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x73,
	0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x48,
	0x00, 0x52, 0x0d, 0x68, 0x74, 0x74, 0x70, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72,
	0x12, 0x5e, 0x0a, 0x13, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x69, 0x74, 0x79, 0x5f, 0x76, 0x61,
	0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2b, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x76, 0x61, 0x6c, 0x69,
	0x64, 0x61, 0x74, 0x6f, 0x72, 0x73, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x69, 0x74, 0x79,
	0x2e, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x48, 0x00, 0x52, 0x12, 0x69, 0x6e,
	0x74, 0x65, 0x67, 0x72, 0x69, 0x74, 0x79, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72,
	0x12, 0x16, 0x0a, 0x05, 0x72, 0x65, 0x67, 0x65, 0x78, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48,
	0x00, 0x52, 0x05, 0x72, 0x65, 0x67, 0x65, 0x78, 0x42, 0x06, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65,
	0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72,
	0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_validators_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_validators_proto_config_proto_goTypes = []interface{}{
	(*Validator)(nil),        // 0: cloudprober.validators.Validator
	(*proto.Validator)(nil),  // 1: cloudprober.validators.http.Validator
	(*proto1.Validator)(nil), // 2: cloudprober.validators.integrity.Validator
}
var file_github_com_cloudprober_cloudprober_validators_proto_config_proto_depIdxs = []int32{
	1, // 0: cloudprober.validators.Validator.http_validator:type_name -> cloudprober.validators.http.Validator
	2, // 1: cloudprober.validators.Validator.integrity_validator:type_name -> cloudprober.validators.integrity.Validator
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_validators_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_validators_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_validators_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_validators_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Validator); i {
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
	file_github_com_cloudprober_cloudprober_validators_proto_config_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Validator_HttpValidator)(nil),
		(*Validator_IntegrityValidator)(nil),
		(*Validator_Regex)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_validators_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_validators_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_validators_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_validators_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_validators_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_validators_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_validators_proto_config_proto_depIdxs = nil
}
