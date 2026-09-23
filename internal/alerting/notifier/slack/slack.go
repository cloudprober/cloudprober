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

// Package slack implements slack notifications for Cloudprober alert events.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/logger"
)

const (
	// DefaultSlackWebhookURLEnvVar is the default environment variable
	// to use for the Slack webhook URL.
	DefaultSlackWebhookURLEnvVar = "SLACK_WEBHOOK_URL"
)

// Client is a Slack client.
type Client struct {
	httpClient *http.Client
	logger     *logger.Logger
	webhookURL string
}

// New creates a new Slack client.
func New(slackcfg *configpb.Slack, l *logger.Logger) (*Client, error) {
	webhookURL, err := lookupWebhookUrl(slackcfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{},
		logger:     l,
		webhookURL: webhookURL,
	}, nil
}

// lookupWebhookUrl looks up the webhook URL to use for the Slack client,
// in order of precedence:
// 1. Webhook URL in the config
// 2. Webhook URL environment variable
func lookupWebhookUrl(slackcfg *configpb.Slack) (string, error) {
	// check if the webhook URL is set in the config
	if slackcfg.GetWebhookUrl() != "" {
		return slackcfg.GetWebhookUrl(), nil
	}

	// check if the environment variable is set for the webhook URL
	if webhookURL, exists := os.LookupEnv(webhookUrlEnvVar(slackcfg)); exists {
		return webhookURL, nil
	}

	return "", fmt.Errorf("no Slack webhook URL found")
}

// webhookUrlEnvVar returns the environment variable to use for the Slack webhook URL.
func webhookUrlEnvVar(slackcfg *configpb.Slack) string {
	if slackcfg.GetWebhookUrlEnvVar() != "" {
		return slackcfg.GetWebhookUrlEnvVar()
	}

	return DefaultSlackWebhookURLEnvVar
}

// webhookMessage is the message that is sent to the Slack webhook.
type webhookMessage struct {
	Text string `json:"text"`
}

// Notify sends a notification to Slack.
func (c *Client) Notify(ctx context.Context, alertFields map[string]string) error {
	message := createMessage(alertFields)

	jsonBody, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check status code, return error if not 200
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack webhook returned error; statusCode: %d, response: %s", resp.StatusCode, string(b))
	}

	return nil
}

// createMessage creates a new Slack webhook message, from the alertFields
func createMessage(alertFields map[string]string) webhookMessage {
	return webhookMessage{
		Text: alertFields["details"] + "\n\nDetails:\n" + alertinfo.FieldsToString(alertFields, "details", "summary"),
	}
}
