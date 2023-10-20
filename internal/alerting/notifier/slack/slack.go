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
	// DEFAULT_SLACK_WEBHOOK_URL_ENV_VAR is the default environment variable
	// to use for the Slack webhook URL.
	DEFAULT_SLACK_WEBHOOK_URL_ENV_VAR = "SLACK_WEBHOOK_URL"
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
// in order of precendence:
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

// webhookUrlEnvVar returns the environment variable to use for the Slack
func webhookUrlEnvVar(slackcfg *configpb.Slack) string {
	if slackcfg.GetWebhookUrlEnvVar() != "" {
		return slackcfg.GetWebhookUrlEnvVar()
	}

	return DEFAULT_SLACK_WEBHOOK_URL_ENV_VAR
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

	req, err := http.NewRequest(http.MethodPost, c.webhookURL, bytes.NewBuffer(jsonBody))
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
