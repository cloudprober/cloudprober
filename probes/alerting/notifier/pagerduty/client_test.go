package notifier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestSendEventV2(t *testing.T) {
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
		body := &PagerDutyEventV2Request{}
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

	p := NewPagerDutyClient(server.URL, "test-api-token")

	event := &PagerDutyEventV2Request{
		RoutingKey:  "test-routing-key",
		EventAction: Trigger,
		Payload: PagerDutyEventV2Payload{
			Summary:  "test-summary",
			Source:   "test-source",
			Severity: "critical",
		},
	}

	resp, err := p.SendEventV2(event)
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

func TestSendEventV2Authentication(t *testing.T) {
	server := newPagerDutyEventV2TestServer(func(r *http.Request) error {
		if r.Header.Get("Authorization") != "Token token=test-api-token" {
			t.Errorf("Expected Authorization token, got %s", r.Header.Get("Authorization"))
		}

		return nil
	})
	defer server.Close()

	p := NewPagerDutyClient(server.URL, "test-api-token")

	event := &PagerDutyEventV2Request{
		RoutingKey:  "test-routing-key",
		EventAction: Trigger,
		Payload: PagerDutyEventV2Payload{
			Summary:  "test-summary",
			Source:   "test-source",
			Severity: "critical",
		},
	}

	_, err := p.SendEventV2(event)
	if err != nil {
		t.Errorf("Error sending event: %v", err)
	}
}
