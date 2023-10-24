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
	httpreqpb "github.com/cloudprober/cloudprober/internal/httpreq/proto"
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
	l   *logger.Logger
	cfg *configpb.NotifyConfig

	summaryTmpl      string
	detailsTmpl      string
	dashboardURLTmpl string
	playbookURLTmpl  string
	severity         configpb.AlertConf_Severity
	otherInfo        map[string]string

	cmdNotifier       *commandNotifier
	emailNotifier     *emailNotifier
	pagerdutyNotifier *pagerduty.Client
	opsgenieNotifier  *opsgenie.Client
	slackNotifier     *slack.Client
	httpNotifier      *httpreqpb.HTTPRequest
}

func (n *Notifier) alertFields(alertInfo *alertinfo.AlertInfo) map[string]string {
	templateDetails := map[string]string{
		"summary":       n.summaryTmpl,
		"details":       n.detailsTmpl,
		"dashboard_url": n.dashboardURLTmpl,
		"playbook_url":  n.playbookURLTmpl,
	}
	for k, v := range n.otherInfo {
		templateDetails[k] = v
	}

	fields := alertInfo.Fields(templateDetails)

	if n.severity != configpb.AlertConf_UNKNOWN_SEVERITY {
		fields["severity"] = n.severity.String()
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

	if n.httpNotifier != nil {
		err := n.httpNotify(ctx, fields)
		if err != nil {
			n.l.Errorf("Error sending HTTP notification: %v", err)
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
		cfg: alertcfg.GetNotify(),
		l:   l,

		severity:         alertcfg.GetSeverity(),
		otherInfo:        alertcfg.GetOtherInfo(),
		summaryTmpl:      alertcfg.GetSummaryTemplate(),
		detailsTmpl:      alertcfg.GetDetailsTemplate(),
		dashboardURLTmpl: alertcfg.GetDashboardUrlTemplate(),
		playbookURLTmpl:  alertcfg.GetPlaybookUrlTemplate(),
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

	if n.cfg == nil {
		return n, nil
	}

	if n.cfg.Command != "" {
		cmdParts, err := newCommandNotifier(n.cfg.Command, l)
		if err != nil {
			return nil, fmt.Errorf("error parsing notify command: %v", err)
		}
		n.cmdNotifier = cmdParts
	}

	if n.cfg.GetEmail() != nil {
		en, err := newEmailNotifier(n.cfg.GetEmail(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring email notifier: %v", err)
		}
		n.emailNotifier = en
	}

	if n.cfg.GetPagerDuty() != nil {
		pd, err := pagerduty.New(n.cfg.GetPagerDuty(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring PagerDuty notifier: %v", err)
		}
		n.pagerdutyNotifier = pd
	}

	if n.cfg.GetOpsgenie() != nil {
		og, err := opsgenie.New(n.cfg.GetOpsgenie(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring OpsGenie notifier: %v", err)
		}
		n.opsgenieNotifier = og
	}

	if n.cfg.GetSlack() != nil {
		slack, err := slack.New(n.cfg.GetSlack(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring Slack notifier: %v", err)
		}
		n.slackNotifier = slack
	}

	if n.cfg.GetHttpNotify() != nil {
		n.httpNotifier = n.cfg.GetHttpNotify()
	}

	return n, nil
}
