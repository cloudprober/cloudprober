// Copyright 2026 The Cloudprober Authors.
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

// Package teams implements Microsoft Teams notifications for Cloudprober alert
// events.
package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/logger"
)

const (
	// DefaultTeamsWebhookURLEnvVar is the default environment variable
	// to use for the Teams webhook URL.
	DefaultTeamsWebhookURLEnvVar = "TEAMS_WEBHOOK_URL"
)

// Client is a Teams client.
type Client struct {
	httpClient *http.Client
	logger     *logger.Logger
	webhookURL string
}

// New creates a new Teams client.
func New(teamscfg *configpb.Teams, l *logger.Logger) (*Client, error) {
	webhookURL, err := lookupWebhookUrl(teamscfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{},
		logger:     l,
		webhookURL: webhookURL,
	}, nil
}

// lookupWebhookUrl looks up the webhook URL to use for the Teams client,
// in order of precedence:
// 1. Webhook URL in the config
// 2. Webhook URL environment variable
func lookupWebhookUrl(teamscfg *configpb.Teams) (string, error) {
	// check if the webhook URL is set in the config
	if teamscfg.GetWebhookUrl() != "" {
		return teamscfg.GetWebhookUrl(), nil
	}

	// check if the environment variable is set for the webhook URL
	if webhookURL, exists := os.LookupEnv(webhookUrlEnvVar(teamscfg)); exists {
		return webhookURL, nil
	}

	return "", fmt.Errorf("no Teams webhook URL found")
}

// webhookUrlEnvVar returns the environment variable to use for the Teams
func webhookUrlEnvVar(teamscfg *configpb.Teams) string {
	if teamscfg.GetWebhookUrlEnvVar() != "" {
		return teamscfg.GetWebhookUrlEnvVar()
	}

	return DefaultTeamsWebhookURLEnvVar
}

// webhookMessage is the message that is sent to the Teams webhook.
type webhookMessage struct {
	Type        string       `json:"type"`
	Attachments []attachment `json:"attachments"`
}

type attachment struct {
	ContentType string       `json:"contentType"`
	Content     adaptiveCard `json:"content"`
}

type adaptiveCard struct {
	SchemaURL string    `json:"$schema"`
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	MSTeams   teamsCard `json:"msteams"`
	Body      []any     `json:"body"`
	Actions   []action  `json:"actions,omitempty"`
}

type teamsCard struct {
	Width string `json:"width"`
}

type textBlock struct {
	Type   string `json:"type"`
	Size   string `json:"size,omitempty"`
	Weight string `json:"weight,omitempty"`
	Text   string `json:"text"`
	Wrap   bool   `json:"wrap,omitempty"`
}

type factSet struct {
	Type  string `json:"type"`
	Facts []fact `json:"facts"`
}

type fact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

type action struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Notify sends a notification to Teams.
func (c *Client) Notify(ctx context.Context, alertFields map[string]string) error {
	message := createMessage(alertFields)

	jsonBody, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.webhookURL,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check status code, return error if not 2xx
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("teams webhook returned error; statusCode: %d, response: %s", resp.StatusCode, string(b))
	}

	return nil
}

// createMessage creates a new Teams webhook message, from the alertFields
func createMessage(alertFields map[string]string) webhookMessage {
	card := adaptiveCard{
		SchemaURL: "http://adaptivecards.io/schemas/adaptive-card.json",
		Type:      "AdaptiveCard",
		Version:   "1.2",
		MSTeams: teamsCard{
			Width: "Full",
		},
		Body: []any{
			textBlock{
				Type:   "TextBlock",
				Size:   "Large",
				Weight: "Bolder",
				Text:   "Cloudprober alert",
			},
			factSet{
				Type: "FactSet",
				Facts: []fact{
					{Title: "Alert", Value: alertFields["alert"]},
					{Title: "Target", Value: alertFields["target"]},
					{
						Title: "Failures",
						Value: fmt.Sprintf(
							"%s out of %s probes",
							alertFields["failures"],
							alertFields["total"],
						),
					},
					{Title: "Failing since", Value: alertFields["since"]},
					{Title: "Probe", Value: alertFields["probe"]},
				},
			},
		},
	}

	if alertFields["dashboard_url"] != "" {
		card.Actions = []action{{
			Type:  "Action.OpenUrl",
			Title: "Open Dashboard",
			URL:   alertFields["dashboard_url"],
		}}
	}

	return webhookMessage{
		Type: "message",
		Attachments: []attachment{{
			ContentType: "application/vnd.microsoft.card.adaptive",
			Content:     card,
		}},
	}
}
