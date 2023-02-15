// Provides all configuration necessary to list targets for a cloudprober probe.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/targets/proto/targets.proto

package proto

import (
	proto "github.com/cloudprober/cloudprober/rds/client/proto"
	proto1 "github.com/cloudprober/cloudprober/rds/proto"
	proto3 "github.com/cloudprober/cloudprober/targets/file/proto"
	proto2 "github.com/cloudprober/cloudprober/targets/gce/proto"
	proto4 "github.com/cloudprober/cloudprober/targets/lameduck/proto"
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

type RDSTargets struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// RDS server options, for example:
	// rds_server_options {
	//   server_address: "rds-server.xyz:9314"
	//   oauth_config: {
	//     ...
	//   }
	// }
	RdsServerOptions *proto.ClientConf_ServerOptions `protobuf:"bytes,1,opt,name=rds_server_options,json=rdsServerOptions" json:"rds_server_options,omitempty"`
	// Resource path specifies the resources to return. Resources paths have the
	// following format:
	// <resource_provider>://<resource_type>/<additional_params>
	//
	// Examples:
	// For GCE instances in projectA: "gcp://gce_instances/<projectA>"
	// Kubernetes Pods : "k8s://pods"
	ResourcePath *string `protobuf:"bytes,2,opt,name=resource_path,json=resourcePath" json:"resource_path,omitempty"`
	// Filters to filter resources by.
	Filter []*proto1.Filter `protobuf:"bytes,3,rep,name=filter" json:"filter,omitempty"`
	// IP config to specify the IP address to pick for a resource.
	IpConfig *proto1.IPConfig `protobuf:"bytes,4,opt,name=ip_config,json=ipConfig" json:"ip_config,omitempty"`
}

func (x *RDSTargets) Reset() {
	*x = RDSTargets{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSTargets) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSTargets) ProtoMessage() {}

func (x *RDSTargets) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSTargets.ProtoReflect.Descriptor instead.
func (*RDSTargets) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{0}
}

func (x *RDSTargets) GetRdsServerOptions() *proto.ClientConf_ServerOptions {
	if x != nil {
		return x.RdsServerOptions
	}
	return nil
}

func (x *RDSTargets) GetResourcePath() string {
	if x != nil && x.ResourcePath != nil {
		return *x.ResourcePath
	}
	return ""
}

func (x *RDSTargets) GetFilter() []*proto1.Filter {
	if x != nil {
		return x.Filter
	}
	return nil
}

func (x *RDSTargets) GetIpConfig() *proto1.IPConfig {
	if x != nil {
		return x.IpConfig
	}
	return nil
}

type K8STargets struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace *string `protobuf:"bytes,1,opt,name=namespace" json:"namespace,omitempty"`
	// labelSelector uses the same format as kubernetes API calls.
	// Example:
	//   labelSelector: "k8s-app"       # label k8s-app exists
	//   labelSelector: "role=frontend" # label role=frontend
	//   labelSelector: "!canary"       # canary label doesn't exist
	LabelSelector []string `protobuf:"bytes,2,rep,name=labelSelector" json:"labelSelector,omitempty"`
	// Types that are assignable to Resources:
	//	*K8STargets_Services
	//	*K8STargets_Endpoints
	//	*K8STargets_Ingresses
	//	*K8STargets_Pods
	Resources isK8STargets_Resources `protobuf_oneof:"resources"`
	// How often to re-check k8s API servers. Note this field will be irrelevant
	// when (and if) we move to the watch API.
	ReEvalSec        *int32                          `protobuf:"varint,19,opt,name=re_eval_sec,json=reEvalSec" json:"re_eval_sec,omitempty"`
	RdsServerOptions *proto.ClientConf_ServerOptions `protobuf:"bytes,20,opt,name=rds_server_options,json=rdsServerOptions" json:"rds_server_options,omitempty"`
}

func (x *K8STargets) Reset() {
	*x = K8STargets{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *K8STargets) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*K8STargets) ProtoMessage() {}

func (x *K8STargets) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use K8STargets.ProtoReflect.Descriptor instead.
func (*K8STargets) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{1}
}

func (x *K8STargets) GetNamespace() string {
	if x != nil && x.Namespace != nil {
		return *x.Namespace
	}
	return ""
}

