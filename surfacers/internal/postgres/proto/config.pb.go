// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/surfacers/internal/postgres/proto/config.proto

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
	// Postgres connection string.
	// Example:
	//
	//	"postgresql://root:${PASSWORD}@localhost/cloudprober?sslmode=disable"
	ConnectionString *string `protobuf:"bytes,1,req,name=connection_string,json=connectionString" json:"connection_string,omitempty"`
	// Metrics table name.
	// To create table (when storing all labels in single column in JSON format):
	// CREATE TABLE metrics (
	//
	//	time timestamp, metric_name varchar(80), value float8, labels jsonb
	//
	// )
	MetricsTableName *string `protobuf:"bytes,2,req,name=metrics_table_name,json=metricsTableName" json:"metrics_table_name,omitempty"`
	// Adding label_to_column fields changes how labels are stored in a Postgres
	// table. If this field is not specified at all, all the labels are stored as
	// jsonb values as the 'labels' column (this mode impacts performance
	// negatively). If label_to_colum entries are specified for some labels,
	// those labels are stored in their dedicated columns; all the labels that
	// don't have a mapping will be dropped.
	LabelToColumn     []*LabelToColumn `protobuf:"bytes,4,rep,name=label_to_column,json=labelToColumn" json:"label_to_column,omitempty"`
	MetricsBufferSize *int64           `protobuf:"varint,3,opt,name=metrics_buffer_size,json=metricsBufferSize,def=10000" json:"metrics_buffer_size,omitempty"`
	// The maximum number of metric events will be commited in one transaction at one
	// time. Metrics will be stored locally until this limit is reached. Metrics will
	// be commited  to postgres when the timer expires, or the buffer is full, whichever
	// happens first.
	MetricsBatchSize *int32 `protobuf:"varint,5,opt,name=metrics_batch_size,json=metricsBatchSize,def=1" json:"metrics_batch_size,omitempty"`
	// The maximum amount of time to hold metrics in the buffer (above).
	// Metrics will be commited  to postgres when the timer expires, or the buffer is full,
	// whichever happens first.
	BatchTimerSec *int32 `protobuf:"varint,6,opt,name=batch_timer_sec,json=batchTimerSec,def=1" json:"batch_timer_sec,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for SurfacerConf fields.
const (
	Default_SurfacerConf_MetricsBufferSize = int64(10000)
	Default_SurfacerConf_MetricsBatchSize  = int32(1)
	Default_SurfacerConf_BatchTimerSec     = int32(1)
)

func (x *SurfacerConf) Reset() {
	*x = SurfacerConf{}
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SurfacerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerConf) ProtoMessage() {}

func (x *SurfacerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_msgTypes[0]
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
	return file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *SurfacerConf) GetConnectionString() string {
	if x != nil && x.ConnectionString != nil {
		return *x.ConnectionString
	}
	return ""
}

func (x *SurfacerConf) GetMetricsTableName() string {
	if x != nil && x.MetricsTableName != nil {
		return *x.MetricsTableName
	}
	return ""
}

func (x *SurfacerConf) GetLabelToColumn() []*LabelToColumn {
	if x != nil {
		return x.LabelToColumn
	}
	return nil
}

func (x *SurfacerConf) GetMetricsBufferSize() int64 {
	if x != nil && x.MetricsBufferSize != nil {
		return *x.MetricsBufferSize
	}
	return Default_SurfacerConf_MetricsBufferSize
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

type LabelToColumn struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Label name
	Label *string `protobuf:"bytes,1,req,name=label" json:"label,omitempty"`
	// Column to map this label to:
	Column        *string `protobuf:"bytes,2,req,name=column" json:"column,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LabelToColumn) Reset() {
	*x = LabelToColumn{}
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LabelToColumn) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LabelToColumn) ProtoMessage() {}

func (x *LabelToColumn) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LabelToColumn.ProtoReflect.Descriptor instead.
func (*LabelToColumn) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *LabelToColumn) GetLabel() string {
	if x != nil && x.Label != nil {
		return *x.Label
	}
	return ""
}

func (x *LabelToColumn) GetColumn() string {
	if x != nil && x.Column != nil {
		return *x.Column
	}
	return ""
}

var File_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDesc = string([]byte{
	0x0a, 0x51, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72,
	0x65, 0x73, 0x22, 0xd2, 0x02, 0x0a, 0x0c, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43,
	0x6f, 0x6e, 0x66, 0x12, 0x2b, 0x0a, 0x11, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x5f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x52, 0x10,
	0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x12, 0x2c, 0x0a, 0x12, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x74, 0x61, 0x62, 0x6c,
	0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x02, 0x28, 0x09, 0x52, 0x10, 0x6d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x73, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x54,
	0x0a, 0x0f, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x74, 0x6f, 0x5f, 0x63, 0x6f, 0x6c, 0x75, 0x6d,
	0x6e, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x70,
	0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x54, 0x6f, 0x43,
	0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x52, 0x0d, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x54, 0x6f, 0x43, 0x6f,
	0x6c, 0x75, 0x6d, 0x6e, 0x12, 0x35, 0x0a, 0x13, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f,
	0x62, 0x75, 0x66, 0x66, 0x65, 0x72, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x03, 0x3a, 0x05, 0x31, 0x30, 0x30, 0x30, 0x30, 0x52, 0x11, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x42, 0x75, 0x66, 0x66, 0x65, 0x72, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x2f, 0x0a, 0x12, 0x6d,
	0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x62, 0x61, 0x74, 0x63, 0x68, 0x5f, 0x73, 0x69, 0x7a,
	0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x01, 0x31, 0x52, 0x10, 0x6d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x42, 0x61, 0x74, 0x63, 0x68, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x29, 0x0a, 0x0f,
	0x62, 0x61, 0x74, 0x63, 0x68, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x72, 0x5f, 0x73, 0x65, 0x63, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x01, 0x31, 0x52, 0x0d, 0x62, 0x61, 0x74, 0x63, 0x68, 0x54,
	0x69, 0x6d, 0x65, 0x72, 0x53, 0x65, 0x63, 0x22, 0x3d, 0x0a, 0x0d, 0x4c, 0x61, 0x62, 0x65, 0x6c,
	0x54, 0x6f, 0x43, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x61, 0x62, 0x65,
	0x6c, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x52, 0x05, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x12, 0x16,
	0x0a, 0x06, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x18, 0x02, 0x20, 0x02, 0x28, 0x09, 0x52, 0x06,
	0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x42, 0x46, 0x5a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
})

var (
	file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_goTypes = []any{
	(*SurfacerConf)(nil),  // 0: cloudprober.surfacer.postgres.SurfacerConf
	(*LabelToColumn)(nil), // 1: cloudprober.surfacer.postgres.LabelToColumn
}
var file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_depIdxs = []int32{
	1, // 0: cloudprober.surfacer.postgres.SurfacerConf.label_to_column:type_name -> cloudprober.surfacer.postgres.LabelToColumn
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_init()
}
func file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_internal_postgres_proto_config_proto_depIdxs = nil
}
