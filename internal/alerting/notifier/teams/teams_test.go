package teams

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
)

func TestTeamsNew(t *testing.T) {
	tests := map[string]struct {
		cfg         *configpb.Teams
		envVars     map[string]string
		wantWebhook string
	}{
		"valid config": {
			cfg: &configpb.Teams{
				WebhookUrl: "test-webhook-url",
			},
			envVars:     map[string]string{},
			wantWebhook: "test-webhook-url",
		},
		"valid config with env var": {
			cfg: &configpb.Teams{
				WebhookUrl: "test-webhook-url",
			},
			envVars: map[string]string{
				"TEAMS_WEBHOOK_URL": "test-webhook-url-env-var",
			},
			wantWebhook: "test-webhook-url",
		},
		"env var": {
			cfg: &configpb.Teams{},
			envVars: map[string]string{
				"TEAMS_WEBHOOK_URL": "test-webhook-url-env-var",
			},
			wantWebhook: "test-webhook-url-env-var",
		},
		"env var override": {
			cfg: &configpb.Teams{
				WebhookUrlEnvVar: "TEAMS_WEBHOOK_URL_OVERRIDE",
			},
			envVars: map[string]string{
				"TEAMS_WEBHOOK_URL_OVERRIDE": "test-webhook-url-env-var",
				"TEAMS_WEBHOOK_URL":          "test-webhook-url-env-var-2",
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

func TestTeamsCreateWebhookMessage(t *testing.T) {
	tests := map[string]struct {
		alertFields map[string]string
		want        webhookMessage
	}{
		"valid": {
			alertFields: map[string]string{
				"alert":         "teams_webhook_test",
				"target":        "127.0.0.1:8000",
				"failures":      "1",
				"total":         "1",
				"since":         "2026-05-27T15:20:18-04:00",
				"probe":         "teams-test",
				"playbook_url":  "",
				"dashboard_url": "http://localhost:9313/status?probe=teams-test",
			},
			want: webhookMessage{
				Type: "message",
				Attachments: []attachment{{
					ContentType: "application/vnd.microsoft.card.adaptive",
					Content: adaptiveCard{
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
									{Title: "Alert", Value: "teams_webhook_test"},
									{Title: "Target", Value: "127.0.0.1:8000"},
									{Title: "Failures", Value: "1 out of 1 probes"},
									{Title: "Failing since", Value: "2026-05-27T15:20:18-04:00"},
									{Title: "Probe", Value: "teams-test"},
								},
							},
						},
						Actions: []action{
							{
								Type:  "Action.OpenUrl",
								Title: "Open Dashboard",
								URL:   "http://localhost:9313/status?probe=teams-test",
							},
						},
					},
				}},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := createMessage(tc.alertFields)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("createMessage() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTeamsNotify(t *testing.T) {
	var gotBody []byte
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer httpServer.Close()

	tests := map[string]struct {
		alertFields map[string]string
		want        webhookMessage
		wantErr     bool
	}{
		"valid": {
			alertFields: map[string]string{
				"alert":         "teams_webhook_test",
				"target":        "127.0.0.1:8000",
				"failures":      "1",
				"total":         "1",
				"since":         "2026-05-27T15:20:18-04:00",
				"probe":         "teams-test",
				"playbook_url":  "",
				"dashboard_url": "http://localhost:9313/status?probe=teams-test",
			},
			want: webhookMessage{
				Type: "message",
				Attachments: []attachment{{
					ContentType: "application/vnd.microsoft.card.adaptive",
					Content: adaptiveCard{
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
									{Title: "Alert", Value: "teams_webhook_test"},
									{Title: "Target", Value: "127.0.0.1:8000"},
									{Title: "Failures", Value: "1 out of 1 probes"},
									{Title: "Failing since", Value: "2026-05-27T15:20:18-04:00"},
									{Title: "Probe", Value: "teams-test"},
								},
							},
						},
						Actions: []action{
							{
								Type:  "Action.OpenUrl",
								Title: "Open Dashboard",
								URL:   "http://localhost:9313/status?probe=teams-test",
							},
						},
					},
				}},
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
			wantBody, err := json.Marshal(tc.want)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			var gotJSON, wantJSON any
			if err := json.Unmarshal(gotBody, &gotJSON); err != nil {
				t.Fatalf("Unmarshal(gotBody) error = %v", err)
			}
			if err := json.Unmarshal(wantBody, &wantJSON); err != nil {
				t.Fatalf("Unmarshal(wantBody) error = %v", err)
			}
			if !reflect.DeepEqual(gotJSON, wantJSON) {
				t.Errorf("Notify() sent = %s, want %s", string(gotBody), string(wantBody))
			}
		})
	}
}

func TestTeamsNotifyError(t *testing.T) {
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
