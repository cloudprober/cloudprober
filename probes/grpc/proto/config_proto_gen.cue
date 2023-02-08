package proto

import (
	"github.com/cloudprober/cloudprober/common/oauth/proto"
	proto_1 "github.com/cloudprober/cloudprober/common/tlsconfig/proto"
)

// Next tag: 13
#ProbeConf: {
	// Optional oauth config. For GOOGLE_DEFAULT_CREDENTIALS, use:
	// oauth_config: { bearer_token { gce_service_account: "default" } }
	oauthConfig?: proto.#Config @protobuf(1,oauth.Config,name=oauth_config)

	// ALTS is a gRPC security method supported by some Google services.
	// If enabled, peers, with the help of a handshaker service (e.g. metadata
	// server of GCE instances), use credentials attached to the service accounts
	// to authenticate each other. See
	// https://cloud.google.com/security/encryption-in-transit/#service_integrity_encryption
	// for more details.
	#ALTSConfig: {
		// If provided, ALTS verifies that peer is using one of the given service
		// accounts.
		targetServiceAccount?: [...string] @protobuf(1,string,name=target_service_account)

		// Handshaker service address. Default is to use the local metadata server.
		// For most of the ALTS use cases, default address should be okay.
		handshakerServiceAddress?: string @protobuf(2,string,name=handshaker_service_address)
	}

	// If alts_config is provided, gRPC client uses ALTS for authentication and
	// encryption. For default alts configs, use:
	// alts_config: {}
	altsConfig?: #ALTSConfig @protobuf(2,ALTSConfig,name=alts_config)

	// If TLSConfig is specified, it's used for authentication.
	// Note that only one of ALTSConfig and TLSConfig can be enabled at a time.
	tlsConfig?: proto_1.#TLSConfig @protobuf(9,tlsconfig.TLSConfig,name=tls_config)

	// if insecure_transport is set to true, TLS will not be used.
	insecureTransport?: bool @protobuf(12,bool,name=insecure_transport)

	#MethodType: {"ECHO", #enumValue: 1} |
		{"READ", #enumValue: 2} |
		{"WRITE", #enumValue: 3} | {
			"HEALTH_CHECK"// gRPC healthcheck service.
			#enumValue: 4
		}

	#MethodType_value: {
		ECHO:         1
		READ:         2
		WRITE:        3
		HEALTH_CHECK: 4
	}
	method?: #MethodType @protobuf(3,MethodType,"default=ECHO")

	// Blob size for ECHO, READ, and WRITE methods.
	blobSize?: int32 @protobuf(4,int32,name=blob_size,"default=1024")

	// For HEALTH_CHECK, name of the service to health check.
	healthCheckService?: string @protobuf(10,string,name=health_check_service)

	// For HEALTH_CHECK, ignore status. By default, HEALTH_CHECK test passes
	// only if response-status is SERVING. Setting the following option makes
	// HEALTH_CHECK pass regardless of the response-status.
	healthCheckIgnoreStatus?: bool  @protobuf(11,bool,name=health_check_ignore_status)
	numConns?:                int32 @protobuf(5,int32,name=num_conns,"default=2")
	keepAlive?:               bool  @protobuf(6,bool,name=keep_alive,default)

	// If connect_timeout is not specified, reuse probe timeout.
	connectTimeoutMsec?: int32 @protobuf(7,int32,name=connect_timeout_msec)

	// URI scheme allows gRPC to use different resolvers
	// Example URI scheme: "google-c2p:///"
	// See https://github.com/grpc/grpc/blob/master/doc/naming.md for more details
	uriScheme?: string @protobuf(8,string,name=uri_scheme)
}
