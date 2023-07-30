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
package notifier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
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
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"success","message":"Event processed","incident_key":"test-incident-key"}`)
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

	p := newPagerDutyClient(server.URL, "test-api-token")

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

	if resp.IncidentKey != "test-incident-key" {
		t.Errorf("Expected incident key 'test-incident-key', got %s", resp.IncidentKey)
	}
}

func TestPagerDutySendEventV2Authentication(t *testing.T) {
	server := newPagerDutyEventV2TestServer(func(r *http.Request) error {
		if r.Header.Get("Authorization") != "Token token=test-api-token" {
			t.Errorf("Expected Authorization token, got %s", r.Header.Get("Authorization"))
		}

		return nil
	})
	defer server.Close()

	p := newPagerDutyClient(server.URL, "test-api-token")

	event := &EventV2Request{
		RoutingKey:  "test-routing-key",
		EventAction: Trigger,
		Payload: EventV2Payload{
			Summary:  "test-summary",
			Source:   "test-source",
			Severity: "critical",
		},
	}

	_, err := p.sendEventV2(event)
	if err != nil {
		t.Errorf("Error sending event: %v", err)
	}
}

func TestPagerDutySendEventV2Error(t *testing.T) {
	server := newPagerDutyEventV2TestServer(func(r *http.Request) error {
		return fmt.Errorf("test-error")
	})
	defer server.Close()

	p := newPagerDutyClient(server.URL, "test-api-token")
	_, err := p.sendEventV2(&EventV2Request{})
	if err == nil {
		t.Errorf("Expected error sending event")
	}
}

func TestPagerDutyDedupeKey(t *testing.T) {
	tests := map[string]struct {
		alertFields map[string]string
		want        string
	}{
		"simple": {
			alertFields: map[string]string{
				"alert":        "test-alert",
				"probe":        "test-probe",
				"target":       "test-target",
				"condition_id": "test-condition-id",
			},
			want: "test-alert-test-probe-test-target-test-condition-id",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := dedupeKey(tc.alertFields)
			if got != tc.want {
				t.Errorf("dedupeKey(%v) = %s, want %s", tc.alertFields, got, tc.want)
			}
		})
	}
}

func TestPagerDutyCreateEventV2Request(t *testing.T) {
	tests := map[string]struct {
		alertFields map[string]string
		want        *EventV2Request
	}{
		"simple": {
			alertFields: map[string]string{
				"alert":        "test-alert",
				"summary":      "test-summary",
				"probe":        "test-probe",
				"target":       "test-target",
				"condition_id": "test-condition-id",
				"failures":     "1",
				"total":        "2",
				"since":        "2020-01-01T00:00:00Z",
			},
			want: &EventV2Request{
				RoutingKey:  "test-api-token",
				DedupKey:    "test-alert-test-probe-test-target-test-condition-id",
				EventAction: Trigger,
				Client:      "Cloudprober",
				ClientURL:   PAGERDUTY_CLIENT_URL,
				Payload: EventV2Payload{
					Summary:   "test-summary",
					Source:    "test-target",
					Severity:  "critical",
					Timestamp: "2020-01-01T00:00:00Z",
					Component: "test-probe",
					Group:     "test-condition-id",
					CustomDetails: map[string]string{
						"alert":        "test-alert",
						"summary":      "test-summary",
						"probe":        "test-probe",
						"target":       "test-target",
						"condition_id": "test-condition-id",
						"failures":     "1",
						"total":        "2",
						"since":        "2020-01-01T00:00:00Z",
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := newPagerDutyClient("test-hostname", "test-api-token")
			got := p.createEventV2Request(tc.alertFields)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("createEventV2Request(%v) = \n%+v\n, want \n%+v\n", got, tc.want)
			}
		})
	}
}
