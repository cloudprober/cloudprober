// Copyright 2017-2019 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/internal/servers/http/proto/config.proto

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

// tls_cert_file and tls_key_file field should be set for HTTPS.
type ServerConf_ProtocolType int32

const (
	ServerConf_HTTP  ServerConf_ProtocolType = 0
	ServerConf_HTTPS ServerConf_ProtocolType = 1
)

// Enum value maps for ServerConf_ProtocolType.
var (
	ServerConf_ProtocolType_name = map[int32]string{
		0: "HTTP",
		1: "HTTPS",
	}
	ServerConf_ProtocolType_value = map[string]int32{
		"HTTP":  0,
		"HTTPS": 1,
	}
)

func (x ServerConf_ProtocolType) Enum() *ServerConf_ProtocolType {
	p := new(ServerConf_ProtocolType)
	*p = x
	return p
}

func (x ServerConf_ProtocolType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ServerConf_ProtocolType) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_enumTypes[0].Descriptor()
}

func (ServerConf_ProtocolType) Type() protoreflect.EnumType {
	return &file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_enumTypes[0]
}

func (x ServerConf_ProtocolType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ServerConf_ProtocolType) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ServerConf_ProtocolType(num)
	return nil
}

// Deprecated: Use ServerConf_ProtocolType.Descriptor instead.
func (ServerConf_ProtocolType) EnumDescriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescGZIP(), []int{0, 0}
}

