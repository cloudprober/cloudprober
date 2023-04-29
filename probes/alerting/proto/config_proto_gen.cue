package proto

#Notify: {
	// Command to run when alert is fired. In the command line following fields
	// are substituted:
	//  @alert@: Alert name
	//  @probe@: Probe name
	//  @target@: Target name, or target and port if port is specified.
	//  @target.label.<label>@: Label <label> value, e.g. target.label.role.
	//  @value@: Value that triggered the alert.
	//  @threshold@: Threshold that was crossed.
	//  @since@: Time since the alert condition started.
	//
	// For example, if you want to send an email when an alert is fired, you can
	// use the following command:
	// command: "/usr/bin/mail -s 'Alert @alert@ fired for @target@' manu@a.b"
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
