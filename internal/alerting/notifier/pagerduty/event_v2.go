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

// Pagerduty EventV2. This file, part of the pagerduty package, implements
// the EventsV2 interface to send alerts to PagerDuty. Specifically, it
// implements the "Alert Event"component of the EventsV2 API.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event

package pagerduty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
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
	Severity      EventV2Severity   `json:"severity"` // required
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

	c.l.Infof("Sending event to PagerDuty (URL: %s): %s", req.URL.String(), event.Payload.Summary)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// check status code, return error if not 202
	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("PagerDuty API returned status code %d, response: %s", resp.StatusCode, string(b))
	}

	body := &EventV2Response{}
	err = json.Unmarshal(b, body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func severityFromAlertFields(alertFields map[string]string) EventV2Severity {
	severity := strings.ToLower(alertFields["severity"])
	if severity != "critical" && severity != "error" && severity != "warning" && severity != "info" {
		return Error
	}
	return EventV2Severity(severity)
}

func payloadFromAlertFields(alertFields map[string]string) EventV2Payload {
	customDetails := make(map[string]string)
	for k, v := range alertFields {
		if k != "summary" && k != "details" && k != "severity" {
			customDetails[k] = v
		}
	}

	return EventV2Payload{
		Summary:       alertFields["summary"],
		Source:        alertFields["target"],
		Severity:      severityFromAlertFields(alertFields),
		Timestamp:     alertFields["since"],
		Component:     alertFields["probe"],
		CustomDetails: customDetails,
	}
}

// createTriggerRequest creates a new PagerDuty event, from the alertFields
// that are passed in from the alerting package.
func (c *Client) createTriggerRequest(alertInfo *alertinfo.AlertInfo, alertFields map[string]string) *EventV2Request {
	event := &EventV2Request{
		RoutingKey:  c.routingKey,
		DedupKey:    eventV2DedupeKey(alertInfo),
		EventAction: Trigger,
		Client:      "Cloudprober",
		ClientURL:   "https://cloudprober.org/",
		Payload:     payloadFromAlertFields(alertFields),
	}

	// add links for the dashboard and playbook to the event if they
	// are present in the alertFields
	if links := generateLinks(alertFields); len(links) > 0 {
		event.Links = links
	}

	return event
}

// createTriggerRequest creates a new PagerDuty event, from the alertFields
// that are passed in from the alerting package.
func (c *Client) createResolveRequest(alertInfo *alertinfo.AlertInfo, alertFields map[string]string) *EventV2Request {
	return &EventV2Request{
		RoutingKey:  c.routingKey,
		DedupKey:    eventV2DedupeKey(alertInfo),
		EventAction: Resolve,
		Payload: EventV2Payload{
			Summary:  alertFields["summary"],
			Source:   alertFields["target"],
			Severity: severityFromAlertFields(alertFields),
		},
	}
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
func eventV2DedupeKey(alertInfo *alertinfo.AlertInfo) string {
	return alertInfo.DeduplicationID
}
