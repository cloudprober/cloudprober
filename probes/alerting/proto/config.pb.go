// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v3.21.5
// source: github.com/cloudprober/cloudprober/probes/alerting/proto/config.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Email struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Email addresses to send the alert to.
	To []string `protobuf:"bytes,1,rep,name=to,proto3" json:"to,omitempty"`
	// From address in the alert email.
	// If not set, defaults to the value of smtp_user if smtp_user is set,
	// otherwise defaults to cloudprober-alert@<hostname>.
	From string `protobuf:"bytes,2,opt,name=from,proto3" json:"from,omitempty"`
	// Default: Environment variable SMTP_SERVER
	SmtpServer string `protobuf:"bytes,3,opt,name=smtp_server,json=smtpServer,proto3" json:"smtp_server,omitempty"`
	// Default: Environment variable SMTP_USERNAME
	SmtpUsername string `protobuf:"bytes,4,opt,name=smtp_username,json=smtpUsername,proto3" json:"smtp_username,omitempty"`
	// Default: Environment variable SMTP_PASSWORD
	SmtpPassword string `protobuf:"bytes,5,opt,name=smtp_password,json=smtpPassword,proto3" json:"smtp_password,omitempty"`
}

func (x *Email) Reset() {
	*x = Email{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Email) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Email) ProtoMessage() {}

func (x *Email) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Email.ProtoReflect.Descriptor instead.
func (*Email) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *Email) GetTo() []string {
	if x != nil {
		return x.To
	}
	return nil
}

func (x *Email) GetFrom() string {
	if x != nil {
		return x.From
	}
	return ""
}

func (x *Email) GetSmtpServer() string {
	if x != nil {
		return x.SmtpServer
	}
	return ""
}

func (x *Email) GetSmtpUsername() string {
	if x != nil {
		return x.SmtpUsername
	}
	return ""
}

func (x *Email) GetSmtpPassword() string {
	if x != nil {
		return x.SmtpPassword
	}
	return ""
}

type PagerDuty struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// PagerDuty Routing Key.
	// The routing key is used to determine which service the alerts are sent to
	// and is generated with the service. The routing key is found under the
	// service, when the events v2 integration is enabled, under integrations,
	// in the pagerduty console.
	// Note: set either routing_key or routing_key_env_var. routing_key
	// takes precedence over routing_key_env_var.
	RoutingKey string `protobuf:"bytes,1,opt,name=routing_key,json=routingKey,proto3" json:"routing_key,omitempty"`
	// The environment variable that is used to contain the pagerduty routing
	// key.
	RoutingKeyEnvVar string `protobuf:"bytes,2,opt,name=routing_key_env_var,json=routingKeyEnvVar,proto3" json:"routing_key_env_var,omitempty"` // Default: PAGERDUTY_ROUTING_KEY;
	// PagerDuty API URL.
	// Used to overwrite the default PagerDuty API URL.
	ApiUrl string `protobuf:"bytes,3,opt,name=api_url,json=apiUrl,proto3" json:"api_url,omitempty"` // Default: https://event.pagerduty.com
	// Whether to send resolve notifications or not. Default is to send resolve
	// notifications.
	DisableSendResolved bool `protobuf:"varint,4,opt,name=disable_send_resolved,json=disableSendResolved,proto3" json:"disable_send_resolved,omitempty"` // Default: false
}

func (x *PagerDuty) Reset() {
	*x = PagerDuty{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PagerDuty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PagerDuty) ProtoMessage() {}

func (x *PagerDuty) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PagerDuty.ProtoReflect.Descriptor instead.
func (*PagerDuty) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{1}
}

func (x *PagerDuty) GetRoutingKey() string {
	if x != nil {
		return x.RoutingKey
	}
	return ""
}

func (x *PagerDuty) GetRoutingKeyEnvVar() string {
	if x != nil {
		return x.RoutingKeyEnvVar
	}
	return ""
}

func (x *PagerDuty) GetApiUrl() string {
	if x != nil {
		return x.ApiUrl
	}
	return ""
}

