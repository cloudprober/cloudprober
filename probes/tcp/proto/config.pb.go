// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/probes/tcp/proto/config.proto

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

// Next tag: 4
type ProbeConf struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Port for TCP requests. If not specfied, and port is provided by the
	// targets (e.g. kubernetes endpoint or service), that port is used.
	Port *int32 `protobuf:"varint,1,opt,name=port" json:"port,omitempty"`
	// Whether to resolve the target before making the request. If set to false,
	// we hand over the target golang's net.Dial module, Otherwise, we resolve
	// the target first to an IP address and make a request using that. By
	// default we resolve first if it's a discovered resource, e.g., a k8s
	// endpoint.
	ResolveFirst *bool `protobuf:"varint,2,opt,name=resolve_first,json=resolveFirst" json:"resolve_first,omitempty"`
	// Interval between targets.
	IntervalBetweenTargetsMsec *int32 `protobuf:"varint,3,opt,name=interval_between_targets_msec,json=intervalBetweenTargetsMsec,def=10" json:"interval_between_targets_msec,omitempty"`
	unknownFields              protoimpl.UnknownFields
	sizeCache                  protoimpl.SizeCache
}

// Default values for ProbeConf fields.
const (
	Default_ProbeConf_IntervalBetweenTargetsMsec = int32(10)
)

func (x *ProbeConf) Reset() {
	*x = ProbeConf{}
	mi := &file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ProbeConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProbeConf) ProtoMessage() {}

func (x *ProbeConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_msgTypes[0]
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
	return file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *ProbeConf) GetPort() int32 {
	if x != nil && x.Port != nil {
		return *x.Port
	}
	return 0
}

func (x *ProbeConf) GetResolveFirst() bool {
	if x != nil && x.ResolveFirst != nil {
		return *x.ResolveFirst
	}
	return false
}

func (x *ProbeConf) GetIntervalBetweenTargetsMsec() int32 {
	if x != nil && x.IntervalBetweenTargetsMsec != nil {
		return *x.IntervalBetweenTargetsMsec
	}
	return Default_ProbeConf_IntervalBetweenTargetsMsec
}

var File_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto protoreflect.FileDescriptor

const file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDesc = "" +
	"\n" +
	"@github.com/cloudprober/cloudprober/probes/tcp/proto/config.proto\x12\x16cloudprober.probes.tcp\"\x8b\x01\n" +
	"\tProbeConf\x12\x12\n" +
	"\x04port\x18\x01 \x01(\x05R\x04port\x12#\n" +
	"\rresolve_first\x18\x02 \x01(\bR\fresolveFirst\x12E\n" +
	"\x1dinterval_between_targets_msec\x18\x03 \x01(\x05:\x0210R\x1aintervalBetweenTargetsMsecB5Z3github.com/cloudprober/cloudprober/probes/tcp/proto"

var (
	file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_goTypes = []any{
	(*ProbeConf)(nil), // 0: cloudprober.probes.tcp.ProbeConf
}
var file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_probes_tcp_proto_config_proto_depIdxs = nil
}
