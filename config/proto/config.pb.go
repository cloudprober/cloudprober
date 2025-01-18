// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.3
// 	protoc        v5.27.5
// source: github.com/cloudprober/cloudprober/config/proto/config.proto

package proto

import (
	proto3 "github.com/cloudprober/cloudprober/internal/rds/server/proto"
	proto2 "github.com/cloudprober/cloudprober/internal/servers/proto"
	proto4 "github.com/cloudprober/cloudprober/internal/tlsconfig/proto"
	proto "github.com/cloudprober/cloudprober/probes/proto"
	proto1 "github.com/cloudprober/cloudprober/surfacers/proto"
	proto5 "github.com/cloudprober/cloudprober/targets/proto"
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

// Cloudprober config proto defines the config schema. Cloudprober config can
// either be in YAML or textproto format.
//
// Cloudprober uses Go text templates along with Sprig functions[1] to add
// programming capabilities to the configs.
// [1]- http://masterminds.github.io/sprig/
// Config template examples:
// https://github.com/cloudprober/cloudprober/tree/master/examples/templates
type ProberConfig struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Probes to run.
	Probe []*proto.ProbeDef `protobuf:"bytes,1,rep,name=probe" json:"probe,omitempty"`
	// Surfacers are used to export probe results for further processing.
	// If no surfacer is configured, a prometheus and a file surfacer are
	// initialized:
	//   - Prometheus makes probe results available at http://<host>:9313/metrics.
	//   - File surfacer writes results to stdout.
	//
	// You can disable default surfacers (in case you want no surfacer at all), by
	// adding the following to your config:
	//
	//	surfacer {}
	Surfacer []*proto1.SurfacerDef `protobuf:"bytes,2,rep,name=surfacer" json:"surfacer,omitempty"`
	// Servers to run inside cloudprober. These servers can serve as targets for
	// other probes.
	Server []*proto2.ServerDef `protobuf:"bytes,3,rep,name=server" json:"server,omitempty"`
	// Shared targets allow you to re-use the same targets copy across multiple
	// probes.
	// Example:
	//
	//	shared_targets {
	//	  name: "internal-vms"
	//	  targets {
	//	    rds_targets {...}
	//	  }
	//	}
	//
	//	probe {
	//	  name: "vm-ping"
	//	  type: PING
	//	  targets {
	//	    shared_targets: "internal-vms"
	//	  }
	//	}
	//
	//	probe {
	//	  name: "vm-http"
	//	  type: HTTP
	//	  targets {
	//	    shared_targets: "internal-vms"
	//	  }
	//	}
	SharedTargets []*SharedTargets `protobuf:"bytes,4,rep,name=shared_targets,json=sharedTargets" json:"shared_targets,omitempty"`
	// Resource discovery server
	RdsServer *proto3.ServerConf `protobuf:"bytes,95,opt,name=rds_server,json=rdsServer" json:"rds_server,omitempty"`
	// Port for the default HTTP server. This port is also used for prometheus
	// exporter (URL /metrics). Default port is 9313. If not specified in the
	// config, default port can be overridden by the environment variable
	// CLOUDPROBER_PORT.
	Port *int32 `protobuf:"varint,96,opt,name=port" json:"port,omitempty"`
	// Port to run the default gRPC server on. If not specified, and if
	// environment variable CLOUDPROBER_GRPC_PORT is set, CLOUDPROBER_GRPC_PORT is
	// used for the default gRPC server. If CLOUDPROBER_GRPC_PORT is not set as
	// well, default gRPC server is not started.
	GrpcPort *int32 `protobuf:"varint,104,opt,name=grpc_port,json=grpcPort" json:"grpc_port,omitempty"`
	// TLS config, it can be used to:
	//   - Specify client's CA cert for client cert verification:
	//     grpc_tls_config {
	//     ca_cert_file: "...."
	//     }
	//
	//   - Specify TLS cert and key:
	//     grpc_tls_config {
	//     tls_cert_file: "..."
	//     tls_key_file: "..."
	//     }
	GrpcTlsConfig *proto4.TLSConfig `protobuf:"bytes,105,opt,name=grpc_tls_config,json=grpcTlsConfig" json:"grpc_tls_config,omitempty"`
	// Host for the default HTTP server. Default listens on all addresses. If not
	// specified in the config, default port can be overridden by the environment
	// variable CLOUDPROBER_HOST.
	Host *string `protobuf:"bytes,101,opt,name=host" json:"host,omitempty"`
	// Probes are staggered across time to avoid executing all of them at the
	// same time. This behavior can be disabled by setting the following option
	// to true.
	DisableJitter *bool `protobuf:"varint,102,opt,name=disable_jitter,json=disableJitter,def=0" json:"disable_jitter,omitempty"`
	// How often to export system variables. To learn more about system variables:
	// http://godoc.org/github.com/cloudprober/cloudprober/internal/sysvars.
	SysvarsIntervalMsec *int32 `protobuf:"varint,97,opt,name=sysvars_interval_msec,json=sysvarsIntervalMsec,def=10000" json:"sysvars_interval_msec,omitempty"`
	// Variables specified in this environment variable are exported as it is.
	// This is specifically useful to export information about system environment,
	// for example, docker image tag/digest-id, OS version etc. See
	// tools/cloudprober_startup.sh in the cloudprober directory for an example on
	// how to use these variables.
	SysvarsEnvVar *string `protobuf:"bytes,98,opt,name=sysvars_env_var,json=sysvarsEnvVar,def=SYSVARS" json:"sysvars_env_var,omitempty"`
	// Time between triggering cancelation of various goroutines and exiting the
	// process. If --stop_time flag is also configured, that gets priority.
	// You may want to set it to 0 if cloudprober is running as a backend for
	// the probes and you don't want time lost in stop and start.
	StopTimeSec *int32 `protobuf:"varint,99,opt,name=stop_time_sec,json=stopTimeSec,def=5" json:"stop_time_sec,omitempty"`
	// Global targets options. Per-probe options are specified within the probe
	// stanza.
	GlobalTargetsOptions *proto5.GlobalTargetsOptions `protobuf:"bytes,100,opt,name=global_targets_options,json=globalTargetsOptions" json:"global_targets_options,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

// Default values for ProberConfig fields.
const (
	Default_ProberConfig_DisableJitter       = bool(false)
	Default_ProberConfig_SysvarsIntervalMsec = int32(10000)
	Default_ProberConfig_SysvarsEnvVar       = string("SYSVARS")
	Default_ProberConfig_StopTimeSec         = int32(5)
)

func (x *ProberConfig) Reset() {
	*x = ProberConfig{}
	mi := &file_github_com_cloudprober_cloudprober_config_proto_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ProberConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProberConfig) ProtoMessage() {}

func (x *ProberConfig) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_config_proto_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProberConfig.ProtoReflect.Descriptor instead.
func (*ProberConfig) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *ProberConfig) GetProbe() []*proto.ProbeDef {
	if x != nil {
		return x.Probe
	}
	return nil
}

func (x *ProberConfig) GetSurfacer() []*proto1.SurfacerDef {
	if x != nil {
		return x.Surfacer
	}
	return nil
}

func (x *ProberConfig) GetServer() []*proto2.ServerDef {
	if x != nil {
		return x.Server
	}
	return nil
}

func (x *ProberConfig) GetSharedTargets() []*SharedTargets {
	if x != nil {
		return x.SharedTargets
	}
	return nil
}

func (x *ProberConfig) GetRdsServer() *proto3.ServerConf {
	if x != nil {
		return x.RdsServer
	}
	return nil
}

func (x *ProberConfig) GetPort() int32 {
	if x != nil && x.Port != nil {
		return *x.Port
	}
	return 0
}

func (x *ProberConfig) GetGrpcPort() int32 {
	if x != nil && x.GrpcPort != nil {
		return *x.GrpcPort
	}
	return 0
}

func (x *ProberConfig) GetGrpcTlsConfig() *proto4.TLSConfig {
	if x != nil {
		return x.GrpcTlsConfig
	}
	return nil
}

func (x *ProberConfig) GetHost() string {
	if x != nil && x.Host != nil {
		return *x.Host
	}
	return ""
}

func (x *ProberConfig) GetDisableJitter() bool {
	if x != nil && x.DisableJitter != nil {
		return *x.DisableJitter
	}
	return Default_ProberConfig_DisableJitter
}

func (x *ProberConfig) GetSysvarsIntervalMsec() int32 {
	if x != nil && x.SysvarsIntervalMsec != nil {
		return *x.SysvarsIntervalMsec
	}
	return Default_ProberConfig_SysvarsIntervalMsec
}

func (x *ProberConfig) GetSysvarsEnvVar() string {
	if x != nil && x.SysvarsEnvVar != nil {
		return *x.SysvarsEnvVar
	}
	return Default_ProberConfig_SysvarsEnvVar
}

func (x *ProberConfig) GetStopTimeSec() int32 {
	if x != nil && x.StopTimeSec != nil {
		return *x.StopTimeSec
	}
	return Default_ProberConfig_StopTimeSec
}

func (x *ProberConfig) GetGlobalTargetsOptions() *proto5.GlobalTargetsOptions {
	if x != nil {
		return x.GlobalTargetsOptions
	}
	return nil
}

type SharedTargets struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          *string                `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	Targets       *proto5.TargetsDef     `protobuf:"bytes,2,req,name=targets" json:"targets,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SharedTargets) Reset() {
	*x = SharedTargets{}
	mi := &file_github_com_cloudprober_cloudprober_config_proto_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SharedTargets) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SharedTargets) ProtoMessage() {}

func (x *SharedTargets) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_config_proto_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SharedTargets.ProtoReflect.Descriptor instead.
func (*SharedTargets) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *SharedTargets) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *SharedTargets) GetTargets() *proto5.TargetsDef {
	if x != nil {
		return x.Targets
	}
	return nil
}

// This is used to parse surfacers config from a separate file.
type SurfacersConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Surfacer      []*proto1.SurfacerDef  `protobuf:"bytes,1,rep,name=surfacer" json:"surfacer,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SurfacersConfig) Reset() {
	*x = SurfacersConfig{}
	mi := &file_github_com_cloudprober_cloudprober_config_proto_config_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SurfacersConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacersConfig) ProtoMessage() {}

