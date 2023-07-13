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
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
)

type emailNotifier struct {
	to     []string
	from   string
	server string
	auth   smtp.Auth
	l      *logger.Logger
}

func (en *emailNotifier) Notify(ctx context.Context, alertFields map[string]string) error {
	msg := fmt.Sprintf("From: %s\r\n", en.from)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(en.to, ","))
	msg += fmt.Sprintf("Subject: %s\r\n", alertFields["summary"])
	msg += fmt.Sprintf("\r\n%s\r\n", alertFields["details"])
	en.l.Infof("Sending email notification to: %v \nserver: %s\nmsg:\n%s", en.to, en.server, msg)

	if err := smtp.SendMail(en.server, en.auth, en.from, en.to, []byte(msg)); err != nil {
		return fmt.Errorf("error while sending email notification: %v", err)
	}
	return nil
}

func newEmailNotifier(notifyCfg *configpb.NotifyConfig, l *logger.Logger) (*emailNotifier, error) {
	server := os.Getenv("SMTP_SERVER")
	if server == "" {
		l.Warningf("SMTP_SERVER environment variable not set, using localhost for email notification")
		server = "localhost"
	}

	username := os.Getenv("SMTP_USERNAME")
	if username == "" {
		// Complain only if we are not using localhost.
		if server != "localhost" {
			l.Warningf("SMTP_USERNAME environment variable not set, will skip SMTP authentication")
		}
	}

	var password string
	if username != "" {
		password = os.Getenv("SMTP_PASSWORD")
		if password == "" {
			return nil, fmt.Errorf("SMTP_PASSWORD environment variable not set")
		}
	}

	from, to := notifyCfg.EmailFrom, notifyCfg.Email
	if from == "" {
		if username != "" {
			from = username
		} else {
			from = "no-reply@cloudprober." + os.Getenv("HOSTNAME")
		}
	}

	en := &emailNotifier{
		to:     to,
		from:   from,
		server: server,
		l:      l,
	}
	if username != "" {
		en.auth = smtp.PlainAuth("", username, password, strings.Split(server, ":")[0])
	}

	return en, nil
}
