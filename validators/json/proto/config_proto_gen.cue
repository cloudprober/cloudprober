package proto

// JSON validator configuration.
#Validator: {
	// If jq filter is specified, validator passes only if applying jq_filter to
	// the probe output, e.g. HTTP API response, results in 'true' boolean.
	// See the following test file for some examples:
	// https://github.com/cloudprober/cloudprober/blob/master/validators/json/json_test.go
	jqFilter?: string @protobuf(1,string,name=jq_filter)
}
