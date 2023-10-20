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
	"net/smtp"
	"os"
	"testing"

	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewEmailNotifier(t *testing.T) {
	const smtpServer = "smtp.gmail.com:587"
	const smtpHost = "smtp.gmail.com"
	var to = []string{"a@cloudprober.org", "b@cloudprober.org"}

	tests := []struct {
		name     string
		emailCfg *configpb.Email
		env      map[string]string
		want     *emailNotifier
		wantErr  bool
	}{
		{
			name: "default_localhost",
			emailCfg: &configpb.Email{
				To:   to,
				From: "",
			},
			want: &emailNotifier{
				to:     to,
				from:   "cloudprober-alert@localhost",
				server: "localhost",
			},
		},
		{
			name:     "error_no_to",
			emailCfg: &configpb.Email{},
			wantErr:  true,
		},
		{
			name: "error_no_password",
			emailCfg: &configpb.Email{
				To:           []string{"b@cloudprober.org"},
				SmtpServer:   "smtp1.gmail.com:587",
				SmtpUsername: "user1@gmail.com",
			},
			wantErr: true,
		},
		{
			name: "config",
			emailCfg: &configpb.Email{
				To:           to,
				SmtpServer:   smtpServer,
				SmtpUsername: "user2@gmail.com",
				SmtpPassword: "password",
			},
			want: &emailNotifier{
				to:     to,
				from:   "user2@gmail.com",
				server: smtpServer,
				auth:   smtp.PlainAuth("", "user2@gmail.com", "password", smtpHost),
			},
		},
		{
			name: "config_from_env",
			emailCfg: &configpb.Email{
				To:         to[:1],
				SmtpServer: smtpServer,
			},
			env: map[string]string{
				"SMTP_SERVER":   "smtp1.yahoo.com:587",
				"SMTP_USERNAME": "user1@gmail.com",
				"SMTP_PASSWORD": "password",
			},
			want: &emailNotifier{
				to:     to[:1],
				from:   "user1@gmail.com",
				server: smtpServer,
				auth:   smtp.PlainAuth("", "user1@gmail.com", "password", smtpHost),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}
			got, err := newEmailNotifier(tt.emailCfg, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("newEmailNotifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEmailNotifierNotify(t *testing.T) {
	tests := []struct {
		name    string
		to      []string
		wantTo  []string
		wantMsg string
	}{
		{
			name:    "default",
			to:      []string{"user1@gmail.com"},
			wantTo:  []string{"user1@gmail.com"},
			wantMsg: "From: f@gmail.com\r\nTo: user1@gmail.com\r\nSubject: summary1\r\n\r\ndetails1\nDetails:\ntarget.label.owner: manugarg@cloudprober.org\nteam: sre\r\n",
		},
		{
			name:    "email_substitution",
			to:      []string{"@target.label.owner@"},
			wantTo:  []string{"manugarg@cloudprober.org"},
			wantMsg: "From: f@gmail.com\r\nTo: manugarg@cloudprober.org\r\nSubject: summary1\r\n\r\ndetails1\nDetails:\ntarget.label.owner: manugarg@cloudprober.org\nteam: sre\r\n",
		},
	}

	for _, tt := range tests {
		var (
			smtpServer  = "smtp.gmail.com:587"
			from        = "f@gmail.com"
			wantFrom    = from
			wantServer  = smtpServer
			alertFields = map[string]string{
				"summary":            "summary1",
				"details":            "details1",
				"target.label.owner": "manugarg@cloudprober.org",
				"team":               "sre",
			}
		)

		en := &emailNotifier{
			to:     tt.to,
			from:   from,
			server: smtpServer,
		}

		var gotMsg, gotFrom, gotServer string
		var gotTo []string

		en.sendMailFunc = func(server string, auth smtp.Auth, from string, to []string, msg []byte) error {
			gotMsg = string(msg)
			gotFrom = from
			gotTo = to
			gotServer = server
			return nil
		}

		en.Notify(context.Background(), alertFields)

		assert.Equal(t, tt.wantMsg, gotMsg)
		assert.Equal(t, tt.wantTo, gotTo)
		assert.Equal(t, wantServer, gotServer)
		assert.Equal(t, wantFrom, gotFrom)
	}
}

func TestEmaiFrom(t *testing.T) {
	tests := []struct {
		name        string
		user        string
		envHostname string
		emailCfg    *configpb.Email
		want        string
	}{
		{
			name:     "configured",
			user:     "user1@gmail.com",
			emailCfg: &configpb.Email{From: "fromuser@gmail.com"},
			want:     "fromuser@gmail.com",
		},
		{
			name:     "from_user",
			user:     "user1@gmail.com",
			emailCfg: &configpb.Email{},
			want:     "user1@gmail.com",
		},
		{
			name:        "default_from_env",
			emailCfg:    &configpb.Email{},
			envHostname: "cloudprober.org",
			want:        "cloudprober-alert@cloudprober.org",
		},
		{
			name:     "default",
			emailCfg: &configpb.Email{},
			want:     "cloudprober-alert@localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envHostname != "" {
				os.Setenv("HOSTNAME", tt.envHostname)
				defer os.Unsetenv("HOSTNAME")
			}
			assert.Equal(t, tt.want, emailFrom(tt.user, tt.emailCfg))
		})
	}
}
