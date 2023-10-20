package proto

import "github.com/cloudprober/cloudprober/internal/rds/client/proto"

#Options: {
	// How often to check for lame-ducked targets
	reEvalSec?: int32 @protobuf(1,int32,name=re_eval_sec,"default=10")

	// Runtime config project. If running on GCE, this defaults to the project
	// containing the VM.
	runtimeconfigProject?: string @protobuf(2,string,name=runtimeconfig_project)

	// Lame duck targets runtime config name. An operator will create a variable
	// here to mark a target as lame-ducked.
	runtimeconfigName?: string @protobuf(3,string,name=runtimeconfig_name,#"default="lame-duck-targets""#)

	// Lame duck targets pubsub topic name. An operator will create a message
	// here to mark a target as lame-ducked.
	pubsubTopic?: string @protobuf(7,string,name=pubsub_topic)

	// Lame duck expiration time. We ignore variables (targets) that have been
	// updated more than these many seconds ago. This is a safety mechanism for
	// failing to cleanup. Also, the idea is that if a target has actually
	// disappeared, automatic targets expansion will take care of that some time
	// during this expiration period.
	expirationSec?: int32 @protobuf(4,int32,name=expiration_sec,"default=300")

	// Use an RDS client to get lame-duck-targets.
	// This option is always true now and will be removed after v0.10.7.
	useRds?: bool @protobuf(5,bool,name=use_rds,deprecated)

	// RDS server options, for example:
	// rds_server_options {
	//   server_address: "rds-server.xyz:9314"
	//   oauth_config: {
	//     ...
	//   }
	rdsServerOptions?: proto.#ClientConf.#ServerOptions @protobuf(6,rds.ClientConf.ServerOptions,name=rds_server_options)
}
