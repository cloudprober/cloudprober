// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/probes/browser/proto/config.proto

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

type TestMetricsOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DisableTestMetrics *bool `protobuf:"varint,1,opt,name=disable_test_metrics,json=disableTestMetrics" json:"disable_test_metrics,omitempty"`
	DisableAggregation *bool `protobuf:"varint,2,opt,name=disable_aggregation,json=disableAggregation" json:"disable_aggregation,omitempty"`
	EnableStepMetrics  *bool `protobuf:"varint,3,opt,name=enable_step_metrics,json=enableStepMetrics" json:"enable_step_metrics,omitempty"`
}

func (x *TestMetricsOptions) Reset() {
	*x = TestMetricsOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestMetricsOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMetricsOptions) ProtoMessage() {}

func (x *TestMetricsOptions) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestMetricsOptions.ProtoReflect.Descriptor instead.
func (*TestMetricsOptions) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *TestMetricsOptions) GetDisableTestMetrics() bool {
	if x != nil && x.DisableTestMetrics != nil {
		return *x.DisableTestMetrics
	}
	return false
}

func (x *TestMetricsOptions) GetDisableAggregation() bool {
	if x != nil && x.DisableAggregation != nil {
		return *x.DisableAggregation
	}
	return false
}

func (x *TestMetricsOptions) GetEnableStepMetrics() bool {
	if x != nil && x.EnableStepMetrics != nil {
		return *x.EnableStepMetrics
	}
	return false
}

type ProbeConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Path to the playwright test spec file. We'll search for it in the the
	// current working directory and the workdir directory.
	TestSpecFile *string `protobuf:"bytes,1,opt,name=test_spec_file,json=testSpecFile" json:"test_spec_file,omitempty"`
	// Inline playwright test spec.
	TestSpec *string `protobuf:"bytes,2,opt,name=test_spec,json=testSpec" json:"test_spec,omitempty"`
	// Workdir is path to the working directory. It should be writable. If not
	// specified, we try to create a temporary directory. All the output files
	// and reports are stored under <workdir>/output/<runid_timestamp>.
	// If you need to be able access the output files, you should set this
	// field to a persistent location, e.g. a persistent volume.
	Workdir *string `protobuf:"bytes,3,opt,name=workdir" json:"workdir,omitempty"`
	// Path to the playwright installation location. P
	// run from this location.
	PlaywrightDir *string `protobuf:"bytes,4,opt,name=playwright_dir,json=playwrightDir,def=/playwright" json:"playwright_dir,omitempty"`
	// Whether to enable screenshots for successful tests as well.
	// Note that screenshots are always enabled for failed tests, and you can
	// always save screenshots explicitly in the test spec.
	EnableScreenshotsForSuccess *bool `protobuf:"varint,5,opt,name=enable_screenshots_for_success,json=enableScreenshotsForSuccess,def=0" json:"enable_screenshots_for_success,omitempty"`
	// Traces are expensive and can slow down the test. We recommend to enable
	// this only when needed.
	EnableTraces *bool `protobuf:"varint,6,opt,name=enable_traces,json=enableTraces,def=0" json:"enable_traces,omitempty"`
	// By default, we export all test metrica as counters. You can change how
	// metrics are exported by setting the following options.
	TestMetricsOptions *TestMetricsOptions `protobuf:"bytes,7,opt,name=test_metrics_options,json=testMetricsOptions" json:"test_metrics_options,omitempty"`
	// Requests per probe.
	// Number of DNS requests per probe. Requests are executed concurrently and
	// each DNS request contributes to probe results. For example, if you run two
	// requests per probe, "total" counter will be incremented by 2.
	RequestsPerProbe *int32 `protobuf:"varint,98,opt,name=requests_per_probe,json=requestsPerProbe,def=1" json:"requests_per_probe,omitempty"`
	// How long to wait between two requests to the same target. Only relevant
	// if requests_per_probe is also configured.
	//
	// This value should be less than (interval - timeout) / requests_per_probe.
	// This is to ensure that all requests are executed within one probe interval
	// and all of them get sufficient time. For example, if probe interval is 2s,
	// timeout is 1s, and requests_per_probe is 10,  requests_interval_msec
	// should be less than 10ms.
	RequestsIntervalMsec *int32 `protobuf:"varint,99,opt,name=requests_interval_msec,json=requestsIntervalMsec,def=0" json:"requests_interval_msec,omitempty"`
}

// Default values for ProbeConf fields.
const (
	Default_ProbeConf_PlaywrightDir               = string("/playwright")
	Default_ProbeConf_EnableScreenshotsForSuccess = bool(false)
	Default_ProbeConf_EnableTraces                = bool(false)
	Default_ProbeConf_RequestsPerProbe            = int32(1)
	Default_ProbeConf_RequestsIntervalMsec        = int32(0)
)

func (x *ProbeConf) Reset() {
	*x = ProbeConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProbeConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProbeConf) ProtoMessage() {}

