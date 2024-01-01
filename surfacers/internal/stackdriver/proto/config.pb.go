// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/surfacers/internal/stackdriver/proto/config.proto

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

type SurfacerConf_MetricPrefix int32

const (
	SurfacerConf_NONE        SurfacerConf_MetricPrefix = 0 // monitoring_url/metric_name
	SurfacerConf_PROBE       SurfacerConf_MetricPrefix = 1 // monitoring_url/probe/metric_name
	SurfacerConf_PTYPE_PROBE SurfacerConf_MetricPrefix = 2 // monitoring_url/ptype/probe/metric_name
)

// Enum value maps for SurfacerConf_MetricPrefix.
var (
	SurfacerConf_MetricPrefix_name = map[int32]string{
		0: "NONE",
		1: "PROBE",
		2: "PTYPE_PROBE",
	}
	SurfacerConf_MetricPrefix_value = map[string]int32{
		"NONE":        0,
		"PROBE":       1,
		"PTYPE_PROBE": 2,
	}
)

func (x SurfacerConf_MetricPrefix) Enum() *SurfacerConf_MetricPrefix {
	p := new(SurfacerConf_MetricPrefix)
	*p = x
	return p
}

func (x SurfacerConf_MetricPrefix) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (SurfacerConf_MetricPrefix) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_enumTypes[0].Descriptor()
}

func (SurfacerConf_MetricPrefix) Type() protoreflect.EnumType {
	return &file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_enumTypes[0]
}

func (x SurfacerConf_MetricPrefix) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *SurfacerConf_MetricPrefix) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = SurfacerConf_MetricPrefix(num)
	return nil
}

// Deprecated: Use SurfacerConf_MetricPrefix.Descriptor instead.
func (SurfacerConf_MetricPrefix) EnumDescriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescGZIP(), []int{0, 0}
}

type SurfacerConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// GCP project name for stackdriver. If not specified and running on GCP,
	// project is used.
	Project *string `protobuf:"bytes,1,opt,name=project" json:"project,omitempty"`
	// How often to export metrics to stackdriver.
	BatchTimerSec *uint64 `protobuf:"varint,2,opt,name=batch_timer_sec,json=batchTimerSec,def=10" json:"batch_timer_sec,omitempty"`
	// If allowed_metrics_regex is specified, only metrics matching the given
	// regular expression will be exported to stackdriver. Since probe type and
	// probe name are part of the metric name, you can use this field to restrict
	// stackdriver metrics to a particular probe.
	// Example:
	// allowed_metrics_regex: ".*(http|ping).*(success|validation_failure).*"
	//
	// Deprecated: Please use the common surfacer options to filter metrics:
	// https://cloudprober.org/docs/surfacers/overview/#filtering-metrics
	AllowedMetricsRegex *string `protobuf:"bytes,3,opt,name=allowed_metrics_regex,json=allowedMetricsRegex" json:"allowed_metrics_regex,omitempty"`
	// Monitoring URL base. Full metric URL looks like the following:
	// <monitoring_url>/<ptype>/<probe>/<metric>
	// Example:
	// custom.googleapis.com/cloudprober/http/google-homepage/latency
	MonitoringUrl *string `protobuf:"bytes,4,opt,name=monitoring_url,json=monitoringUrl,def=custom.googleapis.com/cloudprober/" json:"monitoring_url,omitempty"`
	// How many metrics entries to buffer. Incoming metrics
	// processing is paused while serving data to Stackdriver. This buffer is to
	// make writes to Stackdriver surfacer non-blocking.
	MetricsBufferSize *int64                     `protobuf:"varint,5,opt,name=metrics_buffer_size,json=metricsBufferSize,def=10000" json:"metrics_buffer_size,omitempty"`
	MetricsPrefix     *SurfacerConf_MetricPrefix `protobuf:"varint,6,opt,name=metrics_prefix,json=metricsPrefix,enum=cloudprober.surfacer.stackdriver.SurfacerConf_MetricPrefix,def=2" json:"metrics_prefix,omitempty"` // using current behavior as default
}

// Default values for SurfacerConf fields.
const (
	Default_SurfacerConf_BatchTimerSec     = uint64(10)
	Default_SurfacerConf_MonitoringUrl     = string("custom.googleapis.com/cloudprober/")
	Default_SurfacerConf_MetricsBufferSize = int64(10000)
	Default_SurfacerConf_MetricsPrefix     = SurfacerConf_PTYPE_PROBE
)

func (x *SurfacerConf) Reset() {
	*x = SurfacerConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SurfacerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerConf) ProtoMessage() {}

func (x *SurfacerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_msgTypes[0]
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
	return file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *SurfacerConf) GetProject() string {
	if x != nil && x.Project != nil {
		return *x.Project
	}
	return ""
}

