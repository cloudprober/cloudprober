package proto

#Config: {
	{} | {
		bearerToken: #BearerToken @protobuf(1,BearerToken,name=bearer_token)
	} | {
		googleCredentials: #GoogleCredentials @protobuf(2,GoogleCredentials,name=google_credentials)
	}
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
	refreshIntervalSec?: float32 @protobuf(90,float,name=refresh_interval_sec,"default=60")
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
