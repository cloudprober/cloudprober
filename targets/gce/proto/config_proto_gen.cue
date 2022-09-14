package proto

// TargetsConf represents GCE targets, e.g. instances, forwarding rules etc.
#TargetsConf: {
	// If running on GCE, this defaults to the local project.
	// Note: Multiple projects support in targets is experimental and may go away
	// with future iterations.
	project?: [...string] @protobuf(1,string)
	{} | {
		instances: #Instances @protobuf(2,Instances)
	} | {
		forwardingRules: #ForwardingRules @protobuf(3,ForwardingRules,name=forwarding_rules)
	}
}

// Represents GCE instances
#Instances: {
	// IP address resolving options.

	// Use DNS to resolve target names (instances). If set to false (default),
	// IP addresses specified in the compute.Instance resource is used. If set
	// to true all the other resolving options are ignored.
	useDnsToResolve?: bool @protobuf(1,bool,name=use_dns_to_resolve,"default=false")

	// Get the IP address from Network Interface
	#NetworkInterface: {
		index?: int32 @protobuf(1,int32,"default=0")

		#IPType: {
			// Private IP address.
			"PRIVATE"
			#enumValue: 0
		} | {
			// IP address of the first access config.
			"PUBLIC"
			#enumValue: 1
		} | {
			// First IP address from the first Alias IP range. For example, for
			// alias IP range "192.168.12.0/24", 192.168.12.0 will be returned.
			"ALIAS"
			#enumValue: 2
		}

		#IPType_value: {
			PRIVATE: 0
			PUBLIC:  1
			ALIAS:   2
		}
		ipType?: #IPType @protobuf(2,IPType,name=ip_type,"default=PRIVATE")
	}
	networkInterface?: #NetworkInterface @protobuf(2,NetworkInterface,name=network_interface)

	// Labels to filter instances by ("key:value-regex" format).
	label?: [...string] @protobuf(3,string)
}

// Represents GCE forwarding rules. Does not support multiple projects
#ForwardingRules: {
	// Important: if multiple probes use forwarding_rules targets, only the
	// settings in the definition will take effect.
	// TODO(manugarg): Fix this behavior.
	//
	// For regional forwarding rules, regions to return forwarding rules for.
	// Default is to return forwarding rules from the region that the VM is
	// running in. To return forwarding rules from all regions, specify region as
	// "all".
	region?: [...string] @protobuf(1,string)

	// For global forwarding rules, if it is set to true,  it will ignore
	// the value for the above region property.
	globalRule?: bool @protobuf(2,bool,name=global_rule,"default=false")
}

// Global GCE targets options. These options are independent of the per-probe
// targets which are defined by the "GCETargets" type above.
#GlobalOptions: {
	// How often targets should be evaluated/expanded
	reEvalSec?: int32 @protobuf(1,int32,name=re_eval_sec,"default=900") // default 15 min

	// Compute API version.
	apiVersion?: string @protobuf(2,string,name=api_version,#"default="v1""#)
}
