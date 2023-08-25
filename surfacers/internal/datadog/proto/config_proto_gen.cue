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

	// The maximum number of metrics that will be published at one
	// time. Metrics will be stored locally in a cache until this
	// limit is reached. Datadog's SubmitMetric API has a maximum payload
	// size of 500 kilobytes (512000 bytes). Compressed payloads must have a
	// decompressed size of less than 5 megabytes (5242880 bytes).
	// Metrics will be published when the timer expires, or the buffer is
	// full, whichever happens first.
	metricsBatchSize?: int32 @protobuf(5,int32,name=metrics_batch_size,"default=1000")

	// The maximum amount of time to hold metrics in the buffer (above).
	// Metrics will be published when the timer expires, or the buffer is
	// full, whichever happens first.
	batchTimerSec?: int32 @protobuf(6,int32,name=batch_timer_sec,"default=30")

	// Disable gzip compression of metric payload, when sending metrics to Datadog.
	// Compression is enabled by default.
	disableCompression?: bool @protobuf(7,bool,name=disable_compression)
}
