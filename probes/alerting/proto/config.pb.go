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
	// Email addresses to send alerts to. For email notifications to work,
	// following environment variables must be set:
	// - SMTP_SERVER: SMTP server and port to use for sending emails.
	// - SMTP_USERNAME: SMTP user name.
	// - SMTP_PASSWORD: SMTP password.
	Email     []string `protobuf:"bytes,11,rep,name=email,proto3" json:"email,omitempty"`
	EmailFrom string   `protobuf:"bytes,12,opt,name=email_from,json=emailFrom,proto3" json:"email_from,omitempty"` // Default: SMTP_USERNAME
	// PagerDuty Routing Key.
	// The routing key is used to determine which service the alerts are sent to
	// and is generated with the service. The routing key is found under the service,
	// when the events v2 integration is enabled.
	// If the pagerduty routing key is set then pagerduty alerts will be enabled.
	PagerdutyRoutingKey string `protobuf:"bytes,30,opt,name=pagerduty_routing_key,json=pagerdutyRoutingKey,proto3" json:"pagerduty_routing_key,omitempty"`
	// PagerDuty API URL.
	PagerdutyApiUrl string `protobuf:"bytes,31,opt,name=pagerduty_api_url,json=pagerdutyApiUrl,proto3" json:"pagerduty_api_url,omitempty"` // Default: https://event.pagerduty.com
}

func (x *NotifyConfig) Reset() {
	*x = NotifyConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NotifyConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NotifyConfig) ProtoMessage() {}

func (x *NotifyConfig) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use NotifyConfig.ProtoReflect.Descriptor instead.
func (*NotifyConfig) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *NotifyConfig) GetCommand() string {
	if x != nil {
		return x.Command
	}
	return ""
}

func (x *NotifyConfig) GetEmail() []string {
	if x != nil {
		return x.Email
	}
	return nil
}

func (x *NotifyConfig) GetEmailFrom() string {
	if x != nil {
		return x.EmailFrom
	}
	return ""
}

func (x *NotifyConfig) GetPagerdutyRoutingKey() string {
	if x != nil {
		return x.PagerdutyRoutingKey
	}
	return ""
}

func (x *NotifyConfig) GetPagerdutyApiUrl() string {
	if x != nil {
		return x.PagerdutyApiUrl
	}
	return ""
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
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Condition) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Condition) ProtoMessage() {}

func (x *Condition) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use Condition.ProtoReflect.Descriptor instead.
func (*Condition) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{1}
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

	// Name of the alert. Default is to use the probe name.
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
	// Default: "Cloudprober alert @alert@ for @target@"
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
		mi := &file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AlertConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AlertConf) ProtoMessage() {}

