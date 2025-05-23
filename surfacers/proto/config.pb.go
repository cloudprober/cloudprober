// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.27.5
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
	unsafe "unsafe"
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
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           *string                `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value         *string                `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
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
	state protoimpl.MessageState `protogen:"open.v1"`
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
	// Types that are valid to be assigned to Surfacer:
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
	Surfacer      isSurfacerDef_Surfacer `protobuf_oneof:"surfacer"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
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

func (x *SurfacerDef) GetSurfacer() isSurfacerDef_Surfacer {
	if x != nil {
		return x.Surfacer
	}
	return nil
}

func (x *SurfacerDef) GetPrometheusSurfacer() *proto.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_PrometheusSurfacer); ok {
			return x.PrometheusSurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetStackdriverSurfacer() *proto1.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_StackdriverSurfacer); ok {
			return x.StackdriverSurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetFileSurfacer() *proto2.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_FileSurfacer); ok {
			return x.FileSurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetPostgresSurfacer() *proto3.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_PostgresSurfacer); ok {
			return x.PostgresSurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetPubsubSurfacer() *proto4.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_PubsubSurfacer); ok {
			return x.PubsubSurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetCloudwatchSurfacer() *proto5.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_CloudwatchSurfacer); ok {
			return x.CloudwatchSurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetDatadogSurfacer() *proto6.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_DatadogSurfacer); ok {
			return x.DatadogSurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetProbestatusSurfacer() *proto7.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_ProbestatusSurfacer); ok {
			return x.ProbestatusSurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetBigquerySurfacer() *proto8.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_BigquerySurfacer); ok {
			return x.BigquerySurfacer
		}
	}
	return nil
}

func (x *SurfacerDef) GetOtelSurfacer() *proto9.SurfacerConf {
	if x != nil {
		if x, ok := x.Surfacer.(*SurfacerDef_OtelSurfacer); ok {
			return x.OtelSurfacer
		}
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

const file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc = "" +
	"\n" +
	"?github.com/cloudprober/cloudprober/surfacers/proto/config.proto\x12\x14cloudprober.surfacer\x1aSgithub.com/cloudprober/cloudprober/surfacers/internal/cloudwatch/proto/config.proto\x1aPgithub.com/cloudprober/cloudprober/surfacers/internal/datadog/proto/config.proto\x1aMgithub.com/cloudprober/cloudprober/surfacers/internal/file/proto/config.proto\x1aMgithub.com/cloudprober/cloudprober/surfacers/internal/otel/proto/config.proto\x1aQgithub.com/cloudprober/cloudprober/surfacers/internal/postgres/proto/config.proto\x1aTgithub.com/cloudprober/cloudprober/surfacers/internal/probestatus/proto/config.proto\x1aSgithub.com/cloudprober/cloudprober/surfacers/internal/prometheus/proto/config.proto\x1aOgithub.com/cloudprober/cloudprober/surfacers/internal/pubsub/proto/config.proto\x1aTgithub.com/cloudprober/cloudprober/surfacers/internal/stackdriver/proto/config.proto\x1aQgithub.com/cloudprober/cloudprober/surfacers/internal/bigquery/proto/config.proto\"5\n" +
	"\vLabelFilter\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value\"\xd0\f\n" +
	"\vSurfacerDef\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12.\n" +
	"\x04type\x18\x02 \x01(\x0e2\x1a.cloudprober.surfacer.TypeR\x04type\x125\n" +
	"\x13metrics_buffer_size\x18\x03 \x01(\x03:\x0510000R\x11metricsBufferSize\x12Z\n" +
	"\x18allow_metrics_with_label\x18\x04 \x03(\v2!.cloudprober.surfacer.LabelFilterR\x15allowMetricsWithLabel\x12\\\n" +
	"\x19ignore_metrics_with_label\x18\x05 \x03(\v2!.cloudprober.surfacer.LabelFilterR\x16ignoreMetricsWithLabel\x125\n" +
	"\x17allow_metrics_with_name\x18\x06 \x01(\tR\x14allowMetricsWithName\x127\n" +
	"\x18ignore_metrics_with_name\x18\a \x01(\tR\x15ignoreMetricsWithName\x122\n" +
	"\x12add_failure_metric\x18\b \x01(\b:\x04trueR\x10addFailureMetric\x12&\n" +
	"\x0fexport_as_gauge\x18\t \x01(\bR\rexportAsGauge\x12E\n" +
	"\x16latency_metric_pattern\x183 \x01(\t:\x0f^(.+_|)latency$R\x14latencyMetricPattern\x12X\n" +
	"\x19additional_labels_env_var\x184 \x01(\t:\x1dCLOUDPROBER_ADDITIONAL_LABELSR\x16additionalLabelsEnvVar\x12`\n" +
	"\x13prometheus_surfacer\x18\n" +
	" \x01(\v2-.cloudprober.surfacer.prometheus.SurfacerConfH\x00R\x12prometheusSurfacer\x12c\n" +
	"\x14stackdriver_surfacer\x18\v \x01(\v2..cloudprober.surfacer.stackdriver.SurfacerConfH\x00R\x13stackdriverSurfacer\x12N\n" +
	"\rfile_surfacer\x18\f \x01(\v2'.cloudprober.surfacer.file.SurfacerConfH\x00R\ffileSurfacer\x12Z\n" +
	"\x11postgres_surfacer\x18\r \x01(\v2+.cloudprober.surfacer.postgres.SurfacerConfH\x00R\x10postgresSurfacer\x12T\n" +
	"\x0fpubsub_surfacer\x18\x0e \x01(\v2).cloudprober.surfacer.pubsub.SurfacerConfH\x00R\x0epubsubSurfacer\x12`\n" +
	"\x13cloudwatch_surfacer\x18\x0f \x01(\v2-.cloudprober.surfacer.cloudwatch.SurfacerConfH\x00R\x12cloudwatchSurfacer\x12W\n" +
	"\x10datadog_surfacer\x18\x10 \x01(\v2*.cloudprober.surfacer.datadog.SurfacerConfH\x00R\x0fdatadogSurfacer\x12c\n" +
	"\x14probestatus_surfacer\x18\x11 \x01(\v2..cloudprober.surfacer.probestatus.SurfacerConfH\x00R\x13probestatusSurfacer\x12Z\n" +
	"\x11bigquery_surfacer\x18\x12 \x01(\v2+.cloudprober.surfacer.bigquery.SurfacerConfH\x00R\x10bigquerySurfacer\x12N\n" +
	"\rotel_surfacer\x18\x13 \x01(\v2'.cloudprober.surfacer.otel.SurfacerConfH\x00R\fotelSurfacerB\n" +
	"\n" +
	"\bsurfacer*\xad\x01\n" +
	"\x04Type\x12\b\n" +
	"\x04NONE\x10\x00\x12\x0e\n" +
	"\n" +
	"PROMETHEUS\x10\x01\x12\x0f\n" +
	"\vSTACKDRIVER\x10\x02\x12\b\n" +
	"\x04FILE\x10\x03\x12\f\n" +
	"\bPOSTGRES\x10\x04\x12\n" +
	"\n" +
	"\x06PUBSUB\x10\x05\x12\x0e\n" +
	"\n" +
	"CLOUDWATCH\x10\x06\x12\v\n" +
	"\aDATADOG\x10\a\x12\x0f\n" +
	"\vPROBESTATUS\x10\b\x12\f\n" +
	"\bBIGQUERY\x10\t\x12\b\n" +
	"\x04OTEL\x10\n" +
	"\x12\x10\n" +
	"\fUSER_DEFINED\x10cB4Z2github.com/cloudprober/cloudprober/surfacers/proto"

var (
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc)))
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
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_rawDesc)),
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
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_proto_config_proto_depIdxs = nil
}
