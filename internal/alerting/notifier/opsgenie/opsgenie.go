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

// Package opsgenie implements Opsgenie notifications for Cloudprober
// alert events.
package opsgenie

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
	cfg        *configpb.Opsgenie
	httpClient *http.Client
	l          *logger.Logger
	apiURL     string
	ogKey      string
}

const (
	// DefaultOpsgenieAPIURL is the base URL for the Opsgenie API.
	DefaultOpsgenieAPIURL = "https://api.opsgenie.com/v2/alerts"

	// DEFAULT_opsgenie_ROUTING_KEY_ENV_VAR is the default environment variable
	// to use for the Opsgenie routing key.
	DefaultApiKeyEnvVar = "OPSGENIE_API_KEY"
)

func New(cfg *configpb.Opsgenie, l *logger.Logger) (*Client, error) {
	apiURL := DefaultOpsgenieAPIURL
	if cfg.GetApiUrl() != "" {
		apiURL = cfg.GetApiUrl()
		l.Debugf("Using Opsgenie API URL: %s", apiURL)
	}

	ogKey, err := lookupOpsgenieKey(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		cfg:        cfg,
		httpClient: http.DefaultClient,
		l:          l,
		apiURL:     apiURL,
		ogKey:      ogKey,
	}, nil
}

// lookupOpsgenieKey looks up the routing key to use for the Opsgenie client,
// in order of precendence:
// 1. Routing key supplied by the user in the config
// 2. Routing key environment variable
func lookupOpsgenieKey(cfg *configpb.Opsgenie) (string, error) {
	// check if the user supplied a routing key
	if cfg.GetApiKey() != "" {
		return cfg.GetApiKey(), nil
	}

	envVar := cfg.GetApiKeyEnvVar()
	if envVar == "" {
		envVar = DefaultApiKeyEnvVar
	}
	// check if the environment variable is set for the routing key
	if routingKey, exists := os.LookupEnv(envVar); exists {
		return routingKey, nil
	}

	return "", errors.New("no genie key found")
}

func (c *Client) Notify(ctx context.Context, alertInfo *alertinfo.AlertInfo, alertFields map[string]string) error {
	alertMsg := c.createAlertMessage(alertInfo, alertFields)

	req, err := c.alertRequest(alertMsg)
	if err != nil {
		return err
	}

	return c.sendRequest(req, alertMsg.Message)
}

func (c *Client) NotifyResolve(ctx context.Context, alertInfo *alertinfo.AlertInfo, alertFields map[string]string) error {
	if c.cfg.GetDisableSendResolved() {
		return nil
	}

	alertMsg := c.createAlertMessage(alertInfo, alertFields)

	req, err := c.closeRequest(alertMsg.Alias)
	if err != nil {
		return err
	}

	return c.sendRequest(req, alertMsg.Message)
}
