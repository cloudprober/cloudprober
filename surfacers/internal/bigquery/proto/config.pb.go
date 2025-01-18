// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.3
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/surfacers/internal/bigquery/proto/config.proto

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

type SurfacerConf struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	ProjectName     *string                `protobuf:"bytes,1,opt,name=project_name,json=projectName" json:"project_name,omitempty"`
	BigqueryDataset *string                `protobuf:"bytes,2,opt,name=bigquery_dataset,json=bigqueryDataset" json:"bigquery_dataset,omitempty"`
	BigqueryTable   *string                `protobuf:"bytes,3,opt,name=bigquery_table,json=bigqueryTable" json:"bigquery_table,omitempty"`
	// It represents the bigquery table columns.
	//
	//	bigquery_columns {
	//	  label: "id",
	//	  column_name: "id",
	//	  column_type: "string",
	//	}
	BigqueryColumns []*BQColumn `protobuf:"bytes,4,rep,name=bigquery_columns,json=bigqueryColumns" json:"bigquery_columns,omitempty"`
	// It represents bigquery client timeout in seconds. So, if bigquery insertion
	// is not completed within this time period then the request will fail and the
	// failed rows will be retried later.
	BigqueryTimeoutSec *int64 `protobuf:"varint,5,opt,name=bigquery_timeout_sec,json=bigqueryTimeoutSec,def=30" json:"bigquery_timeout_sec,omitempty"`
	MetricsBufferSize  *int64 `protobuf:"varint,6,opt,name=metrics_buffer_size,json=metricsBufferSize,def=100000" json:"metrics_buffer_size,omitempty"`
	// This denotes the time interval after which data will be inserted in
	// bigquery. Default is 10 seconds. So after every 10 seconds all the em in
	// current will be inserted in bigquery in a default batch size of 1000
	BatchTimerSec    *int64 `protobuf:"varint,7,opt,name=batch_timer_sec,json=batchTimerSec,def=10" json:"batch_timer_sec,omitempty"`
	MetricsBatchSize *int64 `protobuf:"varint,8,opt,name=metrics_batch_size,json=metricsBatchSize,def=1000" json:"metrics_batch_size,omitempty"`
	// Column name for metrics name, value and timestamp
	MetricTimeColName  *string `protobuf:"bytes,9,opt,name=metric_time_col_name,json=metricTimeColName,def=metric_time" json:"metric_time_col_name,omitempty"`
	MetricNameColName  *string `protobuf:"bytes,10,opt,name=metric_name_col_name,json=metricNameColName,def=metric_name" json:"metric_name_col_name,omitempty"`
	MetricValueColName *string `protobuf:"bytes,11,opt,name=metric_value_col_name,json=metricValueColName,def=metric_value" json:"metric_value_col_name,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

// Default values for SurfacerConf fields.
const (
	Default_SurfacerConf_BigqueryTimeoutSec = int64(30)
	Default_SurfacerConf_MetricsBufferSize  = int64(100000)
	Default_SurfacerConf_BatchTimerSec      = int64(10)
	Default_SurfacerConf_MetricsBatchSize   = int64(1000)
	Default_SurfacerConf_MetricTimeColName  = string("metric_time")
	Default_SurfacerConf_MetricNameColName  = string("metric_name")
	Default_SurfacerConf_MetricValueColName = string("metric_value")
)

func (x *SurfacerConf) Reset() {
	*x = SurfacerConf{}
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SurfacerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerConf) ProtoMessage() {}

func (x *SurfacerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_msgTypes[0]
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
	return file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *SurfacerConf) GetProjectName() string {
	if x != nil && x.ProjectName != nil {
		return *x.ProjectName
	}
	return ""
}

func (x *SurfacerConf) GetBigqueryDataset() string {
	if x != nil && x.BigqueryDataset != nil {
		return *x.BigqueryDataset
	}
	return ""
}

func (x *SurfacerConf) GetBigqueryTable() string {
	if x != nil && x.BigqueryTable != nil {
		return *x.BigqueryTable
	}
	return ""
}

func (x *SurfacerConf) GetBigqueryColumns() []*BQColumn {
	if x != nil {
		return x.BigqueryColumns
	}
	return nil
}

func (x *SurfacerConf) GetBigqueryTimeoutSec() int64 {
	if x != nil && x.BigqueryTimeoutSec != nil {
		return *x.BigqueryTimeoutSec
	}
	return Default_SurfacerConf_BigqueryTimeoutSec
}

func (x *SurfacerConf) GetMetricsBufferSize() int64 {
	if x != nil && x.MetricsBufferSize != nil {
		return *x.MetricsBufferSize
	}
	return Default_SurfacerConf_MetricsBufferSize
}

func (x *SurfacerConf) GetBatchTimerSec() int64 {
	if x != nil && x.BatchTimerSec != nil {
		return *x.BatchTimerSec
	}
	return Default_SurfacerConf_BatchTimerSec
}

func (x *SurfacerConf) GetMetricsBatchSize() int64 {
	if x != nil && x.MetricsBatchSize != nil {
		return *x.MetricsBatchSize
	}
	return Default_SurfacerConf_MetricsBatchSize
}

func (x *SurfacerConf) GetMetricTimeColName() string {
	if x != nil && x.MetricTimeColName != nil {
		return *x.MetricTimeColName
	}
	return Default_SurfacerConf_MetricTimeColName
}

func (x *SurfacerConf) GetMetricNameColName() string {
	if x != nil && x.MetricNameColName != nil {
		return *x.MetricNameColName
	}
	return Default_SurfacerConf_MetricNameColName
}

func (x *SurfacerConf) GetMetricValueColName() string {
	if x != nil && x.MetricValueColName != nil {
		return *x.MetricValueColName
	}
	return Default_SurfacerConf_MetricValueColName
}

type BQColumn struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Label         *string                `protobuf:"bytes,1,opt,name=label" json:"label,omitempty"`
	ColumnName    *string                `protobuf:"bytes,2,opt,name=column_name,json=columnName" json:"column_name,omitempty"`
	ColumnType    *string                `protobuf:"bytes,3,opt,name=column_type,json=columnType" json:"column_type,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BQColumn) Reset() {
	*x = BQColumn{}
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BQColumn) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BQColumn) ProtoMessage() {}

