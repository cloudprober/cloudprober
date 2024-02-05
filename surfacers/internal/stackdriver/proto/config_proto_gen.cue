package proto

#SurfacerConf: {
	// GCP project name for stackdriver. If not specified and running on GCP,
	// project is used.
	project?: string @protobuf(1,string)

	// How often to export metrics to stackdriver.
	batchTimerSec?: uint64 @protobuf(2,uint64,name=batch_timer_sec,"default=10")

	// If allowed_metrics_regex is specified, only metrics matching the given
	// regular expression will be exported to stackdriver. Since probe type and
	// probe name are part of the metric name, you can use this field to restrict
	// stackdriver metrics to a particular probe.
	// Example:
	// allowed_metrics_regex: ".*(http|ping).*(success|validation_failure).*"
	//
	// Deprecated: Please use the common surfacer options to filter metrics:
	// https://cloudprober.org/docs/surfacers/overview/#filtering-metrics
	allowedMetricsRegex?: string @protobuf(3,string,name=allowed_metrics_regex)

	// Monitoring URL base. Full metric URL looks like the following:
	// <monitoring_url>/<ptype>/<probe>/<metric>
	// Example:
	// custom.googleapis.com/cloudprober/http/google-homepage/latency
	monitoringUrl?: string @protobuf(4,string,name=monitoring_url,#"default="custom.googleapis.com/cloudprober/""#)

	// How many metrics entries to buffer. Incoming metrics
	// processing is paused while serving data to Stackdriver. This buffer is to
	// make writes to Stackdriver surfacer non-blocking.
	metricsBufferSize?: int64 @protobuf(5,int64,name=metrics_buffer_size,"default=10000")

	#MetricPrefix: {
		"NONE"// monitoring_url/metric_name
		#enumValue: 0
	} | {
		"PROBE"// monitoring_url/probe/metric_name
		#enumValue: 1
	} | {
		"PTYPE_PROBE"// monitoring_url/ptype/probe/metric_name
		#enumValue: 2
	}

	#MetricPrefix_value: {
		NONE:        0
		PROBE:       1
		PTYPE_PROBE: 2
	}

	// Metric prefix to use for stackdriver metrics. If not specified, default
	// is PTYPE_PROBE.
	metricsPrefix?: #MetricPrefix @protobuf(6,MetricPrefix,name=metrics_prefix,"default=PTYPE_PROBE")
}
