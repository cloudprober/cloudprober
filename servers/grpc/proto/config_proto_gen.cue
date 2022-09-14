package proto

#ServerConf: {
	port?: int32 @protobuf(1,int32,"default=3142")

	// Enables gRPC reflection for publicly visible services, allowing grpc_cli to
	// work. See https://grpc.io/grpc/core/md_doc_server_reflection_tutorial.html.
	enableReflection?: bool @protobuf(2,bool,name=enable_reflection,"default=false")

	// If use_dedicated_server is set to true, then create a new gRPC server
	// to handle probes. Otherwise, attempt to reuse gRPC server from runconfig
	// if that was set.
	useDedicatedServer?: bool @protobuf(3,bool,name=use_dedicated_server,default)
}
