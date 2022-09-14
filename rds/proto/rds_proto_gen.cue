package proto

#ListResourcesRequest: {
	// Resources to list and the associated IP address that we are interested in.
	// Example:
	// {
	//   provider: "gcp"
	//   resource_path: "gce_instances/project-1"
	//   filter {
	//     key: "name"
	//     value: "ig-us-central1-.*"
	//   }
	//   ip_config {
	//     ip_type: PUBLIC
	//   }
	// }

	// Provider is the resource list provider, for example: "gcp", "aws", etc.
	provider?: string @protobuf(1,string)

	// Provider specific resource path. For example: for GCP, it could be
	// "gce_instances/<project>", "regional_forwarding_rules/<project>", etc.
	resourcePath?: string @protobuf(2,string,name=resource_path)

	// Filters for the resources list. Filters are ANDed: all filters should
	// succeed for an item to included in the result list.
	filter?: [...#Filter] @protobuf(3,Filter)

	// Optional. If resource has an IP (and a NIC) address, following
	// fields determine which IP address will be included in the results.
	ipConfig?: #IPConfig @protobuf(4,IPConfig,name=ip_config)

	// If specified, and if provider supports it, server will send resources in
	// the response only if they have changed since the given timestamp. Since
	// there may be no resources in the response for non-caching reasons as well,
	// clients should use the "last_modified" field in the response to determine
	// if they need to update the local cache or not.
	ifModifiedSince?: int64 @protobuf(5,int64,name=if_modified_since)
}

#Filter: {
	key?:   string @protobuf(1,string)
	value?: string @protobuf(2,string)
}

#IPConfig: {
	// NIC index
	nicIndex?: int32 @protobuf(1,int32,name=nic_index,"default=0")

	#IPType: {
		// Default IP of the resource.
		//  - Private IP for instance resource
		//  - Forwarding rule IP for forwarding rule.
		"DEFAULT"
		#enumValue: 0
	} | {
		// Instance's external IP.
		"PUBLIC"
		#enumValue: 1
	} | {
		// First IP address from the first Alias IP range. For example, for
		// alias IP range "192.168.12.0/24", 192.168.12.0 will be returned.
		// Supported only on GCE.
		"ALIAS"
		#enumValue: 2
	}

	#IPType_value: {
		DEFAULT: 0
		PUBLIC:  1
		ALIAS:   2
	}
	ipType?: #IPType @protobuf(3,IPType,name=ip_type)

	#IPVersion: {"IP_VERSION_UNSPECIFIED", #enumValue: 0} |
		{"IPV4", #enumValue: 1} |
		{"IPV6", #enumValue: 2}

	#IPVersion_value: {
		IP_VERSION_UNSPECIFIED: 0
		IPV4:                   1
		IPV6:                   2
	}
	ipVersion?: #IPVersion @protobuf(2,IPVersion,name=ip_version)
}

#Resource: {
	// Resource name.
	name?: string @protobuf(1,string)

	// Resource's IP address, selected based on the request's ip_config.
	ip?: string @protobuf(2,string)

	// Resource's port, if any.
	port?: int32 @protobuf(5,int32)

	// Resource's labels, if any.
	labels?: {
		[string]: string
	} @protobuf(6,map[string]string)

	// Last updated (in unix epoch).
	lastUpdated?: int64 @protobuf(7,int64,name=last_updated)

	// Id associated with the resource, if any.
	id?: string @protobuf(3,string)

	// Optional info associated with the resource. Some resource type may make use
	// of it.
	info?: bytes @protobuf(4,bytes)
}

#ListResourcesResponse: {
	// There may not be any resources in the response if request contains the
	// "if_modified_since" field and provider "knows" that nothing has changed since
	// the if_modified_since timestamp.
	resources?: [...#Resource] @protobuf(1,Resource)

	// When were resources last modified. This field will always be set if
	// provider has a way of figuring out last_modified timestamp for its
	// resources.
	lastModified?: int64 @protobuf(2,int64,name=last_modified)
}
