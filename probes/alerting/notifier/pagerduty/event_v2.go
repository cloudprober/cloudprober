/*
	Pagerduty EventV2. This package implements the EventsV2 interface to send alerts to PagerDuty.
	Specifically, it implements the "Alert Event" component of the EventsV2 API.

	https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event
*/

package pagerduty

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
	RoutingKey  string          `json:"routing_key"` // required
	DedupKey    string          `json:"dedup_key,omitempty"`
	EventAction EventV2Action   `json:"event_action"` // required
	Client      string          `json:"client,omitempty"`
	ClientURL   string          `json:"client_url,omitempty"`
	Payload     EventV2Payload  `json:"payload"` // required
	Images      []EventV2Images `json:"images,omitempty"`
	Links       []EventV2Links  `json:"links,omitempty"`
}

// EventV2Payload is the data structure for the payload of a PagerDuty event.
// This struct encapsulates metadata about the alarm.
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

// EventV2Images is the data structure for an image in a PagerDuty event.
// Images are optional.
type EventV2Images struct {
	Src  string `json:"src,omitempty"`
	Alt  string `json:"alt,omitempty"`
	Href string `json:"href,omitempty"`
}

// EventV2Links is the data structure for a link in a PagerDuty event.
// Links are optional
type EventV2Links struct {
	Href string `json:"href,omitempty"`
	Text string `json:"text,omitempty"`
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
	return &EventV2Request{
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
			Group:     alertFields["condition_id"],
			// pass all alert fields as custom details
			CustomDetails: alertFields,
		},
	}
}

// dedupeKey returns a unique key for the alert.
func eventV2DedupeKey(alertFields map[string]string) string {
	return fmt.Sprintf("%s-%s-%s", alertFields["alert"], alertFields["probe"], alertFields["target"])
}
