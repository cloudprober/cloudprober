package proto

import "github.com/cloudprober/cloudprober/internal/httpreq/proto"

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

#Opsgenie: {
	// API key to access Opsgenie. It's usually tied to a team and is
	// obtained by creating a new API integration or using an existing one.
	apiKey?: string @protobuf(1,string,name=api_key)

	// Environment variable name Default: OPSGENIE_API_KEY
	apiKeyEnvVar?: string @protobuf(2,string,name=api_key_env_var)

	#Responder: {
		{} | {
			id: string @protobuf(1,string)
		} | {
			name: string @protobuf(2,string)
		}

		#Type: {"UNKNOWN_RESPONDER", #enumValue: 0} |
			{"USER", #enumValue: 1} |
			{"TEAM", #enumValue: 2} |
			{"ESCALATION", #enumValue: 3} |
			{"SCHEDULE", #enumValue: 4}

		#Type_value: {
			UNKNOWN_RESPONDER: 0
			USER:              1
			TEAM:              2
			ESCALATION:        3
			SCHEDULE:          4
		}
		type?: #Type @protobuf(3,Type)
	}

	// Opsgenie responders. Opsgenie uses the responders to route the alerts if
	// API key doesn't belong to a team integration.
	// Example:
	//  responders {
	//    id: "4513b7ea-3b91-438f-b7e4-e3e54af9147c"
	//    type: TEAM
	//  }
	responders?: [...#Responder] @protobuf(3,Responder)

	// Opsgenie API URL.
	// Default: https://api.opsgenie.com/v2/alerts
	apiUrl?: string @protobuf(4,string,name=api_url)

	// Whether to send resolve notifications or not. Default is to send resolve
	// notifications.
	disableSendResolved?: bool @protobuf(5,bool,name=disable_send_resolved) // Default: false
}

#PagerDuty: {
	// PagerDuty Routing Key.
	// The routing key is used to authenticate to PagerDuty and is tied to a
	// service. You can obtain the routing key from the service page, under the
	// integrations tab.
	// Note: set either routing_key or routing_key_env_var. routing_key
	// takes precedence over routing_key_env_var.
	routingKey?: string @protobuf(1,string,name=routing_key)

	// The environment variable containing the pagerduty routing key.
	// Default: PAGERDUTY_ROUTING_KEY;
	routingKeyEnvVar?: string @protobuf(2,string,name=routing_key_env_var)

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
	// Command to run when alert is fired. You can use this command to do
	// various things, e.g.:
	//  - Send a notification using a method not supported by Cloudprober.
	//  - Collect more information, e.g. send mtr report on ping failures.
	//  - Attempt fix the issue, e.g. restart a pod or clear cache.
	//
	// In the command line following fields are substituted:
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

	// Opsgenie configuration.
	opsgenie?: #Opsgenie @protobuf(14,Opsgenie)

	// Notify using an HTTP request. HTTP request fields are expanded using the
	// same template expansion rules as "command" above:
	// For example, to send a notification using rest API:
	//  http_notify {
	//    url: "http://localhost:8080/alert"
	//    method: POST
	//    header {
	//      key: "Authorization"
	//      value: "Bearer {{env 'AUTH_TOKEN'}}"
	//    }
	//    data: "{\"message\": \"@alert@ fired for @target@\", \"details\": \"name\"}"
	//  }
	httpNotify?: proto.#HTTPRequest @protobuf(3,utils.httpreq.HTTPRequest,name=http_notify)
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
	detailsTemplate?: string @protobuf(7,string,name=details_template) // Default: ""

	// Key values to be included in the alert. These fields are expanded
	// using the same template expansion rules as summary_template and
	// details_template (see above).
	otherInfo?: {
		[string]: string
	} @protobuf(9,map[string]string,other_info)

	// Severity of the alert. If provided it's included in the alert
	// notifications. If severity is not defined, we set it to ERROR for
	// PagerDuty notifications.
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
	severity?: #Severity @protobuf(10,Severity)

	// How often to repeat notification for the same alert. Default is 1hr.
	// To disable any kind of notification throttling, set this to 0.
	repeatIntervalSec?: int32 @protobuf(8,int32,name=repeat_interval_sec) // Default: 1hr
}
