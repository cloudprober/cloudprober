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

// The pagerduty package implements pagerduty notifications for Cloudprober
// alert events.

import (
	"context"
	"net/http"

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
)

func New(notifycfg *configpb.NotifyConfig, l *logger.Logger) (*Client, error) {
	hostname := PAGERDUTY_API_URL
	if notifycfg.GetPagerdutyApiUrl() != "" {
		hostname = notifycfg.GetPagerdutyApiUrl()
		l.Debugf("Using PagerDuty API URL: %s", hostname)
	}

	return &Client{
		httpClient: &http.Client{},
		logger:     l,
		hostname:   hostname,
		routingKey: notifycfg.PagerdutyRoutingKey,
	}, nil
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
