// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.1
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/probes/browser/proto/config.proto

package proto

import (
	proto "github.com/cloudprober/cloudprober/internal/oauth/proto"
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
	state              protoimpl.MessageState `protogen:"open.v1"`
	DisableTestMetrics *bool                  `protobuf:"varint,1,opt,name=disable_test_metrics,json=disableTestMetrics" json:"disable_test_metrics,omitempty"`
	DisableAggregation *bool                  `protobuf:"varint,2,opt,name=disable_aggregation,json=disableAggregation" json:"disable_aggregation,omitempty"`
	EnableStepMetrics  *bool                  `protobuf:"varint,3,opt,name=enable_step_metrics,json=enableStepMetrics" json:"enable_step_metrics,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *TestMetricsOptions) Reset() {
	*x = TestMetricsOptions{}
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TestMetricsOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMetricsOptions) ProtoMessage() {}

func (x *TestMetricsOptions) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[0]
	if x != nil {
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

// S3 storage backend configuration.
type S3 struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	Bucket          *string                `protobuf:"bytes,1,opt,name=bucket" json:"bucket,omitempty"`
	Path            *string                `protobuf:"bytes,2,opt,name=path" json:"path,omitempty"`
	Region          *string                `protobuf:"bytes,3,opt,name=region" json:"region,omitempty"`
	AccessKeyId     *string                `protobuf:"bytes,4,opt,name=access_key_id,json=accessKeyId" json:"access_key_id,omitempty"`
	SecretAccessKey *string                `protobuf:"bytes,5,opt,name=secret_access_key,json=secretAccessKey" json:"secret_access_key,omitempty"`
	// S3 endpoint. If not specified, default endpoint for the region is used.
	Endpoint      *string `protobuf:"bytes,6,opt,name=endpoint" json:"endpoint,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *S3) Reset() {
	*x = S3{}
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *S3) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*S3) ProtoMessage() {}

func (x *S3) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use S3.ProtoReflect.Descriptor instead.
func (*S3) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *S3) GetBucket() string {
	if x != nil && x.Bucket != nil {
		return *x.Bucket
	}
	return ""
}

func (x *S3) GetPath() string {
	if x != nil && x.Path != nil {
		return *x.Path
	}
	return ""
}

func (x *S3) GetRegion() string {
	if x != nil && x.Region != nil {
		return *x.Region
	}
	return ""
}

func (x *S3) GetAccessKeyId() string {
	if x != nil && x.AccessKeyId != nil {
		return *x.AccessKeyId
	}
	return ""
}

func (x *S3) GetSecretAccessKey() string {
	if x != nil && x.SecretAccessKey != nil {
		return *x.SecretAccessKey
	}
	return ""
}

func (x *S3) GetEndpoint() string {
	if x != nil && x.Endpoint != nil {
		return *x.Endpoint
	}
	return ""
}

type GCS struct {
	state       protoimpl.MessageState   `protogen:"open.v1"`
	Bucket      *string                  `protobuf:"bytes,1,opt,name=bucket" json:"bucket,omitempty"`
	Path        *string                  `protobuf:"bytes,2,opt,name=path" json:"path,omitempty"`
	Credentials *proto.GoogleCredentials `protobuf:"bytes,3,opt,name=credentials" json:"credentials,omitempty"`
	// GCS endpoint.
	Endpoint      *string `protobuf:"bytes,4,opt,name=endpoint,def=https://storage.googleapis.com" json:"endpoint,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for GCS fields.
const (
	Default_GCS_Endpoint = string("https://storage.googleapis.com")
)

func (x *GCS) Reset() {
	*x = GCS{}
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GCS) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GCS) ProtoMessage() {}

func (x *GCS) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GCS.ProtoReflect.Descriptor instead.
func (*GCS) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP(), []int{2}
}

func (x *GCS) GetBucket() string {
	if x != nil && x.Bucket != nil {
		return *x.Bucket
	}
	return ""
}

func (x *GCS) GetPath() string {
	if x != nil && x.Path != nil {
		return *x.Path
	}
	return ""
}

func (x *GCS) GetCredentials() *proto.GoogleCredentials {
	if x != nil {
		return x.Credentials
	}
	return nil
}

func (x *GCS) GetEndpoint() string {
	if x != nil && x.Endpoint != nil {
		return *x.Endpoint
	}
	return Default_GCS_Endpoint
}

type Storage struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to Storage:
	//
	//	*Storage_LocalStorageDir
	//	*Storage_S3
	//	*Storage_Gcs
	Storage       isStorage_Storage `protobuf_oneof:"storage"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Storage) Reset() {
	*x = Storage{}
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Storage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Storage) ProtoMessage() {}

