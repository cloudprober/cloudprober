// Copyright 2023 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pagerduty

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/stretchr/testify/assert"
)

func newPagerDutyEventV2TestServer(testCallback func(r *http.Request) error) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/enqueue", func(w http.ResponseWriter, r *http.Request) {
		err := testCallback(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, err)
			return
		}

		// Respond
		w.WriteHeader(http.StatusAccepted)

		fmt.Fprintln(w, `{"status":"success","message":"Event processed","dedup_key":"test-dedupe-key"}`)
	})

	return httptest.NewServer(mux)
}

func TestPagerDutySendEventV2(t *testing.T) {
	server := newPagerDutyEventV2TestServer(func(r *http.Request) error {
		// Check request
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}

		if r.URL.Path != "/v2/enqueue" {
			t.Errorf("Expected path /v2/enqueue, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected content type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Check body
		body := &EventV2Request{}
		err := json.NewDecoder(r.Body).Decode(body)
		if err != nil {
			t.Errorf("Error decoding body: %v", err)
		}

		if body.RoutingKey != "test-routing-key" {
			t.Errorf("Expected routing key 'test-routing-key', got %s", body.RoutingKey)
		}

		if body.EventAction != Trigger {
			t.Errorf("Expected event action 'trigger', got %s", body.EventAction)
		}

		if body.Payload.Summary != "test-summary" {
			t.Errorf("Expected payload summary 'test-summary', got %s", body.Payload.Summary)
		}

		return nil
	})
	defer server.Close()

	pagerdutyConfig := &configpb.PagerDuty{
		ApiUrl:     server.URL,
		RoutingKey: "test-routing-key",
	}

	p, err := New(pagerdutyConfig, nil)
	if err != nil {
		t.Errorf("Error creating PagerDuty client: %v", err)
	}

	event := &EventV2Request{
		RoutingKey:  "test-routing-key",
		EventAction: Trigger,
		Payload: EventV2Payload{
			Summary:  "test-summary",
			Source:   "test-source",
			Severity: "critical",
		},
	}

	resp, err := p.sendEventV2(event)
	if err != nil {
		t.Errorf("Error sending event: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Expected status success, got %s", resp.Status)
	}

	if resp.Message != "Event processed" {
		t.Errorf("Expected message 'Event processed', got %s", resp.Message)
	}

	if resp.DedupKey != "test-dedupe-key" {
		t.Errorf("Expected Dedupe key 'test-dedupe-key', got %s", resp.DedupKey)
	}
}

func TestPagerDutySendEventV2Error(t *testing.T) {
	server := newPagerDutyEventV2TestServer(func(r *http.Request) error {
		return fmt.Errorf("test-error")
	})
	defer server.Close()

	pagerdutyConfig := &configpb.PagerDuty{
		ApiUrl:     server.URL,
		RoutingKey: "test-routing-key",
	}

	p, err := New(pagerdutyConfig, nil)
	if err != nil {
		t.Errorf("Error creating PagerDuty client: %v", err)
	}

	_, err = p.sendEventV2(&EventV2Request{})
	if err == nil {
		t.Errorf("Expected error sending event")
	}
}

func TestPagerDutyEventV2DedupeKey(t *testing.T) {
	assert.Equal(t, "testConditionId", eventV2DedupeKey(&alertinfo.AlertInfo{DeduplicationID: "testConditionId"}))
}

func TestPagerDutyCreateTriggerRequest(t *testing.T) {
	tests := map[string]struct {
		severity     string
		details      string
		wantSeverity EventV2Severity
	}{
		"default_severity": {
			severity:     "",
			wantSeverity: Error,
		},
		"uppercase_sev": {
			severity:     "CRITICAL",
			wantSeverity: Critical,
		},
		"unknown_sec": {
			severity:     "fatal",
			wantSeverity: Error,
		},
		"details_dont_show": {
			details:      "test-details",
			wantSeverity: Error,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pagerdutyConfig := &configpb.PagerDuty{
				RoutingKey: "test-routing-key",
			}

			p, err := New(pagerdutyConfig, nil)
			if err != nil {
				t.Errorf("Error creating PagerDuty client: %v", err)
			}

			alertInfo := &alertinfo.AlertInfo{
				DeduplicationID: "test-condition-id",
			}

			alertFields := map[string]string{
				"alert":         "test-alert",
				"summary":       "test-summary",
				"probe":         "test-probe",
				"target":        "test-target",
				"failures":      "1",
				"total":         "2",
				"since":         "2020-01-01T00:00:00Z",
				"dashboard_url": "test_dashboard_url",
				"playbook_url":  "test_playbook_url",
			}
			if tc.severity != "" {
				alertFields["severity"] = tc.severity
			}
			if tc.details != "" {
				alertFields["details"] = tc.details
			}

			want := &EventV2Request{
				RoutingKey:  "test-routing-key",
				DedupKey:    "test-condition-id",
				EventAction: Trigger,
				Client:      "Cloudprober",
				ClientURL:   "https://cloudprober.org/",
				Links: []EventV2Links{
					{
						Href: "test_dashboard_url",
						Text: "Dashboard",
					},
					{
						Href: "test_playbook_url",
						Text: "Playbook",
					},
				},
				Payload: EventV2Payload{
					Summary:   "test-summary",
					Source:    "test-target",
					Severity:  tc.wantSeverity,
					Timestamp: "2020-01-01T00:00:00Z",
					Component: "test-probe",
					CustomDetails: map[string]string{
						"alert":         "test-alert",
						"probe":         "test-probe",
						"target":        "test-target",
						"failures":      "1",
						"total":         "2",
						"since":         "2020-01-01T00:00:00Z",
						"dashboard_url": "test_dashboard_url",
						"playbook_url":  "test_playbook_url",
					},
				},
			}

			assert.Equal(t, want, p.createTriggerRequest(alertInfo, alertFields), "Requests don't match")
		})
	}
}

