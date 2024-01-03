package proto

#EC2Instances: {
	// How often resources should be refreshed.
	reEvalSec?: int32 @protobuf(98,int32,name=re_eval_sec,"default=600") // default 10 mins
}

// ElastiCacheReplicationGroups discovery options.
#ElastiCacheReplicationGroups: {
	// How often resources should be refreshed.
	reEvalSec?: int32 @protobuf(98,int32,name=re_eval_sec,"default=600") // default 10 mins
}

// ElastiCacheClusters discovery options.
#ElastiCacheClusters: {
	// How often resources should be refreshed.
	reEvalSec?: int32 @protobuf(98,int32,name=re_eval_sec,"default=600") // default 10 mins
}

// RDS (Amazon Relational Databases) Clusters discovery options.
#RDSClusters: {
	// DB cluster identifier or the Amazon Resource Name (ARN) of the DB cluster
	// if specified, only the corresponding cluster information is returned.
	identifier?: string @protobuf(1,string)

	// Filters to be added to the discovery and search.
	filter?: [...string] @protobuf(2,string)

	// Whether to includes information about clusters shared from other AWS accounts.
	includeShared?: bool  @protobuf(3,bool,name=include_shared)
	reEvalSec?:     int32 @protobuf(98,int32,name=re_eval_sec,"default=600") // default 10 mins
}

// RDS (Amazon Relational Databases) Clusters discovery options.
#RDSInstances: {
	// DB cluster identifier or the Amazon Resource Name (ARN) of the DB cluster
	// if specified, only the corresponding cluster information is returned.
	identifier?: string @protobuf(1,string)

	// Filters to be added to the discovery and search.
	filter?: [...string] @protobuf(2,string)

	// Whether to includes information about clusters shared from other AWS accounts.
	includeShared?: bool  @protobuf(3,bool,name=include_shared)
	reEvalSec?:     int32 @protobuf(98,int32,name=re_eval_sec,"default=600") // default 10 mins
}

// LoadBalancers discovery options.
#LoadBalancers: {
	// Amazon Resource Name (ARN) of the load balancer
	// if specified, only the corresponding load balancer information is returned.
	name?: [...string] @protobuf(1,string)
}

// AWS provider config.
#ProviderConfig: {
	// Profile for the session.
	profileName?: string @protobuf(1,string,name=profile_name)

	// AWS region
	region?: string @protobuf(2,string)

	// ECS instances discovery options. This field should be declared for the AWS
	// instances discovery to be enabled.
	ec2Instances?: #EC2Instances @protobuf(3,EC2Instances,name=ec2_instances)

	// ElastiCacheReplicationGroups discovery options. This field should be declared for the
	// elasticache replication groups discovery to be enabled.
	elasticacheReplicationgroups?: #ElastiCacheReplicationGroups @protobuf(4,ElastiCacheReplicationGroups,name=elasticache_replicationgroups)

	// ElastiCacheClusters discovery options. This field should be declared for the
	// elasticache cluster discovery to be enabled.
	elasticacheClusters?: #ElastiCacheClusters @protobuf(5,ElastiCacheClusters,name=elasticache_clusters)

	// RDS instances discovery options.
	rdsInstances?: #RDSInstances @protobuf(6,RDSInstances,name=rds_instances)

	// RDS clusters discovery options.
	rdsClusters?: #RDSClusters @protobuf(7,RDSClusters,name=rds_clusters)
}