func (x *AlertConf) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use AlertConf.ProtoReflect.Descriptor instead.
func (*AlertConf) Descriptor() ([]byte, []int) {
	return file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDescGZIP(), []int{2}
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
	0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x61, 0x6c, 0x65, 0x72,
	0x74, 0x69, 0x6e, 0x67, 0x22, 0xbd, 0x01, 0x0a, 0x0c, 0x4e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x18, 0x0a, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12,
	0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x0b, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05,
	0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x1d, 0x0a, 0x0a, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x5f, 0x66,
	0x72, 0x6f, 0x6d, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x65, 0x6d, 0x61, 0x69, 0x6c,
	0x46, 0x72, 0x6f, 0x6d, 0x12, 0x32, 0x0a, 0x15, 0x70, 0x61, 0x67, 0x65, 0x72, 0x64, 0x75, 0x74,
	0x79, 0x5f, 0x72, 0x6f, 0x75, 0x74, 0x69, 0x6e, 0x67, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x1e, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x13, 0x70, 0x61, 0x67, 0x65, 0x72, 0x64, 0x75, 0x74, 0x79, 0x52, 0x6f,
	0x75, 0x74, 0x69, 0x6e, 0x67, 0x4b, 0x65, 0x79, 0x12, 0x2a, 0x0a, 0x11, 0x70, 0x61, 0x67, 0x65,
	0x72, 0x64, 0x75, 0x74, 0x79, 0x5f, 0x61, 0x70, 0x69, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x1f, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0f, 0x70, 0x61, 0x67, 0x65, 0x72, 0x64, 0x75, 0x74, 0x79, 0x41, 0x70,
	0x69, 0x55, 0x72, 0x6c, 0x22, 0x3d, 0x0a, 0x09, 0x43, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x08, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x73, 0x12, 0x14, 0x0a,
	0x05, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x74, 0x6f,
	0x74, 0x61, 0x6c, 0x22, 0xc8, 0x03, 0x0a, 0x09, 0x41, 0x6c, 0x65, 0x72, 0x74, 0x43, 0x6f, 0x6e,
	0x66, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x49, 0x0a, 0x09, 0x63, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69,
	0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x61, 0x6c,
	0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x2e, 0x43, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x48, 0x00, 0x52, 0x09, 0x63, 0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x88, 0x01, 0x01,
	0x12, 0x41, 0x0a, 0x06, 0x6e, 0x6f, 0x74, 0x69, 0x66, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x61, 0x6c, 0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x2e, 0x4e,
	0x6f, 0x74, 0x69, 0x66, 0x79, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x06, 0x6e, 0x6f, 0x74,
	0x69, 0x66, 0x79, 0x12, 0x34, 0x0a, 0x16, 0x64, 0x61, 0x73, 0x68, 0x62, 0x6f, 0x61, 0x72, 0x64,
	0x5f, 0x75, 0x72, 0x6c, 0x5f, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x14, 0x64, 0x61, 0x73, 0x68, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x55, 0x72,
	0x6c, 0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x12, 0x32, 0x0a, 0x15, 0x70, 0x6c, 0x61,
	0x79, 0x62, 0x6f, 0x6f, 0x6b, 0x5f, 0x75, 0x72, 0x6c, 0x5f, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61,
	0x74, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x70, 0x6c, 0x61, 0x79, 0x62, 0x6f,
	0x6f, 0x6b, 0x55, 0x72, 0x6c, 0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x12, 0x29, 0x0a,
	0x10, 0x73, 0x75, 0x6d, 0x6d, 0x61, 0x72, 0x79, 0x5f, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74,
	0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x73, 0x75, 0x6d, 0x6d, 0x61, 0x72, 0x79,
	0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x12, 0x29, 0x0a, 0x10, 0x64, 0x65, 0x74, 0x61,
	0x69, 0x6c, 0x73, 0x5f, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x18, 0x07, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0f, 0x64, 0x65, 0x74, 0x61, 0x69, 0x6c, 0x73, 0x54, 0x65, 0x6d, 0x70, 0x6c,
	0x61, 0x74, 0x65, 0x12, 0x33, 0x0a, 0x13, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x5f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x18, 0x08, 0x20, 0x01, 0x28, 0x05,
	0x48, 0x01, 0x52, 0x11, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76,
	0x61, 0x6c, 0x53, 0x65, 0x63, 0x88, 0x01, 0x01, 0x42, 0x0c, 0x0a, 0x0a, 0x5f, 0x63, 0x6f, 0x6e,
	0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x16, 0x0a, 0x14, 0x5f, 0x72, 0x65, 0x70, 0x65, 0x61,
	0x74, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x5f, 0x73, 0x65, 0x63, 0x42, 0x3a,
	0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2f, 0x61, 0x6c, 0x65, 0x72,
	0x74, 0x69, 0x6e, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
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

var file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_goTypes = []interface{}{
	(*NotifyConfig)(nil), // 0: cloudprober.probes.alerting.NotifyConfig
	(*Condition)(nil),    // 1: cloudprober.probes.alerting.Condition
	(*AlertConf)(nil),    // 2: cloudprober.probes.alerting.AlertConf
}
var file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_depIdxs = []int32{
	1, // 0: cloudprober.probes.alerting.AlertConf.condition:type_name -> cloudprober.probes.alerting.Condition
	0, // 1: cloudprober.probes.alerting.AlertConf.notify:type_name -> cloudprober.probes.alerting.NotifyConfig
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_init() }
func file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_init() {
	if File_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
		file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
	file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_msgTypes[2].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_cloudprober_cloudprober_probes_alerting_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
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
