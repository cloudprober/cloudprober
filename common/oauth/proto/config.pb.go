// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/common/oauth/proto/config.proto

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

type Config struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Type:
	//	*Config_BearerToken
	//	*Config_GoogleCredentials
	//	*Config_Jwt
	Type isConfig_Type `protobuf_oneof:"type"`
}

func (x *Config) Reset() {
	*x = Config{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config.ProtoReflect.Descriptor instead.
func (*Config) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescGZIP(), []int{0}
}

func (m *Config) GetType() isConfig_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *Config) GetBearerToken() *BearerToken {
	if x, ok := x.GetType().(*Config_BearerToken); ok {
		return x.BearerToken
	}
	return nil
}

func (x *Config) GetGoogleCredentials() *GoogleCredentials {
	if x, ok := x.GetType().(*Config_GoogleCredentials); ok {
		return x.GoogleCredentials
	}
	return nil
}

func (x *Config) GetJwt() *JWT {
	if x, ok := x.GetType().(*Config_Jwt); ok {
		return x.Jwt
	}
	return nil
}

type isConfig_Type interface {
	isConfig_Type()
}

type Config_BearerToken struct {
	BearerToken *BearerToken `protobuf:"bytes,1,opt,name=bearer_token,json=bearerToken,proto3,oneof"`
}

type Config_GoogleCredentials struct {
	GoogleCredentials *GoogleCredentials `protobuf:"bytes,2,opt,name=google_credentials,json=googleCredentials,proto3,oneof"`
}

type Config_Jwt struct {
	Jwt *JWT `protobuf:"bytes,3,opt,name=jwt,proto3,oneof"`
}

func (*Config_BearerToken) isConfig_Type() {}

func (*Config_GoogleCredentials) isConfig_Type() {}

func (*Config_Jwt) isConfig_Type() {}

type JWT struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TokenUrl string `protobuf:"bytes,1,opt,name=token_url,json=tokenUrl,proto3" json:"token_url,omitempty"`
	Method   string `protobuf:"bytes,2,opt,name=method,proto3" json:"method,omitempty"`
	// data can be repeated. If it is repeated we combine the values with a '&'
	// in between. If data appears to be a valid json, we automatically add the
	// content-type header: "Content-Type: application/json" (can be overridden).
	Data []string `protobuf:"bytes,3,rep,name=data,proto3" json:"data,omitempty"`
	// HTTP request headers
	Header []*JWT_Header `protobuf:"bytes,8,rep,name=header,proto3" json:"header,omitempty"`
}

func (x *JWT) Reset() {
	*x = JWT{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JWT) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JWT) ProtoMessage() {}

func (x *JWT) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JWT.ProtoReflect.Descriptor instead.
func (*JWT) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *JWT) GetTokenUrl() string {
	if x != nil {
		return x.TokenUrl
	}
	return ""
}

func (x *JWT) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *JWT) GetData() []string {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *JWT) GetHeader() []*JWT_Header {
	if x != nil {
		return x.Header
	}
	return nil
}

// Bearer token is added to the HTTP request through an HTTP header:
// "Authorization: Bearer <access_token>"
type BearerToken struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Source:
	//	*BearerToken_File
	//	*BearerToken_Cmd
	//	*BearerToken_GceServiceAccount
	Source isBearerToken_Source `protobuf_oneof:"source"`
	// How often to refresh token. As OAuth token usually expire, we need to
	// refresh them on a regular interval. If set to 0, caching is disabled.
	// Default is 60s.
	RefreshIntervalSec *float32 `protobuf:"fixed32,90,opt,name=refresh_interval_sec,json=refreshIntervalSec,proto3,oneof" json:"refresh_interval_sec,omitempty"`
}

func (x *BearerToken) Reset() {
	*x = BearerToken{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BearerToken) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BearerToken) ProtoMessage() {}

func (x *BearerToken) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BearerToken.ProtoReflect.Descriptor instead.
func (*BearerToken) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescGZIP(), []int{2}
}

func (m *BearerToken) GetSource() isBearerToken_Source {
	if m != nil {
		return m.Source
	}
	return nil
}

func (x *BearerToken) GetFile() string {
	if x, ok := x.GetSource().(*BearerToken_File); ok {
		return x.File
	}
	return ""
}

func (x *BearerToken) GetCmd() string {
	if x, ok := x.GetSource().(*BearerToken_Cmd); ok {
		return x.Cmd
	}
	return ""
}

func (x *BearerToken) GetGceServiceAccount() string {
	if x, ok := x.GetSource().(*BearerToken_GceServiceAccount); ok {
		return x.GceServiceAccount
	}
	return ""
}

