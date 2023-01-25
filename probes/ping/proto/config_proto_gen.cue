package proto

// Next tag: 1
#ProbeConf: {
	// Packets per probe
	packetsPerProbe?: int32 @protobuf(6,int32,name=packets_per_probe,"default=2")

	// How long to wait between two packets to the same target
	packetsIntervalMsec?: int32 @protobuf(7,int32,name=packets_interval_msec,"default=25")

	// Resolve targets after these many probes
	resolveTargetsInterval?: int32 @protobuf(9,int32,name=resolve_targets_interval,"default=5") // =10s

	// Ping payload size in bytes. It cannot be smaller than 8, number of bytes
	// required for the nanoseconds timestamp.
	payloadSize?: int32 @protobuf(10,int32,name=payload_size,"default=56")

	// Use datagram socket for ICMP.
	// This option enables unprivileged pings (that is, you don't require root
	// privilege to send ICMP packets). Note that most of the Linux distributions
	// don't allow unprivileged pings by default. To enable unprivileged pings on
	// some Linux distributions, you may need to run the following command:
	//
	//     sudo sysctl -w net.ipv4.ping_group_range="0 <large valid group id>"
	//
	// net.ipv4.ping_group_range system setting takes two integers that specify
	// the group id range that is allowed to execute the unprivileged pings. Note
	// that the same setting (with ipv4 in the path) applies to IPv6 as well.
	//
	// Note: This option is not supported on Windows and is automatically
	// disabled there.
	useDatagramSocket?: bool @protobuf(12,bool,name=use_datagram_socket,default)

	// Disable integrity checks. To detect data courruption in the network, we
	// craft the outgoing ICMP packet payload in a certain format and verify that
	// the reply payload matches the same format.
	disableIntegrityCheck?: bool @protobuf(13,bool,name=disable_integrity_check,"default=false")

	// Do not allow OS-level fragmentation, only works on Linux systems.
	disableFragmentation?: bool @protobuf(14,bool,name=disable_fragmentation,"default=false")
}
