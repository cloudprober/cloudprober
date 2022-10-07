package proto

// JSON validator configuration.
#Validator: {
	// If jq filter is specified, validator passes only applying jq_filter to the
	// probe output, e.g. HTTP API response, results in a truthy value.
	jqFilter?: string @protobuf(1,string,name=jq_filter)
}
