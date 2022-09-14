package proto

#SurfacerConf: {
	// The cloudwatch metric namespace
	namespace?: string @protobuf(1,string,#"default="cloudprober""#)

	// The cloudwatch resolution value, lowering this below 60 will incur
	// additional charges as the metrics will be charged at a high resolution
	// rate.
	resolution?: int64 @protobuf(2,int64,"default=60")

	// The AWS Region, used to create a CloudWatch session.
	// The order of fallback for evaluating the AWS Region:
	// 1. This config value.
	// 2. EC2 metadata endpoint, via cloudprober sysvars.
	// 3. AWS_REGION environment value.
	// 4. AWS_DEFAULT_REGION environment value, if AWS_SDK_LOAD_CONFIG is set.
	// https://docs.aws.amazon.com/sdk-for-go/api/aws/session/
	region?: string @protobuf(3,string)
}