func (x *PagerDuty) GetDisableSendResolved() bool {
	if x != nil {
		return x.DisableSendResolved
	}
	return false
}

type Slack struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Webhook URL
	// The Slack notifications use a webhook URL to send the notifications to
	// a Slack channel. The webhook URL can be found in the Slack console under
	// the "Incoming Webhooks" section.
	// https://api.slack.com/messaging/webhooks
	// Note: set either webhook_url or webhook_url_env_var. webhook_url
	// takes precedence over webhook_url_env_var.
	WebhookUrl string `protobuf:"bytes,1,opt,name=webhook_url,json=webhookUrl,proto3" json:"webhook_url,omitempty"`
	// The environment variable that is used to contain the slack webhook URL.
	WebhookUrlEnvVar string `protobuf:"bytes,2,opt,name=webhook_url_env_var,json=webhookUrlEnvVar,proto3" json:"webhook_url_env_var,omitempty"` // Default: SLACK_WEBHOOK_URL;
}

func (x *Slack) Reset() {
	*x = Slack{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Slack) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Slack) ProtoMessage() {}

func (x *Slack) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Slack.ProtoReflect.Descriptor instead.
func (*Slack) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{2}
}

func (x *Slack) GetWebhookUrl() string {
	if x != nil {
		return x.WebhookUrl
	}
	return ""
}

func (x *Slack) GetWebhookUrlEnvVar() string {
	if x != nil {
		return x.WebhookUrlEnvVar
	}
	return ""
}

type NotifyConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Command to run when alert is fired. In the command line following fields
	// are substituted:
	//
	//	@alert@: Alert name
	//	@probe@: Probe name
	//	@target@: Target name, or target and port if port is specified.
	//	@target.label.<label>@: Label <label> value, e.g. target.label.role.
	//	@failures@: Count of failures.
	//	@total@: Out of.
	//	@since@: Time since the alert condition started.
	//	@json@: JSON representation of the alert fields.
	//
	// For example, if you want to send an email when an alert is fired, you can
	// use the following command:
	// command: "/usr/bin/mail -s 'Alert @alert@ fired for @target@' manu@a.b"
	Command string `protobuf:"bytes,10,opt,name=command,proto3" json:"command,omitempty"`
	// Email notification configuration.
	Email *Email `protobuf:"bytes,11,opt,name=email,proto3" json:"email,omitempty"`
	// PagerDuty configuration.
	PagerDuty *PagerDuty `protobuf:"bytes,12,opt,name=pager_duty,json=pagerDuty,proto3" json:"pager_duty,omitempty"`
	// Slack configuration.
	Slack *Slack `protobuf:"bytes,13,opt,name=slack,proto3" json:"slack,omitempty"`
}

func (x *NotifyConfig) Reset() {
	*x = NotifyConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotifyConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotifyConfig) ProtoMessage() {}

func (x *NotifyConfig) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NotifyConfig.ProtoReflect.Descriptor instead.
func (*NotifyConfig) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{3}
}

func (x *NotifyConfig) GetCommand() string {
	if x != nil {
		return x.Command
	}
	return ""
}

func (x *NotifyConfig) GetEmail() *Email {
	if x != nil {
		return x.Email
	}
	return nil
}

func (x *NotifyConfig) GetPagerDuty() *PagerDuty {
	if x != nil {
		return x.PagerDuty
	}
	return nil
}

func (x *NotifyConfig) GetSlack() *Slack {
	if x != nil {
		return x.Slack
	}
	return nil
}

type Condition struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Failures int32 `protobuf:"varint,1,opt,name=failures,proto3" json:"failures,omitempty"`
	Total    int32 `protobuf:"varint,2,opt,name=total,proto3" json:"total,omitempty"`
}

func (x *Condition) Reset() {
	*x = Condition{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Condition) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Condition) ProtoMessage() {}

func (x *Condition) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Condition.ProtoReflect.Descriptor instead.
func (*Condition) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{4}
}

func (x *Condition) GetFailures() int32 {
	if x != nil {
		return x.Failures
	}
	return 0
}

func (x *Condition) GetTotal() int32 {
	if x != nil {
		return x.Total
	}
	return 0
}

type AlertConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the alert. Default is to use the probe name. If you have multiple
	// alerts for the same probe, you must specify a name for each alert.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Condition for the alert. Default is to alert on any failure.
	// Example:
	// # Alert if 6 out of 10 probes fail.
	//
	//	condition {
	//	  failures: 6
	//	  total: 10
	//	}
	Condition *Condition `protobuf:"bytes,2,opt,name=condition,proto3,oneof" json:"condition,omitempty"`
	// How to notify in case of alert.
	Notify *NotifyConfig `protobuf:"bytes,3,opt,name=notify,proto3" json:"notify,omitempty"`
	// Dashboard URL template.
	// Default: http://localhost:9313/status?probe=@probe@
	DashboardUrlTemplate string `protobuf:"bytes,4,opt,name=dashboard_url_template,json=dashboardUrlTemplate,proto3" json:"dashboard_url_template,omitempty"` // Default: ""
	PlaybookUrlTemplate  string `protobuf:"bytes,5,opt,name=playbook_url_template,json=playbookUrlTemplate,proto3" json:"playbook_url_template,omitempty"`    // Default: ""
	// Default: Cloudprober alert "@alert@" for "@target@"
	SummaryTemplate string `protobuf:"bytes,6,opt,name=summary_template,json=summaryTemplate,proto3" json:"summary_template,omitempty"`
	// Default:
	// Cloudprober alert "@alert@" for "@target@":
	// Failures: @failures@ out of @total@ probes
	// Failing since: @since@
	// Probe: @probe@
	// Dashboard: @dashboard_url@
	// Playbook: @playbook_url@
	// Condition ID: @condition_id@
	DetailsTemplate string `protobuf:"bytes,7,opt,name=details_template,json=detailsTemplate,proto3" json:"details_template,omitempty"` // Default: ""
	// How often to repeat notification for the same alert. Default is 1hr.
	// To disable any kind of notification throttling, set this to 0.
	RepeatIntervalSec *int32 `protobuf:"varint,8,opt,name=repeat_interval_sec,json=repeatIntervalSec,proto3,oneof" json:"repeat_interval_sec,omitempty"` // Default: 1hr
}

func (x *AlertConf) Reset() {
	*x = AlertConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AlertConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AlertConf) ProtoMessage() {}

func (x *AlertConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AlertConf.ProtoReflect.Descriptor instead.
func (*AlertConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{5}
}

func (x *AlertConf) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *AlertConf) GetCondition() *Condition {
	if x != nil {
		return x.Condition
	}
	return nil
}

func (x *AlertConf) GetNotify() *NotifyConfig {
	if x != nil {
		return x.Notify
	}
	return nil
}

func (x *AlertConf) GetDashboardUrlTemplate() string {
	if x != nil {
		return x.DashboardUrlTemplate
	}
	return ""
}

func (x *AlertConf) GetPlaybookUrlTemplate() string {
	if x != nil {
		return x.PlaybookUrlTemplate
	}
	return ""
}

func (x *AlertConf) GetSummaryTemplate() string {
	if x != nil {
		return x.SummaryTemplate
	}
	return ""
}

func (x *AlertConf) GetDetailsTemplate() string {
	if x != nil {
		return x.DetailsTemplate
	}
	return ""
}

func (x *AlertConf) GetRepeatIntervalSec() int32 {
	if x != nil && x.RepeatIntervalSec != nil {
		return *x.RepeatIntervalSec
	}
	return 0
}

var File_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto protoreflect.FileDescriptor

