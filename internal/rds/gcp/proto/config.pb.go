// Configuration proto for GCP provider.
// Example config:
// {
//   project: 'test-project-id'
//
//   # GCE instances
//   gce_instances {}
//
//   # RTC variables from the config lame-duck-targets, re-evaluated every 10s.
//   rtc_variables {
//     rtc_config {
//       name: "lame-duck-targets"
//     }
//   }
//
//   # Pub/Sub messages from the topic lame-duck-targets.
//   pubsub_messages {
//     subscription {
//       name: "lame-duck-targets-{{.hostname}}"
//       topic_name: "lame-duck-targets"
//     }
//   }
// }

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.2
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/internal/rds/gcp/proto/config.proto

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

type GCEInstances struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Optional zone filter regex to limit discovery to the specific zones
	// For example, zone_filter: "us-east1-*" will limit instances discovery to
	// only to the zones in the "us-east1" region.
	ZoneFilter *string `protobuf:"bytes,1,opt,name=zone_filter,json=zoneFilter" json:"zone_filter,omitempty"`
	// How often resources should be refreshed.
	ReEvalSec     *int32 `protobuf:"varint,98,opt,name=re_eval_sec,json=reEvalSec,def=300" json:"re_eval_sec,omitempty"` // default 5 min
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for GCEInstances fields.
const (
	Default_GCEInstances_ReEvalSec = int32(300)
)

func (x *GCEInstances) Reset() {
	*x = GCEInstances{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GCEInstances) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GCEInstances) ProtoMessage() {}

func (x *GCEInstances) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GCEInstances.ProtoReflect.Descriptor instead.
func (*GCEInstances) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *GCEInstances) GetZoneFilter() string {
	if x != nil && x.ZoneFilter != nil {
		return *x.ZoneFilter
	}
	return ""
}

func (x *GCEInstances) GetReEvalSec() int32 {
	if x != nil && x.ReEvalSec != nil {
		return *x.ReEvalSec
	}
	return Default_GCEInstances_ReEvalSec
}

type ForwardingRules struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Optionl region filter regex to limit discovery to specific regions, e.g.
	// "region_filter:europe-*"
	RegionFilter *string `protobuf:"bytes,1,opt,name=region_filter,json=regionFilter" json:"region_filter,omitempty"`
	// How often resources should be refreshed.
	ReEvalSec     *int32 `protobuf:"varint,98,opt,name=re_eval_sec,json=reEvalSec,def=300" json:"re_eval_sec,omitempty"` // default 5 min
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for ForwardingRules fields.
const (
	Default_ForwardingRules_ReEvalSec = int32(300)
)

func (x *ForwardingRules) Reset() {
	*x = ForwardingRules{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ForwardingRules) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ForwardingRules) ProtoMessage() {}

func (x *ForwardingRules) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ForwardingRules.ProtoReflect.Descriptor instead.
func (*ForwardingRules) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *ForwardingRules) GetRegionFilter() string {
	if x != nil && x.RegionFilter != nil {
		return *x.RegionFilter
	}
	return ""
}

func (x *ForwardingRules) GetReEvalSec() int32 {
	if x != nil && x.ReEvalSec != nil {
		return *x.ReEvalSec
	}
	return Default_ForwardingRules_ReEvalSec
}

