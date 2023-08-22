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

	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewEmailNotifier(t *testing.T) {
	const smtpServer = "smtp.gmail.com:587"
	const smtpHost = "smtp.gmail.com"

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
				To:   []string{"a@cloudprober.org"},
				From: "",
			},
			want: &emailNotifier{
				to:     []string{"a@cloudprober.org"},
				from:   "alert-notification@cloudprober.org",
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
				To:         []string{"b@cloudprober.org"},
				SmtpServer: "smtp1.gmail.com:587",
				SmtpUser:   "user1@gmail.com",
			},
			wantErr: true,
		},
		{
			name: "config",
			emailCfg: &configpb.Email{
				To:           []string{"a@cloudprober.org"},
				SmtpServer:   smtpServer,
				SmtpUser:     "user2@gmail.com",
				SmtpPassword: "password",
			},
			want: &emailNotifier{
				to:     []string{"a@cloudprober.org"},
				from:   "user2@gmail.com",
				server: smtpServer,
				auth:   smtp.PlainAuth("", "user2@gmail.com", "password", smtpHost),
			},
		},
		{
			name: "config_from_env",
			emailCfg: &configpb.Email{
				To:         []string{"a@cloudprober.org"},
				SmtpServer: smtpServer,
			},
			env: map[string]string{
				"SMTP_USER":     "user1@gmail.com",
				"SMTP_PASSWORD": "password",
			},
			want: &emailNotifier{
				to:     []string{"a@cloudprober.org"},
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
	var (
		smtpServer  = "smtp.gmail.com:587"
		to          = []string{"user@gmail.com"}
		from        = "f@gmail.com"
		alertFields = map[string]string{
			"summary": "summary1",
			"details": "details1",
		}
		wantFrom   = from
		wantTo     = to
		wantServer = smtpServer
		wantMsg    = "From: f@gmail.com\r\nTo: user@gmail.com\r\nSubject: summary1\r\n\r\ndetails1\r\n"
	)

	en := &emailNotifier{
		to:     to,
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

	assert.Equal(t, wantMsg, gotMsg)
	assert.Equal(t, wantTo, gotTo)
	assert.Equal(t, wantServer, gotServer)
	assert.Equal(t, wantFrom, gotFrom)
}
