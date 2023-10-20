package proto

import (
	"github.com/cloudprober/cloudprober/probes/proto"
	proto_1 "github.com/cloudprober/cloudprober/surfacers/proto"
	proto_5 "github.com/cloudprober/cloudprober/internal/servers/proto"
	proto_A "github.com/cloudprober/cloudprober/internal/rds/server/proto"
	proto_8 "github.com/cloudprober/cloudprober/internal/tlsconfig/proto"
	proto_E "github.com/cloudprober/cloudprober/targets/proto"
)

// Cloudprober config proto defines the config schema. Cloudprober config can
// either be in YAML or textproto format.
//
// Cloudprober uses Go text templates along with Sprig functions[1] to add
// programming capabilities to the configs.
// [1]- http://masterminds.github.io/sprig/
// Config template examples:
// https://github.com/cloudprober/cloudprober/tree/master/examples/templates
#ProberConfig: {
	// Probes to run.
	probe?: [...proto.#ProbeDef] @protobuf(1,probes.ProbeDef)

	// Surfacers are used to export probe results for further processing.
	// If no surfacer is configured, a prometheus and a file surfacer are
	// initialized:
	//  - Prometheus makes probe results available at http://<host>:9313/metrics.
	//  - File surfacer writes results to stdout.
	//
	// You can disable default surfacers (in case you want no surfacer at all), by
	// adding the following to your config:
	//   surfacer {}
	surfacer?: [...proto_1.#SurfacerDef] @protobuf(2,surfacer.SurfacerDef)

	// Servers to run inside cloudprober. These servers can serve as targets for
	// other probes.
	server?: [...proto_5.#ServerDef] @protobuf(3,servers.ServerDef)

	// Shared targets allow you to re-use the same targets copy across multiple
	// probes.
	// Example:
	// shared_targets {
	//   name: "internal-vms"
	//   targets {
	//     rds_targets {...}
	//   }
	// }
	//
	// probe {
	//   name: "vm-ping"
	//   type: PING
	//   targets {
	//     shared_targets: "internal-vms"
	//   }
	// }
	//
	// probe {
	//   name: "vm-http"
	//   type: HTTP
	//   targets {
	//     shared_targets: "internal-vms"
	//   }
	// }
	sharedTargets?: [...#SharedTargets] @protobuf(4,SharedTargets,name=shared_targets)
	// Common services related options.
	// Next tag: 106

	// Resource discovery server
	rdsServer?: proto_A.#ServerConf @protobuf(95,rds.ServerConf,name=rds_server)

	// Port for the default HTTP server. This port is also used for prometheus
	// exporter (URL /metrics). Default port is 9313. If not specified in the
	// config, default port can be overridden by the environment variable
	// CLOUDPROBER_PORT.
	port?: int32 @protobuf(96,int32)

	// Port to run the default gRPC server on. If not specified, and if
	// environment variable CLOUDPROBER_GRPC_PORT is set, CLOUDPROBER_GRPC_PORT is
	// used for the default gRPC server. If CLOUDPROBER_GRPC_PORT is not set as
	// well, default gRPC server is not started.
	grpcPort?: int32 @protobuf(104,int32,name=grpc_port)

	// TLS config, it can be used to:
	// - Specify client's CA cert for client cert verification:
	//     grpc_tls_config {
	//       ca_cert_file: "...."
	//     }
	//
	// - Specify TLS cert and key:
	//     grpc_tls_config {
	//       tls_cert_file: "..."
	//       tls_key_file: "..."
	//     }
	grpcTlsConfig?: proto_8.#TLSConfig @protobuf(105,tlsconfig.TLSConfig,name=grpc_tls_config)

	// Host for the default HTTP server. Default listens on all addresses. If not
	// specified in the config, default port can be overridden by the environment
	// variable CLOUDPROBER_HOST.
	host?: string @protobuf(101,string)

	// Probes are staggered across time to avoid executing all of them at the
	// same time. This behavior can be disabled by setting the following option
	// to true.
	disableJitter?: bool @protobuf(102,bool,name=disable_jitter,"default=false")

	// How often to export system variables. To learn more about system variables:
	// http://godoc.org/github.com/cloudprober/cloudprober/internal/sysvars.
	sysvarsIntervalMsec?: int32 @protobuf(97,int32,name=sysvars_interval_msec,"default=10000")

	// Variables specified in this environment variable are exported as it is.
	// This is specifically useful to export information about system environment,
	// for example, docker image tag/digest-id, OS version etc. See
	// tools/cloudprober_startup.sh in the cloudprober directory for an example on
	// how to use these variables.
	sysvarsEnvVar?: string @protobuf(98,string,name=sysvars_env_var,#"default="SYSVARS""#)

	// Time between triggering cancelation of various goroutines and exiting the
	// process. If --stop_time flag is also configured, that gets priority.
	// You may want to set it to 0 if cloudprober is running as a backend for
	// the probes and you don't want time lost in stop and start.
	stopTimeSec?: int32 @protobuf(99,int32,name=stop_time_sec,"default=5")

	// Global targets options. Per-probe options are specified within the probe
	// stanza.
	globalTargetsOptions?: proto_E.#GlobalTargetsOptions @protobuf(100,targets.GlobalTargetsOptions,name=global_targets_options)
}

#SharedTargets: {
	name?:    string              @protobuf(1,string)
	targets?: proto_E.#TargetsDef @protobuf(2,targets.TargetsDef)
}
