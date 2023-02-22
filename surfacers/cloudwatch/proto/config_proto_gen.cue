package proto

#SurfacerConf: {
	// The cloudwatch metric namespace
	namespace?: string @protobuf(1,string,#"default="cloudprober""#)

	// The cloudwatch resolution value, lowering this below 60 will incur
	// additional charges as the metrics will be charged at a high resolution
	// rate.
	resolution?: int32 @protobuf(2,int32,"default=60")

	// The AWS Region, used to create a CloudWatch session.
	// The order of fallback for evaluating the AWS Region:
	// 1. This config value.
	// 2. EC2 metadata endpoint, via cloudprober sysvars.
	// 3. AWS_REGION environment value.
	// 4. AWS_DEFAULT_REGION environment value, if AWS_SDK_LOAD_CONFIG is set.
	// https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
	region?: string @protobuf(3,string)

	// The maximum number of metrics that will be published at one
	// time. Metrics will be stored locally in a cache until this
	// limit is reached. 1000 is the maximum number of metrics
	// supported by the Cloudwatch PutMetricData API.
	// Metrics will be published when the timer expires, or the buffer is
	// full, whichever happens first.
	metricsBatchSize?: int32 @protobuf(4,int32,name=metrics_batch_size,"default=1000")

	// The maximum amount of time to hold metrics in the buffer (above).
	// Metrics will be published when the timer expires, or the buffer is
	// full, whichever happens first.
	batchTimerSec?: int32 @protobuf(5,int32,name=batch_timer_sec,"default=30")
}
