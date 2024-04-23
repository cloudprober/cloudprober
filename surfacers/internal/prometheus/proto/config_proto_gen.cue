package proto

#SurfacerConf: {
	// How many metrics entries (EventMetrics) to buffer. Incoming metrics
	// processing is paused while serving data to prometheus. This buffer is to
	// make writes to prometheus surfacer non-blocking.
	// NOTE: This field is confusing for users and will be removed from the config
	// after v0.10.3.
	metricsBufferSize?: int64 @protobuf(1,int64,name=metrics_buffer_size,"default=10000")

	// Whether to include timestamps in metrics. If enabled (default) each metric
	// string includes the metric timestamp as recorded in the EventMetric.
	// Prometheus associates the scraped values with this timestamp. If disabled,
	// i.e. timestamps are not exported, prometheus associates scraped values with
	// scrape timestamp.
	includeTimestamp?: bool @protobuf(2,bool,name=include_timestamp,default)

	// URL that prometheus scrapes metrics from.
	metricsUrl?: string @protobuf(3,string,name=metrics_url,#"default="/metrics""#)

	// Prefix to add to all metric names. For example setting this field to
	// "cloudprober_" will result in metrics with names:
	// cloudprober_total, cloudprober_success, cloudprober_latency, ..
	//
	// As it's typically useful to set this across the deployment, this field can
	// also be set through the command line flag --prometheus_metrics_prefix.
	metricsPrefix?: string @protobuf(4,string,name=metrics_prefix)
}
