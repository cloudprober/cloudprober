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

package alerting

import (
	"context"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestAlertFields(t *testing.T) {
	testTarget := endpoint.Endpoint{
		Name: "test-target",
		Labels: map[string]string{
			"apptype":  "backend",
			"language": "go",
		},
	}

	tests := []struct {
		name string
		ai   *AlertInfo
		want map[string]string
	}{
		{
			name: "simple",
			ai: &AlertInfo{
				Name:         "test-alert",
				ProbeName:    "test-probe",
				Target:       testTarget,
				Failures:     8,
				Total:        12,
				FailingSince: time.Time{}.Add(time.Second),
			},
			want: map[string]string{
				"alert":                 "test-alert",
				"probe":                 "test-probe",
				"target":                "test-target",
				"failures":              "8",
				"total":                 "12",
				"since":                 "0001-01-01T00:00:01Z",
				"target.label.apptype":  "backend",
				"target.label.language": "go",
				"json":                  `{"alert":"test-alert","failures":"8","probe":"test-probe","since":"0001-01-01T00:00:01Z","target":"test-target","target.label.apptype":"backend","target.label.language":"go","total":"12"}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := alertFields(tt.ai)
			assert.NoError(t, err, "Error getting alert fields")
			assert.Equal(t, tt.want, fields, "Fields don't match")
		})
	}
}

func TestAlertHandlerNotifyCommand(t *testing.T) {
	ah := &AlertHandler{
		name:      "test-alert",
		probeName: "test-probe",
	}

	fields := map[string]string{
		"alert":              "test-alert",
		"probe":              "test-probe",
		"target":             "test-target:1234",
		"target.label.owner": "manugarg@a.b",
	}

	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			command: "/usr/bin/mail -s 'Alert @alert@ fired for the target @target@ - @unmatched@' @target.label.owner@",
			want:    []string{"/usr/bin/mail", "-s", "Alert test-alert fired for the target test-target:1234 - @unmatched@", "manugarg@a.b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ah.notifyConfig = &configpb.NotifyConfig{
				Command: tt.command,
			}
			assert.Equal(t, tt.want, ah.notifyCommand(context.Background(), tt.command, fields, true), "Commands don't match")
		})
	}
}