func (x *K8STargets) GetLabelSelector() []string {
	if x != nil {
		return x.LabelSelector
	}
	return nil
}

func (m *K8STargets) GetResources() isK8STargets_Resources {
	if m != nil {
		return m.Resources
	}
	return nil
}

func (x *K8STargets) GetServices() string {
	if x, ok := x.GetResources().(*K8STargets_Services); ok {
		return x.Services
	}
	return ""
}

func (x *K8STargets) GetEndpoints() string {
	if x, ok := x.GetResources().(*K8STargets_Endpoints); ok {
		return x.Endpoints
	}
	return ""
}

func (x *K8STargets) GetIngresses() string {
	if x, ok := x.GetResources().(*K8STargets_Ingresses); ok {
		return x.Ingresses
	}
	return ""
}

func (x *K8STargets) GetPods() string {
	if x, ok := x.GetResources().(*K8STargets_Pods); ok {
		return x.Pods
	}
	return ""
}

func (x *K8STargets) GetReEvalSec() int32 {
	if x != nil && x.ReEvalSec != nil {
		return *x.ReEvalSec
	}
	return 0
}

func (x *K8STargets) GetRdsServerOptions() *proto.ClientConf_ServerOptions {
	if x != nil {
		return x.RdsServerOptions
	}
	return nil
}

type isK8STargets_Resources interface {
	isK8STargets_Resources()
}

type K8STargets_Services struct {
	Services string `protobuf:"bytes,3,opt,name=services,oneof"`
}

type K8STargets_Endpoints struct {
	Endpoints string `protobuf:"bytes,4,opt,name=endpoints,oneof"`
}

type K8STargets_Ingresses struct {
	Ingresses string `protobuf:"bytes,5,opt,name=ingresses,oneof"`
}

type K8STargets_Pods struct {
	Pods string `protobuf:"bytes,6,opt,name=pods,oneof"`
}

func (*K8STargets_Services) isK8STargets_Resources() {}

func (*K8STargets_Endpoints) isK8STargets_Resources() {}

func (*K8STargets_Ingresses) isK8STargets_Resources() {}

func (*K8STargets_Pods) isK8STargets_Resources() {}

type TargetsDef struct {
	state           protoimpl.MessageState
	sizeCache       protoimpl.SizeCache
	unknownFields   protoimpl.UnknownFields
	extensionFields protoimpl.ExtensionFields

	// Types that are assignable to Type:
	//	*TargetsDef_HostNames
	//	*TargetsDef_SharedTargets
	//	*TargetsDef_GceTargets
	//	*TargetsDef_RdsTargets
	//	*TargetsDef_FileTargets
	//	*TargetsDef_K8S
	//	*TargetsDef_DummyTargets
	Type isTargetsDef_Type `protobuf_oneof:"type"`
	// Regex to apply on the targets.
	Regex *string `protobuf:"bytes,21,opt,name=regex" json:"regex,omitempty"`
	// Exclude lameducks. Lameduck targets can be set through RTC (realtime
	// configurator) service. This functionality works only if lame_duck_options
	// are specified.
	ExcludeLameducks *bool `protobuf:"varint,22,opt,name=exclude_lameducks,json=excludeLameducks,def=1" json:"exclude_lameducks,omitempty"`
}

// Default values for TargetsDef fields.
const (
	Default_TargetsDef_ExcludeLameducks = bool(true)
)

func (x *TargetsDef) Reset() {
	*x = TargetsDef{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TargetsDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TargetsDef) ProtoMessage() {}

func (x *TargetsDef) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TargetsDef.ProtoReflect.Descriptor instead.
func (*TargetsDef) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{2}
}

func (m *TargetsDef) GetType() isTargetsDef_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *TargetsDef) GetHostNames() string {
	if x, ok := x.GetType().(*TargetsDef_HostNames); ok {
		return x.HostNames
	}
	return ""
}

func (x *TargetsDef) GetSharedTargets() string {
	if x, ok := x.GetType().(*TargetsDef_SharedTargets); ok {
		return x.SharedTargets
	}
	return ""
}

func (x *TargetsDef) GetGceTargets() *proto2.TargetsConf {
	if x, ok := x.GetType().(*TargetsDef_GceTargets); ok {
		return x.GceTargets
	}
	return nil
}

