package proto

#Config: {
	{} | {
		httpRequest: #HTTPRequest @protobuf(3,HTTPRequest,name=http_request)
	} | {
		bearerToken: #BearerToken @protobuf(1,BearerToken,name=bearer_token)
	} | {
		googleCredentials: #GoogleCredentials @protobuf(2,GoogleCredentials,name=google_credentials)
	}

	// How long before the expiry do we refresh. Default is 60 (1m). This applies
	// only to http_request and bearer_token types, and only if token presents
	// expiry in some way.
	// TODO(manugarg): Consider setting default based on probe interval.
	refreshExpiryBufferSec?: int32 @protobuf(20,int32,name=refresh_expiry_buffer_sec)
}

#HTTPRequest: {
	tokenUrl?: string @protobuf(1,string,name=token_url)
	method?:   string @protobuf(2,string)

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
	} @protobuf(8,map[string]string)
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
	} | {
		// K8s service account token file:
		// /var/run/secrets/kubernetes.io/serviceaccount/token
		k8sLocalToken: bool @protobuf(4,bool,name=k8s_local_token)
	}

	// If above sources return JSON tokens with an expiry, we use that info to
	// determine when to refresh tokens and refresh_interval_sec is completely
	// ignored. If above sources return a string, we refresh from the source
	// every 30s by default. To disable this behavior set refresh_interval_sec to
	// zero.
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
