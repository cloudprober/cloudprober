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
	"net"
	"net/smtp"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestAlertFields(t *testing.T) {
	testTarget := endpoint.Endpoint{
		Name: "test-target",
		IP:   net.ParseIP("10.11.12.13"),
		Labels: map[string]string{
			"apptype":  "backend",
			"language": "go",
		},
	}

	tests := []struct {
		name string
		ai   *alertinfo.AlertInfo
		want map[string]string
	}{
		{
			name: "simple",
			ai: &alertinfo.AlertInfo{
				Name:            "test-alert",
				ProbeName:       "test-probe",
				DeduplicationID: "122333444",
				Target:          testTarget,
				Failures:        8,
				Total:           12,
				FailingSince:    time.Time{}.Add(time.Second),
			},
			want: map[string]string{
				"alert":                 "test-alert",
				"probe":                 "test-probe",
				"target":                "test-target",
				"target_ip":             "10.11.12.13",
				"failures":              "8",
				"total":                 "12",
				"since":                 "0001-01-01T00:00:01Z",
				"target.label.apptype":  "backend",
				"target.label.language": "go",
				"summary":               "Cloudprober alert \"test-alert\" for \"test-target\"",
				"details":               "Cloudprober alert \"test-alert\" for \"test-target\":\n\nFailures: 8 out of 12 probes\nFailing since: 0001-01-01T00:00:01Z\nProbe: test-probe\nDashboard: http://localhost:9313/status?probe=test-probe\nPlaybook: \n",
				"dashboard_url":         "http://localhost:9313/status?probe=test-probe",
				"playbook_url":          "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, _ := New(nil, nil)
			assert.Equal(t, tt.want, n.alertFields(tt.ai), "Fields don't match")
		})
	}
}

func TestNotify(t *testing.T) {
	alertInfo := &alertinfo.AlertInfo{
		Name:            "test-alert",
		ProbeName:       "test-probe",
		DeduplicationID: "cond-id",
		Target: endpoint.Endpoint{
			Name:   "test-target:1234",
			Labels: map[string]string{"owner": "manugarg@a.b"},
		},
		Failures: 1,
		Total:    2,
	}

	tests := []struct {
		name          string
		conf          *configpb.NotifyConfig
		command       string
		errorContains string
		wantEmailFrom string
		wantEmailMsg  string
	}{
		{
			name: "command",
			conf: &configpb.NotifyConfig{
				Command: "/random-cmd-@alert@-@target.label.owner@ -s 'Alert @alert@ fired for the target @target@ - @unmatched@' @target.label.owner@",
			},
			errorContains: "/random-cmd-test-alert-manugarg@a.b",
		},
		{
			name: "commandAndEmail",
			conf: &configpb.NotifyConfig{
				Command: "/random-cmd-@alert@-@target.label.owner@ -s 'Alert @alert@ fired for the target @target@ - @unmatched@' @target.label.owner@",
				Email: &configpb.Email{
					To: []string{"manugarg@a.b"},
				},
			},
			errorContains: "/random-cmd-test-alert-manugarg@a.b",
			wantEmailFrom: "cloudprober-alert@localhost",
			wantEmailMsg:  "From: cloudprober-alert@localhost\r\nTo: manugarg@a.b\r\nSubject: Cloudprober alert \"test-alert\" for \"test-target:1234\"\r\n\r\nCloudprober alert \"test-alert\" for \"test-target:1234\":\n\nFailures: 1 out of 2 probes\nFailing since: 0001-01-01T00:00:00Z\nProbe: test-probe\nDashboard: http://localhost:9313/status?probe=test-probe\nPlaybook: \n\nDetails:\nalert: test-alert\ndashboard_url: http://localhost:9313/status?probe=test-probe\nfailures: 1\nplaybook_url: \nprobe: test-probe\nsince: 0001-01-01T00:00:00Z\ntarget: test-target:1234\ntarget.label.owner: manugarg@a.b\ntotal: 2\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &configpb.AlertConf{
				Notify: tt.conf,
			}
			n, err := New(conf, nil)

			var gotTo []string
			var gotFrom, gotMsg string

			assert.NotNil(t, n.cmdNotifier, "cmdNotifier is nil")
			if tt.conf.Email != nil {
				assert.NotNil(t, n.emailNotifier, "emailNotifier is nil")
				n.emailNotifier.sendMailFunc = func(server string, auth smtp.Auth, from string, to []string, msg []byte) error {
					gotTo, gotFrom, gotMsg = to, from, string(msg)
					return nil
				}
			}

			assert.NoError(t, err, "Error creating notifier")

			err = n.Notify(context.Background(), alertInfo)

			// Command should result in an error, but email should go through.
			assert.ErrorContains(t, err, tt.errorContains, "notify command error")
			if tt.conf.Email != nil {
				assert.Equal(t, tt.conf.Email.To, gotTo, "email to doesn't match")
				assert.Equal(t, tt.wantEmailFrom, gotFrom, "email from doesn't match")
				assert.Equal(t, tt.wantEmailMsg, gotMsg, "email msg doesn't match")
			}
		})
	}
}
