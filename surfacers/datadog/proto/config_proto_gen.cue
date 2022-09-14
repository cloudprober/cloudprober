package proto

// Surfacer config for datadog surfacer.
#SurfacerConf: {
	// Prefix to add to all metrics.
	prefix?: string @protobuf(1,string,#"default="cloudprober""#)

	// Datadog API key. If not set, DD_API_KEY env variable is used.
	apiKey?: string @protobuf(2,string,name=api_key)

	// Datadog APP key. If not set, DD_APP_KEY env variable is used.
	appKey?: string @protobuf(3,string,name=app_key)

	// Datadog server, default: "api.datadoghq.com"
	server?: string @protobuf(4,string)
}
