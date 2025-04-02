// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/internal/rds/proto/rds.proto

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

type IPConfig_IPType int32

const (
	// Default IP of the resource.
	//   - Private IP for instance resource
	//   - Forwarding rule IP for forwarding rule.
	IPConfig_DEFAULT IPConfig_IPType = 0
	// Instance's external IP.
	IPConfig_PUBLIC IPConfig_IPType = 1
	// First IP address from the first Alias IP range. For example, for
	// alias IP range "192.168.12.0/24", 192.168.12.0 will be returned.
	// Supported only on GCE.
	IPConfig_ALIAS IPConfig_IPType = 2
)

// Enum value maps for IPConfig_IPType.
var (
	IPConfig_IPType_name = map[int32]string{
		0: "DEFAULT",
		1: "PUBLIC",
		2: "ALIAS",
	}
	IPConfig_IPType_value = map[string]int32{
		"DEFAULT": 0,
		"PUBLIC":  1,
		"ALIAS":   2,
	}
)

func (x IPConfig_IPType) Enum() *IPConfig_IPType {
	p := new(IPConfig_IPType)
	*p = x
	return p
}

func (x IPConfig_IPType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (IPConfig_IPType) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_enumTypes[0].Descriptor()
}

func (IPConfig_IPType) Type() protoreflect.EnumType {
	return &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_enumTypes[0]
}

func (x IPConfig_IPType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *IPConfig_IPType) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = IPConfig_IPType(num)
	return nil
}

// Deprecated: Use IPConfig_IPType.Descriptor instead.
func (IPConfig_IPType) EnumDescriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescGZIP(), []int{2, 0}
}

type IPConfig_IPVersion int32

const (
	IPConfig_IP_VERSION_UNSPECIFIED IPConfig_IPVersion = 0
	IPConfig_IPV4                   IPConfig_IPVersion = 1
	IPConfig_IPV6                   IPConfig_IPVersion = 2
)

// Enum value maps for IPConfig_IPVersion.
var (
	IPConfig_IPVersion_name = map[int32]string{
		0: "IP_VERSION_UNSPECIFIED",
		1: "IPV4",
		2: "IPV6",
	}
	IPConfig_IPVersion_value = map[string]int32{
		"IP_VERSION_UNSPECIFIED": 0,
		"IPV4":                   1,
		"IPV6":                   2,
	}
)

func (x IPConfig_IPVersion) Enum() *IPConfig_IPVersion {
	p := new(IPConfig_IPVersion)
	*p = x
	return p
}

func (x IPConfig_IPVersion) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (IPConfig_IPVersion) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_enumTypes[1].Descriptor()
}

func (IPConfig_IPVersion) Type() protoreflect.EnumType {
	return &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_enumTypes[1]
}

func (x IPConfig_IPVersion) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *IPConfig_IPVersion) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = IPConfig_IPVersion(num)
	return nil
}

// Deprecated: Use IPConfig_IPVersion.Descriptor instead.
func (IPConfig_IPVersion) EnumDescriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescGZIP(), []int{2, 1}
}

type ListResourcesRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Provider is the resource list provider, for example: "gcp", "aws", etc.
	Provider *string `protobuf:"bytes,1,req,name=provider" json:"provider,omitempty"`
	// Provider specific resource path. For example: for GCP, it could be
	// "gce_instances/<project>", "regional_forwarding_rules/<project>", etc.
	ResourcePath *string `protobuf:"bytes,2,opt,name=resource_path,json=resourcePath" json:"resource_path,omitempty"`
	// Filters for the resources list. Filters are ANDed: all filters should
	// succeed for an item to included in the result list.
	Filter []*Filter `protobuf:"bytes,3,rep,name=filter" json:"filter,omitempty"`
	// Optional. If resource has an IP (and a NIC) address, following
	// fields determine which IP address will be included in the results.
	IpConfig *IPConfig `protobuf:"bytes,4,opt,name=ip_config,json=ipConfig" json:"ip_config,omitempty"`
	// If specified, and if provider supports it, server will send resources in
	// the response only if they have changed since the given timestamp. Since
	// there may be no resources in the response for non-caching reasons as well,
	// clients should use the "last_modified" field in the response to determine
	// if they need to update the local cache or not.
	IfModifiedSince *int64 `protobuf:"varint,5,opt,name=if_modified_since,json=ifModifiedSince" json:"if_modified_since,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *ListResourcesRequest) Reset() {
	*x = ListResourcesRequest{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListResourcesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListResourcesRequest) ProtoMessage() {}

func (x *ListResourcesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListResourcesRequest.ProtoReflect.Descriptor instead.
func (*ListResourcesRequest) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescGZIP(), []int{0}
}

func (x *ListResourcesRequest) GetProvider() string {
	if x != nil && x.Provider != nil {
		return *x.Provider
	}
	return ""
}

func (x *ListResourcesRequest) GetResourcePath() string {
	if x != nil && x.ResourcePath != nil {
		return *x.ResourcePath
	}
	return ""
}

func (x *ListResourcesRequest) GetFilter() []*Filter {
	if x != nil {
		return x.Filter
	}
	return nil
}

func (x *ListResourcesRequest) GetIpConfig() *IPConfig {
	if x != nil {
		return x.IpConfig
	}
	return nil
}

func (x *ListResourcesRequest) GetIfModifiedSince() int64 {
	if x != nil && x.IfModifiedSince != nil {
		return *x.IfModifiedSince
	}
	return 0
}

type Filter struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           *string                `protobuf:"bytes,1,req,name=key" json:"key,omitempty"`
	Value         *string                `protobuf:"bytes,2,req,name=value" json:"value,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Filter) Reset() {
	*x = Filter{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Filter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Filter) ProtoMessage() {}

func (x *Filter) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Filter.ProtoReflect.Descriptor instead.
func (*Filter) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescGZIP(), []int{1}
}

func (x *Filter) GetKey() string {
	if x != nil && x.Key != nil {
		return *x.Key
	}
	return ""
}

func (x *Filter) GetValue() string {
	if x != nil && x.Value != nil {
		return *x.Value
	}
	return ""
}

type IPConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// NIC index
	NicIndex      *int32              `protobuf:"varint,1,opt,name=nic_index,json=nicIndex,def=0" json:"nic_index,omitempty"`
	IpType        *IPConfig_IPType    `protobuf:"varint,3,opt,name=ip_type,json=ipType,enum=cloudprober.rds.IPConfig_IPType" json:"ip_type,omitempty"`
	IpVersion     *IPConfig_IPVersion `protobuf:"varint,2,opt,name=ip_version,json=ipVersion,enum=cloudprober.rds.IPConfig_IPVersion" json:"ip_version,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for IPConfig fields.
const (
	Default_IPConfig_NicIndex = int32(0)
)

func (x *IPConfig) Reset() {
	*x = IPConfig{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *IPConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IPConfig) ProtoMessage() {}

func (x *IPConfig) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IPConfig.ProtoReflect.Descriptor instead.
func (*IPConfig) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescGZIP(), []int{2}
}

func (x *IPConfig) GetNicIndex() int32 {
	if x != nil && x.NicIndex != nil {
		return *x.NicIndex
	}
	return Default_IPConfig_NicIndex
}

func (x *IPConfig) GetIpType() IPConfig_IPType {
	if x != nil && x.IpType != nil {
		return *x.IpType
	}
	return IPConfig_DEFAULT
}

func (x *IPConfig) GetIpVersion() IPConfig_IPVersion {
	if x != nil && x.IpVersion != nil {
		return *x.IpVersion
	}
	return IPConfig_IP_VERSION_UNSPECIFIED
}

type Resource struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Resource name.
	Name *string `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	// Resource's IP address, selected based on the request's ip_config.
	Ip *string `protobuf:"bytes,2,opt,name=ip" json:"ip,omitempty"`
	// Resource's port, if any.
	Port *int32 `protobuf:"varint,5,opt,name=port" json:"port,omitempty"`
	// Resource's labels, if any.
	Labels map[string]string `protobuf:"bytes,6,rep,name=labels" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// Last updated (in unix epoch).
	LastUpdated *int64 `protobuf:"varint,7,opt,name=last_updated,json=lastUpdated" json:"last_updated,omitempty"`
	// Id associated with the resource, if any.
	Id *string `protobuf:"bytes,3,opt,name=id" json:"id,omitempty"`
	// Optional info associated with the resource. Some resource type may make use
	// of it.
	Info          []byte `protobuf:"bytes,4,opt,name=info" json:"info,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Resource) Reset() {
	*x = Resource{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Resource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Resource) ProtoMessage() {}

