package proto

#SurfacerConf: {
	// GCP project name for pubsub. It's required if not running on GCP,
	// otherwise it's retrieved from the metadata.
	project?: string @protobuf(1,string)

	// Pubsub topic name.
	// Default is cloudprober-{hostname}
	topicName?: string @protobuf(2,string,name=topic_name)

	// Compress data before writing to pubsub.
	compressionEnabled?: bool @protobuf(4,bool,name=compression_enabled,"default=false")
}
