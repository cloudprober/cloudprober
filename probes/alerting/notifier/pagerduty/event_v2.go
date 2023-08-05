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

// Pagerduty EventV2. This package implements the EventsV2 interface to send alerts to PagerDuty.
// Specifically, it implements the "Alert Event" component of the EventsV2 API.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// EventV2Request is the data structure for a Pagerduty event.
// The event can either be an alarm event, or a change event.
type EventV2Request struct {
	RoutingKey  string         `json:"routing_key"` // required
	DedupKey    string         `json:"dedup_key,omitempty"`
	EventAction EventV2Action  `json:"event_action"` // required
	Client      string         `json:"client,omitempty"`
	ClientURL   string         `json:"client_url,omitempty"`
	Links       []EventV2Links `json:"links,omitempty"`
	Payload     EventV2Payload `json:"payload"` // required
}

// EventV2Links is the data structure for the links field of a PagerDuty event.
type EventV2Links struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

// EventV2Payload is the data structure for the payload of a PagerDuty event.
// This struct encapsulates metadata about the alarm.
type EventV2Payload struct {
	Summary       string            `json:"summary"`  // required
	Source        string            `json:"source"`   // required
	Severity      string            `json:"severity"` // required
	Timestamp     string            `json:"timestamp,omitempty"`
	Component     string            `json:"component,omitempty"`
	CustomDetails map[string]string `json:"custom_details,omitempty"`
}

// EventV2Action is the action to be performed on the event, documented as:
// The type of event. Can be trigger, acknowledge or resolve.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event#event-action-behavior
type EventV2Action string

const (
	Trigger     EventV2Action = "trigger"
	Acknowledge               = "acknowledge"
	Resolve                   = "resolve"
)

// EventV2Severity is the severity of the event, documented as:
// The perceived severity of the status the event is describing with respect
// to the affected system. This can be critical, error, warning or info.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event#parameters
type EventV2Severity string

const (
	Critical EventV2Severity = "critical"
	Error                    = "error"
	Warning                  = "warning"
	Info                     = "info"
)

// EventV2Response is the data structure that is returned from PagerDuty when
// sending a EventV2 request.
type EventV2Response struct {
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
	DedupKey string `json:"dedup_key,omitempty"`
}

// sendEventV2 sends an event to PagerDuty using the V2 API.
func (c *Client) sendEventV2(event *EventV2Request) (*EventV2Response, error) {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.hostname+"/v2/enqueue", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// check status code, return error if not 202
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("PagerDuty API returned status code %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	body := &EventV2Response{}
	err = json.Unmarshal(b, body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// createEventV2Request creates a new PagerDuty event, from the alertFields that are passed
// in from the alerting package.
func (c *Client) createEventV2Request(alertFields map[string]string) *EventV2Request {
	event := EventV2Request{
		RoutingKey:  c.routingKey,
		DedupKey:    eventV2DedupeKey(alertFields),
		EventAction: Trigger,
		Client:      "Cloudprober",
		ClientURL:   "https://cloudprober.org/",
		Payload: EventV2Payload{
			Summary:   alertFields["summary"],
			Source:    alertFields["target"],
			Severity:  "critical",
			Timestamp: alertFields["since"],
			Component: alertFields["probe"],
			// pass all alert fields as custom details
			CustomDetails: alertFields,
		},
	}

	// add links for the dashboard and playbook to the event if they
	// are present in the alertFields
	if links := generateLinks(alertFields); len(links) > 0 {
		event.Links = links
	}

	return &event
}

// generateLinks generates a slice of EventV2Links from the alertFields.
func generateLinks(alertFields map[string]string) []EventV2Links {
	links := make([]EventV2Links, 0)

	if url, exists := alertFields["dashboard_url"]; exists && url != "" {
		links = append(links, EventV2Links{
			Href: alertFields["dashboard_url"],
			Text: "Dashboard",
		})
	}

	if url, exists := alertFields["playbook_url"]; exists && url != "" {
		links = append(links, EventV2Links{
			Href: alertFields["playbook_url"],
			Text: "Playbook",
		})
	}

	return links
}

// dedupeKey returns a key that can be used to dedupe PagerDuty events.
// note: submitting subsequent events with the same dedup_key will result in
// those events being applied to an open alert matching that dedup_key.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event#alert-de-duplication
func eventV2DedupeKey(alertFields map[string]string) string {
	return alertFields["condition_id"]
}
