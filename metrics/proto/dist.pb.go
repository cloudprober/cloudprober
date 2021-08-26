// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: github.com/google/cloudprober/metrics/proto/dist.proto

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

// Dist defines a Distribution data type.
type Dist struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Buckets:
	//	*Dist_ExplicitBuckets
	//	*Dist_ExponentialBuckets
	Buckets isDist_Buckets `protobuf_oneof:"buckets"`
}

func (x *Dist) Reset() {
	*x = Dist{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Dist) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Dist) ProtoMessage() {}

func (x *Dist) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Dist.ProtoReflect.Descriptor instead.
func (*Dist) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescGZIP(), []int{0}
}

func (m *Dist) GetBuckets() isDist_Buckets {
	if m != nil {
		return m.Buckets
	}
	return nil
}

func (x *Dist) GetExplicitBuckets() string {
	if x, ok := x.GetBuckets().(*Dist_ExplicitBuckets); ok {
		return x.ExplicitBuckets
	}
	return ""
}

func (x *Dist) GetExponentialBuckets() *ExponentialBuckets {
	if x, ok := x.GetBuckets().(*Dist_ExponentialBuckets); ok {
		return x.ExponentialBuckets
	}
	return nil
}

type isDist_Buckets interface {
	isDist_Buckets()
}

type Dist_ExplicitBuckets struct {
	// Comma-separated list of lower bounds, where each lower bound is a float
	// value. Example: 0.5,1,2,4,8.
	ExplicitBuckets string `protobuf:"bytes,1,opt,name=explicit_buckets,json=explicitBuckets,oneof"`
}

type Dist_ExponentialBuckets struct {
	// Exponentially growing buckets
	ExponentialBuckets *ExponentialBuckets `protobuf:"bytes,2,opt,name=exponential_buckets,json=exponentialBuckets,oneof"`
}

func (*Dist_ExplicitBuckets) isDist_Buckets() {}

func (*Dist_ExponentialBuckets) isDist_Buckets() {}

// ExponentialBucket defines a set of num_buckets+2 buckets:
//   bucket[0] covers (−Inf, 0)
//   bucket[1] covers [0, scale_factor)
//   bucket[2] covers [scale_factor, scale_factor*base)
//   ...
//   bucket[i] covers [scale_factor*base^(i−2), scale_factor*base^(i−1))
//   ...
//   bucket[num_buckets+1] covers [scale_factor*base^(num_buckets−1), +Inf)
// NB: Base must be at least 1.01.
type ExponentialBuckets struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScaleFactor *float32 `protobuf:"fixed32,1,opt,name=scale_factor,json=scaleFactor,def=1" json:"scale_factor,omitempty"`
	Base        *float32 `protobuf:"fixed32,2,opt,name=base,def=2" json:"base,omitempty"`
	NumBuckets  *uint32  `protobuf:"varint,3,opt,name=num_buckets,json=numBuckets,def=20" json:"num_buckets,omitempty"`
}

// Default values for ExponentialBuckets fields.
const (
	Default_ExponentialBuckets_ScaleFactor = float32(1)
	Default_ExponentialBuckets_Base        = float32(2)
	Default_ExponentialBuckets_NumBuckets  = uint32(20)
)

func (x *ExponentialBuckets) Reset() {
	*x = ExponentialBuckets{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExponentialBuckets) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExponentialBuckets) ProtoMessage() {}

func (x *ExponentialBuckets) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExponentialBuckets.ProtoReflect.Descriptor instead.
func (*ExponentialBuckets) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescGZIP(), []int{1}
}

func (x *ExponentialBuckets) GetScaleFactor() float32 {
	if x != nil && x.ScaleFactor != nil {
		return *x.ScaleFactor
	}
	return Default_ExponentialBuckets_ScaleFactor
}

func (x *ExponentialBuckets) GetBase() float32 {
	if x != nil && x.Base != nil {
		return *x.Base
	}
	return Default_ExponentialBuckets_Base
}

func (x *ExponentialBuckets) GetNumBuckets() uint32 {
	if x != nil && x.NumBuckets != nil {
		return *x.NumBuckets
	}
	return Default_ExponentialBuckets_NumBuckets
}

