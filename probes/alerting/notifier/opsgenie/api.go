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

package opsgenie

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cloudprober/cloudprober/probes/alerting/alertinfo"
)

type responder struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// alertMessage is the data structure for a Pagerduty event.
// The event can either be an alarm event, or a change event.
type alertMessage struct {
	Alias       string            `json:"alias,omitempty"`
	Entity      string            `json:"entity,omitempty"`
	Message     string            `json:"message"` // required
	Description string            `json:"description,omitempty"`
	Priority    priority          `json:"priority"` // required
	Details     map[string]string `json:"details,omitempty"`
	Responders  []responder       `json:"responders,omitempty"`
}

// priority is the severity of the event, documented as:
// The perceived severity of the status the event is describing with respect
// to the affected system. This can be critical, error, warning or info.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event#parameters
type priority string

const (
	P1, P2, P3, P4 priority = "P1", "P2", "P3", "P4"
)

// EventV2Response is the data structure that is returned from PagerDuty when
// sending a EventV2 request.
type EventV2Response struct {
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
	DedupKey string `json:"dedup_key,omitempty"`
}

// sendEventV2 sends an event to PagerDuty using the V2 API.
func (c *Client) sendAlert(msg *alertMessage) (*EventV2Response, error) {
	req, err := c.alertRequest(msg)
	if err != nil {
		return nil, err
	}

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
		return nil, fmt.Errorf("opsGenie API returned status code %d, response: %s", resp.StatusCode, string(b))
	}

	return nil, nil
}

// sendEventV2 sends an event to PagerDuty using the V2 API.
func (c *Client) alertRequest(msg *alertMessage) (*http.Request, error) {
	jsonBody, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "GenieKey "+c.ogKey)

	return req, nil
}

func priorityFromAlertFields(alertFields map[string]string) priority {
	p, ok := map[string]priority{
		"CRITICAL": P1,
		"ERROR":    P2,
		"WARNING":  P3,
		"INFO":     P4,
	}[alertFields["severity"]]
	if !ok {
		return P2
	}
	return p
}

// createAlertMessage creates a new PagerDuty event, from the alertFields
// that are passed in from the alerting package.
func (c *Client) createAlertMessage(alertInfo *alertinfo.AlertInfo, alertFields map[string]string) *alertMessage {
	msg := &alertMessage{
		Alias:       dedupeKey(alertInfo),
		Entity:      "Cloudprober",
		Message:     alertFields["summary"],
		Description: alertFields["details"],
		Priority:    priorityFromAlertFields(alertFields),
		Details:     make(map[string]string),
	}

	for k, v := range alertFields {
		if k != "summary" && k != "details" && k != "severity" {
			msg.Details[k] = v
		}
	}

	for _, res := range c.cfg.GetResponders() {
		msg.Responders = append(msg.Responders, responder{
			Type: strings.ToLower(res.GetType().String()),
			Name: res.GetName(),
			ID:   res.GetId(),
		})
	}

	return msg
}

// dedupeKey returns a key that can be used to dedupe PagerDuty events.
// note: submitting subsequent events with the same dedup_key will result in
// those events being applied to an open alert matching that dedup_key.
// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-alert-event#alert-de-duplication
func dedupeKey(alertInfo *alertinfo.AlertInfo) string {
	return alertInfo.ConditionID
}
