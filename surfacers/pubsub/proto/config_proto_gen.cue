package proto

#SurfacerConf: {
	// GCP project name for pubsub. If not specified and running on GCP,
	// project is used.
	project?: string @protobuf(1,string)

	// Pubsub topic name.
	// Default is cloudprober-{hostname}
	topicName?: string @protobuf(2,string,name=topic_name)

	// Compress data before writing to pubsub.
	compressionEnabled?: bool @protobuf(4,bool,name=compression_enabled,"default=false")
}
