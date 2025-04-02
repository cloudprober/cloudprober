// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/surfacers/internal/stackdriver/proto/config.proto

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
	state protoimpl.MessageState `protogen:"open.v1"`
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
	MetricsBufferSize *int64 `protobuf:"varint,5,opt,name=metrics_buffer_size,json=metricsBufferSize,def=10000" json:"metrics_buffer_size,omitempty"`
	// Metric prefix to use for stackdriver metrics. If not specified, default
	// is PTYPE_PROBE.
	MetricsPrefix *SurfacerConf_MetricPrefix `protobuf:"varint,6,opt,name=metrics_prefix,json=metricsPrefix,enum=cloudprober.surfacer.stackdriver.SurfacerConf_MetricPrefix,def=2" json:"metrics_prefix,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
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
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SurfacerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerConf) ProtoMessage() {}

func (x *SurfacerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_msgTypes[0]
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

const file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc = "" +
	"\n" +
	"Tgithub.com/cloudprober/cloudprober/surfacers/internal/stackdriver/proto/config.proto\x12 cloudprober.surfacer.stackdriver\"\xb1\x03\n" +
	"\fSurfacerConf\x12\x18\n" +
	"\aproject\x18\x01 \x01(\tR\aproject\x12*\n" +
	"\x0fbatch_timer_sec\x18\x02 \x01(\x04:\x0210R\rbatchTimerSec\x122\n" +
	"\x15allowed_metrics_regex\x18\x03 \x01(\tR\x13allowedMetricsRegex\x12I\n" +
	"\x0emonitoring_url\x18\x04 \x01(\t:\"custom.googleapis.com/cloudprober/R\rmonitoringUrl\x125\n" +
	"\x13metrics_buffer_size\x18\x05 \x01(\x03:\x0510000R\x11metricsBufferSize\x12o\n" +
	"\x0emetrics_prefix\x18\x06 \x01(\x0e2;.cloudprober.surfacer.stackdriver.SurfacerConf.MetricPrefix:\vPTYPE_PROBER\rmetricsPrefix\"4\n" +
	"\fMetricPrefix\x12\b\n" +
	"\x04NONE\x10\x00\x12\t\n" +
	"\x05PROBE\x10\x01\x12\x0f\n" +
	"\vPTYPE_PROBE\x10\x02BIZGgithub.com/cloudprober/cloudprober/surfacers/internal/stackdriver/proto"

var (
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_goTypes = []any{
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_rawDesc)),
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
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_internal_stackdriver_proto_config_proto_depIdxs = nil
}
