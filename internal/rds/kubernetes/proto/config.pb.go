// Configuration proto for Kubernetes provider.
//
// Example provider config:
// {
//   pods {}
// }
//
// In probe config:
// probe {
//   targets{
//     rds_targets {
//       resource_path: "k8s://pods"
//       filter {
//         key: "namespace"
//         value: "default"
//       }
//       filter {
//         key: "name"
//         value: "cloudprober.*"
//       }
//     }
//   }
// }

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/internal/rds/kubernetes/proto/config.proto

package proto

import (
	proto "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
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

type Pods struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Pods) Reset() {
	*x = Pods{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Pods) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Pods) ProtoMessage() {}

func (x *Pods) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Pods.ProtoReflect.Descriptor instead.
func (*Pods) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescGZIP(), []int{0}
}

type Endpoints struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Endpoints) Reset() {
	*x = Endpoints{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Endpoints) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Endpoints) ProtoMessage() {}

func (x *Endpoints) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Endpoints.ProtoReflect.Descriptor instead.
func (*Endpoints) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescGZIP(), []int{1}
}

type Services struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Services) Reset() {
	*x = Services{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Services) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Services) ProtoMessage() {}

func (x *Services) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Services.ProtoReflect.Descriptor instead.
func (*Services) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescGZIP(), []int{2}
}

type Ingresses struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Ingresses) Reset() {
	*x = Ingresses{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Ingresses) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Ingresses) ProtoMessage() {}

func (x *Ingresses) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Ingresses.ProtoReflect.Descriptor instead.
func (*Ingresses) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescGZIP(), []int{3}
}

// Kubernetes provider config.
type ProviderConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Namespace to list resources for. If not specified, we default to all
	// namespaces.
	Namespace *string `protobuf:"bytes,1,opt,name=namespace" json:"namespace,omitempty"`
	// Pods discovery options. This field should be declared for the pods
	// discovery to be enabled.
	Pods *Pods `protobuf:"bytes,2,opt,name=pods" json:"pods,omitempty"`
	// Endpoints discovery options. This field should be declared for the
	// endpoints discovery to be enabled.
	Endpoints *Endpoints `protobuf:"bytes,3,opt,name=endpoints" json:"endpoints,omitempty"`
	// Services discovery options. This field should be declared for the
	// services discovery to be enabled.
	Services *Services `protobuf:"bytes,4,opt,name=services" json:"services,omitempty"`
	// Ingresses discovery options. This field should be declared for the
	// ingresses discovery to be enabled.
	// Note: Ingress support is experimental and may change in future.
	Ingresses *Ingresses `protobuf:"bytes,5,opt,name=ingresses" json:"ingresses,omitempty"`
	// Label selectors to filter resources. This is useful for large clusters.
	// label_selector: ["app=cloudprober", "env!=dev"]
	LabelSelector []string `protobuf:"bytes,20,rep,name=label_selector,json=labelSelector" json:"label_selector,omitempty"`
	// Kubernetes API server address. If not specified, we assume in-cluster mode
	// and get it from the local environment variables.
	ApiServerAddress *string `protobuf:"bytes,91,opt,name=api_server_address,json=apiServerAddress" json:"api_server_address,omitempty"`
	// TLS config to authenticate communication with the API server.
	TlsConfig *proto.TLSConfig `protobuf:"bytes,93,opt,name=tls_config,json=tlsConfig" json:"tls_config,omitempty"`
	// How often resources should be evaluated/expanded.
	ReEvalSec     *int32 `protobuf:"varint,99,opt,name=re_eval_sec,json=reEvalSec,def=60" json:"re_eval_sec,omitempty"` // default 1 min
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for ProviderConfig fields.
const (
	Default_ProviderConfig_ReEvalSec = int32(60)
)

func (x *ProviderConfig) Reset() {
	*x = ProviderConfig{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ProviderConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProviderConfig) ProtoMessage() {}

func (x *ProviderConfig) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes[4]
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
	return file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescGZIP(), []int{4}
}

func (x *ProviderConfig) GetNamespace() string {
	if x != nil && x.Namespace != nil {
		return *x.Namespace
	}
	return ""
}

func (x *ProviderConfig) GetPods() *Pods {
	if x != nil {
		return x.Pods
	}
	return nil
}

func (x *ProviderConfig) GetEndpoints() *Endpoints {
	if x != nil {
		return x.Endpoints
	}
	return nil
}

func (x *ProviderConfig) GetServices() *Services {
	if x != nil {
		return x.Services
	}
	return nil
}

func (x *ProviderConfig) GetIngresses() *Ingresses {
	if x != nil {
		return x.Ingresses
	}
	return nil
}

func (x *ProviderConfig) GetLabelSelector() []string {
	if x != nil {
		return x.LabelSelector
	}
	return nil
}