func TestPagerDutyCreateResolveRequest(t *testing.T) {
	tests := map[string]struct {
		alertInfo   *alertinfo.AlertInfo
		alertFields map[string]string
		want        *EventV2Request
	}{
		"simple": {
			alertInfo: &alertinfo.AlertInfo{
				DeduplicationID: "test-condition-id",
			},
			alertFields: map[string]string{
				"summary": "test-summary",
				"target":  "test-target",
			},
			want: &EventV2Request{
				RoutingKey:  "test-routing-key",
				DedupKey:    "test-condition-id",
				EventAction: Resolve,
				Payload: EventV2Payload{
					Summary:  "test-summary",
					Severity: Error,
					Source:   "test-target",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pagerdutyConfig := &configpb.PagerDuty{
				RoutingKey: "test-routing-key",
			}

			p, err := New(pagerdutyConfig, nil)
			if err != nil {
				t.Errorf("Error creating PagerDuty client: %v", err)
			}

			assert.Equal(t, tc.want, p.createResolveRequest(tc.alertInfo, tc.alertFields), "Requests don't match")
		})
	}
}

func TestGenerateLinks(t *testing.T) {
	tests := map[string]struct {
		alertFields map[string]string
		want        []EventV2Links
	}{
		"dashboard_and_playback": {
			alertFields: map[string]string{
				"dashboard_url": "test-dashboard-url",
				"playbook_url":  "test-playbook-url",
			},
			want: []EventV2Links{
				{
					Href: "test-dashboard-url",
					Text: "Dashboard",
				},
				{
					Href: "test-playbook-url",
					Text: "Playbook",
				},
			},
		},
		"dashboard_only": {
			alertFields: map[string]string{
				"dashboard_url": "test-dashboard-url",
			},
			want: []EventV2Links{
				{
					Href: "test-dashboard-url",
					Text: "Dashboard",
				},
			},
		},
		"playbook_only": {
			alertFields: map[string]string{
				"playbook_url": "test-playbook-url",
			},
			want: []EventV2Links{
				{
					Href: "test-playbook-url",
					Text: "Playbook",
				},
			},
		},
		"none": {
			alertFields: map[string]string{},
			want:        []EventV2Links{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := generateLinks(tc.alertFields)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("generateLinks = \n%+v\n, want \n%+v\n", got, tc.want)
			}
		})
	}
}

func TestPagerDutyLookupRoutingKey(t *testing.T) {
	tests := map[string]struct {
		pagerdutyConfig *configpb.PagerDuty
		env             map[string]string
		want            string
		wantErr         bool
	}{
		"env_var": {
			pagerdutyConfig: &configpb.PagerDuty{
				RoutingKeyEnvVar: "TEST_ROUTING_KEY",
			},
			env: map[string]string{
				"TEST_ROUTING_KEY": "test-routing-key",
			},
			want:    "test-routing-key",
			wantErr: false,
		},
		"config": {
			pagerdutyConfig: &configpb.PagerDuty{
				RoutingKey: "test-routing-key",
			},
			env:     map[string]string{},
			want:    "test-routing-key",
			wantErr: false,
		},
		"no_routing_key": {
			pagerdutyConfig: &configpb.PagerDuty{},
			env:             map[string]string{},
			want:            "",
			wantErr:         true,
		},
		"config_overrides_env_var": {
			pagerdutyConfig: &configpb.PagerDuty{
				RoutingKey:       "test-routing-key",
				RoutingKeyEnvVar: "TEST_ROUTING_KEY",
			},
			env: map[string]string{
				"TEST_ROUTING_KEY": "test-routing-key-env-var",
			},
			want:    "test-routing-key",
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			got, err := lookupRoutingKey(tc.pagerdutyConfig)
			if (err != nil) != tc.wantErr {
				t.Errorf("lookupRoutingKey() error = %v, wantErr %v", err, tc.wantErr)
			}

			if got != tc.want {
				t.Errorf("lookupRoutingKey() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPagerDutyRoutingKeyEnvVar(t *testing.T) {
	tests := map[string]struct {
		pagerdutyConfig *configpb.PagerDuty
		want            string
	}{
		"env_var": {
			pagerdutyConfig: &configpb.PagerDuty{
				RoutingKeyEnvVar: "TEST_ROUTING_KEY",
			},
			want: "TEST_ROUTING_KEY",
		},
		"default": {
			pagerdutyConfig: &configpb.PagerDuty{},
			want:            DEFAULT_PAGERDUTY_ROUTING_KEY_ENV_VAR,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := routingKeyEnvVar(tc.pagerdutyConfig)
			if got != tc.want {
				t.Errorf("routingKeyEnvVar() = %v, want %v", got, tc.want)
			}
		})
	}
}