func (x *Storage) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Storage.ProtoReflect.Descriptor instead.
func (*Storage) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP(), []int{3}
}

func (x *Storage) GetStorage() isStorage_Storage {
	if x != nil {
		return x.Storage
	}
	return nil
}

func (x *Storage) GetLocalStorageDir() string {
	if x != nil {
		if x, ok := x.Storage.(*Storage_LocalStorageDir); ok {
			return x.LocalStorageDir
		}
	}
	return ""
}

func (x *Storage) GetS3() *S3 {
	if x != nil {
		if x, ok := x.Storage.(*Storage_S3); ok {
			return x.S3
		}
	}
	return nil
}

func (x *Storage) GetGcs() *GCS {
	if x != nil {
		if x, ok := x.Storage.(*Storage_Gcs); ok {
			return x.Gcs
		}
	}
	return nil
}

type isStorage_Storage interface {
	isStorage_Storage()
}

type Storage_LocalStorageDir struct {
	LocalStorageDir string `protobuf:"bytes,1,opt,name=local_storage_dir,json=localStorageDir,oneof"`
}

type Storage_S3 struct {
	S3 *S3 `protobuf:"bytes,2,opt,name=s3,oneof"`
}

type Storage_Gcs struct {
	Gcs *GCS `protobuf:"bytes,3,opt,name=gcs,oneof"`
}

func (*Storage_LocalStorageDir) isStorage_Storage() {}

func (*Storage_S3) isStorage_Storage() {}

func (*Storage_Gcs) isStorage_Storage() {}

type ArtifactsOptions struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Serve test artifacts on Cloudprober's default webserver. This is
	// disabled by default for security reasons.
	ServeOnWeb *bool `protobuf:"varint,1,opt,name=serve_on_web,json=serveOnWeb" json:"serve_on_web,omitempty"`
	// Specify web server path to serve test artifacts on.
	// Default is "/artifacts/<probename>".
	WebServerPath *string `protobuf:"bytes,2,opt,name=web_server_path,json=webServerPath" json:"web_server_path,omitempty"`
	// Storage for test artifacts. Note that test artifacts are always
	// written to the workdir first, and uploaded to the storage backend in a
	// parallel goroutine. This is to make sure that uploads don't block the
	// main probe execution.
	Storage       []*Storage `protobuf:"bytes,3,rep,name=storage" json:"storage,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ArtifactsOptions) Reset() {
	*x = ArtifactsOptions{}
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ArtifactsOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ArtifactsOptions) ProtoMessage() {}

func (x *ArtifactsOptions) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ArtifactsOptions.ProtoReflect.Descriptor instead.
func (*ArtifactsOptions) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP(), []int{4}
}

func (x *ArtifactsOptions) GetServeOnWeb() bool {
	if x != nil && x.ServeOnWeb != nil {
		return *x.ServeOnWeb
	}
	return false
}

func (x *ArtifactsOptions) GetWebServerPath() string {
	if x != nil && x.WebServerPath != nil {
		return *x.WebServerPath
	}
	return ""
}

func (x *ArtifactsOptions) GetStorage() []*Storage {
	if x != nil {
		return x.Storage
	}
	return nil
}

type CleanupOptions struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Maximum age of artifacts in seconds.
	MaxAgeSec *int32 `protobuf:"varint,1,opt,name=max_age_sec,json=maxAgeSec" json:"max_age_sec,omitempty"`
	// Cleanup interval in seconds. Default is 1 hour or max_age_sec, whichever
	// is smaller.
	CleanupIntervalSec *int32 `protobuf:"varint,3,opt,name=cleanup_interval_sec,json=cleanupIntervalSec,def=3600" json:"cleanup_interval_sec,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