func (x *Resource) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Resource.ProtoReflect.Descriptor instead.
func (*Resource) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescGZIP(), []int{3}
}

func (x *Resource) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *Resource) GetIp() string {
	if x != nil && x.Ip != nil {
		return *x.Ip
	}
	return ""
}

func (x *Resource) GetPort() int32 {
	if x != nil && x.Port != nil {
		return *x.Port
	}
	return 0
}

func (x *Resource) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

func (x *Resource) GetLastUpdated() int64 {
	if x != nil && x.LastUpdated != nil {
		return *x.LastUpdated
	}
	return 0
}

func (x *Resource) GetId() string {
	if x != nil && x.Id != nil {
		return *x.Id
	}
	return ""
}

func (x *Resource) GetInfo() []byte {
	if x != nil {
		return x.Info
	}
	return nil
}

type ListResourcesResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// There may not be any resources in the response if request contains the
	// "if_modified_since" field and provider "knows" that nothing has changed since
	// the if_modified_since timestamp.
	Resources []*Resource `protobuf:"bytes,1,rep,name=resources" json:"resources,omitempty"`
	// When were resources last modified. This field will always be set if
	// provider has a way of figuring out last_modified timestamp for its
	// resources.
	LastModified  *int64 `protobuf:"varint,2,opt,name=last_modified,json=lastModified" json:"last_modified,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListResourcesResponse) Reset() {
	*x = ListResourcesResponse{}
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListResourcesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListResourcesResponse) ProtoMessage() {}

func (x *ListResourcesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListResourcesResponse.ProtoReflect.Descriptor instead.
func (*ListResourcesResponse) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescGZIP(), []int{4}
}

func (x *ListResourcesResponse) GetResources() []*Resource {
	if x != nil {
		return x.Resources
	}
	return nil
}

func (x *ListResourcesResponse) GetLastModified() int64 {
	if x != nil && x.LastModified != nil {
		return *x.LastModified
	}
	return 0
}

var File_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto protoreflect.FileDescriptor

const file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDesc = "" +
	"\n" +
	"?github.com/cloudprober/cloudprober/internal/rds/proto/rds.proto\x12\x0fcloudprober.rds\"\xec\x01\n" +
	"\x14ListResourcesRequest\x12\x1a\n" +
	"\bprovider\x18\x01 \x02(\tR\bprovider\x12#\n" +
	"\rresource_path\x18\x02 \x01(\tR\fresourcePath\x12/\n" +
	"\x06filter\x18\x03 \x03(\v2\x17.cloudprober.rds.FilterR\x06filter\x126\n" +
	"\tip_config\x18\x04 \x01(\v2\x19.cloudprober.rds.IPConfigR\bipConfig\x12*\n" +
	"\x11if_modified_since\x18\x05 \x01(\x03R\x0fifModifiedSince\"0\n" +
	"\x06Filter\x12\x10\n" +
	"\x03key\x18\x01 \x02(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x02(\tR\x05value\"\x94\x02\n" +
	"\bIPConfig\x12\x1e\n" +
	"\tnic_index\x18\x01 \x01(\x05:\x010R\bnicIndex\x129\n" +
	"\aip_type\x18\x03 \x01(\x0e2 .cloudprober.rds.IPConfig.IPTypeR\x06ipType\x12B\n" +
	"\n" +
	"ip_version\x18\x02 \x01(\x0e2#.cloudprober.rds.IPConfig.IPVersionR\tipVersion\",\n" +
	"\x06IPType\x12\v\n" +
	"\aDEFAULT\x10\x00\x12\n" +
	"\n" +
	"\x06PUBLIC\x10\x01\x12\t\n" +
	"\x05ALIAS\x10\x02\";\n" +
	"\tIPVersion\x12\x1a\n" +
	"\x16IP_VERSION_UNSPECIFIED\x10\x00\x12\b\n" +
	"\x04IPV4\x10\x01\x12\b\n" +
	"\x04IPV6\x10\x02\"\x83\x02\n" +
	"\bResource\x12\x12\n" +
	"\x04name\x18\x01 \x02(\tR\x04name\x12\x0e\n" +
	"\x02ip\x18\x02 \x01(\tR\x02ip\x12\x12\n" +
	"\x04port\x18\x05 \x01(\x05R\x04port\x12=\n" +
	"\x06labels\x18\x06 \x03(\v2%.cloudprober.rds.Resource.LabelsEntryR\x06labels\x12!\n" +
	"\flast_updated\x18\a \x01(\x03R\vlastUpdated\x12\x0e\n" +
	"\x02id\x18\x03 \x01(\tR\x02id\x12\x12\n" +
	"\x04info\x18\x04 \x01(\fR\x04info\x1a9\n" +
	"\vLabelsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"u\n" +
	"\x15ListResourcesResponse\x127\n" +
	"\tresources\x18\x01 \x03(\v2\x19.cloudprober.rds.ResourceR\tresources\x12#\n" +
	"\rlast_modified\x18\x02 \x01(\x03R\flastModified2u\n" +
	"\x11ResourceDiscovery\x12`\n" +
	"\rListResources\x12%.cloudprober.rds.ListResourcesRequest\x1a&.cloudprober.rds.ListResourcesResponse\"\x00B7Z5github.com/cloudprober/cloudprober/internal/rds/proto"