func (x *BearerToken) GetRefreshIntervalSec() float32 {
	if x != nil && x.RefreshIntervalSec != nil {
		return *x.RefreshIntervalSec
	}
	return 0
}

type isBearerToken_Source interface {
	isBearerToken_Source()
}

type BearerToken_File struct {
	// Path to token file.
	File string `protobuf:"bytes,1,opt,name=file,proto3,oneof"`
}

type BearerToken_Cmd struct {
	// Run a comand to obtain the token, e.g.
	// cat /var/lib/myapp/token, or
	// /var/lib/run/get_token.sh
	Cmd string `protobuf:"bytes,2,opt,name=cmd,proto3,oneof"`
}

type BearerToken_GceServiceAccount struct {
	// GCE metadata token
	GceServiceAccount string `protobuf:"bytes,3,opt,name=gce_service_account,json=gceServiceAccount,proto3,oneof"`
}

func (*BearerToken_File) isBearerToken_Source() {}

func (*BearerToken_Cmd) isBearerToken_Source() {}

func (*BearerToken_GceServiceAccount) isBearerToken_Source() {}

// Google credentials in JSON format. We simply use oauth2/google package to
// use these credentials.
type GoogleCredentials struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	JsonFile string   `protobuf:"bytes,1,opt,name=json_file,json=jsonFile,proto3" json:"json_file,omitempty"`
	Scope    []string `protobuf:"bytes,2,rep,name=scope,proto3" json:"scope,omitempty"`
	// Use encoded JWT directly as access token, instead of implementing the whole
	// OAuth2.0 flow.
	JwtAsAccessToken bool `protobuf:"varint,4,opt,name=jwt_as_access_token,json=jwtAsAccessToken,proto3" json:"jwt_as_access_token,omitempty"`
	// Audience works only if jwt_as_access_token is true.
	Audience string `protobuf:"bytes,3,opt,name=audience,proto3" json:"audience,omitempty"`
}

func (x *GoogleCredentials) Reset() {
	*x = GoogleCredentials{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GoogleCredentials) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GoogleCredentials) ProtoMessage() {}

func (x *GoogleCredentials) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GoogleCredentials.ProtoReflect.Descriptor instead.
func (*GoogleCredentials) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescGZIP(), []int{3}
}

func (x *GoogleCredentials) GetJsonFile() string {
	if x != nil {
		return x.JsonFile
	}
	return ""
}

func (x *GoogleCredentials) GetScope() []string {
	if x != nil {
		return x.Scope
	}
	return nil
}

func (x *GoogleCredentials) GetJwtAsAccessToken() bool {
	if x != nil {
		return x.JwtAsAccessToken
	}
	return false
}

func (x *GoogleCredentials) GetAudience() string {
	if x != nil {
		return x.Audience
	}
	return ""
}

type JWT_Header struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name  *string `protobuf:"bytes,1,opt,name=name,proto3,oneof" json:"name,omitempty"`
	Value *string `protobuf:"bytes,2,opt,name=value,proto3,oneof" json:"value,omitempty"`
}

func (x *JWT_Header) Reset() {
	*x = JWT_Header{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JWT_Header) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JWT_Header) ProtoMessage() {}

func (x *JWT_Header) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JWT_Header.ProtoReflect.Descriptor instead.
func (*JWT_Header) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescGZIP(), []int{1, 0}
}

func (x *JWT_Header) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *JWT_Header) GetValue() string {
	if x != nil && x.Value != nil {
		return *x.Value
	}
	return ""
}