// Default values for CleanupOptions fields.
const (
	Default_CleanupOptions_CleanupIntervalSec = int32(3600)
)

func (x *CleanupOptions) Reset() {
	*x = CleanupOptions{}
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CleanupOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CleanupOptions) ProtoMessage() {}

func (x *CleanupOptions) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CleanupOptions.ProtoReflect.Descriptor instead.
func (*CleanupOptions) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP(), []int{5}
}

func (x *CleanupOptions) GetMaxAgeSec() int32 {
	if x != nil && x.MaxAgeSec != nil {
		return *x.MaxAgeSec
	}
	return 0
}

func (x *CleanupOptions) GetCleanupIntervalSec() int32 {
	if x != nil && x.CleanupIntervalSec != nil {
		return *x.CleanupIntervalSec
	}
	return Default_CleanupOptions_CleanupIntervalSec
}

type ProbeConf struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Playwright test specs. These are passed to playwright as it is. This
	// field works in conjunction with test_dir -- test specs should be under
	// test directory.
	//
	// If test_spec is not specified, all test specs in the test directory are
	// executed, and since default test directory is config file's directory,
	// if you leave both the fields unspecified, all test specs co-located with
	// the config file are executed.
	TestSpec []string `protobuf:"bytes,1,rep,name=test_spec,json=testSpec" json:"test_spec,omitempty"`
	// Test directory. This is the directory where test specs are located.
	// Default test_dir is config file directory ("{{configDir}}").
	TestDir *string `protobuf:"bytes,2,opt,name=test_dir,json=testDir" json:"test_dir,omitempty"`
	// Workdir is path to the working directory. It should be writable. If not
	// specified, we try to create a temporary directory. All the output files
	// and reports are stored under <workdir>/output/.
	// If you need to be able access the output files, you should set this
	// field to a persistent location, e.g. a persistent volume, or configure
	// artifact options.
	Workdir *string `protobuf:"bytes,3,opt,name=workdir" json:"workdir,omitempty"`
	// Path to the playwright installation. We execute tests from this location.
	// If not specified, we'll use the value of environment variable
	// $PLAYWRIGHT_DIR, which is automatically set by the official cloudprober
	// playwright image (tag: "<version>-pw").
	PlaywrightDir *string `protobuf:"bytes,4,opt,name=playwright_dir,json=playwrightDir" json:"playwright_dir,omitempty"`
	// NPX path. Default is to assume npx is in the PATH.
	NpxPath *string `protobuf:"bytes,5,opt,name=npx_path,json=npxPath,def=npx" json:"npx_path,omitempty"`
	// Whether to enable screenshots for successful tests as well.
	// Note that screenshots are always enabled for failed tests, and you can
	// always save screenshots explicitly in the test spec.
	SaveScreenshotsForSuccess *bool `protobuf:"varint,6,opt,name=save_screenshots_for_success,json=saveScreenshotsForSuccess,def=0" json:"save_screenshots_for_success,omitempty"`
	// Traces are expensive and can slow down the test. We recommend to enable
	// this only when needed.
	SaveTraces *bool `protobuf:"varint,7,opt,name=save_traces,json=saveTraces,def=0" json:"save_traces,omitempty"`
	// By default, we export all test metrica as counters. You can change how
	// metrics are exported by setting the following options.
	TestMetricsOptions *TestMetricsOptions `protobuf:"bytes,8,opt,name=test_metrics_options,json=testMetricsOptions" json:"test_metrics_options,omitempty"`
	// Artifacts options.
	ArtifactsOptions *ArtifactsOptions `protobuf:"bytes,9,opt,name=artifacts_options,json=artifactsOptions" json:"artifacts_options,omitempty"`
	// Cleanup options.
	WorkdirCleanupOptions *CleanupOptions `protobuf:"bytes,10,opt,name=workdir_cleanup_options,json=workdirCleanupOptions" json:"workdir_cleanup_options,omitempty"`
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
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

// Default values for ProbeConf fields.
const (
	Default_ProbeConf_NpxPath                   = string("npx")
	Default_ProbeConf_SaveScreenshotsForSuccess = bool(false)
	Default_ProbeConf_SaveTraces                = bool(false)
	Default_ProbeConf_RequestsPerProbe          = int32(1)
	Default_ProbeConf_RequestsIntervalMsec      = int32(0)
)