func (x *ProviderConfig) GetApiServerAddress() string {
	if x != nil && x.ApiServerAddress != nil {
		return *x.ApiServerAddress
	}
	return ""
}

func (x *ProviderConfig) GetTlsConfig() *proto.TLSConfig {
	if x != nil {
		return x.TlsConfig
	}
	return nil
}

func (x *ProviderConfig) GetReEvalSec() int32 {
	if x != nil && x.ReEvalSec != nil {
		return *x.ReEvalSec
	}
	return Default_ProviderConfig_ReEvalSec
}

var File_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDesc = string([]byte{
	0x0a, 0x4d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x64,
	0x73, 0x2f, 0x6b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x1a, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73,
	0x2e, 0x6b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x1a, 0x46, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2f, 0x74, 0x6c, 0x73, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x06, 0x0a, 0x04, 0x50, 0x6f, 0x64, 0x73, 0x22, 0x0b, 0x0a, 0x09, 0x45,
	0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x22, 0x0a, 0x0a, 0x08, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x73, 0x22, 0x0b, 0x0a, 0x09, 0x49, 0x6e, 0x67, 0x72, 0x65, 0x73, 0x73, 0x65,
	0x73, 0x22, 0xea, 0x03, 0x0a, 0x0e, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x12, 0x34, 0x0a, 0x04, 0x70, 0x6f, 0x64, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x20, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72,
	0x64, 0x73, 0x2e, 0x6b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x50, 0x6f,
	0x64, 0x73, 0x52, 0x04, 0x70, 0x6f, 0x64, 0x73, 0x12, 0x43, 0x0a, 0x09, 0x65, 0x6e, 0x64, 0x70,
	0x6f, 0x69, 0x6e, 0x74, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x6b, 0x75,
	0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x45, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e,
	0x74, 0x73, 0x52, 0x09, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x12, 0x40, 0x0a,
	0x08, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x24, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64,
	0x73, 0x2e, 0x6b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x2e, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x73, 0x52, 0x08, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x12,
	0x43, 0x0a, 0x09, 0x69, 0x6e, 0x67, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x25, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x72, 0x64, 0x73, 0x2e, 0x6b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x2e,
	0x49, 0x6e, 0x67, 0x72, 0x65, 0x73, 0x73, 0x65, 0x73, 0x52, 0x09, 0x69, 0x6e, 0x67, 0x72, 0x65,
	0x73, 0x73, 0x65, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x73, 0x65,
	0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x14, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0d, 0x6c, 0x61,
	0x62, 0x65, 0x6c, 0x53, 0x65, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x2c, 0x0a, 0x12, 0x61,
	0x70, 0x69, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x18, 0x5b, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x61, 0x70, 0x69, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x3f, 0x0a, 0x0a, 0x74, 0x6c, 0x73,
	0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x5d, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x6c, 0x73, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x54, 0x4c, 0x53, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52,
	0x09, 0x74, 0x6c, 0x73, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x22, 0x0a, 0x0b, 0x72, 0x65,
	0x5f, 0x65, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x63, 0x20, 0x01, 0x28, 0x05, 0x3a,
	0x02, 0x36, 0x30, 0x52, 0x09, 0x72, 0x65, 0x45, 0x76, 0x61, 0x6c, 0x53, 0x65, 0x63, 0x42, 0x42,
	0x5a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x64,
	0x73, 0x2f, 0x6b, 0x75, 0x62, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f,
})

var (
	file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_goTypes = []any{
	(*Pods)(nil),            // 0: cloudprober.rds.kubernetes.Pods
	(*Endpoints)(nil),       // 1: cloudprober.rds.kubernetes.Endpoints
	(*Services)(nil),        // 2: cloudprober.rds.kubernetes.Services
	(*Ingresses)(nil),       // 3: cloudprober.rds.kubernetes.Ingresses
	(*ProviderConfig)(nil),  // 4: cloudprober.rds.kubernetes.ProviderConfig
	(*proto.TLSConfig)(nil), // 5: cloudprober.tlsconfig.TLSConfig
}
var file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.rds.kubernetes.ProviderConfig.pods:type_name -> cloudprober.rds.kubernetes.Pods
	1, // 1: cloudprober.rds.kubernetes.ProviderConfig.endpoints:type_name -> cloudprober.rds.kubernetes.Endpoints
	2, // 2: cloudprober.rds.kubernetes.ProviderConfig.services:type_name -> cloudprober.rds.kubernetes.Services
	3, // 3: cloudprober.rds.kubernetes.ProviderConfig.ingresses:type_name -> cloudprober.rds.kubernetes.Ingresses
	5, // 4: cloudprober.rds.kubernetes.ProviderConfig.tls_config:type_name -> cloudprober.tlsconfig.TLSConfig
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_init()
}
func file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_internal_rds_kubernetes_proto_config_proto_depIdxs = nil
}