func (x *SurfacerConf) GetBatchTimerSec() uint64 {
	if x != nil && x.BatchTimerSec != nil {
		return *x.BatchTimerSec
	}
	return Default_SurfacerConf_BatchTimerSec
}

func (x *SurfacerConf) GetAllowedMetricsRegex() string {
	if x != nil && x.AllowedMetricsRegex != nil {
		return *x.AllowedMetricsRegex
	}
	return ""
}

func (x *SurfacerConf) GetMonitoringUrl() string {
	if x != nil && x.MonitoringUrl != nil {
		return *x.MonitoringUrl
	}
	return Default_SurfacerConf_MonitoringUrl
}

func (x *SurfacerConf) GetMetricsBufferSize() int64 {
	if x != nil && x.MetricsBufferSize != nil {
		return *x.MetricsBufferSize
	}
	return Default_SurfacerConf_MetricsBufferSize
}

func (x *SurfacerConf) GetMetricsPrefix() SurfacerConf_MetricPrefix {
	if x != nil && x.MetricsPrefix != nil {
		return *x.MetricsPrefix
	}
	return Default_SurfacerConf_MetricsPrefix
}

var File_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc = []byte{
	0x0a, 0x54, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69,
	0x76, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x20, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x73, 0x74, 0x61,
	0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72, 0x22, 0xb1, 0x03, 0x0a, 0x0c, 0x53, 0x75, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x72, 0x6f,
	0x6a, 0x65, 0x63, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70, 0x72, 0x6f, 0x6a,
	0x65, 0x63, 0x74, 0x12, 0x2a, 0x0a, 0x0f, 0x62, 0x61, 0x74, 0x63, 0x68, 0x5f, 0x74, 0x69, 0x6d,
	0x65, 0x72, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x3a, 0x02, 0x31, 0x30,
	0x52, 0x0d, 0x62, 0x61, 0x74, 0x63, 0x68, 0x54, 0x69, 0x6d, 0x65, 0x72, 0x53, 0x65, 0x63, 0x12,
	0x32, 0x0a, 0x15, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x65, 0x64, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69,
	0x63, 0x73, 0x5f, 0x72, 0x65, 0x67, 0x65, 0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13,
	0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x65, 0x64, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x52, 0x65,
	0x67, 0x65, 0x78, 0x12, 0x49, 0x0a, 0x0e, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e,
	0x67, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x22, 0x63, 0x75, 0x73,
	0x74, 0x6f, 0x6d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x61, 0x70, 0x69, 0x73, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x52,
	0x0d, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e, 0x67, 0x55, 0x72, 0x6c, 0x12, 0x35,
	0x0a, 0x13, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x62, 0x75, 0x66, 0x66, 0x65, 0x72,
	0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x3a, 0x05, 0x31, 0x30, 0x30,
	0x30, 0x30, 0x52, 0x11, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x42, 0x75, 0x66, 0x66, 0x65,
	0x72, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x6f, 0x0a, 0x0e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73,
	0x5f, 0x70, 0x72, 0x65, 0x66, 0x69, 0x78, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x3b, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72,
	0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x4d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x3a, 0x0b, 0x50, 0x54, 0x59, 0x50,
	0x45, 0x5f, 0x50, 0x52, 0x4f, 0x42, 0x45, 0x52, 0x0d, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73,
	0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x22, 0x34, 0x0a, 0x0c, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x50, 0x72, 0x65, 0x66, 0x69, 0x78, 0x12, 0x08, 0x0a, 0x04, 0x4e, 0x4f, 0x4e, 0x45, 0x10, 0x00,
	0x12, 0x09, 0x0a, 0x05, 0x50, 0x52, 0x4f, 0x42, 0x45, 0x10, 0x01, 0x12, 0x0f, 0x0a, 0x0b, 0x50,
	0x54, 0x59, 0x50, 0x45, 0x5f, 0x50, 0x52, 0x4f, 0x42, 0x45, 0x10, 0x02, 0x42, 0x49, 0x5a, 0x47,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65,
	0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_goTypes = []interface{}{
	(SurfacerConf_MetricPrefix)(0), // 0: cloudprober.surfacer.stackdriver.SurfacerConf.MetricPrefix
	(*SurfacerConf)(nil),           // 1: cloudprober.surfacer.stackdriver.SurfacerConf
}
var file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.surfacer.stackdriver.SurfacerConf.metrics_prefix:type_name -> cloudprober.surfacer.stackdriver.SurfacerConf.MetricPrefix
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_init()
}
func file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
			RawDescriptor: file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_depIdxs,
		EnumInfos:         file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_enumTypes,
		MessageInfos:      file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_depIdxs = nil
}
