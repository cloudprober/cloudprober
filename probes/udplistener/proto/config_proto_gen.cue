package proto

#ProbeConf: {
	// Port to listen.
	port?: int32 @protobuf(3,int32,"default=32212")

	// Probe response to an incoming packet: echo back or discard.
	#Type: {"INVALID", #enumValue: 0} |
		{"ECHO", #enumValue: 1} |
		{"DISCARD", #enumValue: 2}

	#Type_value: {
		INVALID: 0
		ECHO:    1
		DISCARD: 2
	}
	type?: #Type @protobuf(4,Type)

	// Number of packets sent in a single probe.
	packetsPerProbe?: int32 @protobuf(5,int32,name=packets_per_probe,"default=1")
}
