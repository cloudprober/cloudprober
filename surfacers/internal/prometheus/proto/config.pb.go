// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/surfacers/internal/prometheus/proto/config.proto

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
	// How many metrics entries (EventMetrics) to buffer. Incoming metrics
	// processing is paused while serving data to prometheus. This buffer is to
	// make writes to prometheus surfacer non-blocking.
	// NOTE: This field is confusing for users and will be removed from the config
	// after v0.10.3.
	MetricsBufferSize *int64 `protobuf:"varint,1,opt,name=metrics_buffer_size,json=metricsBufferSize,def=10000" json:"metrics_buffer_size,omitempty"`
	// Whether to include timestamps in metrics. If enabled (default) each metric
	// string includes the metric timestamp as recorded in the EventMetric.
	// Prometheus associates the scraped values with this timestamp. If disabled,
	// i.e. timestamps are not exported, prometheus associates scraped values with
	// scrape timestamp.
	//
	// As it's typically useful to set this across the deployment, this field can
	// also be set through the command line flag --prometheus_include_timestamp.
	// If both are set, the config value takes precedence.
	IncludeTimestamp *bool `protobuf:"varint,2,opt,name=include_timestamp,json=includeTimestamp,def=1" json:"include_timestamp,omitempty"`
	// We automatically delete metrics that haven't been updated for more than
	// 10 min. We do that because prometheus generates warnings while scraping
	// metrics that are more than 10m old and prometheus knows that metrics are
	// more than 10m old because we include timestamp in metrics (by default).
	// This option controls that behavior.
	// It's automatically set to true if include_timestamp is false.
	DisableMetricsExpiration *bool `protobuf:"varint,5,opt,name=disable_metrics_expiration,json=disableMetricsExpiration" json:"disable_metrics_expiration,omitempty"`
	// URL that prometheus scrapes metrics from.
	MetricsUrl *string `protobuf:"bytes,3,opt,name=metrics_url,json=metricsUrl,def=/metrics" json:"metrics_url,omitempty"`
	// Prefix to add to all metric names. For example setting this field to
	// "cloudprober_" will result in metrics with names:
	// cloudprober_total, cloudprober_success, cloudprober_latency, ..
	//
	// As it's typically useful to set this across the deployment, this field can
	// also be set through the command line flag --prometheus_metrics_prefix. If
	// both are set, the config value takes precedence.
	MetricsPrefix *string `protobuf:"bytes,4,opt,name=metrics_prefix,json=metricsPrefix" json:"metrics_prefix,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for SurfacerConf fields.
const (
	Default_SurfacerConf_MetricsBufferSize = int64(10000)
	Default_SurfacerConf_IncludeTimestamp  = bool(true)
	Default_SurfacerConf_MetricsUrl        = string("/metrics")
)

func (x *SurfacerConf) Reset() {
	*x = SurfacerConf{}
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SurfacerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerConf) ProtoMessage() {}

func (x *SurfacerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_msgTypes[0]
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
	return file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *SurfacerConf) GetMetricsBufferSize() int64 {
	if x != nil && x.MetricsBufferSize != nil {
		return *x.MetricsBufferSize
	}
	return Default_SurfacerConf_MetricsBufferSize
}

func (x *SurfacerConf) GetIncludeTimestamp() bool {
	if x != nil && x.IncludeTimestamp != nil {
		return *x.IncludeTimestamp
	}
	return Default_SurfacerConf_IncludeTimestamp
}

func (x *SurfacerConf) GetDisableMetricsExpiration() bool {
	if x != nil && x.DisableMetricsExpiration != nil {
		return *x.DisableMetricsExpiration
	}
	return false
}

func (x *SurfacerConf) GetMetricsUrl() string {
	if x != nil && x.MetricsUrl != nil {
		return *x.MetricsUrl
	}
	return Default_SurfacerConf_MetricsUrl
}

func (x *SurfacerConf) GetMetricsPrefix() string {
	if x != nil && x.MetricsPrefix != nil {
		return *x.MetricsPrefix
	}
	return ""
}

var File_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto protoreflect.FileDescriptor

const file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDesc = "" +
	"\n" +
	"Sgithub.com/cloudprober/cloudprober/surfacers/internal/prometheus/proto/config.proto\x12\x1fcloudprober.surfacer.prometheus\"\x88\x02\n" +
	"\fSurfacerConf\x125\n" +
	"\x13metrics_buffer_size\x18\x01 \x01(\x03:\x0510000R\x11metricsBufferSize\x121\n" +
	"\x11include_timestamp\x18\x02 \x01(\b:\x04trueR\x10includeTimestamp\x12<\n" +
	"\x1adisable_metrics_expiration\x18\x05 \x01(\bR\x18disableMetricsExpiration\x12)\n" +
	"\vmetrics_url\x18\x03 \x01(\t:\b/metricsR\n" +
	"metricsUrl\x12%\n" +
	"\x0emetrics_prefix\x18\x04 \x01(\tR\rmetricsPrefixBHZFgithub.com/cloudprober/cloudprober/surfacers/internal/prometheus/proto"

var (
	file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_goTypes = []any{
	(*SurfacerConf)(nil), // 0: cloudprober.surfacer.prometheus.SurfacerConf
}
var file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_init()
}
func file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_internal_prometheus_proto_config_proto_depIdxs = nil
}
