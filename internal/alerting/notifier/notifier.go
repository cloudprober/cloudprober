// Copyright 2023 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package notifier implements notifications related functionality.
package notifier

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	"github.com/cloudprober/cloudprober/internal/alerting/notifier/opsgenie"
	"github.com/cloudprober/cloudprober/internal/alerting/notifier/pagerduty"
	"github.com/cloudprober/cloudprober/internal/alerting/notifier/slack"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/logger"
)

const (
	DefaultDashboardURLTemplate = "http://localhost:9313/status?probe=@probe@"
	DefaultSummaryTemplate      = `Cloudprober alert "@alert@" for "@target@"`
	DefaultDetailsTemplate      = `Cloudprober alert "@alert@" for "@target@":

Failures: @failures@ out of @total@ probes
Failing since: @since@
Probe: @probe@
Dashboard: @dashboard_url@
Playbook: @playbook_url@
`
)

type Notifier struct {
	l        *logger.Logger
	alertcfg *configpb.AlertConf

	summaryTmpl      string
	detailsTmpl      string
	dashboardURLTmpl string

	cmdNotifier       *commandNotifier
	emailNotifier     *emailNotifier
	pagerdutyNotifier *pagerduty.Client
	opsgenieNotifier  *opsgenie.Client
	slackNotifier     *slack.Client
}

func (n *Notifier) alertFields(alertInfo *alertinfo.AlertInfo) map[string]string {
	templateDetails := map[string]string{
		"summary":       n.summaryTmpl,
		"details":       n.detailsTmpl,
		"dashboard_url": n.dashboardURLTmpl,
		"playbook_url":  n.alertcfg.GetPlaybookUrlTemplate(),
	}
	for k, v := range n.alertcfg.GetOtherInfo() {
		templateDetails[k] = v
	}

	fields := alertInfo.Fields(templateDetails)

	severity := n.alertcfg.GetSeverity()
	if severity != configpb.AlertConf_UNKNOWN_SEVERITY {
		fields["severity"] = severity.String()
	}

	return fields
}

func (n *Notifier) Notify(ctx context.Context, alertInfo *alertinfo.AlertInfo) error {
	fields := n.alertFields(alertInfo)

	var errs error
	if n.cmdNotifier != nil {
		err := n.cmdNotifier.Notify(ctx, fields)
		if err != nil {
			n.l.Errorf("Error running notify command: %v", err)
			errs = errors.Join(errs, err)
		}
	}

	if n.emailNotifier != nil {
		err := n.emailNotifier.Notify(ctx, fields)
		if err == nil {
			n.l.Errorf("Error sending email: %v", err)
			errs = errors.Join(errs, err)
		}
	}

	if n.pagerdutyNotifier != nil {
		err := n.pagerdutyNotifier.Notify(ctx, alertInfo, fields)
		if err != nil {
			n.l.Errorf("Error sending PagerDuty event: %v", err)
			errs = errors.Join(errs, err)
		}
	}

	if n.opsgenieNotifier != nil {
		err := n.opsgenieNotifier.Notify(ctx, alertInfo, fields)
		if err != nil {
			n.l.Errorf("Error sending OpsGenie alert: %v", err)
			errs = errors.Join(errs, err)
		}
	}

	if n.slackNotifier != nil {
		err := n.slackNotifier.Notify(ctx, fields)
		if err != nil {
			n.l.Errorf("Error sending Slack message: %v", err)
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (n *Notifier) NotifyResolve(ctx context.Context, alertInfo *alertinfo.AlertInfo) {
	fields := n.alertFields(alertInfo)

	if n.pagerdutyNotifier != nil {
		if err := n.pagerdutyNotifier.NotifyResolve(ctx, alertInfo, fields); err != nil {
			n.l.Errorf("Error sending PagerDuty resolve event: %v", err)
		}
	}

	if n.opsgenieNotifier != nil {
		if err := n.opsgenieNotifier.NotifyResolve(ctx, alertInfo, fields); err != nil {
			n.l.Errorf("Error closing OpsGenie alert: %v", err)
		}
	}
}

func New(alertcfg *configpb.AlertConf, l *logger.Logger) (*Notifier, error) {
	if alertcfg == nil {
		alertcfg = &configpb.AlertConf{}
	}

	n := &Notifier{
		alertcfg:         alertcfg,
		l:                l,
		summaryTmpl:      alertcfg.GetSummaryTemplate(),
		detailsTmpl:      alertcfg.GetDetailsTemplate(),
		dashboardURLTmpl: alertcfg.GetDashboardUrlTemplate(),
	}

	if n.summaryTmpl == "" {
		n.summaryTmpl = DefaultSummaryTemplate
	}
	if n.detailsTmpl == "" {
		n.detailsTmpl = DefaultDetailsTemplate
	}
	if n.dashboardURLTmpl == "" {
		n.dashboardURLTmpl = DefaultDashboardURLTemplate
	}

	if n.alertcfg.GetNotify() == nil {
		return n, nil
	}

	if n.alertcfg.GetNotify().Command != "" {
		cmdParts, err := newCommandNotifier(n.alertcfg.GetNotify().Command, l)
		if err != nil {
			return nil, fmt.Errorf("error parsing notify command: %v", err)
		}
		n.cmdNotifier = cmdParts
	}

	if n.alertcfg.GetNotify().GetEmail() != nil {
		en, err := newEmailNotifier(n.alertcfg.GetNotify().GetEmail(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring email notifier: %v", err)
		}
		n.emailNotifier = en
	}

	if n.alertcfg.GetNotify().GetPagerDuty() != nil {
		pd, err := pagerduty.New(n.alertcfg.Notify.GetPagerDuty(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring PagerDuty notifier: %v", err)
		}
		n.pagerdutyNotifier = pd
	}

	if n.alertcfg.GetNotify().GetOpsgenie() != nil {
		og, err := opsgenie.New(n.alertcfg.Notify.GetOpsgenie(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring OpsGenie notifier: %v", err)
		}
		n.opsgenieNotifier = og
	}

	if n.alertcfg.GetNotify().GetSlack() != nil {
		slack, err := slack.New(n.alertcfg.Notify.GetSlack(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring Slack notifier: %v", err)
		}
		n.slackNotifier = slack
	}

	return n, nil
}
