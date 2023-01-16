package proto

#ProbeConf: {
	// Port to send UDP Ping to (UDP Echo).  If running with the UDP server that
	// comes with cloudprober, it should be same as
	// ProberConfig.udp_echo_server_port.
	port?: int32 @protobuf(3,int32,"default=31122")

	// Number of sending side ports to use.
	numTxPorts?: int32 @protobuf(4,int32,name=num_tx_ports,"default=16")

	// message max to account for MTU.
	maxLength?: int32 @protobuf(5,int32,name=max_length,"default=1300")

	// Payload size
	payloadSize?: int32 @protobuf(6,int32,name=payload_size)

	// Changes the exported monitoring streams to be per port:
	// 1. Changes the streams names to total-per-port, success-per-port etc.
	// 2. Adds src_port and dst_port as stream labels.
	// Note that the field name is experimental and may change in the future.
	exportMetricsByPort?: bool @protobuf(7,bool,name=export_metrics_by_port,"default=false")

	// Whether to use all transmit ports per probe, per target.
	// Default is to probe each target once per probe and round-robin through the
	// source ports.
	// Setting this field to true changes the behavior to send traffic from all
	// ports to all targets in each probe.
	// For example, if num_tx_ports is set to 16, in every probe cycle, we'll send
	// 16 packets to every target (1 per tx port).
	// Note that setting this field to true will increase the probe traffic.
	useAllTxPortsPerProbe?: bool @protobuf(8,bool,name=use_all_tx_ports_per_probe,"default=false")

	// maxTargets is the maximum number of targets supported by this probe type.
	// If there are more targets, they are pruned from the list to bring targets
	// list under maxTargets.  A large number of targets has impact on resource
	// consumption.
	maxTargets?: int32 @protobuf(9,int32,name=max_targets,"default=500")
}
