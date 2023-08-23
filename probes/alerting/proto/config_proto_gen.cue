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

	// PagerDuty configuration.
	pagerDuty?: #PagerDuty @protobuf(30,PagerDuty,name=pager_duty)

	// Slack configuration.
	slack?: #Slack @protobuf(31,Slack)
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
