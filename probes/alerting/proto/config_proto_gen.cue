package proto

#Email: {
	// Email addresses to send the alert to.
	to?: [...string] @protobuf(1,string)

	// From address in the alert email.
	// If not set, defaults to the value of smtp_user if smtp_user is set,
	// otherwise defaults to cloudprober-alert@<hostname>.
	from?: string @protobuf(2,string)

	// Default: Environment variable SMTP_SERVER
	smtpServer?: string @protobuf(3,string,name=smtp_server)

	// Default: Environment variable SMTP_USERNAME
	smtpUsername?: string @protobuf(4,string,name=smtp_username)

	// Default: Environment variable SMTP_PASSWORD
	smtpPassword?: string @protobuf(5,string,name=smtp_password)
}

#PagerDuty: {
	// PagerDuty Routing Key.
	// The routing key is used to determine which service the alerts are sent to
	// and is generated with the service. The routing key is found under the
	// service, when the events v2 integration is enabled, under integrations,
	// in the pagerduty console.
	// Note: set either routing_key or routing_key_env_var. routing_key
	// takes precedence over routing_key_env_var.
	routingKey?: string @protobuf(1,string,name=routing_key)

	// The environment variable that is used to contain the pagerduty routing
	// key.
	routingKeyEnvVar?: string @protobuf(2,string,name=routing_key_env_var) // Default: PAGERDUTY_ROUTING_KEY;

	// PagerDuty API URL.
	// Used to overwrite the default PagerDuty API URL.
	apiUrl?: string @protobuf(3,string,name=api_url) // Default: https://event.pagerduty.com

	// Whether to send resolve notifications or not. Default is to send resolve
	// notifications.
	disableSendResolved?: bool @protobuf(4,bool,name=disable_send_resolved) // Default: false
}

#Slack: {
	// Webhook URL
	// The Slack notifications use a webhook URL to send the notifications to
	// a Slack channel. The webhook URL can be found in the Slack console under
	// the "Incoming Webhooks" section.
	// https://api.slack.com/messaging/webhooks
	// Note: set either webhook_url or webhook_url_env_var. webhook_url
	// takes precedence over webhook_url_env_var.
	webhookUrl?: string @protobuf(1,string,name=webhook_url)

	// The environment variable that is used to contain the slack webhook URL.
	webhookUrlEnvVar?: string @protobuf(2,string,name=webhook_url_env_var) // Default: SLACK_WEBHOOK_URL;
}

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

	// Email notification configuration.
	email?: #Email @protobuf(11,Email)

	// PagerDuty configuration.
	pagerDuty?: #PagerDuty @protobuf(12,PagerDuty,name=pager_duty)

	// Slack configuration.
	slack?: #Slack @protobuf(13,Slack)
}

#Condition: {
	failures?: int32 @protobuf(1,int32)
	total?:    int32 @protobuf(2,int32)
}

#AlertConf: {
	// Name of the alert. Default is to use the probe name. If you have multiple
	// alerts for the same probe, you must specify a name for each alert.
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

	// Default: Cloudprober alert "@alert@" for "@target@"
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

	// Key values to be included in the alert. These fields are expanded
	// using the same template expansion rules as summary_template and
	// details_template (see above).
	otherInfo?: {
		[string]: string
	} @protobuf(9,map[string]string,other_info)

	// Severity of the alert. Default is "ERROR".
	#Severity: {"UNKNOWN_SEVERITY", #enumValue: 0} |
		{"CRITICAL", #enumValue: 1} |
		{"ERROR", #enumValue: 2} |
		{"WARNING", #enumValue: 3} |
		{"INFO", #enumValue: 4}

	#Severity_value: {
		UNKNOWN_SEVERITY: 0
		CRITICAL:         1
		ERROR:            2
		WARNING:          3
		INFO:             4
	}

	severity?: #Severity @protobuf(10,Severity) // Default: ERROR

	// How often to repeat notification for the same alert. Default is 1hr.
	// To disable any kind of notification throttling, set this to 0.
	repeatIntervalSec?: int32 @protobuf(8,int32,name=repeat_interval_sec) // Default: 1hr
}
