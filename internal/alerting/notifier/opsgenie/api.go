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

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
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

// sendRequest sends an event to PagerDuty using the V2 API.
func (c *Client) sendRequest(req *http.Request, msg string) error {
	c.l.Infof("Sending request to Opsgenie (URL: %s): %s", req.URL.String(), msg)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("opsgenie - error reading API response body: %v", err)
	}

	// check status code, return error if not 202
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("opsGenie API returned status code %d, response: %s", resp.StatusCode, string(b))
	}

	return nil
}

// httpRequest prepares an HTTP request to create an alert.
func (c *Client) httpRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "GenieKey "+c.ogKey)

	return req, nil
}

// alertRequest prepares an HTTP request to create an alert.
func (c *Client) alertRequest(msg *alertMessage) (*http.Request, error) {
	jsonBody, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %v", err)
	}
	return c.httpRequest(http.MethodPost, c.apiURL, bytes.NewBuffer(jsonBody))
}

// closeRequest prepares an HTTP request to close an alert.
func (c *Client) closeRequest(alias string) (*http.Request, error) {
	url := fmt.Sprintf("%s/%s/close?identifierType=alias", c.apiURL, alias)
	return c.httpRequest(http.MethodPost, url, bytes.NewBufferString("{}"))
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

// dedupeKey returns a deduplication key.
func dedupeKey(alertInfo *alertinfo.AlertInfo) string {
	return alertInfo.DeduplicationID
}