var file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDesc = []byte{
	0x0a, 0x45, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2f, 0x61, 0x6c, 0x65, 0x72,
	0x74, 0x69, 0x6e, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2e, 0x61, 0x6c, 0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x22, 0x96, 0x01,
	0x0a, 0x05, 0x45, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x0e, 0x0a, 0x02, 0x74, 0x6f, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x02, 0x74, 0x6f, 0x12, 0x12, 0x0a, 0x04, 0x66, 0x72, 0x6f, 0x6d, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x66, 0x72, 0x6f, 0x6d, 0x12, 0x1f, 0x0a, 0x0b, 0x73,
	0x6d, 0x74, 0x70, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0a, 0x73, 0x6d, 0x74, 0x70, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x12, 0x23, 0x0a, 0x0d,
	0x73, 0x6d, 0x74, 0x70, 0x5f, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0c, 0x73, 0x6d, 0x74, 0x70, 0x55, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x23, 0x0a, 0x0d, 0x73, 0x6d, 0x74, 0x70, 0x5f, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f,
	0x72, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x73, 0x6d, 0x74, 0x70, 0x50, 0x61,
	0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x22, 0xa8, 0x01, 0x0a, 0x09, 0x50, 0x61, 0x67, 0x65, 0x72,
	0x44, 0x75, 0x74, 0x79, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x6f, 0x75, 0x74, 0x69, 0x6e, 0x67, 0x5f,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x6f, 0x75, 0x74, 0x69,
	0x6e, 0x67, 0x4b, 0x65, 0x79, 0x12, 0x2d, 0x0a, 0x13, 0x72, 0x6f, 0x75, 0x74, 0x69, 0x6e, 0x67,
	0x5f, 0x6b, 0x65, 0x79, 0x5f, 0x65, 0x6e, 0x76, 0x5f, 0x76, 0x61, 0x72, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x10, 0x72, 0x6f, 0x75, 0x74, 0x69, 0x6e, 0x67, 0x4b, 0x65, 0x79, 0x45, 0x6e,
	0x76, 0x56, 0x61, 0x72, 0x12, 0x17, 0x0a, 0x07, 0x61, 0x70, 0x69, 0x5f, 0x75, 0x72, 0x6c, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x61, 0x70, 0x69, 0x55, 0x72, 0x6c, 0x12, 0x32, 0x0a,
	0x15, 0x64, 0x69, 0x73, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x73, 0x65, 0x6e, 0x64, 0x5f, 0x72, 0x65,
	0x73, 0x6f, 0x6c, 0x76, 0x65, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x13, 0x64, 0x69,
	0x73, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x65, 0x6e, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x65,
	0x64, 0x22, 0x57, 0x0a, 0x05, 0x53, 0x6c, 0x61, 0x63, 0x6b, 0x12, 0x1f, 0x0a, 0x0b, 0x77, 0x65,
	0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0a, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x55, 0x72, 0x6c, 0x12, 0x2d, 0x0a, 0x13, 0x77,
	0x65, 0x62, 0x68, 0x6f, 0x6f, 0x6b, 0x5f, 0x75, 0x72, 0x6c, 0x5f, 0x65, 0x6e, 0x76, 0x5f, 0x76,
	0x61, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x77, 0x65, 0x62, 0x68, 0x6f, 0x6f,
	0x6b, 0x55, 0x72, 0x6c, 0x45, 0x6e, 0x76, 0x56, 0x61, 0x72, 0x22, 0xce, 0x01, 0x0a, 0x0c, 0x4e,
	0x6f, 0x74, 0x69, 0x66, 0x79, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x18, 0x0a, 0x07, 0x63,
	0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f,
	0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x31, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x0b,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2e, 0x61, 0x6c, 0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x2e, 0x45, 0x6d, 0x61, 0x69,
	0x6c, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x3e, 0x0a, 0x0a, 0x70, 0x61, 0x67, 0x65,
	0x72, 0x5f, 0x64, 0x75, 0x74, 0x79, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x61, 0x6c, 0x65, 0x72, 0x74,
	0x69, 0x6e, 0x67, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x72, 0x44, 0x75, 0x74, 0x79, 0x52, 0x09, 0x70,
	0x61, 0x67, 0x65, 0x72, 0x44, 0x75, 0x74, 0x79, 0x12, 0x31, 0x0a, 0x05, 0x73, 0x6c, 0x61, 0x63,
	0x6b, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x61, 0x6c, 0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x2e, 0x53,
	0x6c, 0x61, 0x63, 0x6b, 0x52, 0x05, 0x73, 0x6c, 0x61, 0x63, 0x6b, 0x22, 0x3d, 0x0a, 0x09, 0x43,
	0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x66, 0x61, 0x69, 0x6c,
	0x75, 0x72, 0x65, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x66, 0x61, 0x69, 0x6c,
	0x75, 0x72, 0x65, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x05, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x22, 0xba, 0x03, 0x0a, 0x09, 0x41,
	0x6c, 0x65, 0x72, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x42, 0x0a, 0x09,
	0x63, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1f, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x61, 0x6c,
	0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x2e, 0x43, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x48, 0x00, 0x52, 0x09, 0x63, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x88, 0x01, 0x01,
	0x12, 0x3a, 0x0a, 0x06, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x22, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x61,
	0x6c, 0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x2e, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x52, 0x06, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x12, 0x34, 0x0a, 0x16,
	0x64, 0x61, 0x73, 0x68, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x5f, 0x75, 0x72, 0x6c, 0x5f, 0x74, 0x65,
	0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x64, 0x61,
	0x73, 0x68, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x55, 0x72, 0x6c, 0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61,
	0x74, 0x65, 0x12, 0x32, 0x0a, 0x15, 0x70, 0x6c, 0x61, 0x79, 0x62, 0x6f, 0x6f, 0x6b, 0x5f, 0x75,
	0x72, 0x6c, 0x5f, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x13, 0x70, 0x6c, 0x61, 0x79, 0x62, 0x6f, 0x6f, 0x6b, 0x55, 0x72, 0x6c, 0x54, 0x65,
	0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x12, 0x29, 0x0a, 0x10, 0x73, 0x75, 0x6d, 0x6d, 0x61, 0x72,
	0x79, 0x5f, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0f, 0x73, 0x75, 0x6d, 0x6d, 0x61, 0x72, 0x79, 0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74,
	0x65, 0x12, 0x29, 0x0a, 0x10, 0x64, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x5f, 0x74, 0x65, 0x6d,
	0x70, 0x6c, 0x61, 0x74, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x64, 0x65, 0x74,
	0x61, 0x69, 0x6c, 0x73, 0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x12, 0x33, 0x0a, 0x13,
	0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x5f,
	0x73, 0x65, 0x63, 0x18, 0x08, 0x20, 0x01, 0x28, 0x05, 0x48, 0x01, 0x52, 0x11, 0x72, 0x65, 0x70,
	0x65, 0x61, 0x74, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x53, 0x65, 0x63, 0x88, 0x01,
	0x01, 0x42, 0x0c, 0x0a, 0x0a, 0x5f, 0x63, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x42,
	0x16, 0x0a, 0x14, 0x5f, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x42, 0x3a, 0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x73, 0x2f, 0x61, 0x6c, 0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescOnce sync.Once
	file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescData = file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDesc
)

