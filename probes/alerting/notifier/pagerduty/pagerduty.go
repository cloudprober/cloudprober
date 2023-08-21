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

// Package pagerduty implements pagerduty notifications for Cloudprober
// alert events.
package pagerduty

import (
	"context"
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
	routingKey string
}

const (
	// PAGERDUTY_API_URL is the base URL for the PagerDuty API.
	PAGERDUTY_API_URL = "https://events.pagerduty.com"

	// DEFAULT_PAGERDUTY_ROUTING_KEY_ENV_VAR is the default environment variable
	// to use for the PagerDuty routing key.
	DEFAULT_PAGERDUTY_ROUTING_KEY_ENV_VAR = "PAGERDUTY_ROUTING_KEY"
)

func New(pagerdutycfg *configpb.PagerDuty, l *logger.Logger) (*Client, error) {
	hostname := PAGERDUTY_API_URL
	if pagerdutycfg.GetApiUrl() != "" {
		hostname = pagerdutycfg.GetApiUrl()
		l.Debugf("Using PagerDuty API URL: %s", hostname)
	}

	routingKey, err := lookupRoutingKey(pagerdutycfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{},
		logger:     l,
		hostname:   hostname,
		routingKey: routingKey,
	}, nil
}

// lookupRoutingKey looks up the routing key to use for the PagerDuty client,
// in order of precendence:
// 1. Routing key environment variable
// 2. Routing key in the config
func lookupRoutingKey(pagerdutycfg *configpb.PagerDuty) (string, error) {
	// check if the environment variable is set for the routing key
	if routingKey, exists := os.LookupEnv(routingKeyEnvVar(pagerdutycfg)); exists {
		return routingKey, nil
	}

	// check if the user supplied a routing key
	if pagerdutycfg.GetRoutingKey() != "" {
		return pagerdutycfg.GetRoutingKey(), nil
	}

	return "", fmt.Errorf("No routing key found")
}

// routingKeyEnvVar returns the environment variable to use for the routing key,
// or the default if none is set.
func routingKeyEnvVar(pagerdutycfg *configpb.PagerDuty) string {
	if pagerdutycfg.GetRoutingKeyEnvVar() != "" {
		return pagerdutycfg.GetRoutingKeyEnvVar()
	}

	return DEFAULT_PAGERDUTY_ROUTING_KEY_ENV_VAR
}

func (c *Client) Notify(ctx context.Context, alertFields map[string]string) error {
	// Create the event
	event := c.createEventV2Request(alertFields)

	// Send the event
	response, err := c.sendEventV2(event)
	if err != nil {
		return err
	}

	c.logger.Debugf("PagerDuty event sent successfully. Dedupe key: %s, message: %s", response.DedupKey, response.Message)

	return nil
}
