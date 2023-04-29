package proto

// Notify is not implemented yet. We just log a warning when there is an alert.
#Notify: {
	// Command to run when alert is fired.
	command?: string @protobuf(1,string)
}

#AlertConf: {
	// Name of the alert. Default is to use the probe name.
	name?: string @protobuf(1,string)

	// Thresholds for the alert.
	failureThreshold?: float32 @protobuf(2,float,name=failure_threshold)

	// Duration threshold in seconds. If duration_threshold_sec is set, alert
	// will be fired only if alert condition is true for
	// duration_threshold_sec.
	durationThresholdSec?: int32 @protobuf(3,int32,name=duration_threshold_sec)

	// How to notify in case of alert.
	notify?: #Notify @protobuf(4,Notify)
}
