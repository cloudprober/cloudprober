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
	"errors"
	"net/http"
	"os"

	"github.com/cloudprober/cloudprober/internal/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/logger"
)

type Client struct {
	httpClient          *http.Client
	l                   *logger.Logger
	hostname            string
	routingKey          string
	disableSendResolved bool
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
		httpClient:          &http.Client{},
		l:                   l,
		hostname:            hostname,
		routingKey:          routingKey,
		disableSendResolved: pagerdutycfg.GetDisableSendResolved(),
	}, nil
}

// lookupRoutingKey looks up the routing key to use for the PagerDuty client,
// in order of precendence:
// 1. Routing key supplied by the user in the config
// 2. Routing key environment variable
func lookupRoutingKey(pagerdutycfg *configpb.PagerDuty) (string, error) {
	// check if the user supplied a routing key
	if pagerdutycfg.GetRoutingKey() != "" {
		return pagerdutycfg.GetRoutingKey(), nil
	}

	// check if the environment variable is set for the routing key
	if routingKey, exists := os.LookupEnv(routingKeyEnvVar(pagerdutycfg)); exists {
		return routingKey, nil
	}

	return "", errors.New("No routing key found")
}

// routingKeyEnvVar returns the environment variable to use for the routing key,
// or the default if none is set.
func routingKeyEnvVar(pagerdutycfg *configpb.PagerDuty) string {
	if pagerdutycfg.GetRoutingKeyEnvVar() != "" {
		return pagerdutycfg.GetRoutingKeyEnvVar()
	}

	return DEFAULT_PAGERDUTY_ROUTING_KEY_ENV_VAR
}

func (c *Client) Notify(ctx context.Context, alertInfo *alertinfo.AlertInfo, alertFields map[string]string) error {
	// Create the event
	event := c.createTriggerRequest(alertInfo, alertFields)

	// Send the event
	response, err := c.sendEventV2(event)
	if err != nil {
		return err
	}

	c.l.Debugf("PagerDuty: trigger event sent successfully. Dedupe key: %s, message: %s", response.DedupKey, response.Message)

	return nil
}

func (c *Client) NotifyResolve(ctx context.Context, alertInfo *alertinfo.AlertInfo, alertFields map[string]string) error {
	if c.disableSendResolved {
		return nil
	}

	// Create the event
	event := c.createResolveRequest(alertInfo, alertFields)

	c.l.Infof("PagerDuty: sending resolve event: dedupe key: %s", event.DedupKey)

	// Send the event
	_, err := c.sendEventV2(event)
	if err != nil {
		c.l.Errorf("PagerDuty: error sending resolve event: %v", err)
		return err
	}
	return nil
}