// Runtime configurator variables.
type RTCVariables struct {
	state         protoimpl.MessageState    `protogen:"open.v1"`
	RtcConfig     []*RTCVariables_RTCConfig `protobuf:"bytes,1,rep,name=rtc_config,json=rtcConfig" json:"rtc_config,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RTCVariables) Reset() {
	*x = RTCVariables{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RTCVariables) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RTCVariables) ProtoMessage() {}

func (x *RTCVariables) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RTCVariables.ProtoReflect.Descriptor instead.
func (*RTCVariables) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescGZIP(), []int{2}
}

func (x *RTCVariables) GetRtcConfig() []*RTCVariables_RTCConfig {
	if x != nil {
		return x.RtcConfig
	}
	return nil
}

// Runtime configurator variables.
type PubSubMessages struct {
	state        protoimpl.MessageState         `protogen:"open.v1"`
	Subscription []*PubSubMessages_Subscription `protobuf:"bytes,1,rep,name=subscription" json:"subscription,omitempty"`
	// Only for testing.
	ApiEndpoint   *string `protobuf:"bytes,2,opt,name=api_endpoint,json=apiEndpoint" json:"api_endpoint,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PubSubMessages) Reset() {
	*x = PubSubMessages{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PubSubMessages) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PubSubMessages) ProtoMessage() {}

func (x *PubSubMessages) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PubSubMessages.ProtoReflect.Descriptor instead.
func (*PubSubMessages) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescGZIP(), []int{3}
}

func (x *PubSubMessages) GetSubscription() []*PubSubMessages_Subscription {
	if x != nil {
		return x.Subscription
	}
	return nil
}

func (x *PubSubMessages) GetApiEndpoint() string {
	if x != nil && x.ApiEndpoint != nil {
		return *x.ApiEndpoint
	}
	return ""
}

// GCP provider config.
type ProviderConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// GCP projects. If running on GCE, it defaults to the local project.
	Project []string `protobuf:"bytes,1,rep,name=project" json:"project,omitempty"`
	// GCE instances discovery options. This field should be declared for the GCE
	// instances discovery to be enabled.
	GceInstances *GCEInstances `protobuf:"bytes,2,opt,name=gce_instances,json=gceInstances" json:"gce_instances,omitempty"`
	// Forwarding rules discovery options. This field should be declared for the
	// forwarding rules discovery to be enabled.
	// Note that RDS supports only regional forwarding rules.
	ForwardingRules *ForwardingRules `protobuf:"bytes,3,opt,name=forwarding_rules,json=forwardingRules" json:"forwarding_rules,omitempty"`
	// RTC variables discovery options.
	RtcVariables *RTCVariables `protobuf:"bytes,4,opt,name=rtc_variables,json=rtcVariables" json:"rtc_variables,omitempty"`
	// PubSub messages discovery options.
	PubsubMessages *PubSubMessages `protobuf:"bytes,5,opt,name=pubsub_messages,json=pubsubMessages" json:"pubsub_messages,omitempty"`
	// Compute API version.
	ApiVersion *string `protobuf:"bytes,99,opt,name=api_version,json=apiVersion,def=v1" json:"api_version,omitempty"`
	// Compute API endpoint. Currently supported only for GCE instances and
	// forwarding rules.
	ApiEndpoint   *string `protobuf:"bytes,100,opt,name=api_endpoint,json=apiEndpoint,def=https://www.googleapis.com/compute/" json:"api_endpoint,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for ProviderConfig fields.
const (
	Default_ProviderConfig_ApiVersion  = string("v1")
	Default_ProviderConfig_ApiEndpoint = string("https://www.googleapis.com/compute/")
)

func (x *ProviderConfig) Reset() {
	*x = ProviderConfig{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ProviderConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProviderConfig) ProtoMessage() {}

func (x *ProviderConfig) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProviderConfig.ProtoReflect.Descriptor instead.
func (*ProviderConfig) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescGZIP(), []int{4}
}

func (x *ProviderConfig) GetProject() []string {
	if x != nil {
		return x.Project
	}
	return nil
}

func (x *ProviderConfig) GetGceInstances() *GCEInstances {
	if x != nil {
		return x.GceInstances
	}
	return nil
}

func (x *ProviderConfig) GetForwardingRules() *ForwardingRules {
	if x != nil {
		return x.ForwardingRules
	}
	return nil
}

func (x *ProviderConfig) GetRtcVariables() *RTCVariables {
	if x != nil {
		return x.RtcVariables
	}
	return nil
}

func (x *ProviderConfig) GetPubsubMessages() *PubSubMessages {
	if x != nil {
		return x.PubsubMessages
	}
	return nil
}

func (x *ProviderConfig) GetApiVersion() string {
	if x != nil && x.ApiVersion != nil {
		return *x.ApiVersion
	}
	return Default_ProviderConfig_ApiVersion
}

func (x *ProviderConfig) GetApiEndpoint() string {
	if x != nil && x.ApiEndpoint != nil {
		return *x.ApiEndpoint
	}
	return Default_ProviderConfig_ApiEndpoint
}

type RTCVariables_RTCConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	Name  *string                `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	// How often RTC variables should be evaluated/expanded.
	ReEvalSec     *int32 `protobuf:"varint,2,opt,name=re_eval_sec,json=reEvalSec,def=10" json:"re_eval_sec,omitempty"` // default 10 sec
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for RTCVariables_RTCConfig fields.
const (
	Default_RTCVariables_RTCConfig_ReEvalSec = int32(10)
)

