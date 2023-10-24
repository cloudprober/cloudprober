package proto

#HTTPRequest: {
	url?: string @protobuf(1,string)

	// HTTP method
	#Method: {"GET", #enumValue: 0} |
		{"POST", #enumValue: 1} |
		{"PUT", #enumValue: 2} |
		{"DELETE", #enumValue: 3} |
		{"HEAD", #enumValue: 4} |
		{"OPTIONS", #enumValue: 5} |
		{"PATCH", #enumValue: 6}

	#Method_value: {
		GET:     0
		POST:    1
		PUT:     2
		DELETE:  3
		HEAD:    4
		OPTIONS: 5
		PATCH:   6
	}
	method?: #Method @protobuf(2,Method)

	// Data to be sent as request body. If there are multiple "data" fields, we combine
	// their values with a '&' in between. Note: 1) If data appears to be a valid json,
	// we automatically set the content-type header to "application/json", 2) If data
	// appears to be a query string we set content-type to
	// "application/x-www-form-urlencoded". Content type header can still be overridden
	// using the header field below.
	data?: [...string] @protobuf(3,string)

	// HTTP request headers
	header?: {
		[string]: string
	} @protobuf(4,map[string]string)
}
