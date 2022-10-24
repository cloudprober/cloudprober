package proto

// Next tag: 4
#ProbeConf: {
	// Port for TCP requests. If not specfied, and port is provided by the
	// targets (e.g. kubernetes endpoint or service), that port is used.
	port?: int32 @protobuf(1,int32)

	// Whether to resolve the target before making the request. If set to false,
	// we hand over the target golang's net.Dial module, Otherwise, we resolve
	// the target first to an IP address and make a request using that. By
	// default we resolve first if it's a discovered resource, e.g., a k8s
	// endpoint.
	resolveFirst?: bool @protobuf(2,bool,name=resolve_first)

	// Interval between targets.
	intervalBetweenTargetsMsec?: int32 @protobuf(3,int32,name=interval_between_targets_msec,"default=10")
}