func (x *BQColumn) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BQColumn.ProtoReflect.Descriptor instead.
func (*BQColumn) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *BQColumn) GetLabel() string {
	if x != nil && x.Label != nil {
		return *x.Label
	}
	return ""
}

func (x *BQColumn) GetColumnName() string {
	if x != nil && x.ColumnName != nil {
		return *x.ColumnName
	}
	return ""
}

func (x *BQColumn) GetColumnType() string {
	if x != nil && x.ColumnType != nil {
		return *x.ColumnType
	}
	return ""
}

var File_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDesc = []byte{
	0x0a, 0x51, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x22, 0xe2, 0x04, 0x0a, 0x0c, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43,
	0x6f, 0x6e, 0x66, 0x12, 0x21, 0x0a, 0x0c, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x70, 0x72, 0x6f, 0x6a, 0x65,
	0x63, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x29, 0x0a, 0x10, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x73, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0f, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72, 0x79, 0x44, 0x61, 0x74, 0x61, 0x73, 0x65,
	0x74, 0x12, 0x25, 0x0a, 0x0e, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x74, 0x61,
	0x62, 0x6c, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x62, 0x69, 0x67, 0x71, 0x75,
	0x65, 0x72, 0x79, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x52, 0x0a, 0x10, 0x62, 0x69, 0x67, 0x71,
	0x75, 0x65, 0x72, 0x79, 0x5f, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73, 0x18, 0x04, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x27, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65,
	0x72, 0x79, 0x2e, 0x42, 0x51, 0x43, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x52, 0x0f, 0x62, 0x69, 0x67,
	0x71, 0x75, 0x65, 0x72, 0x79, 0x43, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x73, 0x12, 0x34, 0x0a, 0x14,
	0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72, 0x79, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74,
	0x5f, 0x73, 0x65, 0x63, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x3a, 0x02, 0x33, 0x30, 0x52, 0x12,
	0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72, 0x79, 0x54, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x53,
	0x65, 0x63, 0x12, 0x36, 0x0a, 0x13, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x62, 0x75,
	0x66, 0x66, 0x65, 0x72, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x3a,
	0x06, 0x31, 0x30, 0x30, 0x30, 0x30, 0x30, 0x52, 0x11, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73,
	0x42, 0x75, 0x66, 0x66, 0x65, 0x72, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x2a, 0x0a, 0x0f, 0x62, 0x61,
	0x74, 0x63, 0x68, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x72, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x03, 0x3a, 0x02, 0x31, 0x30, 0x52, 0x0d, 0x62, 0x61, 0x74, 0x63, 0x68, 0x54, 0x69,
	0x6d, 0x65, 0x72, 0x53, 0x65, 0x63, 0x12, 0x32, 0x0a, 0x12, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x5f, 0x62, 0x61, 0x74, 0x63, 0x68, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x08, 0x20, 0x01,
	0x28, 0x03, 0x3a, 0x04, 0x31, 0x30, 0x30, 0x30, 0x52, 0x10, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x42, 0x61, 0x74, 0x63, 0x68, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x3c, 0x0a, 0x14, 0x6d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x63, 0x6f, 0x6c, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x0b, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x5f, 0x74, 0x69, 0x6d, 0x65, 0x52, 0x11, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x54, 0x69, 0x6d,
	0x65, 0x43, 0x6f, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x3c, 0x0a, 0x14, 0x6d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x5f, 0x63, 0x6f, 0x6c, 0x5f, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x0b, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x52, 0x11, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x4e, 0x61, 0x6d, 0x65, 0x43,
	0x6f, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x3f, 0x0a, 0x15, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x5f, 0x63, 0x6f, 0x6c, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x0b, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x0c, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x5f, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x52, 0x12, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x56, 0x61, 0x6c, 0x75, 0x65,
	0x43, 0x6f, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x62, 0x0a, 0x08, 0x42, 0x51, 0x43, 0x6f, 0x6c,
	0x75, 0x6d, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6f, 0x6c,
	0x75, 0x6d, 0x6e, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a,
	0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6f,
	0x6c, 0x75, 0x6d, 0x6e, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0a, 0x63, 0x6f, 0x6c, 0x75, 0x6d, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x42, 0x46, 0x5a, 0x44, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x62, 0x69, 0x67, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_goTypes = []any{
	(*SurfacerConf)(nil), // 0: cloudprober.surfacer.bigquery.SurfacerConf
	(*BQColumn)(nil),     // 1: cloudprober.surfacer.bigquery.BQColumn
}
var file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_depIdxs = []int32{
	1, // 0: cloudprober.surfacer.bigquery.SurfacerConf.bigquery_columns:type_name -> cloudprober.surfacer.bigquery.BQColumn
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_init()
}
func file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_surfacers_internal_bigquery_proto_config_proto_depIdxs = nil
}
