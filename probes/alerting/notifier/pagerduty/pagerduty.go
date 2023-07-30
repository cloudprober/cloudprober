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
	"os"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
)

type Client struct {
	httpClient *http.Client
	logger     *logger.Logger
	hostname   string
	apiToken   string
	routingKey string
}

const (
	// PAGERDUTY_API_URL is the base URL for the PagerDuty API.
	PAGERDUTY_API_URL = "https://api.pagerduty.com"

	// CLIENT_URL is the URL for the client that is triggering the event.
	// This is used in the client_url field of the event payload.
	PAGERDUTY_CLIENT_URL = "https://cloudprober.org/"

	// PAGERDUTY_API_TOKEN_ENV_VAR is the environment variable that
	// contains the PagerDuty API token.
	PAGERDUTY_API_TOKEN_ENV_VAR = "PAGERDUTY_API_TOKEN"

	// PAGERDUTY_ROUTING_KEY_ENV_VAR is the environment variable that
	// contains the PagerDuty routing key.
	PAGERDUTY_ROUTING_KEY_ENV_VAR = "PAGERDUTY_ROUTING_KEY"
)

func New(notifycfg *configpb.NotifyConfig, l *logger.Logger) (*Client, error) {
	hostname := PAGERDUTY_API_URL
	if notifycfg.GetPagerdutyUrl() != "" {
		hostname = notifycfg.GetPagerdutyUrl()
	}
	l.Debugf("Using PagerDuty API URL: %s", hostname)

	apiToken, err := apiToken(notifycfg)
	if err != nil {
		return nil, err
	}
	l.Debugf("Using PagerDuty API token: %s", apiToken)

	routingKey, err := routingKey(notifycfg)
	if err != nil {
		return nil, err
	}
	l.Debugf("Using PagerDuty routing key: %s", routingKey)

	return &Client{
		httpClient: &http.Client{},
		logger:     l,
		hostname:   hostname,
		apiToken:   apiToken,
		routingKey: routingKey,
	}, nil
}

func apiToken(notifycfg *configpb.NotifyConfig) (string, error) {
	if token, ok := os.LookupEnv(PAGERDUTY_API_TOKEN_ENV_VAR); ok {
		return token, nil
	}

	if notifycfg.GetPagerdutyApiToken() != "" {
		return notifycfg.GetPagerdutyApiToken(), nil
	}

	return "", fmt.Errorf("PagerDuty API token not found in environment variable %s or in the config", PAGERDUTY_API_TOKEN_ENV_VAR)
}

func routingKey(notifycfg *configpb.NotifyConfig) (string, error) {
	if token, ok := os.LookupEnv(PAGERDUTY_ROUTING_KEY_ENV_VAR); ok {
		return token, nil
	}

	if notifycfg.GetPagerdutyRoutingKey() != "" {
		return notifycfg.GetPagerdutyRoutingKey(), nil
	}

	return "", fmt.Errorf("PagerDuty routing key not found in environment variable %s or in the config", PAGERDUTY_ROUTING_KEY_ENV_VAR)
}

func (c *Client) Notify(ctx context.Context, alertFields map[string]string) error {
	// Create the event
	event := c.createEventV2Request(alertFields)

	// Send the event
	_, err := c.sendEventV2(event)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Refactor this as not all fields are correct.
// createEventV2Request creates a new PagerDuty event, using the V2 API.
func (c *Client) createEventV2Request(alertFields map[string]string) *EventV2Request {
	return &EventV2Request{
		RoutingKey:  c.routingKey,
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

func (c *Client) sendEventV2(event *EventV2Request) (*EventV2Response, error) {
	req, err := event.httpRequest(c.hostname)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
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
func (c *Client) do(req *http.Request) (*http.Response, error) {
	// The PagerDuty API requires authentication, using the API token.
	// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTUx-authentication
	req.Header.Set("Authorization", "Token token="+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