var File_github_com_google_cloudprober_metrics_proto_dist_proto protoreflect.FileDescriptor

var file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDesc = []byte{
	0x0a, 0x36, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x64, 0x69,
	0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x22, 0x9a, 0x01,
	0x0a, 0x04, 0x44, 0x69, 0x73, 0x74, 0x12, 0x2b, 0x0a, 0x10, 0x65, 0x78, 0x70, 0x6c, 0x69, 0x63,
	0x69, 0x74, 0x5f, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x48, 0x00, 0x52, 0x0f, 0x65, 0x78, 0x70, 0x6c, 0x69, 0x63, 0x69, 0x74, 0x42, 0x75, 0x63, 0x6b,
	0x65, 0x74, 0x73, 0x12, 0x5a, 0x0a, 0x13, 0x65, 0x78, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x69,
	0x61, 0x6c, 0x5f, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x27, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x6d,
	0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x45, 0x78, 0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x69,
	0x61, 0x6c, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73, 0x48, 0x00, 0x52, 0x12, 0x65, 0x78, 0x70,
	0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73, 0x42,
	0x09, 0x0a, 0x07, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73, 0x22, 0x76, 0x0a, 0x12, 0x45, 0x78,
	0x70, 0x6f, 0x6e, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x42, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73,
	0x12, 0x24, 0x0a, 0x0c, 0x73, 0x63, 0x61, 0x6c, 0x65, 0x5f, 0x66, 0x61, 0x63, 0x74, 0x6f, 0x72,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x02, 0x3a, 0x01, 0x31, 0x52, 0x0b, 0x73, 0x63, 0x61, 0x6c, 0x65,
	0x46, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x15, 0x0a, 0x04, 0x62, 0x61, 0x73, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x02, 0x3a, 0x01, 0x32, 0x52, 0x04, 0x62, 0x61, 0x73, 0x65, 0x12, 0x23, 0x0a,
	0x0b, 0x6e, 0x75, 0x6d, 0x5f, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x73, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0d, 0x3a, 0x02, 0x32, 0x30, 0x52, 0x0a, 0x6e, 0x75, 0x6d, 0x42, 0x75, 0x63, 0x6b, 0x65,
	0x74, 0x73, 0x42, 0x2d, 0x5a, 0x2b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f,
}

var (
	file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescOnce sync.Once
	file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescData = file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDesc
)

func file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescGZIP() []byte {
	file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescOnce.Do(func() {
		file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescData)
	})
	return file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDescData
}

var file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_google_cloudprober_metrics_proto_dist_proto_goTypes = []interface{}{
	(*Dist)(nil),               // 0: cloudprober.metrics.Dist
	(*ExponentialBuckets)(nil), // 1: cloudprober.metrics.ExponentialBuckets
}
var file_github_com_google_cloudprober_metrics_proto_dist_proto_depIdxs = []int32{
	1, // 0: cloudprober.metrics.Dist.exponential_buckets:type_name -> cloudprober.metrics.ExponentialBuckets
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_github_com_google_cloudprober_metrics_proto_dist_proto_init() }
func file_github_com_google_cloudprober_metrics_proto_dist_proto_init() {
	if File_github_com_google_cloudprober_metrics_proto_dist_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Dist); i {
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
		file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExponentialBuckets); i {
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
	file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Dist_ExplicitBuckets)(nil),
		(*Dist_ExponentialBuckets)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_google_cloudprober_metrics_proto_dist_proto_goTypes,
		DependencyIndexes: file_github_com_google_cloudprober_metrics_proto_dist_proto_depIdxs,
		MessageInfos:      file_github_com_google_cloudprober_metrics_proto_dist_proto_msgTypes,
	}.Build()
	File_github_com_google_cloudprober_metrics_proto_dist_proto = out.File
	file_github_com_google_cloudprober_metrics_proto_dist_proto_rawDesc = nil
	file_github_com_google_cloudprober_metrics_proto_dist_proto_goTypes = nil
	file_github_com_google_cloudprober_metrics_proto_dist_proto_depIdxs = nil
}