func (x *ProbeConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProbeConf.ProtoReflect.Descriptor instead.
func (*ProbeConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *ProbeConf) GetTestSpecFile() string {
	if x != nil && x.TestSpecFile != nil {
		return *x.TestSpecFile
	}
	return ""
}

func (x *ProbeConf) GetTestSpec() string {
	if x != nil && x.TestSpec != nil {
		return *x.TestSpec
	}
	return ""
}

func (x *ProbeConf) GetWorkdir() string {
	if x != nil && x.Workdir != nil {
		return *x.Workdir
	}
	return ""
}

func (x *ProbeConf) GetPlaywrightDir() string {
	if x != nil && x.PlaywrightDir != nil {
		return *x.PlaywrightDir
	}
	return Default_ProbeConf_PlaywrightDir
}

func (x *ProbeConf) GetEnableScreenshotsForSuccess() bool {
	if x != nil && x.EnableScreenshotsForSuccess != nil {
		return *x.EnableScreenshotsForSuccess
	}
	return Default_ProbeConf_EnableScreenshotsForSuccess
}

func (x *ProbeConf) GetEnableTraces() bool {
	if x != nil && x.EnableTraces != nil {
		return *x.EnableTraces
	}
	return Default_ProbeConf_EnableTraces
}

func (x *ProbeConf) GetTestMetricsOptions() *TestMetricsOptions {
	if x != nil {
		return x.TestMetricsOptions
	}
	return nil
}

func (x *ProbeConf) GetRequestsPerProbe() int32 {
	if x != nil && x.RequestsPerProbe != nil {
		return *x.RequestsPerProbe
	}
	return Default_ProbeConf_RequestsPerProbe
}

func (x *ProbeConf) GetRequestsIntervalMsec() int32 {
	if x != nil && x.RequestsIntervalMsec != nil {
		return *x.RequestsIntervalMsec
	}
	return Default_ProbeConf_RequestsIntervalMsec
}

var File_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDesc = []byte{
	0x0a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2f, 0x62, 0x72, 0x6f, 0x77,
	0x73, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1a, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x62, 0x72, 0x6f, 0x77, 0x73,
	0x65, 0x72, 0x22, 0xa7, 0x01, 0x0a, 0x12, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x72, 0x69,
	0x63, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x30, 0x0a, 0x14, 0x64, 0x69, 0x73,
	0x61, 0x62, 0x6c, 0x65, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x12, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65,
	0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x12, 0x2f, 0x0a, 0x13, 0x64,
	0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x12, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c,
	0x65, 0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2e, 0x0a, 0x13,
	0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x73, 0x74, 0x65, 0x70, 0x5f, 0x6d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x11, 0x65, 0x6e, 0x61, 0x62, 0x6c,
	0x65, 0x53, 0x74, 0x65, 0x70, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x22, 0xe0, 0x03, 0x0a,
	0x09, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x24, 0x0a, 0x0e, 0x74, 0x65,
	0x73, 0x74, 0x5f, 0x73, 0x70, 0x65, 0x63, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0c, 0x74, 0x65, 0x73, 0x74, 0x53, 0x70, 0x65, 0x63, 0x46, 0x69, 0x6c, 0x65,
	0x12, 0x1b, 0x0a, 0x09, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x73, 0x70, 0x65, 0x63, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x08, 0x74, 0x65, 0x73, 0x74, 0x53, 0x70, 0x65, 0x63, 0x12, 0x18, 0x0a,
	0x07, 0x77, 0x6f, 0x72, 0x6b, 0x64, 0x69, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x77, 0x6f, 0x72, 0x6b, 0x64, 0x69, 0x72, 0x12, 0x32, 0x0a, 0x0e, 0x70, 0x6c, 0x61, 0x79, 0x77,
	0x72, 0x69, 0x67, 0x68, 0x74, 0x5f, 0x64, 0x69, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x3a,
	0x0b, 0x2f, 0x70, 0x6c, 0x61, 0x79, 0x77, 0x72, 0x69, 0x67, 0x68, 0x74, 0x52, 0x0d, 0x70, 0x6c,
	0x61, 0x79, 0x77, 0x72, 0x69, 0x67, 0x68, 0x74, 0x44, 0x69, 0x72, 0x12, 0x4a, 0x0a, 0x1e, 0x65,
	0x6e, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x73, 0x63, 0x72, 0x65, 0x65, 0x6e, 0x73, 0x68, 0x6f, 0x74,
	0x73, 0x5f, 0x66, 0x6f, 0x72, 0x5f, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x08, 0x3a, 0x05, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x52, 0x1b, 0x65, 0x6e, 0x61, 0x62,
	0x6c, 0x65, 0x53, 0x63, 0x72, 0x65, 0x65, 0x6e, 0x73, 0x68, 0x6f, 0x74, 0x73, 0x46, 0x6f, 0x72,
	0x53, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x2a, 0x0a, 0x0d, 0x65, 0x6e, 0x61, 0x62, 0x6c,
	0x65, 0x5f, 0x74, 0x72, 0x61, 0x63, 0x65, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x3a, 0x05,
	0x66, 0x61, 0x6c, 0x73, 0x65, 0x52, 0x0c, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x54, 0x72, 0x61,
	0x63, 0x65, 0x73, 0x12, 0x60, 0x0a, 0x14, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x2e, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x2e, 0x54,
	0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x52, 0x12, 0x74, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x2f, 0x0a, 0x12, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x73, 0x5f, 0x70, 0x65, 0x72, 0x5f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x18, 0x62, 0x20, 0x01, 0x28,
	0x05, 0x3a, 0x01, 0x31, 0x52, 0x10, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x73, 0x50, 0x65,
	0x72, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x12, 0x37, 0x0a, 0x16, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x73, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x5f, 0x6d, 0x73, 0x65, 0x63,
	0x18, 0x63, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x01, 0x30, 0x52, 0x14, 0x72, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x73, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x4d, 0x73, 0x65, 0x63, 0x42,
	0x39, 0x5a, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2f, 0x62, 0x72, 0x6f,
	0x77, 0x73, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_goTypes = []any{
	(*TestMetricsOptions)(nil), // 0: cloudprober.probes.browser.TestMetricsOptions
	(*ProbeConf)(nil),          // 1: cloudprober.probes.browser.ProbeConf
}
var file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.probes.browser.ProbeConf.test_metrics_options:type_name -> cloudprober.probes.browser.TestMetricsOptions
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*TestMetricsOptions); i {
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
		file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*ProbeConf); i {
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
			RawDescriptor: file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_depIdxs = nil
}