func (x *TargetsDef) GetRdsTargets() *RDSTargets {
	if x, ok := x.GetType().(*TargetsDef_RdsTargets); ok {
		return x.RdsTargets
	}
	return nil
}

func (x *TargetsDef) GetFileTargets() *proto3.TargetsConf {
	if x, ok := x.GetType().(*TargetsDef_FileTargets); ok {
		return x.FileTargets
	}
	return nil
}

func (x *TargetsDef) GetK8S() *K8STargets {
	if x, ok := x.GetType().(*TargetsDef_K8S); ok {
		return x.K8S
	}
	return nil
}

func (x *TargetsDef) GetDummyTargets() *DummyTargets {
	if x, ok := x.GetType().(*TargetsDef_DummyTargets); ok {
		return x.DummyTargets
	}
	return nil
}

func (x *TargetsDef) GetRegex() string {
	if x != nil && x.Regex != nil {
		return *x.Regex
	}
	return ""
}

func (x *TargetsDef) GetExcludeLameducks() bool {
	if x != nil && x.ExcludeLameducks != nil {
		return *x.ExcludeLameducks
	}
	return Default_TargetsDef_ExcludeLameducks
}

type isTargetsDef_Type interface {
	isTargetsDef_Type()
}

type TargetsDef_HostNames struct {
	// Static host names, for example:
	// host_name: "www.google.com,8.8.8.8,en.wikipedia.org"
	HostNames string `protobuf:"bytes,1,opt,name=host_names,json=hostNames,oneof"`
}

type TargetsDef_SharedTargets struct {
	// Shared targets are accessed through their names.
	// Example:
	// shared_targets {
	//   name:"backend-vms"
	//   targets {
	//     rds_targets {
	//       ..
	//     }
	//   }
	// }
	//
	// probe {
	//   targets {
	//     shared_targets: "backend-vms"
	//   }
	// }
	SharedTargets string `protobuf:"bytes,5,opt,name=shared_targets,json=sharedTargets,oneof"`
}

type TargetsDef_GceTargets struct {
	// GCE targets: instances and forwarding_rules, for example:
	// gce_targets {
	//   instances {}
	// }
	GceTargets *proto2.TargetsConf `protobuf:"bytes,2,opt,name=gce_targets,json=gceTargets,oneof"`
}

type TargetsDef_RdsTargets struct {
	// ResourceDiscovery service based targets.
	// Example:
	// rds_targets {
	//   resource_path: "gcp://gce_instances/{{.project}}"
	//   filter {
	//     key: "name"
	//     value: ".*backend.*"
	//   }
	// }
	RdsTargets *RDSTargets `protobuf:"bytes,3,opt,name=rds_targets,json=rdsTargets,oneof"`
}

type TargetsDef_FileTargets struct {
	// File based targets.
	// Example:
	// file_targets {
	//   file_path: "/var/run/cloudprober/vips.textpb"
	// }
	FileTargets *proto3.TargetsConf `protobuf:"bytes,4,opt,name=file_targets,json=fileTargets,oneof"`
}

type TargetsDef_K8S struct {
	// K8s targets.
	// Example:
	// k8s {
	//   namespace: "qa"
	//   labelSelector: "k8s-app"
	//   services: ""
	// }
	K8S *K8STargets `protobuf:"bytes,6,opt,name=k8s,oneof"`
}

type TargetsDef_DummyTargets struct {
	// Empty targets to meet the probe definition requirement where there are
	// actually no targets, for example in case of some external probes.
	DummyTargets *DummyTargets `protobuf:"bytes,20,opt,name=dummy_targets,json=dummyTargets,oneof"`
}

func (*TargetsDef_HostNames) isTargetsDef_Type() {}

func (*TargetsDef_SharedTargets) isTargetsDef_Type() {}

func (*TargetsDef_GceTargets) isTargetsDef_Type() {}

func (*TargetsDef_RdsTargets) isTargetsDef_Type() {}

func (*TargetsDef_FileTargets) isTargetsDef_Type() {}

func (*TargetsDef_K8S) isTargetsDef_Type() {}

func (*TargetsDef_DummyTargets) isTargetsDef_Type() {}

// DummyTargets represent empty targets, which are useful for external
// probes that do not have any "proper" targets.  Such as ilbprober.
type DummyTargets struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DummyTargets) Reset() {
	*x = DummyTargets{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DummyTargets) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DummyTargets) ProtoMessage() {}

