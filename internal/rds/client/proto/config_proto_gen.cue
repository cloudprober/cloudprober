package proto

import (
	"github.com/cloudprober/cloudprober/internal/oauth/proto"
	proto_1 "github.com/cloudprober/cloudprober/internal/tlsconfig/proto"
	proto_5 "github.com/cloudprober/cloudprober/internal/rds/proto"
)

// ClientConf represents resource discovery service (RDS) based targets.
// Next tag: 6
#ClientConf: {

	#ServerOptions: {
		serverAddress?: string @protobuf(1,string,name=server_address)

		// Optional oauth config for authentication.
		oauthConfig?: proto.#Config @protobuf(2,oauth.Config,name=oauth_config)

		// TLS config, it can be used to:
		// - Specify a CA cert for server cert verification:
		//     tls_config {
		//       ca_cert_file: "...."
		//     }
		//
		// - Specify client's TLS cert and key:
		//     tls_config {
		//       tls_cert_file: "..."
		//       tls_key_file: "..."
		//     }
		tlsConfig?: proto_1.#TLSConfig @protobuf(3,tlsconfig.TLSConfig,name=tls_config)
	}
	serverOptions?: #ServerOptions                @protobuf(1,ServerOptions,name=server_options)
	request?:       proto_5.#ListResourcesRequest @protobuf(2,ListResourcesRequest)

	// How often targets should be evaluated. Any number less than or equal to 0
	// will result in no target caching (targets will be reevaluated on demand).
	// Note that individual target types may have their own caches implemented
	// (specifically GCE instances/forwarding rules). This does not impact those
	// caches.
	reEvalSec?: int32 @protobuf(3,int32,name=re_eval_sec,"default=30")
}
