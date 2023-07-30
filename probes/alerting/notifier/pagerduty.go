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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type pagerDutyClient struct {
	httpClient *http.Client
	hostname   string
	apiToken   string
}

const (
	// PAGERDUTY_API_URL is the base URL for the PagerDuty API.
	PAGERDUTY_API_URL = "https://api.pagerduty.com"

	// CLIENT_URL is the URL for the client that is triggering the event.
	// This is used in the client_url field of the event payload.
	PAGERDUTY_CLIENT_URL = "https://cloudprober.org/"
)

func newPagerDutyClient(hostname, apiToken string) *pagerDutyClient {
	return &pagerDutyClient{
		httpClient: &http.Client{},
		hostname:   hostname,
		apiToken:   apiToken,
	}
}

func (p *pagerDutyClient) Notify(ctx context.Context, alertFields map[string]string) error {
	// Create the event
	event := p.createEventV2Request(alertFields)

	// Send the event
	_, err := p.sendEventV2(event)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Refactor this as not all fields are correct.
// createEventV2Request creates a new PagerDuty event, using the V2 API.
func (p *pagerDutyClient) createEventV2Request(alertFields map[string]string) *EventV2Request {
	return &EventV2Request{
		RoutingKey:  p.apiToken,
		DedupKey:    dedupeKey(alertFields),
		EventAction: Trigger,
		Client:      "Cloudprober",
		ClientURL:   PAGERDUTY_CLIENT_URL,
		Payload: EventV2Payload{
			Summary:   alertFields["summary"],
			Source:    alertFields["target"],
			Severity:  "critical",
			Timestamp: alertFields["since"],
			Component: alertFields["probe"],
			Group:     alertFields["condition_id"],
			// pass all alert fields as custom details
			CustomDetails: alertFields,
		},
	}
}

// dedupeKey returns a unique key for the alert.
func dedupeKey(alertFields map[string]string) string {
	return fmt.Sprintf("%s-%s-%s-%s", alertFields["alert"], alertFields["probe"], alertFields["target"], alertFields["condition_id"])
}

func (p *pagerDutyClient) sendEventV2(event *EventV2Request) (*EventV2Response, error) {
	req, err := event.newHTTPRequest(p.hostname)
	if err != nil {
		return nil, err
	}

	resp, err := p.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// check status code, return error if not 200
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PagerDuty API returned status code %d, body: %s", resp.StatusCode, resp.Body)
	}

	body := &EventV2Response{}
	err = json.NewDecoder(resp.Body).Decode(body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Do sends an HTTP request and returns an HTTP response, ensuring that the API
// token is set in the request header.
func (p *pagerDutyClient) do(req *http.Request) (*http.Response, error) {
	// The PagerDuty API requires authentication, using the API token.
	// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTUx-authentication
	req.Header.Set("Authorization", "Token token="+p.apiToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// EventV2Request is the data structure for a PagerDuty event, using the V2 API.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event
type EventV2Request struct {
	RoutingKey  string          `json:"routing_key"` // required
	DedupKey    string          `json:"dedup_key,omitempty"`
	EventAction EventV2Action   `json:"event_action"` // required
	Client      string          `json:"client,omitempty"`
	ClientURL   string          `json:"client_url,omitempty"`
	Payload     EventV2Payload  `json:"payload"` // required
	Images      []EventV2Images `json:"images,omitempty"`
	Links       []EventV2Links  `json:"links,omitempty"`
}

// EventV2Payload is the data structure for the payload of a PagerDuty event, using the V2 API.
type EventV2Payload struct {
	Summary       string            `json:"summary"`  // required
	Source        string            `json:"source"`   // required
	Severity      string            `json:"severity"` // required
	Timestamp     string            `json:"timestamp,omitempty"`
	Component     string            `json:"component,omitempty"`
	Group         string            `json:"group,omitempty"`
	Class         string            `json:"class,omitempty"`
	CustomDetails map[string]string `json:"custom_details,omitempty"`
}

// EventV2Images is the data structure for an image in a PagerDuty event, using the V2 API.
type EventV2Images struct {
	Src  string `json:"src,omitempty"`
	Alt  string `json:"alt,omitempty"`
	Href string `json:"href,omitempty"`
}

// PagerDutyEventV2Links is the data structure for a link in a PagerDuty event, using the V2 API.
type EventV2Links struct {
	Href string `json:"href,omitempty"`
	Text string `json:"text,omitempty"`
}

// EventAction is the action to be performed on the event
type EventV2Action string

const (
	Trigger     EventV2Action = "trigger"
	Acknowledge               = "acknowledge"
	Resolve                   = "resolve"
)

// EventV2Response is the data structure for a PagerDuty event response, using the V2 API.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event
type EventV2Response struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	IncidentKey string `json:"incident_key"`
}

// NewRequest creates a new HTTP request for the PagerDuty event.
func (e *EventV2Request) newHTTPRequest(hostname string) (*http.Request, error) {
	jsonBody, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, hostname+"/v2/enqueue", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// The PagerDuty API requires the Accept header to be set to
	// application/vnd.pagerduty+json;version=2 to version the response.
	// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTUy-versioning
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	return req, nil
}