func (x *DummyTargets) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DummyTargets.ProtoReflect.Descriptor instead.
func (*DummyTargets) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{3}
}

// Global targets options. These options are independent of the per-probe
// targets which are defined by the "Targets" type above.
//
// Currently these options are used only for GCE targets to control things like
// how often to re-evaluate the targets and whether to check for lame ducks or
// not.
type GlobalTargetsOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// RDS server address
	// Deprecated: This option is now deprecated, please use rds_server_options
	// instead.
	//
	// Deprecated: Do not use.
	RdsServerAddress *string `protobuf:"bytes,3,opt,name=rds_server_address,json=rdsServerAddress" json:"rds_server_address,omitempty"`
	// RDS server options, for example:
	// rds_server_options {
	//   server_address: "rds-server.xyz:9314"
	//   oauth_config: {
	//     ...
	//   }
	// }
	RdsServerOptions *proto.ClientConf_ServerOptions `protobuf:"bytes,4,opt,name=rds_server_options,json=rdsServerOptions" json:"rds_server_options,omitempty"`
	// GCE targets options.
	GlobalGceTargetsOptions *proto2.GlobalOptions `protobuf:"bytes,1,opt,name=global_gce_targets_options,json=globalGceTargetsOptions" json:"global_gce_targets_options,omitempty"`
	// Lame duck options. If provided, targets module checks for the lame duck
	// targets and removes them from the targets list.
	LameDuckOptions *proto4.Options `protobuf:"bytes,2,opt,name=lame_duck_options,json=lameDuckOptions" json:"lame_duck_options,omitempty"`
}

func (x *GlobalTargetsOptions) Reset() {
	*x = GlobalTargetsOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GlobalTargetsOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GlobalTargetsOptions) ProtoMessage() {}

func (x *GlobalTargetsOptions) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GlobalTargetsOptions.ProtoReflect.Descriptor instead.
func (*GlobalTargetsOptions) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{4}
}

// Deprecated: Do not use.
func (x *GlobalTargetsOptions) GetRdsServerAddress() string {
	if x != nil && x.RdsServerAddress != nil {
		return *x.RdsServerAddress
	}
	return ""
}

func (x *GlobalTargetsOptions) GetRdsServerOptions() *proto.ClientConf_ServerOptions {
	if x != nil {
		return x.RdsServerOptions
	}
	return nil
}

func (x *GlobalTargetsOptions) GetGlobalGceTargetsOptions() *proto2.GlobalOptions {
	if x != nil {
		return x.GlobalGceTargetsOptions
	}
	return nil
}

func (x *GlobalTargetsOptions) GetLameDuckOptions() *proto4.Options {
	if x != nil {
		return x.LameDuckOptions
	}
	return nil
}