// Next available tag = 10
type ServerConf struct {
	state    protoimpl.MessageState   `protogen:"open.v1"`
	Port     *int32                   `protobuf:"varint,1,opt,name=port,def=3141" json:"port,omitempty"`
	Protocol *ServerConf_ProtocolType `protobuf:"varint,6,opt,name=protocol,enum=cloudprober.servers.http.ServerConf_ProtocolType,def=0" json:"protocol,omitempty"`
	// Maximum duration for reading the entire request, including the body.
	ReadTimeoutMs *int32 `protobuf:"varint,2,opt,name=read_timeout_ms,json=readTimeoutMs,def=10000" json:"read_timeout_ms,omitempty"` // default: 10s
	// Maximum duration before timing out writes of the response.
	WriteTimeoutMs *int32 `protobuf:"varint,3,opt,name=write_timeout_ms,json=writeTimeoutMs,def=10000" json:"write_timeout_ms,omitempty"` // default: 10s
	// Maximum amount of time to wait for the next request when keep-alives are
	// enabled.
	IdleTimeoutMs *int32 `protobuf:"varint,4,opt,name=idle_timeout_ms,json=idleTimeoutMs,def=60000" json:"idle_timeout_ms,omitempty"` // default: 1m
	// Certificate file to use for HTTPS servers.
	TlsCertFile *string `protobuf:"bytes,7,opt,name=tls_cert_file,json=tlsCertFile" json:"tls_cert_file,omitempty"`
	// Private key file corresponding to the certificate above.
	TlsKeyFile *string `protobuf:"bytes,8,opt,name=tls_key_file,json=tlsKeyFile" json:"tls_key_file,omitempty"`
	// Disable HTTP/2 for HTTPS servers.
	DisableHttp2 *bool `protobuf:"varint,9,opt,name=disable_http2,json=disableHttp2" json:"disable_http2,omitempty"`
	// Pattern data handler returns pattern data at the url /data_<size_in_bytes>,
	// e.g. "/data_2048".
	PatternDataHandler []*ServerConf_PatternDataHandler `protobuf:"bytes,5,rep,name=pattern_data_handler,json=patternDataHandler" json:"pattern_data_handler,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

// Default values for ServerConf fields.
const (
	Default_ServerConf_Port           = int32(3141)
	Default_ServerConf_Protocol       = ServerConf_HTTP
	Default_ServerConf_ReadTimeoutMs  = int32(10000)
	Default_ServerConf_WriteTimeoutMs = int32(10000)
	Default_ServerConf_IdleTimeoutMs  = int32(60000)
)

func (x *ServerConf) Reset() {
	*x = ServerConf{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConf) ProtoMessage() {}

func (x *ServerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerConf.ProtoReflect.Descriptor instead.
func (*ServerConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *ServerConf) GetPort() int32 {
	if x != nil && x.Port != nil {
		return *x.Port
	}
	return Default_ServerConf_Port
}

func (x *ServerConf) GetProtocol() ServerConf_ProtocolType {
	if x != nil && x.Protocol != nil {
		return *x.Protocol
	}
	return Default_ServerConf_Protocol
}

func (x *ServerConf) GetReadTimeoutMs() int32 {
	if x != nil && x.ReadTimeoutMs != nil {
		return *x.ReadTimeoutMs
	}
	return Default_ServerConf_ReadTimeoutMs
}

func (x *ServerConf) GetWriteTimeoutMs() int32 {
	if x != nil && x.WriteTimeoutMs != nil {
		return *x.WriteTimeoutMs
	}
	return Default_ServerConf_WriteTimeoutMs
}

func (x *ServerConf) GetIdleTimeoutMs() int32 {
	if x != nil && x.IdleTimeoutMs != nil {
		return *x.IdleTimeoutMs
	}
	return Default_ServerConf_IdleTimeoutMs
}

func (x *ServerConf) GetTlsCertFile() string {
	if x != nil && x.TlsCertFile != nil {
		return *x.TlsCertFile
	}
	return ""
}

func (x *ServerConf) GetTlsKeyFile() string {
	if x != nil && x.TlsKeyFile != nil {
		return *x.TlsKeyFile
	}
	return ""
}

func (x *ServerConf) GetDisableHttp2() bool {
	if x != nil && x.DisableHttp2 != nil {
		return *x.DisableHttp2
	}
	return false
}

func (x *ServerConf) GetPatternDataHandler() []*ServerConf_PatternDataHandler {
	if x != nil {
		return x.PatternDataHandler
	}
	return nil
}

type ServerConf_PatternDataHandler struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Response sizes to server, e.g. 1024.
	ResponseSize *int32 `protobuf:"varint,1,req,name=response_size,json=responseSize" json:"response_size,omitempty"`
	// Pattern is repeated to build the response, with "response_size mod
	// pattern_size" filled by '0' bytes.
	Pattern       *string `protobuf:"bytes,2,opt,name=pattern,def=cloudprober" json:"pattern,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

// Default values for ServerConf_PatternDataHandler fields.
const (
	Default_ServerConf_PatternDataHandler_Pattern = string("cloudprober")
)

func (x *ServerConf_PatternDataHandler) Reset() {
	*x = ServerConf_PatternDataHandler{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerConf_PatternDataHandler) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConf_PatternDataHandler) ProtoMessage() {}

func (x *ServerConf_PatternDataHandler) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerConf_PatternDataHandler.ProtoReflect.Descriptor instead.
func (*ServerConf_PatternDataHandler) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescGZIP(), []int{0, 0}
}

func (x *ServerConf_PatternDataHandler) GetResponseSize() int32 {
	if x != nil && x.ResponseSize != nil {
		return *x.ResponseSize
	}
	return 0
}

func (x *ServerConf_PatternDataHandler) GetPattern() string {
	if x != nil && x.Pattern != nil {
		return *x.Pattern
	}
	return Default_ServerConf_PatternDataHandler_Pattern
}

