package notifier

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// PagerDutyEventV2Request is the data structure for a PagerDuty event, using the V2 API.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event
type PagerDutyEventV2Request struct {
	RoutingKey  string                   `json:"routing_key"` // required
	DedupKey    string                   `json:"dedup_key,omitempty"`
	EventAction PagerDutyEventV2Action   `json:"event_action"` // required
	Client      string                   `json:"client,omitempty"`
	ClientURL   string                   `json:"client_url,omitempty"`
	Payload     PagerDutyEventV2Payload  `json:"payload"` // required
	Images      []PagerDutyEventV2Images `json:"images,omitempty"`
	Links       []PagerDutyEventV2Links  `json:"links,omitempty"`
}

// PagerDutyEventV2Payload is the data structure for the payload of a PagerDuty event, using the V2 API.
type PagerDutyEventV2Payload struct {
	Summary       string            `json:"summary"`  // required
	Source        string            `json:"source"`   // required
	Severity      string            `json:"severity"` // required
	Timestamp     string            `json:"timestamp,omitempty"`
	Component     string            `json:"component,omitempty"`
	Group         string            `json:"group,omitempty"`
	Class         string            `json:"class,omitempty"`
	CustomDetails map[string]string `json:"custom_details,omitempty"`
}

// PagerDutyEventV2Images is the data structure for an image in a PagerDuty event, using the V2 API.
type PagerDutyEventV2Images struct {
	Src  string `json:"src,omitempty"`
	Alt  string `json:"alt,omitempty"`
	Href string `json:"href,omitempty"`
}

// PagerDutyEventV2Links is the data structure for a link in a PagerDuty event, using the V2 API.
type PagerDutyEventV2Links struct {
	Href string `json:"href,omitempty"`
	Text string `json:"text,omitempty"`
}

// EventAction is the action to be performed on the event
type PagerDutyEventV2Action string

const (
	Trigger     PagerDutyEventV2Action = "trigger"
	Acknowledge                        = "acknowledge"
	Resolve                            = "resolve"
)

// PagerDutyEventV2Response is the data structure for a PagerDuty event response, using the V2 API.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event
type PagerDutyEventV2Response struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	IncidentKey string `json:"incident_key"`
}

// NewRequest creates a new HTTP request for the PagerDuty event.
func (e *PagerDutyEventV2Request) newHTTPRequest(hostname string) (*http.Request, error) {
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
