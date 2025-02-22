// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/internal/servers/grpc/proto/grpcservice.proto

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

type EchoMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Blob          []byte                 `protobuf:"bytes,1,opt,name=blob" json:"blob,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EchoMessage) Reset() {
	*x = EchoMessage{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EchoMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EchoMessage) ProtoMessage() {}

func (x *EchoMessage) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EchoMessage.ProtoReflect.Descriptor instead.
func (*EchoMessage) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescGZIP(), []int{0}
}

func (x *EchoMessage) GetBlob() []byte {
	if x != nil {
		return x.Blob
	}
	return nil
}

type StatusRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ClientName    *string                `protobuf:"bytes,1,opt,name=client_name,json=clientName" json:"client_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StatusRequest) Reset() {
	*x = StatusRequest{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusRequest) ProtoMessage() {}

func (x *StatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusRequest.ProtoReflect.Descriptor instead.
func (*StatusRequest) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescGZIP(), []int{1}
}

func (x *StatusRequest) GetClientName() string {
	if x != nil && x.ClientName != nil {
		return *x.ClientName
	}
	return ""
}

type StatusResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UptimeUs      *int64                 `protobuf:"varint,1,opt,name=uptime_us,json=uptimeUs" json:"uptime_us,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StatusResponse) Reset() {
	*x = StatusResponse{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StatusResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusResponse) ProtoMessage() {}

func (x *StatusResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StatusResponse.ProtoReflect.Descriptor instead.
func (*StatusResponse) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescGZIP(), []int{2}
}

func (x *StatusResponse) GetUptimeUs() int64 {
	if x != nil && x.UptimeUs != nil {
		return *x.UptimeUs
	}
	return 0
}

type BlobReadRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Size          *int32                 `protobuf:"varint,1,opt,name=size" json:"size,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BlobReadRequest) Reset() {
	*x = BlobReadRequest{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BlobReadRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlobReadRequest) ProtoMessage() {}

func (x *BlobReadRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlobReadRequest.ProtoReflect.Descriptor instead.
func (*BlobReadRequest) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescGZIP(), []int{3}
}

func (x *BlobReadRequest) GetSize() int32 {
	if x != nil && x.Size != nil {
		return *x.Size
	}
	return 0
}

type BlobReadResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Blob          []byte                 `protobuf:"bytes,1,opt,name=blob" json:"blob,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BlobReadResponse) Reset() {
	*x = BlobReadResponse{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BlobReadResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlobReadResponse) ProtoMessage() {}

func (x *BlobReadResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlobReadResponse.ProtoReflect.Descriptor instead.
func (*BlobReadResponse) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescGZIP(), []int{4}
}

func (x *BlobReadResponse) GetBlob() []byte {
	if x != nil {
		return x.Blob
	}
	return nil
}

type BlobWriteRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Blob          []byte                 `protobuf:"bytes,1,opt,name=blob" json:"blob,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BlobWriteRequest) Reset() {
	*x = BlobWriteRequest{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BlobWriteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlobWriteRequest) ProtoMessage() {}

func (x *BlobWriteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlobWriteRequest.ProtoReflect.Descriptor instead.
func (*BlobWriteRequest) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescGZIP(), []int{5}
}

func (x *BlobWriteRequest) GetBlob() []byte {
	if x != nil {
		return x.Blob
	}
	return nil
}

type BlobWriteResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Size          *int32                 `protobuf:"varint,1,opt,name=size" json:"size,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BlobWriteResponse) Reset() {
	*x = BlobWriteResponse{}
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BlobWriteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlobWriteResponse) ProtoMessage() {}

func (x *BlobWriteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlobWriteResponse.ProtoReflect.Descriptor instead.
func (*BlobWriteResponse) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescGZIP(), []int{6}
}

func (x *BlobWriteResponse) GetSize() int32 {
	if x != nil && x.Size != nil {
		return *x.Size
	}
	return 0
}

var File_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDesc = string([]byte{
	0x0a, 0x50, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x67, 0x72, 0x70, 0x63, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x18, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e,
	0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x22, 0x21, 0x0a, 0x0b,
	0x45, 0x63, 0x68, 0x6f, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x62,
	0x6c, 0x6f, 0x62, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x62, 0x6c, 0x6f, 0x62, 0x22,
	0x30, 0x0a, 0x0d, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x4e, 0x61, 0x6d,
	0x65, 0x22, 0x2d, 0x0a, 0x0e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x75, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65, 0x55, 0x73,
	0x22, 0x25, 0x0a, 0x0f, 0x42, 0x6c, 0x6f, 0x62, 0x52, 0x65, 0x61, 0x64, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x22, 0x26, 0x0a, 0x10, 0x42, 0x6c, 0x6f, 0x62, 0x52,
	0x65, 0x61, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x62,
	0x6c, 0x6f, 0x62, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x62, 0x6c, 0x6f, 0x62, 0x22,
	0x26, 0x0a, 0x10, 0x42, 0x6c, 0x6f, 0x62, 0x57, 0x72, 0x69, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x62, 0x6c, 0x6f, 0x62, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x04, 0x62, 0x6c, 0x6f, 0x62, 0x22, 0x27, 0x0a, 0x11, 0x42, 0x6c, 0x6f, 0x62, 0x57,
	0x72, 0x69, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04,
	0x73, 0x69, 0x7a, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x73, 0x69, 0x7a, 0x65,
	0x32, 0x92, 0x03, 0x0a, 0x06, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x12, 0x56, 0x0a, 0x04, 0x45,
	0x63, 0x68, 0x6f, 0x12, 0x25, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x45,
	0x63, 0x68, 0x6f, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x1a, 0x25, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x45, 0x63, 0x68, 0x6f, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x22, 0x00, 0x12, 0x63, 0x0a, 0x08, 0x42, 0x6c, 0x6f, 0x62, 0x52, 0x65, 0x61, 0x64, 0x12,
	0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x52,
	0x65, 0x61, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2a, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x52, 0x65, 0x61, 0x64, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x63, 0x0a, 0x0c, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x27, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x28, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e,
	0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x66, 0x0a,
	0x09, 0x42, 0x6c, 0x6f, 0x62, 0x57, 0x72, 0x69, 0x74, 0x65, 0x12, 0x2a, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x57, 0x72, 0x69, 0x74, 0x65, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2b, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x67, 0x72, 0x70,
	0x63, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x57, 0x72, 0x69, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x40, 0x5a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x67, 0x72, 0x70,
	0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
})

var (
	file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescData []byte
)

func file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDesc)))
	})
	return file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_goTypes = []any{
	(*EchoMessage)(nil),       // 0: cloudprober.servers.grpc.EchoMessage
	(*StatusRequest)(nil),     // 1: cloudprober.servers.grpc.StatusRequest
	(*StatusResponse)(nil),    // 2: cloudprober.servers.grpc.StatusResponse
	(*BlobReadRequest)(nil),   // 3: cloudprober.servers.grpc.BlobReadRequest
	(*BlobReadResponse)(nil),  // 4: cloudprober.servers.grpc.BlobReadResponse
	(*BlobWriteRequest)(nil),  // 5: cloudprober.servers.grpc.BlobWriteRequest
	(*BlobWriteResponse)(nil), // 6: cloudprober.servers.grpc.BlobWriteResponse
}
var file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_depIdxs = []int32{
	0, // 0: cloudprober.servers.grpc.Prober.Echo:input_type -> cloudprober.servers.grpc.EchoMessage
	3, // 1: cloudprober.servers.grpc.Prober.BlobRead:input_type -> cloudprober.servers.grpc.BlobReadRequest
	1, // 2: cloudprober.servers.grpc.Prober.ServerStatus:input_type -> cloudprober.servers.grpc.StatusRequest
	5, // 3: cloudprober.servers.grpc.Prober.BlobWrite:input_type -> cloudprober.servers.grpc.BlobWriteRequest
	0, // 4: cloudprober.servers.grpc.Prober.Echo:output_type -> cloudprober.servers.grpc.EchoMessage
	4, // 5: cloudprober.servers.grpc.Prober.BlobRead:output_type -> cloudprober.servers.grpc.BlobReadResponse
	2, // 6: cloudprober.servers.grpc.Prober.ServerStatus:output_type -> cloudprober.servers.grpc.StatusResponse
	6, // 7: cloudprober.servers.grpc.Prober.BlobWrite:output_type -> cloudprober.servers.grpc.BlobWriteResponse
	4, // [4:8] is the sub-list for method output_type
	0, // [0:4] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() {
	file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_init()
}
func file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_init() {
	if File_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDesc), len(file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto = out.File
	file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_internal_servers_grpc_proto_grpcservice_proto_depIdxs = nil
}