func (x *ProbeConf) Reset() {
	*x = ProbeConf{}
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ProbeConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProbeConf) ProtoMessage() {}

func (x *ProbeConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[6]
	if x != nil {
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
	return file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDescGZIP(), []int{6}
}

func (x *ProbeConf) GetTestSpec() []string {
	if x != nil {
		return x.TestSpec
	}
	return nil
}

func (x *ProbeConf) GetTestDir() string {
	if x != nil && x.TestDir != nil {
		return *x.TestDir
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
	return ""
}

func (x *ProbeConf) GetNpxPath() string {
	if x != nil && x.NpxPath != nil {
		return *x.NpxPath
	}
	return Default_ProbeConf_NpxPath
}

func (x *ProbeConf) GetSaveScreenshotsForSuccess() bool {
	if x != nil && x.SaveScreenshotsForSuccess != nil {
		return *x.SaveScreenshotsForSuccess
	}
	return Default_ProbeConf_SaveScreenshotsForSuccess
}

func (x *ProbeConf) GetSaveTraces() bool {
	if x != nil && x.SaveTraces != nil {
		return *x.SaveTraces
	}
	return Default_ProbeConf_SaveTraces
}

func (x *ProbeConf) GetTestMetricsOptions() *TestMetricsOptions {
	if x != nil {
		return x.TestMetricsOptions
	}
	return nil
}

func (x *ProbeConf) GetArtifactsOptions() *ArtifactsOptions {
	if x != nil {
		return x.ArtifactsOptions
	}
	return nil
}

func (x *ProbeConf) GetWorkdirCleanupOptions() *CleanupOptions {
	if x != nil {
		return x.WorkdirCleanupOptions
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
	0x65, 0x72, 0x1a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x6f, 0x61, 0x75, 0x74, 0x68, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa7, 0x01, 0x0a, 0x12, 0x54, 0x65, 0x73,
	0x74, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12,
	0x30, 0x0a, 0x14, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f,
	0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x12, 0x64,
	0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x12, 0x2f, 0x0a, 0x13, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x61, 0x67, 0x67,
	0x72, 0x65, 0x67, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x12,
	0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x41, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x2e, 0x0a, 0x13, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x73, 0x74, 0x65,
	0x70, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x11, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x74, 0x65, 0x70, 0x4d, 0x65, 0x74, 0x72, 0x69,
	0x63, 0x73, 0x22, 0xb4, 0x01, 0x0a, 0x02, 0x53, 0x33, 0x12, 0x16, 0x0a, 0x06, 0x62, 0x75, 0x63,
	0x6b, 0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x62, 0x75, 0x63, 0x6b, 0x65,
	0x74, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e, 0x12, 0x22, 0x0a,
	0x0d, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x5f, 0x6b, 0x65, 0x79, 0x5f, 0x69, 0x64, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x4b, 0x65, 0x79, 0x49,
	0x64, 0x12, 0x2a, 0x0a, 0x11, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x5f, 0x61, 0x63, 0x63, 0x65,
	0x73, 0x73, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x73, 0x65,
	0x63, 0x72, 0x65, 0x74, 0x41, 0x63, 0x63, 0x65, 0x73, 0x73, 0x4b, 0x65, 0x79, 0x12, 0x1a, 0x0a,
	0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x22, 0xb5, 0x01, 0x0a, 0x03, 0x47, 0x43,
	0x53, 0x12, 0x16, 0x0a, 0x06, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x62, 0x75, 0x63, 0x6b, 0x65, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74,
	0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x46, 0x0a,
	0x0b, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x24, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x2e, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x43, 0x72, 0x65,
	0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x52, 0x0b, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e,
	0x74, 0x69, 0x61, 0x6c, 0x73, 0x12, 0x3a, 0x0a, 0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x1e, 0x68, 0x74, 0x74, 0x70, 0x73, 0x3a, 0x2f,
	0x2f, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x61,
	0x70, 0x69, 0x73, 0x2e, 0x63, 0x6f, 0x6d, 0x52, 0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x22, 0xa9, 0x01, 0x0a, 0x07, 0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x12, 0x2c, 0x0a,
	0x11, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x5f, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5f, 0x64,
	0x69, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0f, 0x6c, 0x6f, 0x63, 0x61,
	0x6c, 0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x44, 0x69, 0x72, 0x12, 0x30, 0x0a, 0x02, 0x73,
	0x33, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x62, 0x72, 0x6f,
	0x77, 0x73, 0x65, 0x72, 0x2e, 0x53, 0x33, 0x48, 0x00, 0x52, 0x02, 0x73, 0x33, 0x12, 0x33, 0x0a,
	0x03, 0x67, 0x63, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e,
	0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x2e, 0x47, 0x43, 0x53, 0x48, 0x00, 0x52, 0x03, 0x67,
	0x63, 0x73, 0x42, 0x09, 0x0a, 0x07, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x22, 0x9b, 0x01,
	0x0a, 0x10, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x12, 0x20, 0x0a, 0x0c, 0x73, 0x65, 0x72, 0x76, 0x65, 0x5f, 0x6f, 0x6e, 0x5f, 0x77,
	0x65, 0x62, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x73, 0x65, 0x72, 0x76, 0x65, 0x4f,
	0x6e, 0x57, 0x65, 0x62, 0x12, 0x26, 0x0a, 0x0f, 0x77, 0x65, 0x62, 0x5f, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x77,
	0x65, 0x62, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x50, 0x61, 0x74, 0x68, 0x12, 0x3d, 0x0a, 0x07,
	0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x73, 0x2e, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x2e, 0x53, 0x74, 0x6f, 0x72, 0x61,
	0x67, 0x65, 0x52, 0x07, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x22, 0x68, 0x0a, 0x0e, 0x43,
	0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x1e, 0x0a,
	0x0b, 0x6d, 0x61, 0x78, 0x5f, 0x61, 0x67, 0x65, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x09, 0x6d, 0x61, 0x78, 0x41, 0x67, 0x65, 0x53, 0x65, 0x63, 0x12, 0x36, 0x0a,
	0x14, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61,
	0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x04, 0x33, 0x36, 0x30,
	0x30, 0x52, 0x12, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76,
	0x61, 0x6c, 0x53, 0x65, 0x63, 0x22, 0x9f, 0x05, 0x0a, 0x09, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x43,
	0x6f, 0x6e, 0x66, 0x12, 0x1b, 0x0a, 0x09, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x73, 0x70, 0x65, 0x63,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x08, 0x74, 0x65, 0x73, 0x74, 0x53, 0x70, 0x65, 0x63,
	0x12, 0x19, 0x0a, 0x08, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x64, 0x69, 0x72, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x07, 0x74, 0x65, 0x73, 0x74, 0x44, 0x69, 0x72, 0x12, 0x18, 0x0a, 0x07, 0x77,
	0x6f, 0x72, 0x6b, 0x64, 0x69, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x77, 0x6f,
	0x72, 0x6b, 0x64, 0x69, 0x72, 0x12, 0x25, 0x0a, 0x0e, 0x70, 0x6c, 0x61, 0x79, 0x77, 0x72, 0x69,
	0x67, 0x68, 0x74, 0x5f, 0x64, 0x69, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x70,
	0x6c, 0x61, 0x79, 0x77, 0x72, 0x69, 0x67, 0x68, 0x74, 0x44, 0x69, 0x72, 0x12, 0x1e, 0x0a, 0x08,
	0x6e, 0x70, 0x78, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x03,
	0x6e, 0x70, 0x78, 0x52, 0x07, 0x6e, 0x70, 0x78, 0x50, 0x61, 0x74, 0x68, 0x12, 0x46, 0x0a, 0x1c,
	0x73, 0x61, 0x76, 0x65, 0x5f, 0x73, 0x63, 0x72, 0x65, 0x65, 0x6e, 0x73, 0x68, 0x6f, 0x74, 0x73,
	0x5f, 0x66, 0x6f, 0x72, 0x5f, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x08, 0x3a, 0x05, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x52, 0x19, 0x73, 0x61, 0x76, 0x65, 0x53,
	0x63, 0x72, 0x65, 0x65, 0x6e, 0x73, 0x68, 0x6f, 0x74, 0x73, 0x46, 0x6f, 0x72, 0x53, 0x75, 0x63,
	0x63, 0x65, 0x73, 0x73, 0x12, 0x26, 0x0a, 0x0b, 0x73, 0x61, 0x76, 0x65, 0x5f, 0x74, 0x72, 0x61,
	0x63, 0x65, 0x73, 0x18, 0x07, 0x20, 0x01, 0x28, 0x08, 0x3a, 0x05, 0x66, 0x61, 0x6c, 0x73, 0x65,
	0x52, 0x0a, 0x73, 0x61, 0x76, 0x65, 0x54, 0x72, 0x61, 0x63, 0x65, 0x73, 0x12, 0x60, 0x0a, 0x14,
	0x74, 0x65, 0x73, 0x74, 0x5f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x6f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e,
	0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x12, 0x74, 0x65, 0x73, 0x74,
	0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x59,
	0x0a, 0x11, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73, 0x5f, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x62,
	0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x2e, 0x41, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63, 0x74, 0x73,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x10, 0x61, 0x72, 0x74, 0x69, 0x66, 0x61, 0x63,
	0x74, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x62, 0x0a, 0x17, 0x77, 0x6f, 0x72,
	0x6b, 0x64, 0x69, 0x72, 0x5f, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70, 0x5f, 0x6f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e,
	0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x2e, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x15, 0x77, 0x6f, 0x72, 0x6b, 0x64, 0x69, 0x72, 0x43,
	0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x2f, 0x0a,
	0x12, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x73, 0x5f, 0x70, 0x65, 0x72, 0x5f, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x18, 0x62, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x01, 0x31, 0x52, 0x10, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x73, 0x50, 0x65, 0x72, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x12, 0x37,
	0x0a, 0x16, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x73, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x76, 0x61, 0x6c, 0x5f, 0x6d, 0x73, 0x65, 0x63, 0x18, 0x63, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x01,
	0x30, 0x52, 0x14, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x73, 0x49, 0x6e, 0x74, 0x65, 0x72,
	0x76, 0x61, 0x6c, 0x4d, 0x73, 0x65, 0x63, 0x42, 0x39, 0x5a, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x73, 0x2f, 0x62, 0x72, 0x6f, 0x77, 0x73, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f,
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

var file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_goTypes = []any{
	(*TestMetricsOptions)(nil),      // 0: cloudprober.probes.browser.TestMetricsOptions
	(*S3)(nil),                      // 1: cloudprober.probes.browser.S3
	(*GCS)(nil),                     // 2: cloudprober.probes.browser.GCS
	(*Storage)(nil),                 // 3: cloudprober.probes.browser.Storage
	(*ArtifactsOptions)(nil),        // 4: cloudprober.probes.browser.ArtifactsOptions
	(*CleanupOptions)(nil),          // 5: cloudprober.probes.browser.CleanupOptions
	(*ProbeConf)(nil),               // 6: cloudprober.probes.browser.ProbeConf
	(*proto.GoogleCredentials)(nil), // 7: cloudprober.oauth.GoogleCredentials
}
var file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_depIdxs = []int32{
	7, // 0: cloudprober.probes.browser.GCS.credentials:type_name -> cloudprober.oauth.GoogleCredentials
	1, // 1: cloudprober.probes.browser.Storage.s3:type_name -> cloudprober.probes.browser.S3
	2, // 2: cloudprober.probes.browser.Storage.gcs:type_name -> cloudprober.probes.browser.GCS
	3, // 3: cloudprober.probes.browser.ArtifactsOptions.storage:type_name -> cloudprober.probes.browser.Storage
	0, // 4: cloudprober.probes.browser.ProbeConf.test_metrics_options:type_name -> cloudprober.probes.browser.TestMetricsOptions
	4, // 5: cloudprober.probes.browser.ProbeConf.artifacts_options:type_name -> cloudprober.probes.browser.ArtifactsOptions
	5, // 6: cloudprober.probes.browser.ProbeConf.workdir_cleanup_options:type_name -> cloudprober.probes.browser.CleanupOptions
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto != nil {
		return
	}
	file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_msgTypes[3].OneofWrappers = []any{
		(*Storage_LocalStorageDir)(nil),
		(*Storage_S3)(nil),
		(*Storage_Gcs)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_probes_browser_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
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