var File_github_com_cloudprober_cloudprober_targets_proto_targets_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDesc = []byte{
	0x0a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x13, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x1a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x72, 0x64, 0x73, 0x2f, 0x63, 0x6c,
	0x69, 0x65, 0x6e, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x36, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x72, 0x64, 0x73, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x72, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x42, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x41, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f,
	0x67, 0x63, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x2f, 0x6c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63, 0x6b, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xf3,
	0x01, 0x0a, 0x0a, 0x52, 0x44, 0x53, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12, 0x57, 0x0a,
	0x12, 0x72, 0x64, 0x73, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x43, 0x6c, 0x69, 0x65,
	0x6e, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x52, 0x10, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x61, 0x74, 0x68, 0x12, 0x2f, 0x0a, 0x06, 0x66,
	0x69, 0x6c, 0x74, 0x65, 0x72, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x46, 0x69,
	0x6c, 0x74, 0x65, 0x72, 0x52, 0x06, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x36, 0x0a, 0x09,
	0x69, 0x70, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x19, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64,
	0x73, 0x2e, 0x49, 0x50, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x08, 0x69, 0x70, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x22, 0xca, 0x02, 0x0a, 0x0a, 0x4b, 0x38, 0x73, 0x54, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x12, 0x24, 0x0a, 0x0d, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74,
	0x6f, 0x72, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0d, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x53,
	0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x1c, 0x0a, 0x08, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x08, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x73, 0x12, 0x1e, 0x0a, 0x09, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x09, 0x65, 0x6e, 0x64, 0x70,
	0x6f, 0x69, 0x6e, 0x74, 0x73, 0x12, 0x1e, 0x0a, 0x09, 0x69, 0x6e, 0x67, 0x72, 0x65, 0x73, 0x73,
	0x65, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x09, 0x69, 0x6e, 0x67, 0x72,
	0x65, 0x73, 0x73, 0x65, 0x73, 0x12, 0x14, 0x0a, 0x04, 0x70, 0x6f, 0x64, 0x73, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x04, 0x70, 0x6f, 0x64, 0x73, 0x12, 0x1e, 0x0a, 0x0b, 0x72,
	0x65, 0x5f, 0x65, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x13, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x09, 0x72, 0x65, 0x45, 0x76, 0x61, 0x6c, 0x53, 0x65, 0x63, 0x12, 0x57, 0x0a, 0x12, 0x72,
	0x64, 0x73, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x18, 0x14, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74,
	0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x52, 0x10, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x42, 0x0b, 0x0a, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x22, 0x8a, 0x04, 0x0a, 0x0a, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x44, 0x65, 0x66,
	0x12, 0x1f, 0x0a, 0x0a, 0x68, 0x6f, 0x73, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x09, 0x68, 0x6f, 0x73, 0x74, 0x4e, 0x61, 0x6d, 0x65,
	0x73, 0x12, 0x27, 0x0a, 0x0e, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0d, 0x73, 0x68, 0x61,
	0x72, 0x65, 0x64, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12, 0x47, 0x0a, 0x0b, 0x67, 0x63,
	0x65, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x24, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x67, 0x63, 0x65, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x73, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0a, 0x67, 0x63, 0x65, 0x54, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x12, 0x42, 0x0a, 0x0b, 0x72, 0x64, 0x73, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x52,
	0x44, 0x53, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x48, 0x00, 0x52, 0x0a, 0x72, 0x64, 0x73,
	0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12, 0x4a, 0x0a, 0x0c, 0x66, 0x69, 0x6c, 0x65, 0x5f,
	0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
	0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0b, 0x66, 0x69, 0x6c, 0x65, 0x54, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x12, 0x33, 0x0a, 0x03, 0x6b, 0x38, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1f, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x4b, 0x38, 0x73, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x73, 0x48, 0x00, 0x52, 0x03, 0x6b, 0x38, 0x73, 0x12, 0x48, 0x0a, 0x0d, 0x64, 0x75, 0x6d, 0x6d,
	0x79, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x14, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x44, 0x75, 0x6d, 0x6d, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x48, 0x00, 0x52, 0x0c, 0x64, 0x75, 0x6d, 0x6d, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x65, 0x67, 0x65, 0x78, 0x18, 0x15, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x72, 0x65, 0x67, 0x65, 0x78, 0x12, 0x31, 0x0a, 0x11, 0x65, 0x78, 0x63, 0x6c,
	0x75, 0x64, 0x65, 0x5f, 0x6c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63, 0x6b, 0x73, 0x18, 0x16, 0x20,
	0x01, 0x28, 0x08, 0x3a, 0x04, 0x74, 0x72, 0x75, 0x65, 0x52, 0x10, 0x65, 0x78, 0x63, 0x6c, 0x75,
	0x64, 0x65, 0x4c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63, 0x6b, 0x73, 0x2a, 0x09, 0x08, 0xc8, 0x01,
	0x10, 0x80, 0x80, 0x80, 0x80, 0x02, 0x42, 0x06, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x0e,
	0x0a, 0x0c, 0x44, 0x75, 0x6d, 0x6d, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x22, 0xd9,
	0x02, 0x0a, 0x14, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x30, 0x0a, 0x12, 0x72, 0x64, 0x73, 0x5f, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x02, 0x18, 0x01, 0x52, 0x10, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x57, 0x0a, 0x12, 0x72, 0x64, 0x73,
	0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x43, 0x6f,
	0x6e, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x52, 0x10, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x12, 0x63, 0x0a, 0x1a, 0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x5f, 0x67, 0x63, 0x65,
	0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x67, 0x63, 0x65,
	0x2e, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x17,
	0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x47, 0x63, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x51, 0x0a, 0x11, 0x6c, 0x61, 0x6d, 0x65, 0x5f,
	0x64, 0x75, 0x63, 0x6b, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x25, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x6c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63,
	0x6b, 0x2e, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x0f, 0x6c, 0x61, 0x6d, 0x65, 0x44,
	0x75, 0x63, 0x6b, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x42, 0x32, 0x5a, 0x30, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescData = file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_goTypes = []interface{}{
	(*RDSTargets)(nil),                     // 0: cloudprober.targets.RDSTargets
	(*K8STargets)(nil),                     // 1: cloudprober.targets.K8sTargets
	(*TargetsDef)(nil),                     // 2: cloudprober.targets.TargetsDef
	(*DummyTargets)(nil),                   // 3: cloudprober.targets.DummyTargets
	(*GlobalTargetsOptions)(nil),           // 4: cloudprober.targets.GlobalTargetsOptions
	(*proto.ClientConf_ServerOptions)(nil), // 5: cloudprober.rds.ClientConf.ServerOptions
	(*proto1.Filter)(nil),                  // 6: cloudprober.rds.Filter
	(*proto1.IPConfig)(nil),                // 7: cloudprober.rds.IPConfig
	(*proto2.TargetsConf)(nil),             // 8: cloudprober.targets.gce.TargetsConf
	(*proto3.TargetsConf)(nil),             // 9: cloudprober.targets.file.TargetsConf
	(*proto2.GlobalOptions)(nil),           // 10: cloudprober.targets.gce.GlobalOptions
	(*proto4.Options)(nil),                 // 11: cloudprober.targets.lameduck.Options
}
var file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_depIdxs = []int32{
	5,  // 0: cloudprober.targets.RDSTargets.rds_server_options:type_name -> cloudprober.rds.ClientConf.ServerOptions
	6,  // 1: cloudprober.targets.RDSTargets.filter:type_name -> cloudprober.rds.Filter
	7,  // 2: cloudprober.targets.RDSTargets.ip_config:type_name -> cloudprober.rds.IPConfig
	5,  // 3: cloudprober.targets.K8sTargets.rds_server_options:type_name -> cloudprober.rds.ClientConf.ServerOptions
	8,  // 4: cloudprober.targets.TargetsDef.gce_targets:type_name -> cloudprober.targets.gce.TargetsConf
	0,  // 5: cloudprober.targets.TargetsDef.rds_targets:type_name -> cloudprober.targets.RDSTargets
	9,  // 6: cloudprober.targets.TargetsDef.file_targets:type_name -> cloudprober.targets.file.TargetsConf
	1,  // 7: cloudprober.targets.TargetsDef.k8s:type_name -> cloudprober.targets.K8sTargets
	3,  // 8: cloudprober.targets.TargetsDef.dummy_targets:type_name -> cloudprober.targets.DummyTargets
	5,  // 9: cloudprober.targets.GlobalTargetsOptions.rds_server_options:type_name -> cloudprober.rds.ClientConf.ServerOptions
	10, // 10: cloudprober.targets.GlobalTargetsOptions.global_gce_targets_options:type_name -> cloudprober.targets.gce.GlobalOptions
	11, // 11: cloudprober.targets.GlobalTargetsOptions.lame_duck_options:type_name -> cloudprober.targets.lameduck.Options
	12, // [12:12] is the sub-list for method output_type
	12, // [12:12] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_init() }
func file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_init() {
	if File_github_com_cloudprober_cloudprober_targets_proto_targets_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSTargets); i {
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
		file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*K8STargets); i {
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
		file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TargetsDef); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			case 3:
				return &v.extensionFields
			default:
				return nil
			}
		}
		file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DummyTargets); i {
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
		file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GlobalTargetsOptions); i {
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
	file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*K8STargets_Services)(nil),
		(*K8STargets_Endpoints)(nil),
		(*K8STargets_Ingresses)(nil),
		(*K8STargets_Pods)(nil),
	}
	file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*TargetsDef_HostNames)(nil),
		(*TargetsDef_SharedTargets)(nil),
		(*TargetsDef_GceTargets)(nil),
		(*TargetsDef_RdsTargets)(nil),
		(*TargetsDef_FileTargets)(nil),
		(*TargetsDef_K8S)(nil),
		(*TargetsDef_DummyTargets)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_targets_proto_targets_proto = out.File
	file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_targets_proto_targets_proto_depIdxs = nil
}
