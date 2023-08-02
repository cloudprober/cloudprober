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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
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

	notifyConfig := &configpb.NotifyConfig{
		PagerdutyApiUrl:     server.URL,
		PagerdutyRoutingKey: "test-routing-key",
	}

	l, err := logger.New(context.TODO(), "test")
	if err != nil {
		t.Errorf("Error creating logger: %v", err)
	}

	p, err := New(notifyConfig, l)
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

	notifyConfig := &configpb.NotifyConfig{
		PagerdutyApiUrl:     server.URL,
		PagerdutyRoutingKey: "test-routing-key",
	}

	l, err := logger.New(context.TODO(), "test")
	if err != nil {
		t.Errorf("Error creating logger: %v", err)
	}

	p, err := New(notifyConfig, l)
	if err != nil {
		t.Errorf("Error creating PagerDuty client: %v", err)
	}

	_, err = p.sendEventV2(&EventV2Request{})
	if err == nil {
		t.Errorf("Expected error sending event")
	}
}

func TestPagerDutyEventV2DedupeKey(t *testing.T) {
	tests := map[string]struct {
		alertFields map[string]string
		want        string
	}{
		"simple": {
			alertFields: map[string]string{
				"alert":  "test-alert",
				"probe":  "test-probe",
				"target": "test-target",
			},
			want: "test-alert-test-probe-test-target",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := eventV2DedupeKey(tc.alertFields)
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
				RoutingKey:  "test-routing-key",
				DedupKey:    "test-alert-test-probe-test-target",
				EventAction: Trigger,
				Client:      "Cloudprober",
				ClientURL:   "https://cloudprober.org/",
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
			notifyConfig := &configpb.NotifyConfig{
				PagerdutyApiUrl:     "test-hostname",
				PagerdutyRoutingKey: "test-routing-key",
			}

			l, err := logger.New(context.TODO(), "test")
			if err != nil {
				t.Errorf("Error creating logger: %v", err)
			}

			p, err := New(notifyConfig, l)
			if err != nil {
				t.Errorf("Error creating PagerDuty client: %v", err)
			}

			got := p.createEventV2Request(tc.alertFields)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("createEventV2Request = \n%+v\n, want \n%+v\n", got, tc.want)
			}
		})
	}
}
