// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.1
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/surfacers/proto/config.proto

package proto

import (
	proto8 "github.com/cloudprober/cloudprober/surfacers/internal/bigquery/proto"
	proto5 "github.com/cloudprober/cloudprober/surfacers/internal/cloudwatch/proto"
	proto6 "github.com/cloudprober/cloudprober/surfacers/internal/datadog/proto"
	proto2 "github.com/cloudprober/cloudprober/surfacers/internal/file/proto"
	proto9 "github.com/cloudprober/cloudprober/surfacers/internal/otel/proto"
	proto3 "github.com/cloudprober/cloudprober/surfacers/internal/postgres/proto"
	proto7 "github.com/cloudprober/cloudprober/surfacers/internal/probestatus/proto"
	proto "github.com/cloudprober/cloudprober/surfacers/internal/prometheus/proto"
	proto4 "github.com/cloudprober/cloudprober/surfacers/internal/pubsub/proto"
	proto1 "github.com/cloudprober/cloudprober/surfacers/internal/stackdriver/proto"
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

// Enumeration for each type of surfacer we can parse and create
type Type int32

const (
	Type_NONE         Type = 0
	Type_PROMETHEUS   Type = 1
	Type_STACKDRIVER  Type = 2
	Type_FILE         Type = 3
	Type_POSTGRES     Type = 4
	Type_PUBSUB       Type = 5
	Type_CLOUDWATCH   Type = 6 // Experimental mode.
	Type_DATADOG      Type = 7 // Experimental mode.
	Type_PROBESTATUS  Type = 8
	Type_BIGQUERY     Type = 9 // Experimental mode.
	Type_OTEL         Type = 10
	Type_USER_DEFINED Type = 99
)

// Enum value maps for Type.
var (
	Type_name = map[int32]string{
		0:  "NONE",
		1:  "PROMETHEUS",
		2:  "STACKDRIVER",
		3:  "FILE",
		4:  "POSTGRES",
		5:  "PUBSUB",
		6:  "CLOUDWATCH",
		7:  "DATADOG",
		8:  "PROBESTATUS",
		9:  "BIGQUERY",
		10: "OTEL",
		99: "USER_DEFINED",
	}
	Type_value = map[string]int32{
		"NONE":         0,
		"PROMETHEUS":   1,
		"STACKDRIVER":  2,
		"FILE":         3,
		"POSTGRES":     4,
		"PUBSUB":       5,
		"CLOUDWATCH":   6,
		"DATADOG":      7,
		"PROBESTATUS":  8,
		"BIGQUERY":     9,
		"OTEL":         10,
		"USER_DEFINED": 99,
	}
)

func (x Type) Enum() *Type {
	p := new(Type)
	*p = x
	return p
}

func (x Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Type) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_enumTypes[0].Descriptor()
}

func (Type) Type() protoreflect.EnumType {
	return &file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_enumTypes[0]
}

func (x Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *Type) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = Type(num)
	return nil
}

// Deprecated: Use Type.Descriptor instead.
func (Type) EnumDescriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescGZIP(), []int{0}
}

