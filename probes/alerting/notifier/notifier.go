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

package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

type Notifier struct {
	l             *logger.Logger
	alertcfg      *configpb.AlertConf
	cmdNotifier   *commandNotifier
	emailNotifier *emailNotifier
}

// AlertInfo contains information about an alert.
type AlertInfo struct {
	Name         string
	ProbeName    string
	ConditionID  string
	Target       endpoint.Endpoint
	Failures     int
	Total        int
	FailingSince time.Time
}

func (n *Notifier) alertFields(alertInfo *AlertInfo) (map[string]string, error) {
	fields := map[string]string{
		"alert":        alertInfo.Name,
		"probe":        alertInfo.ProbeName,
		"target":       alertInfo.Target.Dst(),
		"condition_id": alertInfo.ConditionID,
		"failures":     strconv.Itoa(alertInfo.Failures),
		"total":        strconv.Itoa(alertInfo.Total),
		"since":        alertInfo.FailingSince.Format(time.RFC3339),
	}

	for k, v := range alertInfo.Target.Labels {
		fields["target.label."+k] = v
	}

	alertJSON, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("error marshalling alert fields into json: %v", err)
	}

	fields["json"] = string(alertJSON)

	summary, _ := strtemplate.SubstituteLabels(n.alertcfg.GetSummaryTemplate(), fields)
	fields["summary"] = summary

	if n.alertcfg.GetDashboardUrlTemplate() != "" {
		url, _ := strtemplate.SubstituteLabels(n.alertcfg.GetDashboardUrlTemplate(), fields)
		fields["dashboard_url"] = url
	}

	fields["playbook_url"] = ""
	if n.alertcfg.GetPlaybookUrlTemplate() != "" {
		url, _ := strtemplate.SubstituteLabels(n.alertcfg.GetPlaybookUrlTemplate(), fields)
		fields["playbook_url"] = url
	}

	details, _ := strtemplate.SubstituteLabels(n.alertcfg.GetDetailsTemplate(), fields)
	fields["details"] = details

	return fields, nil
}

func (n *Notifier) Notify(ctx context.Context, alertInfo *AlertInfo) error {
	var err error

	fields, err := n.alertFields(alertInfo)
	if err != nil {
		n.l.Errorf("Error getting alert fields: %v", err)
	}

	if n.cmdNotifier != nil {
		cmdErr := n.cmdNotifier.Notify(ctx, fields)
		if cmdErr != nil {
			n.l.Errorf("Error running notify command: %v", cmdErr)
			err = errors.Join(err, cmdErr)
		}
	}

	if n.emailNotifier != nil {
		emailerr := n.emailNotifier.Notify(ctx, fields)
		if emailerr == nil {
			n.l.Errorf("Error sending email: %v", emailerr)
			err = errors.Join(err, emailerr)
		}
	}

	return err
}

func New(alertcfg *configpb.AlertConf, l *logger.Logger) (*Notifier, error) {
	if alertcfg == nil {
		alertcfg = &configpb.AlertConf{}
	}

	n := &Notifier{
		alertcfg: alertcfg,
		l:        l,
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

	if n.alertcfg.GetNotify().Email != nil {
		en, err := newEmailNotifier(n.alertcfg.GetNotify(), l)
		if err != nil {
			return nil, fmt.Errorf("error configuring email notifier: %v", err)
		}
		n.emailNotifier = en
	}

	return n, nil
}
