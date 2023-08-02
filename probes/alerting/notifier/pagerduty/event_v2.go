package pagerduty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
	DedupKey string `json:"dedup_key,omitempty"`
}

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

// createEventV2Request creates a new PagerDuty event, using the V2 API.
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
