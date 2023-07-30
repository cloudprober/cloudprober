package proto

#NotifyConfig: {
	// Command to run when alert is fired. In the command line following fields
	// are substituted:
	//  @alert@: Alert name
	//  @probe@: Probe name
	//  @target@: Target name, or target and port if port is specified.
	//  @target.label.<label>@: Label <label> value, e.g. target.label.role.
	//  @failures@: Count of failures.
	//  @total@: Out of.
	//  @since@: Time since the alert condition started.
	//  @json@: JSON representation of the alert fields.
	//
	// For example, if you want to send an email when an alert is fired, you can
	// use the following command:
	// command: "/usr/bin/mail -s 'Alert @alert@ fired for @target@' manu@a.b"
	command?: string @protobuf(10,string)

	// Email addresses to send alerts to. For email notifications to work,
	// following environment variables must be set:
	// - SMTP_SERVER: SMTP server and port to use for sending emails.
	// - SMTP_USERNAME: SMTP user name.
	// - SMTP_PASSWORD: SMTP password.
	email?: [...string] @protobuf(11,string)
	emailFrom?: string @protobuf(12,string,name=email_from) // Default: SMTP_USERNAME

	// Enable PagerDuty alerts. If set to true, alerts will be sent to
	// PagerDuty.
	pagerduty?: bool @protobuf(31,bool) // Default: false

	// PagerDuty URL.
	pagerdutyUrl?: string @protobuf(32,string,name=pagerduty_url) // Default: https://api.pagerduty.com

	// PagerDuty API Token.
	// PagerDuty API key can also be set using the environment variable:
	// PAGERDUTY_API_TOKEN.
	// The environment variable takes precedence over this field.
	pagerdutyApiToken?: string @protobuf(33,string,name=pagerduty_api_token)

	// PagerDuty routing key.
	// PagerDuty routing key can also be set using the environment variable:
	// PAGERDUTY_ROUTING_KEY.
	// The environment variable takes precedence over this field.
	pagerdutyRoutingKey?: string @protobuf(34,string,name=pagerduty_routing_key)
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

	// Dashboard URL template.
	// Default: http://localhost:9313/status?probe=@probe@
	dashboardUrlTemplate?: string @protobuf(4,string,name=dashboard_url_template) // Default: ""
	playbookUrlTemplate?:  string @protobuf(5,string,name=playbook_url_template)  // Default: ""

	// Default: "Cloudprober alert @alert@ for @target@"
	summaryTemplate?: string @protobuf(6,string,name=summary_template)

	// Default:
	// Cloudprober alert "@alert@" for "@target@":
	// Failures: @failures@ out of @total@ probes
	// Failing since: @since@
	// Probe: @probe@
	// Dashboard: @dashboard_url@
	// Playbook: @playbook_url@
	// Condition ID: @condition_id@
	detailsTemplate?: string @protobuf(7,string,name=details_template) // Default: ""

	// How often to repeat notification for the same alert. Default is 1hr.
	// To disable any kind of notification throttling, set this to 0.
	repeatIntervalSec?: int32 @protobuf(8,int32,name=repeat_interval_sec) // Default: 1hr
}