var (
	file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_goTypes = []any{
	(IPConfig_IPType)(0),          // 0: cloudprober.rds.IPConfig.IPType
	(IPConfig_IPVersion)(0),       // 1: cloudprober.rds.IPConfig.IPVersion
	(*ListResourcesRequest)(nil),  // 2: cloudprober.rds.ListResourcesRequest
	(*Filter)(nil),                // 3: cloudprober.rds.Filter
	(*IPConfig)(nil),              // 4: cloudprober.rds.IPConfig
	(*Resource)(nil),              // 5: cloudprober.rds.Resource
	(*ListResourcesResponse)(nil), // 6: cloudprober.rds.ListResourcesResponse
	nil,                           // 7: cloudprober.rds.Resource.LabelsEntry
}
var file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_depIdxs = []int32{
	3, // 0: cloudprober.rds.ListResourcesRequest.filter:type_name -> cloudprober.rds.Filter
	4, // 1: cloudprober.rds.ListResourcesRequest.ip_config:type_name -> cloudprober.rds.IPConfig
	0, // 2: cloudprober.rds.IPConfig.ip_type:type_name -> cloudprober.rds.IPConfig.IPType
	1, // 3: cloudprober.rds.IPConfig.ip_version:type_name -> cloudprober.rds.IPConfig.IPVersion
	7, // 4: cloudprober.rds.Resource.labels:type_name -> cloudprober.rds.Resource.LabelsEntry
	5, // 5: cloudprober.rds.ListResourcesResponse.resources:type_name -> cloudprober.rds.Resource
	2, // 6: cloudprober.rds.ResourceDiscovery.ListResources:input_type -> cloudprober.rds.ListResourcesRequest
	6, // 7: cloudprober.rds.ResourceDiscovery.ListResources:output_type -> cloudprober.rds.ListResourcesResponse
	7, // [7:8] is the sub-list for method output_type
	6, // [6:7] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_init() }
func file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_init() {
	if File_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_rawDesc)),
			NumEnums:      2,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_depIdxs,
		EnumInfos:         file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_enumTypes,
		MessageInfos:      file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto = out.File
	file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_internal_rds_proto_rds_proto_depIdxs = nil
}
