package proto

import (
	"github.com/cloudprober/cloudprober/surfacers/internal/prometheus/proto"
	proto_1 "github.com/cloudprober/cloudprober/surfacers/internal/stackdriver/proto"
	proto_5 "github.com/cloudprober/cloudprober/surfacers/internal/file/proto"
	proto_A "github.com/cloudprober/cloudprober/surfacers/internal/postgres/proto"
	proto_8 "github.com/cloudprober/cloudprober/surfacers/internal/pubsub/proto"
	proto_E "github.com/cloudprober/cloudprober/surfacers/internal/cloudwatch/proto"
	proto_B "github.com/cloudprober/cloudprober/surfacers/internal/datadog/proto"
	proto_36 "github.com/cloudprober/cloudprober/surfacers/internal/probestatus/proto"
	proto_9 "github.com/cloudprober/cloudprober/surfacers/internal/bigquery/proto"
	proto_3 "github.com/cloudprober/cloudprober/surfacers/internal/otel/proto"
)

// Enumeration for each type of surfacer we can parse and create
#Type: {"NONE", #enumValue: 0} |
	{"PROMETHEUS", #enumValue: 1} |
	{"STACKDRIVER", #enumValue: 2} |
	{"FILE", #enumValue: 3} |
	{"POSTGRES", #enumValue: 4} |
	{"PUBSUB", #enumValue: 5} | {
		"CLOUDWATCH"// Experimental mode.
		#enumValue: 6
	} | {
		"DATADOG"// Experimental mode.
					#enumValue: 7
	} | {"PROBESTATUS", #enumValue: 8} | {
		"BIGQUERY"// Experimental mode.
					#enumValue: 9
	} | {"OTEL", #enumValue: 10} |
	{"USER_DEFINED", #enumValue: 99}

#Type_value: {
	NONE:         0
	PROMETHEUS:   1
	STACKDRIVER:  2
	FILE:         3
	POSTGRES:     4
	PUBSUB:       5
	CLOUDWATCH:   6
	DATADOG:      7
	PROBESTATUS:  8
	BIGQUERY:     9
	OTEL:         10
	USER_DEFINED: 99
}

#LabelFilter: {
	key?:   string @protobuf(1,string)
	value?: string @protobuf(2,string)
}

#SurfacerDef: {
	// This name is used for logging. If not defined, it's derived from the type.
	// Note that this field is required for the USER_DEFINED surfacer type and
	// should match with the name that you used while registering the user defined
	// surfacer.
	name?: string @protobuf(1,string)
	type?: #Type  @protobuf(2,Type)

	// How many metrics entries (EventMetrics) to buffer. This is the buffer
	// between incoming metrics and the metrics that are being processed. Default
	// value should work in most cases. You may need to increase it on a busy
	// system, but that's usually a sign that you metrics processing pipeline is
	// slow for some reason, e.g. slow writes to a remote file.
	// Note: Only file and pubsub surfacer supports this option right now.
	metricsBufferSize?: int64 @protobuf(3,int64,name=metrics_buffer_size,"default=10000")

	// If specified, only allow metrics that match any of these label filters.
	// Example:
	// allow_metrics_with_label {
	//   key: "probe",
	//   value: "check_homepage",
	// }
	allowMetricsWithLabel?: [...#LabelFilter] @protobuf(4,LabelFilter,name=allow_metrics_with_label)

	// Ignore metrics that match any of these label filters. Ignore has precedence
	// over allow filters.
	// Example:
	// ignore_metrics_with_label {
	//   key: "probe",
	//   value: "sysvars",
	// }
	ignoreMetricsWithLabel?: [...#LabelFilter] @protobuf(5,LabelFilter,name=ignore_metrics_with_label)

	// Allow and ignore metrics based on their names. You can specify regexes
	// here. Ignore has precendence over allow.
	// Examples:
	//  ignore_metrics_with_name: "validation_failure"
	//  allow_metrics_with_name: "(total|success|latency)"
	//
	// For efficiency reasons, filtering by metric name has to be implemented by
	// individual surfacers (while going through metrics within an EventMetrics).
	// As FILE and PUBSUB surfacers export eventmetrics as is, they don't support
	// this option.
	allowMetricsWithName?:  string @protobuf(6,string,name=allow_metrics_with_name)
	ignoreMetricsWithName?: string @protobuf(7,string,name=ignore_metrics_with_name)

	// Whether to add failure metric or not. This option is enabled by default
	// for all surfacers except FILE and PUBSUB.
	addFailureMetric?: bool @protobuf(8,bool,name=add_failure_metric)

	// If set to true, cloudprober will export all metrics as gauge metrics. Note
	// that cloudprober inherently generates only cumulative metrics. To create
	// gauge metrics from cumulative metrics, we keep a copy of the old metrics
	// and subtract new metrics from the previous metrics. This transformation in
	// metrics has an increased memory-overhead because extra copies required.
	// However, it should not be noticeable unless you're producing large number
	// of metrics (say > 10000 metrics per second).
	exportAsGauge?: bool @protobuf(9,bool,name=export_as_gauge)

	// Latency metric name pattern, used to identify latency metrics, and add
	// EventMetric's LatencyUnit to it.
	latencyMetricPattern?: string @protobuf(51,string,name=latency_metric_pattern,#"default="^(.+_|)latency$""#)
	// Matching surfacer specific configuration (one for each type in the above
	// enum)
	{} | {
		prometheusSurfacer: proto.#SurfacerConf @protobuf(10,prometheus.SurfacerConf,name=prometheus_surfacer)
	} | {
		stackdriverSurfacer: proto_1.#SurfacerConf @protobuf(11,stackdriver.SurfacerConf,name=stackdriver_surfacer)
	} | {
		fileSurfacer: proto_5.#SurfacerConf @protobuf(12,file.SurfacerConf,name=file_surfacer)
	} | {
		postgresSurfacer: proto_A.#SurfacerConf @protobuf(13,postgres.SurfacerConf,name=postgres_surfacer)
	} | {
		pubsubSurfacer: proto_8.#SurfacerConf @protobuf(14,pubsub.SurfacerConf,name=pubsub_surfacer)
	} | {
		cloudwatchSurfacer: proto_E.#SurfacerConf @protobuf(15,cloudwatch.SurfacerConf,name=cloudwatch_surfacer)
	} | {
		datadogSurfacer: proto_B.#SurfacerConf @protobuf(16,datadog.SurfacerConf,name=datadog_surfacer)
	} | {
		probestatusSurfacer: proto_36.#SurfacerConf @protobuf(17,probestatus.SurfacerConf,name=probestatus_surfacer)
	} | {
		bigquerySurfacer: proto_9.#SurfacerConf @protobuf(18,bigquery.SurfacerConf,name=bigquery_surfacer)
	} | {
		otelSurfacer: proto_3.#SurfacerConf @protobuf(19,otel.SurfacerConf,name=otel_surfacer)
	}
}
