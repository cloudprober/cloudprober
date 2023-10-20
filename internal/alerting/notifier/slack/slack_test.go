package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
)

func TestSlackNew(t *testing.T) {
	tests := map[string]struct {
		cfg         *configpb.Slack
		envVars     map[string]string
		wantWebhook string
	}{
		"valid config": {
			cfg: &configpb.Slack{
				WebhookUrl: "test-webhook-url",
			},
			envVars:     map[string]string{},
			wantWebhook: "test-webhook-url",
		},
		"valid config with env var": {
			cfg: &configpb.Slack{
				WebhookUrl: "test-webhook-url",
			},
			envVars: map[string]string{
				"SLACK_WEBHOOK_URL": "test-webhook-url-env-var",
			},
			wantWebhook: "test-webhook-url",
		},
		"env var": {
			cfg: &configpb.Slack{},
			envVars: map[string]string{
				"SLACK_WEBHOOK_URL": "test-webhook-url-env-var",
			},
			wantWebhook: "test-webhook-url-env-var",
		},
		"env var override": {
			cfg: &configpb.Slack{
				WebhookUrlEnvVar: "SLACK_WEBHOOK_URL_OVERRIDE",
			},
			envVars: map[string]string{
				"SLACK_WEBHOOK_URL_OVERRIDE": "test-webhook-url-env-var",
				"SLACK_WEBHOOK_URL":          "test-webhook-url-env-var-2",
			},
			wantWebhook: "test-webhook-url-env-var",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			c, err := New(tc.cfg, nil)
			if err != nil {
				t.Errorf("New() error = %v", err)
			}

			if c.webhookURL != tc.wantWebhook {
				t.Errorf("New() = %v, want %v", c.webhookURL, tc.wantWebhook)
			}
		})
	}
}

func TestSlackCreateWebhookMessage(t *testing.T) {
	tests := map[string]struct {
		alertFields map[string]string
		want        webhookMessage
	}{
		"valid": {
			alertFields: map[string]string{
				"details":       "test-details",
				"severity":      "critical",
				"dashboard_url": "http://localhost:9313/status?probe=test-probe",
			},
			want: webhookMessage{
				Text: `test-details

Details:
dashboard_url: http://localhost:9313/status?probe=test-probe
severity: critical`,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := createMessage(tc.alertFields)
			if got != tc.want {
				t.Errorf("createMessage() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSlackNotify(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer httpServer.Close()

	tests := map[string]struct {
		alertFields map[string]string
		wantErr     bool
	}{
		"valid": {
			alertFields: map[string]string{
				"summary": "test-summary",
			},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Client{
				httpClient: httpServer.Client(),
				webhookURL: httpServer.URL,
			}

			err := c.Notify(context.Background(), tc.alertFields)
			if (err != nil) != tc.wantErr {
				t.Errorf("Notify() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestSlackNotifyError(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer httpServer.Close()

	c := &Client{
		httpClient: httpServer.Client(),
		webhookURL: httpServer.URL,
	}

	err := c.Notify(context.Background(), map[string]string{
		"details": "test-details",
	})

	if err == nil {
		t.Errorf("Notify() error = %v, wantErr %v", err, true)
	}
}