type LabelFilter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   *string `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value *string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
}

func (x *LabelFilter) Reset() {
	*x = LabelFilter{}
	mi := &file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LabelFilter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LabelFilter) ProtoMessage() {}

func (x *LabelFilter) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LabelFilter.ProtoReflect.Descriptor instead.
func (*LabelFilter) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *LabelFilter) GetKey() string {
	if x != nil && x.Key != nil {
		return *x.Key
	}
	return ""
}

func (x *LabelFilter) GetValue() string {
	if x != nil && x.Value != nil {
		return *x.Value
	}
	return ""
}

type SurfacerDef struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// This name is used for logging. If not defined, it's derived from the type.
	// Note that this field is required for the USER_DEFINED surfacer type and
	// should match with the name that you used while registering the user defined
	// surfacer.
	Name *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Type *Type   `protobuf:"varint,2,opt,name=type,enum=cloudprober.surfacer.Type" json:"type,omitempty"`
	// How many metrics entries (EventMetrics) to buffer. This is the buffer
	// between incoming metrics and the metrics that are being processed. Default
	// value should work in most cases. You may need to increase it on a busy
	// system, but that's usually a sign that you metrics processing pipeline is
	// slow for some reason, e.g. slow writes to a remote file.
	// Note: Only file and pubsub surfacer supports this option right now.
	MetricsBufferSize *int64 `protobuf:"varint,3,opt,name=metrics_buffer_size,json=metricsBufferSize,def=10000" json:"metrics_buffer_size,omitempty"`
	// If specified, only allow metrics that match any of these label filters.
	// Example:
	//
	//	allow_metrics_with_label {
	//	  key: "probe",
	//	  value: "check_homepage",
	//	}
	AllowMetricsWithLabel []*LabelFilter `protobuf:"bytes,4,rep,name=allow_metrics_with_label,json=allowMetricsWithLabel" json:"allow_metrics_with_label,omitempty"`
	// Ignore metrics that match any of these label filters. Ignore has precedence
	// over allow filters.
	// Example:
	//
	//	ignore_metrics_with_label {
	//	  key: "probe",
	//	  value: "sysvars",
	//	}
	IgnoreMetricsWithLabel []*LabelFilter `protobuf:"bytes,5,rep,name=ignore_metrics_with_label,json=ignoreMetricsWithLabel" json:"ignore_metrics_with_label,omitempty"`
	// Allow and ignore metrics based on their names. You can specify regexes
	// here. Ignore has precendence over allow.
	// Examples:
	//
	//	ignore_metrics_with_name: "validation_failure"
	//	allow_metrics_with_name: "(total|success|latency)"
	AllowMetricsWithName  *string `protobuf:"bytes,6,opt,name=allow_metrics_with_name,json=allowMetricsWithName" json:"allow_metrics_with_name,omitempty"`
	IgnoreMetricsWithName *string `protobuf:"bytes,7,opt,name=ignore_metrics_with_name,json=ignoreMetricsWithName" json:"ignore_metrics_with_name,omitempty"`
	// Whether to add failure metric or not. This option is enabled by default.
	AddFailureMetric *bool `protobuf:"varint,8,opt,name=add_failure_metric,json=addFailureMetric,def=1" json:"add_failure_metric,omitempty"`
	// If set to true, cloudprober will export all metrics as gauge metrics. Note
	// that cloudprober inherently generates only cumulative metrics. To create
	// gauge metrics from cumulative metrics, we keep a copy of the old metrics
	// and subtract new metrics from the previous metrics. This transformation in
	// metrics has an increased memory-overhead because extra copies required.
	// However, it should not be noticeable unless you're producing large number
	// of metrics (say > 10000 metrics per second).
	ExportAsGauge *bool `protobuf:"varint,9,opt,name=export_as_gauge,json=exportAsGauge" json:"export_as_gauge,omitempty"`
	// Latency metric name pattern, used to identify latency metrics, and add
	// EventMetric's LatencyUnit to it.
	LatencyMetricPattern *string `protobuf:"bytes,51,opt,name=latency_metric_pattern,json=latencyMetricPattern,def=^(.+_|)latency$" json:"latency_metric_pattern,omitempty"`
	// Environment variable containing additional labels to be added to all
	// metrics exported by this surfacer.
	// e.g. "CLOUDPROBER_ADDITIONAL_LABELS=env=prod,app=identity-service"
	// You can disable this feature by setting this field to an empty string.
	// Note: These additional labels have no effect if metrics already have the
	// same label.
	AdditionalLabelsEnvVar *string `protobuf:"bytes,52,opt,name=additional_labels_env_var,json=additionalLabelsEnvVar,def=CLOUDPROBER_ADDITIONAL_LABELS" json:"additional_labels_env_var,omitempty"`
	// Matching surfacer specific configuration (one for each type in the above
	// enum)
	//
	// Types that are assignable to Surfacer:
	//
	//	*SurfacerDef_PrometheusSurfacer
	//	*SurfacerDef_StackdriverSurfacer
	//	*SurfacerDef_FileSurfacer
	//	*SurfacerDef_PostgresSurfacer
	//	*SurfacerDef_PubsubSurfacer
	//	*SurfacerDef_CloudwatchSurfacer
	//	*SurfacerDef_DatadogSurfacer
	//	*SurfacerDef_ProbestatusSurfacer
	//	*SurfacerDef_BigquerySurfacer
	//	*SurfacerDef_OtelSurfacer
	Surfacer isSurfacerDef_Surfacer `protobuf_oneof:"surfacer"`
}

// Default values for SurfacerDef fields.
const (
	Default_SurfacerDef_MetricsBufferSize      = int64(10000)
	Default_SurfacerDef_AddFailureMetric       = bool(true)
	Default_SurfacerDef_LatencyMetricPattern   = string("^(.+_|)latency$")
	Default_SurfacerDef_AdditionalLabelsEnvVar = string("CLOUDPROBER_ADDITIONAL_LABELS")
)

func (x *SurfacerDef) Reset() {
	*x = SurfacerDef{}
	mi := &file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SurfacerDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerDef) ProtoMessage() {}

func (x *SurfacerDef) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SurfacerDef.ProtoReflect.Descriptor instead.
func (*SurfacerDef) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *SurfacerDef) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *SurfacerDef) GetType() Type {
	if x != nil && x.Type != nil {
		return *x.Type
	}
	return Type_NONE
}

func (x *SurfacerDef) GetMetricsBufferSize() int64 {
	if x != nil && x.MetricsBufferSize != nil {
		return *x.MetricsBufferSize
	}
	return Default_SurfacerDef_MetricsBufferSize
}

func (x *SurfacerDef) GetAllowMetricsWithLabel() []*LabelFilter {
	if x != nil {
		return x.AllowMetricsWithLabel
	}
	return nil
}

func (x *SurfacerDef) GetIgnoreMetricsWithLabel() []*LabelFilter {
	if x != nil {
		return x.IgnoreMetricsWithLabel
	}
	return nil
}

func (x *SurfacerDef) GetAllowMetricsWithName() string {
	if x != nil && x.AllowMetricsWithName != nil {
		return *x.AllowMetricsWithName
	}
	return ""
}

func (x *SurfacerDef) GetIgnoreMetricsWithName() string {
	if x != nil && x.IgnoreMetricsWithName != nil {
		return *x.IgnoreMetricsWithName
	}
	return ""
}

func (x *SurfacerDef) GetAddFailureMetric() bool {
	if x != nil && x.AddFailureMetric != nil {
		return *x.AddFailureMetric
	}
	return Default_SurfacerDef_AddFailureMetric
}

func (x *SurfacerDef) GetExportAsGauge() bool {
	if x != nil && x.ExportAsGauge != nil {
		return *x.ExportAsGauge
	}
	return false
}

func (x *SurfacerDef) GetLatencyMetricPattern() string {
	if x != nil && x.LatencyMetricPattern != nil {
		return *x.LatencyMetricPattern
	}
	return Default_SurfacerDef_LatencyMetricPattern
}

func (x *SurfacerDef) GetAdditionalLabelsEnvVar() string {
	if x != nil && x.AdditionalLabelsEnvVar != nil {
		return *x.AdditionalLabelsEnvVar
	}
	return Default_SurfacerDef_AdditionalLabelsEnvVar
}

func (m *SurfacerDef) GetSurfacer() isSurfacerDef_Surfacer {
	if m != nil {
		return m.Surfacer
	}
	return nil
}

func (x *SurfacerDef) GetPrometheusSurfacer() *proto.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_PrometheusSurfacer); ok {
		return x.PrometheusSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetStackdriverSurfacer() *proto1.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_StackdriverSurfacer); ok {
		return x.StackdriverSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetFileSurfacer() *proto2.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_FileSurfacer); ok {
		return x.FileSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetPostgresSurfacer() *proto3.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_PostgresSurfacer); ok {
		return x.PostgresSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetPubsubSurfacer() *proto4.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_PubsubSurfacer); ok {
		return x.PubsubSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetCloudwatchSurfacer() *proto5.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_CloudwatchSurfacer); ok {
		return x.CloudwatchSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetDatadogSurfacer() *proto6.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_DatadogSurfacer); ok {
		return x.DatadogSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetProbestatusSurfacer() *proto7.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_ProbestatusSurfacer); ok {
		return x.ProbestatusSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetBigquerySurfacer() *proto8.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_BigquerySurfacer); ok {
		return x.BigquerySurfacer
	}
	return nil
}

func (x *SurfacerDef) GetOtelSurfacer() *proto9.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_OtelSurfacer); ok {
		return x.OtelSurfacer
	}
	return nil
}

type isSurfacerDef_Surfacer interface {
	isSurfacerDef_Surfacer()
}

type SurfacerDef_PrometheusSurfacer struct {
	PrometheusSurfacer *proto.SurfacerConf `protobuf:"bytes,10,opt,name=prometheus_surfacer,json=prometheusSurfacer,oneof"`
}

type SurfacerDef_StackdriverSurfacer struct {
	StackdriverSurfacer *proto1.SurfacerConf `protobuf:"bytes,11,opt,name=stackdriver_surfacer,json=stackdriverSurfacer,oneof"`
}

type SurfacerDef_FileSurfacer struct {
	FileSurfacer *proto2.SurfacerConf `protobuf:"bytes,12,opt,name=file_surfacer,json=fileSurfacer,oneof"`
}

type SurfacerDef_PostgresSurfacer struct {
	PostgresSurfacer *proto3.SurfacerConf `protobuf:"bytes,13,opt,name=postgres_surfacer,json=postgresSurfacer,oneof"`
}

type SurfacerDef_PubsubSurfacer struct {
	PubsubSurfacer *proto4.SurfacerConf `protobuf:"bytes,14,opt,name=pubsub_surfacer,json=pubsubSurfacer,oneof"`
}

type SurfacerDef_CloudwatchSurfacer struct {
	CloudwatchSurfacer *proto5.SurfacerConf `protobuf:"bytes,15,opt,name=cloudwatch_surfacer,json=cloudwatchSurfacer,oneof"`
}

type SurfacerDef_DatadogSurfacer struct {
	DatadogSurfacer *proto6.SurfacerConf `protobuf:"bytes,16,opt,name=datadog_surfacer,json=datadogSurfacer,oneof"`
}

type SurfacerDef_ProbestatusSurfacer struct {
	ProbestatusSurfacer *proto7.SurfacerConf `protobuf:"bytes,17,opt,name=probestatus_surfacer,json=probestatusSurfacer,oneof"`
}

type SurfacerDef_BigquerySurfacer struct {
	BigquerySurfacer *proto8.SurfacerConf `protobuf:"bytes,18,opt,name=bigquery_surfacer,json=bigquerySurfacer,oneof"`
}

type SurfacerDef_OtelSurfacer struct {
	OtelSurfacer *proto9.SurfacerConf `protobuf:"bytes,19,opt,name=otel_surfacer,json=otelSurfacer,oneof"`
}

func (*SurfacerDef_PrometheusSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_StackdriverSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_FileSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_PostgresSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_PubsubSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_CloudwatchSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_DatadogSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_ProbestatusSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_BigquerySurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_OtelSurfacer) isSurfacerDef_Surfacer() {}

var File_github_com_cloudprober_cloudprober_surfacers_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc = []byte{
	0x0a, 0x3f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x14, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73,
	0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x1a, 0x53, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x77, 0x61, 0x74, 0x63, 0x68, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x50, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2f, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x4d,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x4d, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6f, 0x74, 0x65, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x51, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x54, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x53, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f,
	0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x4f, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73,
	0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x2f, 0x70, 0x75, 0x62, 0x73, 0x75, 0x62, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x54, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e,
	0x61, 0x6c, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x51, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72,
	0x79, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x35, 0x0a, 0x0b, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x46, 0x69, 0x6c,
	0x74, 0x65, 0x72, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0xd0, 0x0c, 0x0a, 0x0b,
	0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x44, 0x65, 0x66, 0x12, 0x12, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12,
	0x2e, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1a, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12,
	0x35, 0x0a, 0x13, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x62, 0x75, 0x66, 0x66, 0x65,
	0x72, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x3a, 0x05, 0x31, 0x30,
	0x30, 0x30, 0x30, 0x52, 0x11, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x42, 0x75, 0x66, 0x66,
	0x65, 0x72, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x5a, 0x0a, 0x18, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x5f,
	0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x77, 0x69, 0x74, 0x68, 0x5f, 0x6c, 0x61, 0x62,
	0x65, 0x6c, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e,
	0x4c, 0x61, 0x62, 0x65, 0x6c, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x15, 0x61, 0x6c, 0x6c,
	0x6f, 0x77, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x57, 0x69, 0x74, 0x68, 0x4c, 0x61, 0x62,
	0x65, 0x6c, 0x12, 0x5c, 0x0a, 0x19, 0x69, 0x67, 0x6e, 0x6f, 0x72, 0x65, 0x5f, 0x6d, 0x65, 0x74,
	0x72, 0x69, 0x63, 0x73, 0x5f, 0x77, 0x69, 0x74, 0x68, 0x5f, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x18,
	0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x4c, 0x61, 0x62,
	0x65, 0x6c, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x16, 0x69, 0x67, 0x6e, 0x6f, 0x72, 0x65,
	0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x57, 0x69, 0x74, 0x68, 0x4c, 0x61, 0x62, 0x65, 0x6c,
	0x12, 0x35, 0x0a, 0x17, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x5f, 0x77, 0x69, 0x74, 0x68, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x14, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x57,
	0x69, 0x74, 0x68, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x37, 0x0a, 0x18, 0x69, 0x67, 0x6e, 0x6f, 0x72,
	0x65, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x77, 0x69, 0x74, 0x68, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x15, 0x69, 0x67, 0x6e, 0x6f, 0x72,
	0x65, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x57, 0x69, 0x74, 0x68, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x32, 0x0a, 0x12, 0x61, 0x64, 0x64, 0x5f, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x5f,
	0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x3a, 0x04, 0x74, 0x72,
	0x75, 0x65, 0x52, 0x10, 0x61, 0x64, 0x64, 0x46, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x4d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x12, 0x26, 0x0a, 0x0f, 0x65, 0x78, 0x70, 0x6f, 0x72, 0x74, 0x5f, 0x61,
	0x73, 0x5f, 0x67, 0x61, 0x75, 0x67, 0x65, 0x18, 0x09, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0d, 0x65,
	0x78, 0x70, 0x6f, 0x72, 0x74, 0x41, 0x73, 0x47, 0x61, 0x75, 0x67, 0x65, 0x12, 0x45, 0x0a, 0x16,
	0x6c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x5f, 0x70,
	0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x18, 0x33, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x0f, 0x5e, 0x28,
	0x2e, 0x2b, 0x5f, 0x7c, 0x29, 0x6c, 0x61, 0x74, 0x65, 0x6e, 0x63, 0x79, 0x24, 0x52, 0x14, 0x6c,
	0x61, 0x74, 0x65, 0x6e, 0x63, 0x79, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x50, 0x61, 0x74, 0x74,
	0x65, 0x72, 0x6e, 0x12, 0x58, 0x0a, 0x19, 0x61, 0x64, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61,
	0x6c, 0x5f, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x5f, 0x65, 0x6e, 0x76, 0x5f, 0x76, 0x61, 0x72,
	0x18, 0x34, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x1d, 0x43, 0x4c, 0x4f, 0x55, 0x44, 0x50, 0x52, 0x4f,
	0x42, 0x45, 0x52, 0x5f, 0x41, 0x44, 0x44, 0x49, 0x54, 0x49, 0x4f, 0x4e, 0x41, 0x4c, 0x5f, 0x4c,
	0x41, 0x42, 0x45, 0x4c, 0x53, 0x52, 0x16, 0x61, 0x64, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61,
	0x6c, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x76, 0x56, 0x61, 0x72, 0x12, 0x60, 0x0a,
	0x13, 0x70, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x5f, 0x73, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65,
	0x72, 0x2e, 0x70, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x2e, 0x53, 0x75, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x12, 0x70, 0x72, 0x6f,
	0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x12,
	0x63, 0x0a, 0x14, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72, 0x5f, 0x73,
	0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72,
	0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52,
	0x13, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72, 0x53, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x12, 0x4e, 0x0a, 0x0d, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x73, 0x75, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72,
	0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0c, 0x66, 0x69, 0x6c, 0x65, 0x53, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x12, 0x5a, 0x0a, 0x11, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73,
	0x5f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x2b, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75,
	0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x2e,
	0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x10,
	0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72,
	0x12, 0x54, 0x0a, 0x0f, 0x70, 0x75, 0x62, 0x73, 0x75, 0x62, 0x5f, 0x73, 0x75, 0x72, 0x66, 0x61,
	0x63, 0x65, 0x72, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72,
	0x2e, 0x70, 0x75, 0x62, 0x73, 0x75, 0x62, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72,
	0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0e, 0x70, 0x75, 0x62, 0x73, 0x75, 0x62, 0x53, 0x75,
	0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x12, 0x60, 0x0a, 0x13, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x77,
	0x61, 0x74, 0x63, 0x68, 0x5f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x0f, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x77, 0x61, 0x74, 0x63, 0x68, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f,
	0x6e, 0x66, 0x48, 0x00, 0x52, 0x12, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x77, 0x61, 0x74, 0x63, 0x68,
	0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x12, 0x57, 0x0a, 0x10, 0x64, 0x61, 0x74, 0x61,
	0x64, 0x6f, 0x67, 0x5f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x10, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f,
	0x67, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00,
	0x52, 0x0f, 0x64, 0x61, 0x74, 0x61, 0x64, 0x6f, 0x67, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65,
	0x72, 0x12, 0x63, 0x0a, 0x14, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x5f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x11, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x2e, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75,
	0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48,
	0x00, 0x52, 0x13, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x53, 0x75,
	0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x12, 0x5a, 0x0a, 0x11, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x5f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x12, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x2b, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e,
	0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72,
	0x79, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00,
	0x52, 0x10, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72, 0x79, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x12, 0x4e, 0x0a, 0x0d, 0x6f, 0x74, 0x65, 0x6c, 0x5f, 0x73, 0x75, 0x72, 0x66, 0x61,
	0x63, 0x65, 0x72, 0x18, 0x13, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72,
	0x2e, 0x6f, 0x74, 0x65, 0x6c, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f,
	0x6e, 0x66, 0x48, 0x00, 0x52, 0x0c, 0x6f, 0x74, 0x65, 0x6c, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x42, 0x0a, 0x0a, 0x08, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2a, 0xad,
	0x01, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x4e, 0x4f, 0x4e, 0x45, 0x10,
	0x00, 0x12, 0x0e, 0x0a, 0x0a, 0x50, 0x52, 0x4f, 0x4d, 0x45, 0x54, 0x48, 0x45, 0x55, 0x53, 0x10,
	0x01, 0x12, 0x0f, 0x0a, 0x0b, 0x53, 0x54, 0x41, 0x43, 0x4b, 0x44, 0x52, 0x49, 0x56, 0x45, 0x52,
	0x10, 0x02, 0x12, 0x08, 0x0a, 0x04, 0x46, 0x49, 0x4c, 0x45, 0x10, 0x03, 0x12, 0x0c, 0x0a, 0x08,
	0x50, 0x4f, 0x53, 0x54, 0x47, 0x52, 0x45, 0x53, 0x10, 0x04, 0x12, 0x0a, 0x0a, 0x06, 0x50, 0x55,
	0x42, 0x53, 0x55, 0x42, 0x10, 0x05, 0x12, 0x0e, 0x0a, 0x0a, 0x43, 0x4c, 0x4f, 0x55, 0x44, 0x57,
	0x41, 0x54, 0x43, 0x48, 0x10, 0x06, 0x12, 0x0b, 0x0a, 0x07, 0x44, 0x41, 0x54, 0x41, 0x44, 0x4f,
	0x47, 0x10, 0x07, 0x12, 0x0f, 0x0a, 0x0b, 0x50, 0x52, 0x4f, 0x42, 0x45, 0x53, 0x54, 0x41, 0x54,
	0x55, 0x53, 0x10, 0x08, 0x12, 0x0c, 0x0a, 0x08, 0x42, 0x49, 0x47, 0x51, 0x55, 0x45, 0x52, 0x59,
	0x10, 0x09, 0x12, 0x08, 0x0a, 0x04, 0x4f, 0x54, 0x45, 0x4c, 0x10, 0x0a, 0x12, 0x10, 0x0a, 0x0c,
	0x55, 0x53, 0x45, 0x52, 0x5f, 0x44, 0x45, 0x46, 0x49, 0x4e, 0x45, 0x44, 0x10, 0x63, 0x42, 0x34,
	0x5a, 0x32, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_goTypes = []any{
	(Type)(0),                   // 0: cloudprober.surfacer.Type
	(*LabelFilter)(nil),         // 1: cloudprober.surfacer.LabelFilter
	(*SurfacerDef)(nil),         // 2: cloudprober.surfacer.SurfacerDef
	(*proto.SurfacerConf)(nil),  // 3: cloudprober.surfacer.prometheus.SurfacerConf
	(*proto1.SurfacerConf)(nil), // 4: cloudprober.surfacer.stackdriver.SurfacerConf
	(*proto2.SurfacerConf)(nil), // 5: cloudprober.surfacer.file.SurfacerConf
	(*proto3.SurfacerConf)(nil), // 6: cloudprober.surfacer.postgres.SurfacerConf
	(*proto4.SurfacerConf)(nil), // 7: cloudprober.surfacer.pubsub.SurfacerConf
	(*proto5.SurfacerConf)(nil), // 8: cloudprober.surfacer.cloudwatch.SurfacerConf
	(*proto6.SurfacerConf)(nil), // 9: cloudprober.surfacer.datadog.SurfacerConf
	(*proto7.SurfacerConf)(nil), // 10: cloudprober.surfacer.probestatus.SurfacerConf
	(*proto8.SurfacerConf)(nil), // 11: cloudprober.surfacer.bigquery.SurfacerConf
	(*proto9.SurfacerConf)(nil), // 12: cloudprober.surfacer.otel.SurfacerConf
}
var file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_depIdxs = []int32{
	0,  // 0: cloudprober.surfacer.SurfacerDef.type:type_name -> cloudprober.surfacer.Type
	1,  // 1: cloudprober.surfacer.SurfacerDef.allow_metrics_with_label:type_name -> cloudprober.surfacer.LabelFilter
	1,  // 2: cloudprober.surfacer.SurfacerDef.ignore_metrics_with_label:type_name -> cloudprober.surfacer.LabelFilter
	3,  // 3: cloudprober.surfacer.SurfacerDef.prometheus_surfacer:type_name -> cloudprober.surfacer.prometheus.SurfacerConf
	4,  // 4: cloudprober.surfacer.SurfacerDef.stackdriver_surfacer:type_name -> cloudprober.surfacer.stackdriver.SurfacerConf
	5,  // 5: cloudprober.surfacer.SurfacerDef.file_surfacer:type_name -> cloudprober.surfacer.file.SurfacerConf
	6,  // 6: cloudprober.surfacer.SurfacerDef.postgres_surfacer:type_name -> cloudprober.surfacer.postgres.SurfacerConf
	7,  // 7: cloudprober.surfacer.SurfacerDef.pubsub_surfacer:type_name -> cloudprober.surfacer.pubsub.SurfacerConf
	8,  // 8: cloudprober.surfacer.SurfacerDef.cloudwatch_surfacer:type_name -> cloudprober.surfacer.cloudwatch.SurfacerConf
	9,  // 9: cloudprober.surfacer.SurfacerDef.datadog_surfacer:type_name -> cloudprober.surfacer.datadog.SurfacerConf
	10, // 10: cloudprober.surfacer.SurfacerDef.probestatus_surfacer:type_name -> cloudprober.surfacer.probestatus.SurfacerConf
	11, // 11: cloudprober.surfacer.SurfacerDef.bigquery_surfacer:type_name -> cloudprober.surfacer.bigquery.SurfacerConf
	12, // 12: cloudprober.surfacer.SurfacerDef.otel_surfacer:type_name -> cloudprober.surfacer.otel.SurfacerConf
	13, // [13:13] is the sub-list for method output_type
	13, // [13:13] is the sub-list for method input_type
	13, // [13:13] is the sub-list for extension type_name
	13, // [13:13] is the sub-list for extension extendee
	0,  // [0:13] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_surfacers_proto_config_proto != nil {
		return
	}
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_msgTypes[1].OneofWrappers = []any{
		(*SurfacerDef_PrometheusSurfacer)(nil),
		(*SurfacerDef_StackdriverSurfacer)(nil),
		(*SurfacerDef_FileSurfacer)(nil),
		(*SurfacerDef_PostgresSurfacer)(nil),
		(*SurfacerDef_PubsubSurfacer)(nil),
		(*SurfacerDef_CloudwatchSurfacer)(nil),
		(*SurfacerDef_DatadogSurfacer)(nil),
		(*SurfacerDef_ProbestatusSurfacer)(nil),
		(*SurfacerDef_BigquerySurfacer)(nil),
		(*SurfacerDef_OtelSurfacer)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_depIdxs,
		EnumInfos:         file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_enumTypes,
		MessageInfos:      file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_surfacers_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_depIdxs = nil
}
