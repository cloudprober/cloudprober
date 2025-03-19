// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/surfacers/internal/cloudwatch/proto/config.proto

package proto

import (
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

type SurfacerConf struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The cloudwatch metric namespace
	Namespace *string `protobuf:"bytes,1,opt,name=namespace,def=cloudprober" json:"namespace,omitempty"`
	// The cloudwatch resolution value, lowering this below 60 will incur
	// additional charges as the metrics will be charged at a high resolution
	// rate.
	Resolution *int32 `protobuf:"varint,2,opt,name=resolution,def=60" json:"resolution,omitempty"`
	// The AWS Region, used to create a CloudWatch session.
	// The order of fallback for evaluating the AWS Region:
	// 1. This config value.
	// 2. EC2 metadata endpoint, via cloudprober sysvars.
	// 3. AWS_REGION environment value.
	// 4. AWS_DEFAULT_REGION environment value, if AWS_SDK_LOAD_CONFIG is set.
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
	Region *string `protobuf:"bytes,3,opt,name=region" json:"region,omitempty"`
	// The maximum number of metrics that will be published at one
	// time. Metrics will be stored locally in a cache until this
	// limit is reached. 1000 is the maximum number of metrics
	// supported by the Cloudwatch PutMetricData API.
	// Metrics will be published when the timer expires, or the buffer is
	// full, whichever happens first.
	MetricsBatchSize *int32 `protobuf:"varint,4,opt,name=metrics_batch_size,json=metricsBatchSize,def=1000" json:"metrics_batch_size,omitempty"`
	// The maximum amount of time to hold metrics in the buffer (above).
	// Metrics will be published when the timer expires, or the buffer is
	// full, whichever happens first.
	BatchTimerSec *int32 `protobuf:"varint,5,opt,name=batch_timer_sec,json=batchTimerSec,def=30" json:"batch_timer_sec,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for SurfacerConf fields.
const (
	Default_SurfacerConf_Namespace        = string("cloudprober")
	Default_SurfacerConf_Resolution       = int32(60)
	Default_SurfacerConf_MetricsBatchSize = int32(1000)
	Default_SurfacerConf_BatchTimerSec    = int32(30)
)

func (x *SurfacerConf) Reset() {
	*x = SurfacerConf{}
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SurfacerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerConf) ProtoMessage() {}

func (x *SurfacerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_msgTypes[0]
	if x != nil {
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
	return file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *SurfacerConf) GetNamespace() string {
	if x != nil && x.Namespace != nil {
		return *x.Namespace
	}
	return Default_SurfacerConf_Namespace
}

func (x *SurfacerConf) GetResolution() int32 {
	if x != nil && x.Resolution != nil {
		return *x.Resolution
	}
	return Default_SurfacerConf_Resolution
}

func (x *SurfacerConf) GetRegion() string {
	if x != nil && x.Region != nil {
		return *x.Region
	}
	return ""
}

func (x *SurfacerConf) GetMetricsBatchSize() int32 {
	if x != nil && x.MetricsBatchSize != nil {
		return *x.MetricsBatchSize
	}
	return Default_SurfacerConf_MetricsBatchSize
}

func (x *SurfacerConf) GetBatchTimerSec() int32 {
	if x != nil && x.BatchTimerSec != nil {
		return *x.BatchTimerSec
	}
	return Default_SurfacerConf_BatchTimerSec
}

var File_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDesc = string([]byte{
	0x0a, 0x53, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x77, 0x61, 0x74,
	0x63, 0x68, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x77, 0x61, 0x74, 0x63, 0x68, 0x22, 0xd5, 0x01, 0x0a, 0x0c, 0x53, 0x75, 0x72, 0x66, 0x61,
	0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x29, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73,
	0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x0b, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x12, 0x22, 0x0a, 0x0a, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x02, 0x36, 0x30, 0x52, 0x0a, 0x72, 0x65, 0x73, 0x6f,
	0x6c, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x12, 0x32,
	0x0a, 0x12, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x62, 0x61, 0x74, 0x63, 0x68, 0x5f,
	0x73, 0x69, 0x7a, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x04, 0x31, 0x30, 0x30, 0x30,
	0x52, 0x10, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x42, 0x61, 0x74, 0x63, 0x68, 0x53, 0x69,
	0x7a, 0x65, 0x12, 0x2a, 0x0a, 0x0f, 0x62, 0x61, 0x74, 0x63, 0x68, 0x5f, 0x74, 0x69, 0x6d, 0x65,
	0x72, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x02, 0x33, 0x30, 0x52,
	0x0d, 0x62, 0x61, 0x74, 0x63, 0x68, 0x54, 0x69, 0x6d, 0x65, 0x72, 0x53, 0x65, 0x63, 0x42, 0x48,
	0x5a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x77, 0x61, 0x74,
	0x63, 0x68, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
})

var (
	file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_goTypes = []any{
	(*SurfacerConf)(nil), // 0: cloudprober.surfacer.cloudwatch.SurfacerConf
}
var file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_init()
}
func file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_internal_cloudwatch_proto_config_proto_depIdxs = nil
}
