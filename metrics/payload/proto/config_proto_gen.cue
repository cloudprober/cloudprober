package proto

import "github.com/cloudprober/cloudprober/metrics/proto"

#OutputMetricsOptions: {
	// MetricsKind specifies whether to treat output metrics as GAUGE or
	// CUMULATIVE. If left unspecified, metrics from ONCE mode probes are treated
	// as GAUGE and metrics from SERVER mode probes are treated as CUMULATIVE.
	#MetricsKind: {"UNDEFINED", #enumValue: 0} |
		{"GAUGE", #enumValue: 1} |
		{"CUMULATIVE", #enumValue: 2}

	#MetricsKind_value: {
		UNDEFINED:  0
		GAUGE:      1
		CUMULATIVE: 2
	}
	metricsKind?: #MetricsKind @protobuf(1,MetricsKind,name=metrics_kind)

	// Additional labels (comma-separated) to attach to the output metrics, e.g.
	// "region=us-east1,zone=us-east1-d". ptype="external" and probe="<probeName>"
	// are attached automatically.
	additionalLabels?: string @protobuf(2,string,name=additional_labels)

	// Whether to aggregate metrics in Cloudprober. If enabled, Cloudprober
	// aggregates the metrics returned by the external probe process -- external
	// probe process should return metrics only since the last probe run.
	// Note that this option is mutually exclusive with GAUGE metrics and
	// cloudprober will fail during initialization if both options are enabled.
	aggregateInCloudprober?: bool @protobuf(3,bool,name=aggregate_in_cloudprober,"default=false")

	// Metrics that should be treated as distributions. These metrics are exported
	// by the external probe program as comma-separated list of values, for
	// example: "op_latency 4.7,5.6,5.9,6.1,4.9". To be able to build distribution
	// from these values, these metrics should be pre-configured in external
	// probe:
	// dist_metric {
	//   key: "op_latency"
	//   value {
	//     explicit_buckets: "1,2,4,8,16,32,64,128,256"
	//   }
	// }
	distMetric?: {
		[string]: proto.#Dist
	} @protobuf(4,map[string]metrics.Dist,dist_metric)
}
