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

	"github.com/cloudprober/cloudprober/common/strtemplate"
	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/logger"
)

type emailNotifier struct {
	to           []string
	from         string
	server       string
	auth         smtp.Auth
	sendMailFunc func(string, smtp.Auth, string, []string, []byte) error
	l            *logger.Logger
}

func (en *emailNotifier) Notify(ctx context.Context, alertFields map[string]string) error {
	var to []string
	for _, t := range en.to {
		newTo, _ := strtemplate.SubstituteLabels(t, alertFields)
		to = append(to, newTo)
	}

	msg := fmt.Sprintf("From: %s\r\n", en.from)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(to, ","))
	msg += fmt.Sprintf("Subject: %s\r\n", alertFields["summary"])
	msgBody := alertFields["details"] + "\nDetails:\n" + alertinfo.FieldsToString(alertFields, "summary", "details")
	msg += fmt.Sprintf("\r\n%s\r\n", msgBody)

	en.l.Infof("Sending email notification to: %v \nserver: %s\nmsg:\n%s", en.to, en.server, msg)

	if en.sendMailFunc == nil {
		en.sendMailFunc = smtp.SendMail
	}
	if err := en.sendMailFunc(en.server, en.auth, en.from, to, []byte(msg)); err != nil {
		return fmt.Errorf("error while sending email notification: %v", err)
	}
	return nil
}

func smtpPassword(user string, emailCfg *configpb.Email) (string, error) {
	password := emailCfg.GetSmtpPassword()
	if password == "" {
		password = os.Getenv("SMTP_PASSWORD")
		if password == "" {
			return "", fmt.Errorf("smtp_password not configured or set through environment variable")
		}
	}
	return password, nil
}

func emailFrom(user string, emailCfg *configpb.Email) string {
	if from := emailCfg.GetFrom(); from != "" {
		return from
	}

	if user != "" {
		return user
	}

	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = "localhost"
	}
	return "cloudprober-alert@" + hostname
}

func newEmailNotifier(emailCfg *configpb.Email, l *logger.Logger) (*emailNotifier, error) {
	server := emailCfg.GetSmtpServer()
	if server == "" {
		server = os.Getenv("SMTP_SERVER")
		if server == "" {
			l.Warningf("Neither smtp_server is configured, nor SMTP_SERVER environment variable is set, using localhost for email notification")
			server = "localhost"
		}
	}

	user := emailCfg.GetSmtpUsername()
	if user == "" {
		user = os.Getenv("SMTP_USERNAME")
		if user == "" && server != "localhost" {
			l.Warningf("smtp_user not configured or set through environment variable, will skip SMTP authentication")
		}
	}

	if len(emailCfg.To) == 0 {
		return nil, fmt.Errorf("no email recipients configured")
	}

	en := &emailNotifier{
		to:     emailCfg.To,
		from:   emailFrom(user, emailCfg),
		server: server,
		l:      l,
	}

	if user != "" {
		password, err := smtpPassword(user, emailCfg)
		if err != nil {
			return nil, err
		}
		en.auth = smtp.PlainAuth("", user, password, strings.Split(server, ":")[0])
	}

	return en, nil
}
