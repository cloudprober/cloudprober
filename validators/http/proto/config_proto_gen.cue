package proto

// HTTP validator configuration. For HTTP validator to succeed, all conditions
// specified in the validator should succeed. Note that failures conditions are
// evaluated before success conditions.
#Validator: {
	// Comma-separated list of success status codes and code ranges.
	// Example: success_stauts_codes: 200-299,301,302
	successStatusCodes?: string @protobuf(1,string,name=success_status_codes)

	// Comma-separated list of failure status codes and code ranges. If HTTP
	// status code matches failure_status_codes, validator fails.
	failureStatusCodes?: string @protobuf(2,string,name=failure_status_codes)

	#Header: {
		// Header name to look for
		name?: string @protobuf(1,string)

		// Header value to match. If omited - check for header existence
		valueRegex?: string @protobuf(2,string,name=value_regex)
	}

	// Header based validations.
	// TODO(manugarg): Add support for specifying multiple success and failure
	// headers.
	//
	// Success Header:
	//   If specified, HTTP response headers should match the success_header for
	//   validation to succeed. Example:
	//     success_header: {
	//       name: "Strict-Transport-Security"
	//       value_regex: "max-age=31536000"
	//     }
	successHeader?: #Header @protobuf(3,Header,name=success_header)

	// Failure Header:
	//   If HTTP response headers match failure_header, validation fails.
	failureHeader?: #Header @protobuf(4,Header,name=failure_header)

	// Last Modified Difference:
	//   If specified, HTTP response's Last-Modified header is checked to be
	//   within the specified time difference from the current time. Example:
	//     max_last_modified_diff_sec: 3600
	//   This will check that the Last-Modified header is within the last hour.
	maxLastModifiedDiffSec?: uint64 @protobuf(5,uint64,name=max_last_modified_diff_sec)
}
