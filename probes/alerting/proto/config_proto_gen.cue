package proto

#NotifyConfig: {
	// Command to run when alert is fired. In the command line following fields
	// are substituted:
	//  @alert@: Alert name
	//  @probe@: Probe name
	//  @target@: Target name, or target and port if port is specified.
	//  @target.label.<label>@: Label <label> value, e.g. target.label.role.
	//  @value@: Value that triggered the alert.
	//  @threshold@: Threshold that was crossed.
	//  @since@: Time since the alert condition started.
	//  @json@: JSON representation of the alert fields.
	//
	// For example, if you want to send an email when an alert is fired, you can
	// use the following command:
	// command: "/usr/bin/mail -s 'Alert @alert@ fired for @target@' manu@a.b"
	command?: string @protobuf(1,string)
}

#Condition: {
	failures?: int32 @protobuf(1,int32)
	total?:    int32 @protobuf(2,int32)
}

#AlertConf: {
	// Name of the alert. Default is to use the probe name.
	name?: string @protobuf(1,string)

	// Condition for the alert. Default is to alert on any failure.
	// Example:
	// # Alert if 6 out of 10 probes fail.
	// condition {
	//   failures: 6
	//   total: 10
	// }
	condition?: #Condition @protobuf(2,Condition)

	// How to notify in case of alert.
	notify?: #NotifyConfig @protobuf(3,NotifyConfig)
}