func (x *RTCVariables_RTCConfig) Reset() {
	*x = RTCVariables_RTCConfig{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RTCVariables_RTCConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RTCVariables_RTCConfig) ProtoMessage() {}

func (x *RTCVariables_RTCConfig) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RTCVariables_RTCConfig.ProtoReflect.Descriptor instead.
func (*RTCVariables_RTCConfig) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescGZIP(), []int{2, 0}
}

func (x *RTCVariables_RTCConfig) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *RTCVariables_RTCConfig) GetReEvalSec() int32 {
	if x != nil && x.ReEvalSec != nil {
		return *x.ReEvalSec
	}
	return Default_RTCVariables_RTCConfig_ReEvalSec
}

type PubSubMessages_Subscription struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Subscription name. If it doesn't exist already, we try to create one.
	Name *string `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	// Topic name. This is used to create the subscription if it doesn't exist
	// already.
	TopicName *string `protobuf:"bytes,2,opt,name=topic_name,json=topicName" json:"topic_name,omitempty"`
	// If subscription already exists, how far back to seek back on restart.
	// Note that duplicate data is fine as we filter by publish time.
	SeekBackDurationSec *int32 `protobuf:"varint,3,opt,name=seek_back_duration_sec,json=seekBackDurationSec,def=3600" json:"seek_back_duration_sec,omitempty"`
	unknownFields       protoimpl.UnknownFields
	sizeCache           protoimpl.SizeCache
}

// Default values for PubSubMessages_Subscription fields.
const (
	Default_PubSubMessages_Subscription_SeekBackDurationSec = int32(3600)
)

func (x *PubSubMessages_Subscription) Reset() {
	*x = PubSubMessages_Subscription{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PubSubMessages_Subscription) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PubSubMessages_Subscription) ProtoMessage() {}

func (x *PubSubMessages_Subscription) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PubSubMessages_Subscription.ProtoReflect.Descriptor instead.
func (*PubSubMessages_Subscription) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescGZIP(), []int{3, 0}
}

func (x *PubSubMessages_Subscription) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *PubSubMessages_Subscription) GetTopicName() string {
	if x != nil && x.TopicName != nil {
		return *x.TopicName
	}
	return ""
}

func (x *PubSubMessages_Subscription) GetSeekBackDurationSec() int32 {
	if x != nil && x.SeekBackDurationSec != nil {
		return *x.SeekBackDurationSec
	}
	return Default_PubSubMessages_Subscription_SeekBackDurationSec
}

var File_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDesc = []byte{
	0x0a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x64,
	0x73, 0x2f, 0x67, 0x63, 0x70, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x67, 0x63, 0x70, 0x22, 0x54, 0x0a,
	0x0c, 0x47, 0x43, 0x45, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x12, 0x1f, 0x0a,
	0x0b, 0x7a, 0x6f, 0x6e, 0x65, 0x5f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0a, 0x7a, 0x6f, 0x6e, 0x65, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x23,
	0x0a, 0x0b, 0x72, 0x65, 0x5f, 0x65, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x62, 0x20,
	0x01, 0x28, 0x05, 0x3a, 0x03, 0x33, 0x30, 0x30, 0x52, 0x09, 0x72, 0x65, 0x45, 0x76, 0x61, 0x6c,
	0x53, 0x65, 0x63, 0x22, 0x5b, 0x0a, 0x0f, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x69, 0x6e,
	0x67, 0x52, 0x75, 0x6c, 0x65, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x67, 0x69, 0x6f, 0x6e,
	0x5f, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72,
	0x65, 0x67, 0x69, 0x6f, 0x6e, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x23, 0x0a, 0x0b, 0x72,
	0x65, 0x5f, 0x65, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x62, 0x20, 0x01, 0x28, 0x05,
	0x3a, 0x03, 0x33, 0x30, 0x30, 0x52, 0x09, 0x72, 0x65, 0x45, 0x76, 0x61, 0x6c, 0x53, 0x65, 0x63,
	0x22, 0x9f, 0x01, 0x0a, 0x0c, 0x52, 0x54, 0x43, 0x56, 0x61, 0x72, 0x69, 0x61, 0x62, 0x6c, 0x65,
	0x73, 0x12, 0x4a, 0x0a, 0x0a, 0x72, 0x74, 0x63, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x67, 0x63, 0x70, 0x2e, 0x52, 0x54, 0x43, 0x56,
	0x61, 0x72, 0x69, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x2e, 0x52, 0x54, 0x43, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x52, 0x09, 0x72, 0x74, 0x63, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x1a, 0x43, 0x0a,
	0x09, 0x52, 0x54, 0x43, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x22,
	0x0a, 0x0b, 0x72, 0x65, 0x5f, 0x65, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x05, 0x3a, 0x02, 0x31, 0x30, 0x52, 0x09, 0x72, 0x65, 0x45, 0x76, 0x61, 0x6c, 0x53,
	0x65, 0x63, 0x22, 0x87, 0x02, 0x0a, 0x0e, 0x50, 0x75, 0x62, 0x53, 0x75, 0x62, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x73, 0x12, 0x54, 0x0a, 0x0c, 0x73, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x30, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x67, 0x63,
	0x70, 0x2e, 0x50, 0x75, 0x62, 0x53, 0x75, 0x62, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73,
	0x2e, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0c, 0x73,
	0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x21, 0x0a, 0x0c, 0x61,
	0x70, 0x69, 0x5f, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x61, 0x70, 0x69, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x1a, 0x7c,
	0x0a, 0x0c, 0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x5f, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x4e, 0x61, 0x6d,
	0x65, 0x12, 0x39, 0x0a, 0x16, 0x73, 0x65, 0x65, 0x6b, 0x5f, 0x62, 0x61, 0x63, 0x6b, 0x5f, 0x64,
	0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x05, 0x3a, 0x04, 0x33, 0x36, 0x30, 0x30, 0x52, 0x13, 0x73, 0x65, 0x65, 0x6b, 0x42, 0x61, 0x63,
	0x6b, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x63, 0x22, 0xc6, 0x03, 0x0a,
	0x0e, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12,
	0x18, 0x0a, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x07, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x46, 0x0a, 0x0d, 0x67, 0x63, 0x65,
	0x5f, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72,
	0x64, 0x73, 0x2e, 0x67, 0x63, 0x70, 0x2e, 0x47, 0x43, 0x45, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e,
	0x63, 0x65, 0x73, 0x52, 0x0c, 0x67, 0x63, 0x65, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65,
	0x73, 0x12, 0x4f, 0x0a, 0x10, 0x66, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x69, 0x6e, 0x67, 0x5f,
	0x72, 0x75, 0x6c, 0x65, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x67, 0x63,
	0x70, 0x2e, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x69, 0x6e, 0x67, 0x52, 0x75, 0x6c, 0x65,
	0x73, 0x52, 0x0f, 0x66, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x69, 0x6e, 0x67, 0x52, 0x75, 0x6c,
	0x65, 0x73, 0x12, 0x46, 0x0a, 0x0d, 0x72, 0x74, 0x63, 0x5f, 0x76, 0x61, 0x72, 0x69, 0x61, 0x62,
	0x6c, 0x65, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x67, 0x63, 0x70, 0x2e,
	0x52, 0x54, 0x43, 0x56, 0x61, 0x72, 0x69, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x52, 0x0c, 0x72, 0x74,
	0x63, 0x56, 0x61, 0x72, 0x69, 0x61, 0x62, 0x6c, 0x65, 0x73, 0x12, 0x4c, 0x0a, 0x0f, 0x70, 0x75,
	0x62, 0x73, 0x75, 0x62, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x67, 0x63, 0x70, 0x2e, 0x50, 0x75, 0x62, 0x53, 0x75, 0x62,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x52, 0x0e, 0x70, 0x75, 0x62, 0x73, 0x75, 0x62,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x12, 0x23, 0x0a, 0x0b, 0x61, 0x70, 0x69, 0x5f,
	0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x63, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x02, 0x76,
	0x31, 0x52, 0x0a, 0x61, 0x70, 0x69, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x46, 0x0a,
	0x0c, 0x61, 0x70, 0x69, 0x5f, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x64, 0x20,
	0x01, 0x28, 0x09, 0x3a, 0x23, 0x68, 0x74, 0x74, 0x70, 0x73, 0x3a, 0x2f, 0x2f, 0x77, 0x77, 0x77,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x61, 0x70, 0x69, 0x73, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x63, 0x6f, 0x6d, 0x70, 0x75, 0x74, 0x65, 0x2f, 0x52, 0x0b, 0x61, 0x70, 0x69, 0x45, 0x6e, 0x64,
	0x70, 0x6f, 0x69, 0x6e, 0x74, 0x42, 0x3b, 0x5a, 0x39, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x64, 0x73, 0x2f, 0x67, 0x63, 0x70, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_goTypes = []any{
	(*GCEInstances)(nil),                // 0: cloudprober.rds.gcp.GCEInstances
	(*ForwardingRules)(nil),             // 1: cloudprober.rds.gcp.ForwardingRules
	(*RTCVariables)(nil),                // 2: cloudprober.rds.gcp.RTCVariables
	(*PubSubMessages)(nil),              // 3: cloudprober.rds.gcp.PubSubMessages
	(*ProviderConfig)(nil),              // 4: cloudprober.rds.gcp.ProviderConfig
	(*RTCVariables_RTCConfig)(nil),      // 5: cloudprober.rds.gcp.RTCVariables.RTCConfig
	(*PubSubMessages_Subscription)(nil), // 6: cloudprober.rds.gcp.PubSubMessages.Subscription
}
var file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_depIdxs = []int32{
	5, // 0: cloudprober.rds.gcp.RTCVariables.rtc_config:type_name -> cloudprober.rds.gcp.RTCVariables.RTCConfig
	6, // 1: cloudprober.rds.gcp.PubSubMessages.subscription:type_name -> cloudprober.rds.gcp.PubSubMessages.Subscription
	0, // 2: cloudprober.rds.gcp.ProviderConfig.gce_instances:type_name -> cloudprober.rds.gcp.GCEInstances
	1, // 3: cloudprober.rds.gcp.ProviderConfig.forwarding_rules:type_name -> cloudprober.rds.gcp.ForwardingRules
	2, // 4: cloudprober.rds.gcp.ProviderConfig.rtc_variables:type_name -> cloudprober.rds.gcp.RTCVariables
	3, // 5: cloudprober.rds.gcp.ProviderConfig.pubsub_messages:type_name -> cloudprober.rds.gcp.PubSubMessages
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_internal_rds_gcp_proto_config_proto_depIdxs = nil
}
