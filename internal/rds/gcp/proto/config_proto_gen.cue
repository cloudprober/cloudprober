package proto

#GCEInstances: {
	// Optional zone filter regex to limit discovery to the specific zones
	// For example, zone_filter: "us-east1-*" will limit instances discovery to
	// only to the zones in the "us-east1" region.
	zoneFilter?: string @protobuf(1,string,name=zone_filter)

	// How often resources should be refreshed.
	reEvalSec?: int32 @protobuf(98,int32,name=re_eval_sec,"default=300") // default 5 min
}

#ForwardingRules: {
	// Optionl region filter regex to limit discovery to specific regions, e.g.
	// "region_filter:europe-*"
	regionFilter?: string @protobuf(1,string,name=region_filter)

	// How often resources should be refreshed.
	reEvalSec?: int32 @protobuf(98,int32,name=re_eval_sec,"default=300") // default 5 min
}

// Runtime configurator variables.
#RTCVariables: {

	#RTCConfig: {
		name?: string @protobuf(1,string)

		// How often RTC variables should be evaluated/expanded.
		reEvalSec?: int32 @protobuf(2,int32,name=re_eval_sec,"default=10") // default 10 sec
	}
	rtcConfig?: [...#RTCConfig] @protobuf(1,RTCConfig,name=rtc_config)
}

// Runtime configurator variables.
#PubSubMessages: {

	#Subscription: {
		// Subscription name. If it doesn't exist already, we try to create one.
		name?: string @protobuf(1,string)

		// Topic name. This is used to create the subscription if it doesn't exist
		// already.
		topicName?: string @protobuf(2,string,name=topic_name)

		// If subscription already exists, how far back to seek back on restart.
		// Note that duplicate data is fine as we filter by publish time.
		seekBackDurationSec?: int32 @protobuf(3,int32,name=seek_back_duration_sec,"default=3600")
	}
	subscription?: [...#Subscription] @protobuf(1,Subscription)

	// Only for testing.
	apiEndpoint?: string @protobuf(2,string,name=api_endpoint)
}

// GCP provider config.
#ProviderConfig: {
	// GCP projects. If running on GCE, it defaults to the local project.
	project?: [...string] @protobuf(1,string)

	// GCE instances discovery options. This field should be declared for the GCE
	// instances discovery to be enabled.
	gceInstances?: #GCEInstances @protobuf(2,GCEInstances,name=gce_instances)

	// Forwarding rules discovery options. This field should be declared for the
	// forwarding rules discovery to be enabled.
	// Note that RDS supports only regional forwarding rules.
	forwardingRules?: #ForwardingRules @protobuf(3,ForwardingRules,name=forwarding_rules)

	// RTC variables discovery options.
	rtcVariables?: #RTCVariables @protobuf(4,RTCVariables,name=rtc_variables)

	// PubSub messages discovery options.
	pubsubMessages?: #PubSubMessages @protobuf(5,PubSubMessages,name=pubsub_messages)

	// Compute API version.
	apiVersion?: string @protobuf(99,string,name=api_version,#"default="v1""#)

	// Compute API endpoint. Currently supported only for GCE instances and
	// forwarding rules.
	apiEndpoint?: string @protobuf(100,string,name=api_endpoint,#"default="https://www.googleapis.com/compute/""#)
}