var File_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDesc = string([]byte{
	0x0a, 0x4b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x18, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x22, 0xe7, 0x04, 0x0a, 0x0a, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x18, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x3a, 0x04, 0x33, 0x31, 0x34, 0x31, 0x52, 0x04, 0x70, 0x6f, 0x72, 0x74,
	0x12, 0x53, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x31, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x53, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f,
	0x6c, 0x54, 0x79, 0x70, 0x65, 0x3a, 0x04, 0x48, 0x54, 0x54, 0x50, 0x52, 0x08, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x12, 0x2d, 0x0a, 0x0f, 0x72, 0x65, 0x61, 0x64, 0x5f, 0x74, 0x69,
	0x6d, 0x65, 0x6f, 0x75, 0x74, 0x5f, 0x6d, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x05,
	0x31, 0x30, 0x30, 0x30, 0x30, 0x52, 0x0d, 0x72, 0x65, 0x61, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x6f,
	0x75, 0x74, 0x4d, 0x73, 0x12, 0x2f, 0x0a, 0x10, 0x77, 0x72, 0x69, 0x74, 0x65, 0x5f, 0x74, 0x69,
	0x6d, 0x65, 0x6f, 0x75, 0x74, 0x5f, 0x6d, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x05,
	0x31, 0x30, 0x30, 0x30, 0x30, 0x52, 0x0e, 0x77, 0x72, 0x69, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65,
	0x6f, 0x75, 0x74, 0x4d, 0x73, 0x12, 0x2d, 0x0a, 0x0f, 0x69, 0x64, 0x6c, 0x65, 0x5f, 0x74, 0x69,
	0x6d, 0x65, 0x6f, 0x75, 0x74, 0x5f, 0x6d, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x05,
	0x36, 0x30, 0x30, 0x30, 0x30, 0x52, 0x0d, 0x69, 0x64, 0x6c, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x6f,
	0x75, 0x74, 0x4d, 0x73, 0x12, 0x22, 0x0a, 0x0d, 0x74, 0x6c, 0x73, 0x5f, 0x63, 0x65, 0x72, 0x74,
	0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x74, 0x6c, 0x73,
	0x43, 0x65, 0x72, 0x74, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x20, 0x0a, 0x0c, 0x74, 0x6c, 0x73, 0x5f,
	0x6b, 0x65, 0x79, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a,
	0x74, 0x6c, 0x73, 0x4b, 0x65, 0x79, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x64, 0x69,
	0x73, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x68, 0x74, 0x74, 0x70, 0x32, 0x18, 0x09, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x0c, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x48, 0x74, 0x74, 0x70, 0x32, 0x12,
	0x69, 0x0a, 0x14, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x5f,
	0x68, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x37, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x73, 0x2e, 0x68, 0x74, 0x74, 0x70, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x43,
	0x6f, 0x6e, 0x66, 0x2e, 0x50, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x44, 0x61, 0x74, 0x61, 0x48,
	0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x52, 0x12, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x44,
	0x61, 0x74, 0x61, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72, 0x1a, 0x60, 0x0a, 0x12, 0x50, 0x61,
	0x74, 0x74, 0x65, 0x72, 0x6e, 0x44, 0x61, 0x74, 0x61, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x72,
	0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x5f, 0x73, 0x69, 0x7a,
	0x65, 0x18, 0x01, 0x20, 0x02, 0x28, 0x05, 0x52, 0x0c, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x25, 0x0a, 0x07, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x0b, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x52, 0x07, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x22, 0x23, 0x0a, 0x0c,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x54, 0x79, 0x70, 0x65, 0x12, 0x08, 0x0a, 0x04,
	0x48, 0x54, 0x54, 0x50, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x48, 0x54, 0x54, 0x50, 0x53, 0x10,
	0x01, 0x42, 0x40, 0x5a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c,
	0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x68, 0x74, 0x74, 0x70, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f,
})

var (
	file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_goTypes = []any{
	(ServerConf_ProtocolType)(0),          // 0: cloudprober.servers.http.ServerConf.ProtocolType
	(*ServerConf)(nil),                    // 1: cloudprober.servers.http.ServerConf
	(*ServerConf_PatternDataHandler)(nil), // 2: cloudprober.servers.http.ServerConf.PatternDataHandler
}
var file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.servers.http.ServerConf.protocol:type_name -> cloudprober.servers.http.ServerConf.ProtocolType
	2, // 1: cloudprober.servers.http.ServerConf.pattern_data_handler:type_name -> cloudprober.servers.http.ServerConf.PatternDataHandler
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_depIdxs,
		EnumInfos:         file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_enumTypes,
		MessageInfos:      file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_internal_servers_http_proto_config_proto_depIdxs = nil
}