func file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescData)
	})
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescData
}

var file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_goTypes = []interface{}{
	(*Email)(nil),        // 0: cloudprober.alerting.Email
	(*PagerDuty)(nil),    // 1: cloudprober.alerting.PagerDuty
	(*Slack)(nil),        // 2: cloudprober.alerting.Slack
	(*NotifyConfig)(nil), // 3: cloudprober.alerting.NotifyConfig
	(*Condition)(nil),    // 4: cloudprober.alerting.Condition
	(*AlertConf)(nil),    // 5: cloudprober.alerting.AlertConf
}
var file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.alerting.NotifyConfig.email:type_name -> cloudprober.alerting.Email
	1, // 1: cloudprober.alerting.NotifyConfig.pager_duty:type_name -> cloudprober.alerting.PagerDuty
	2, // 2: cloudprober.alerting.NotifyConfig.slack:type_name -> cloudprober.alerting.Slack
	4, // 3: cloudprober.alerting.AlertConf.condition:type_name -> cloudprober.alerting.Condition
	3, // 4: cloudprober.alerting.AlertConf.notify:type_name -> cloudprober.alerting.NotifyConfig
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Email); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PagerDuty); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Slack); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NotifyConfig); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Condition); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AlertConf); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[5].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto = out.File
	file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDesc = nil
	file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_goTypes = nil
	file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_depIdxs = nil
}
