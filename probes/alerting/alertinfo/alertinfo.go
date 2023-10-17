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

// Package alertinfo implements AlertInfo struct for sharing alert data
// across modules.
package alertinfo

import (
	"strconv"
	"time"

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

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

func (ai *AlertInfo) Fields(templateDetails map[string]string) map[string]string {
	fields := map[string]string{
		"alert":        ai.Name,
		"probe":        ai.ProbeName,
		"target":       ai.Target.Dst(),
		"condition_id": ai.ConditionID,
		"failures":     strconv.Itoa(ai.Failures),
		"total":        strconv.Itoa(ai.Total),
		"since":        ai.FailingSince.Format(time.RFC3339),
	}

	for k, v := range ai.Target.Labels {
		fields["target.label."+k] = v
	}

	if ai.Target.IP != nil {
		fields["target_ip"] = ai.Target.IP.String()
	}

	// Note that we parse details in the end, that's because details template
	// may use other parsed fields like dashboard_url, playbook_url, etc.
	for k, v := range templateDetails {
		if k != "details" {
			fields[k], _ = strtemplate.SubstituteLabels(v, fields)
		}
	}
	if templateDetails["details"] != "" {
		fields["details"], _ = strtemplate.SubstituteLabels(templateDetails["details"], fields)
	}

	return fields
}
