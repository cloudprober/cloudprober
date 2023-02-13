package proto

#Config: {
	{} | {
		httpRequest: #HTTPRequest @protobuf(3,HTTPRequest,name=http_request)
	} | {
		bearerToken: #BearerToken @protobuf(1,BearerToken,name=bearer_token)
	} | {
		googleCredentials: #GoogleCredentials @protobuf(2,GoogleCredentials,name=google_credentials)
	}
}

#HTTPRequest: {
	tokenUrl?: string @protobuf(1,string,name=token_url)
	method?:   string @protobuf(2,string)

	// data can be repeated. If it is repeated we combine the values with a '&'
	// in between. Note: 1) If data appears to be a valid json, we automatically set
	// the content-type header to "application/json", 2) If data appears to be a query
	// string we set content-type to "application/x-www-form-urlencoded". Content type
	// header can still be overridden using the header field below.
	data?: [...string] @protobuf(3,string)

	#Header: {
		name?:  string @protobuf(1,string)
		value?: string @protobuf(2,string)
	}

	// HTTP request headers
	header?: [...#Header] @protobuf(8,Header)
}

// Bearer token is added to the HTTP request through an HTTP header:
// "Authorization: Bearer <access_token>"
#BearerToken: {
	{} | {
		// Path to token file.
		file: string @protobuf(1,string)
	} | {
		// Run a comand to obtain the token, e.g.
		// cat /var/lib/myapp/token, or
		// /var/lib/run/get_token.sh
		cmd: string @protobuf(2,string)
	} | {
		// GCE metadata token
		gceServiceAccount: string @protobuf(3,string,name=gce_service_account)
	}

	// How often to refresh token. As OAuth token usually expire, we need to
	// refresh them on a regular interval. If set to 0, caching is disabled.
	// Default is 60s.
	refreshIntervalSec?: float32 @protobuf(90,float,name=refresh_interval_sec)
}

// Google credentials in JSON format. We simply use oauth2/google package to
// use these credentials.
#GoogleCredentials: {
	jsonFile?: string @protobuf(1,string,name=json_file)
	scope?: [...string] @protobuf(2,string)

	// Use encoded JWT directly as access token, instead of implementing the whole
	// OAuth2.0 flow.
	jwtAsAccessToken?: bool @protobuf(4,bool,name=jwt_as_access_token)

	// Audience works only if jwt_as_access_token is true.
	audience?: string @protobuf(3,string)
}