var File_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDesc = []byte{
	0x0a, 0x42, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2f, 0x6f, 0x61, 0x75, 0x74,
	0x68, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x11, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x22, 0xd8, 0x01, 0x0a, 0x06, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x12, 0x43, 0x0a, 0x0c, 0x62, 0x65, 0x61, 0x72, 0x65, 0x72, 0x5f, 0x74, 0x6f, 0x6b,
	0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x2e, 0x42, 0x65, 0x61,
	0x72, 0x65, 0x72, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x48, 0x00, 0x52, 0x0b, 0x62, 0x65, 0x61, 0x72,
	0x65, 0x72, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x55, 0x0a, 0x12, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x5f, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x2e, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x43, 0x72,
	0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x48, 0x00, 0x52, 0x11, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x12, 0x2a,
	0x0a, 0x03, 0x6a, 0x77, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x2e,
	0x4a, 0x57, 0x54, 0x48, 0x00, 0x52, 0x03, 0x6a, 0x77, 0x74, 0x42, 0x06, 0x0a, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x22, 0xd6, 0x01, 0x0a, 0x03, 0x4a, 0x57, 0x54, 0x12, 0x1b, 0x0a, 0x09, 0x74, 0x6f,
	0x6b, 0x65, 0x6e, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x74,
	0x6f, 0x6b, 0x65, 0x6e, 0x55, 0x72, 0x6c, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12,
	0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x64,
	0x61, 0x74, 0x61, 0x12, 0x35, 0x0a, 0x06, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x18, 0x08, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x2e, 0x4a, 0x57, 0x54, 0x2e, 0x48, 0x65, 0x61, 0x64,
	0x65, 0x72, 0x52, 0x06, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x1a, 0x4f, 0x0a, 0x06, 0x48, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x12, 0x17, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x48, 0x00, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x88, 0x01, 0x01, 0x12, 0x19, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x01, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x88, 0x01, 0x01, 0x42, 0x07, 0x0a, 0x05, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x42, 0x08, 0x0a, 0x06, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x22, 0xc3, 0x01, 0x0a, 0x0b,
	0x42, 0x65, 0x61, 0x72, 0x65, 0x72, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x14, 0x0a, 0x04, 0x66,
	0x69, 0x6c, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x04, 0x66, 0x69, 0x6c,
	0x65, 0x12, 0x12, 0x0a, 0x03, 0x63, 0x6d, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00,
	0x52, 0x03, 0x63, 0x6d, 0x64, 0x12, 0x30, 0x0a, 0x13, 0x67, 0x63, 0x65, 0x5f, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x5f, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x48, 0x00, 0x52, 0x11, 0x67, 0x63, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x35, 0x0a, 0x14, 0x72, 0x65, 0x66, 0x72, 0x65,
	0x73, 0x68, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18,
	0x5a, 0x20, 0x01, 0x28, 0x02, 0x48, 0x01, 0x52, 0x12, 0x72, 0x65, 0x66, 0x72, 0x65, 0x73, 0x68,
	0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x53, 0x65, 0x63, 0x88, 0x01, 0x01, 0x42, 0x08,
	0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x17, 0x0a, 0x15, 0x5f, 0x72, 0x65, 0x66,
	0x72, 0x65, 0x73, 0x68, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65,
	0x63, 0x22, 0x91, 0x01, 0x0a, 0x11, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x43, 0x72, 0x65, 0x64,
	0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x12, 0x1b, 0x0a, 0x09, 0x6a, 0x73, 0x6f, 0x6e, 0x5f,
	0x66, 0x69, 0x6c, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x6a, 0x73, 0x6f, 0x6e,
	0x46, 0x69, 0x6c, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x63, 0x6f, 0x70, 0x65, 0x18, 0x02, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x05, 0x73, 0x63, 0x6f, 0x70, 0x65, 0x12, 0x2d, 0x0a, 0x13, 0x6a, 0x77,
	0x74, 0x5f, 0x61, 0x73, 0x5f, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x5f, 0x74, 0x6f, 0x6b, 0x65,
	0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x10, 0x6a, 0x77, 0x74, 0x41, 0x73, 0x41, 0x63,
	0x63, 0x65, 0x73, 0x73, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x61, 0x75, 0x64,
	0x69, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x61, 0x75, 0x64,
	0x69, 0x65, 0x6e, 0x63, 0x65, 0x42, 0x37, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6d, 0x6d,
	0x6f, 0x6e, 0x2f, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_goTypes = []interface{}{
	(*Config)(nil),            // 0: cloudprober.oauth.Config
	(*JWT)(nil),               // 1: cloudprober.oauth.JWT
	(*BearerToken)(nil),       // 2: cloudprober.oauth.BearerToken
	(*GoogleCredentials)(nil), // 3: cloudprober.oauth.GoogleCredentials
	(*JWT_Header)(nil),        // 4: cloudprober.oauth.JWT.Header
}
var file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_depIdxs = []int32{
	2, // 0: cloudprober.oauth.Config.bearer_token:type_name -> cloudprober.oauth.BearerToken
	3, // 1: cloudprober.oauth.Config.google_credentials:type_name -> cloudprober.oauth.GoogleCredentials
	1, // 2: cloudprober.oauth.Config.jwt:type_name -> cloudprober.oauth.JWT
	4, // 3: cloudprober.oauth.JWT.header:type_name -> cloudprober.oauth.JWT.Header
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Config); i {
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
		file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JWT); i {
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
		file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BearerToken); i {
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
		file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GoogleCredentials); i {
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
		file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JWT_Header); i {
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
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Config_BearerToken)(nil),
		(*Config_GoogleCredentials)(nil),
		(*Config_Jwt)(nil),
	}
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*BearerToken_File)(nil),
		(*BearerToken_Cmd)(nil),
		(*BearerToken_GceServiceAccount)(nil),
	}
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes[4].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_common_oauth_proto_config_proto_depIdxs = nil
}