func (x *SurfacersConfig) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_config_proto_config_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SurfacersConfig.ProtoReflect.Descriptor instead.
func (*SurfacersConfig) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescGZIP(), []int{2}
}

func (x *SurfacersConfig) GetSurfacer() []*proto1.SurfacerDef {
	if x != nil {
		return x.Surfacer
	}
	return nil
}

var File_github_com_cloudprober_cloudprober_config_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDesc = []byte{
	0x0a, 0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x1a, 0x48, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x6c, 0x73, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x49, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c,
	0x2f, 0x72, 0x64, 0x73, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x46,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x3f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61,
	0x63, 0x65, 0x72, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xe9, 0x05, 0x0a, 0x0c, 0x50, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x32, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x50, 0x72, 0x6f,
	0x62, 0x65, 0x44, 0x65, 0x66, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x12, 0x3d, 0x0a, 0x08,
	0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21,
	0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x44, 0x65,
	0x66, 0x52, 0x08, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x12, 0x36, 0x0a, 0x06, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72,
	0x73, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x44, 0x65, 0x66, 0x52, 0x06, 0x73, 0x65, 0x72,
	0x76, 0x65, 0x72, 0x12, 0x41, 0x0a, 0x0e, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x5f, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x53, 0x68, 0x61, 0x72, 0x65, 0x64,
	0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x52, 0x0d, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x54,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12, 0x3a, 0x0a, 0x0a, 0x72, 0x64, 0x73, 0x5f, 0x73, 0x65,
	0x72, 0x76, 0x65, 0x72, 0x18, 0x5f, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x53, 0x65, 0x72,
	0x76, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x52, 0x09, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x60, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x67, 0x72, 0x70, 0x63, 0x5f, 0x70,
	0x6f, 0x72, 0x74, 0x18, 0x68, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x67, 0x72, 0x70, 0x63, 0x50,
	0x6f, 0x72, 0x74, 0x12, 0x48, 0x0a, 0x0f, 0x67, 0x72, 0x70, 0x63, 0x5f, 0x74, 0x6c, 0x73, 0x5f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x69, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x6c, 0x73, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x54, 0x4c, 0x53, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x0d,
	0x67, 0x72, 0x70, 0x63, 0x54, 0x6c, 0x73, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x12, 0x0a,
	0x04, 0x68, 0x6f, 0x73, 0x74, 0x18, 0x65, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x6f, 0x73,
	0x74, 0x12, 0x2c, 0x0a, 0x0e, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x6a, 0x69, 0x74,
	0x74, 0x65, 0x72, 0x18, 0x66, 0x20, 0x01, 0x28, 0x08, 0x3a, 0x05, 0x66, 0x61, 0x6c, 0x73, 0x65,
	0x52, 0x0d, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x4a, 0x69, 0x74, 0x74, 0x65, 0x72, 0x12,
	0x39, 0x0a, 0x15, 0x73, 0x79, 0x73, 0x76, 0x61, 0x72, 0x73, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x76, 0x61, 0x6c, 0x5f, 0x6d, 0x73, 0x65, 0x63, 0x18, 0x61, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x05,
	0x31, 0x30, 0x30, 0x30, 0x30, 0x52, 0x13, 0x73, 0x79, 0x73, 0x76, 0x61, 0x72, 0x73, 0x49, 0x6e,
	0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x4d, 0x73, 0x65, 0x63, 0x12, 0x2f, 0x0a, 0x0f, 0x73, 0x79,
	0x73, 0x76, 0x61, 0x72, 0x73, 0x5f, 0x65, 0x6e, 0x76, 0x5f, 0x76, 0x61, 0x72, 0x18, 0x62, 0x20,
	0x01, 0x28, 0x09, 0x3a, 0x07, 0x53, 0x59, 0x53, 0x56, 0x41, 0x52, 0x53, 0x52, 0x0d, 0x73, 0x79,
	0x73, 0x76, 0x61, 0x72, 0x73, 0x45, 0x6e, 0x76, 0x56, 0x61, 0x72, 0x12, 0x25, 0x0a, 0x0d, 0x73,
	0x74, 0x6f, 0x70, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x63, 0x20, 0x01,
	0x28, 0x05, 0x3a, 0x01, 0x35, 0x52, 0x0b, 0x73, 0x74, 0x6f, 0x70, 0x54, 0x69, 0x6d, 0x65, 0x53,
	0x65, 0x63, 0x12, 0x5f, 0x0a, 0x16, 0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x5f, 0x74, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x73, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x64, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x54,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x14, 0x67,
	0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x4f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x22, 0x5e, 0x0a, 0x0d, 0x53, 0x68, 0x61, 0x72, 0x65, 0x64, 0x54, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x02,
	0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x39, 0x0a, 0x07, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x18, 0x02, 0x20, 0x02, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e,
	0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x44, 0x65, 0x66, 0x52, 0x07, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x22, 0x50, 0x0a, 0x0f, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x3d, 0x0a, 0x08, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e,
	0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x44, 0x65, 0x66, 0x52, 0x08, 0x73, 0x75, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x72, 0x42, 0x31, 0x5a, 0x2f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_config_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_github_com_cloudprober_cloudprober_config_proto_config_proto_goTypes = []any{
	(*ProberConfig)(nil),                // 0: cloudprober.ProberConfig
	(*SharedTargets)(nil),               // 1: cloudprober.SharedTargets
	(*SurfacersConfig)(nil),             // 2: cloudprober.SurfacersConfig
	(*proto.ProbeDef)(nil),              // 3: cloudprober.probes.ProbeDef
	(*proto1.SurfacerDef)(nil),          // 4: cloudprober.surfacer.SurfacerDef
	(*proto2.ServerDef)(nil),            // 5: cloudprober.servers.ServerDef
	(*proto3.ServerConf)(nil),           // 6: cloudprober.rds.ServerConf
	(*proto4.TLSConfig)(nil),            // 7: cloudprober.tlsconfig.TLSConfig
	(*proto5.GlobalTargetsOptions)(nil), // 8: cloudprober.targets.GlobalTargetsOptions
	(*proto5.TargetsDef)(nil),           // 9: cloudprober.targets.TargetsDef
}
var file_github_com_cloudprober_cloudprober_config_proto_config_proto_depIdxs = []int32{
	3, // 0: cloudprober.ProberConfig.probe:type_name -> cloudprober.probes.ProbeDef
	4, // 1: cloudprober.ProberConfig.surfacer:type_name -> cloudprober.surfacer.SurfacerDef
	5, // 2: cloudprober.ProberConfig.server:type_name -> cloudprober.servers.ServerDef
	1, // 3: cloudprober.ProberConfig.shared_targets:type_name -> cloudprober.SharedTargets
	6, // 4: cloudprober.ProberConfig.rds_server:type_name -> cloudprober.rds.ServerConf
	7, // 5: cloudprober.ProberConfig.grpc_tls_config:type_name -> cloudprober.tlsconfig.TLSConfig
	8, // 6: cloudprober.ProberConfig.global_targets_options:type_name -> cloudprober.targets.GlobalTargetsOptions
	9, // 7: cloudprober.SharedTargets.targets:type_name -> cloudprober.targets.TargetsDef
	4, // 8: cloudprober.SurfacersConfig.surfacer:type_name -> cloudprober.surfacer.SurfacerDef
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_config_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_config_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_config_proto_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_config_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_config_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_config_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_config_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_config_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_config_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_config_proto_config_proto_depIdxs = nil
}
