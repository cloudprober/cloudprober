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

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes/alerting/alertinfo"
	configpb "github.com/cloudprober/cloudprober/probes/alerting/proto"
)

type Client struct {
	cfg        *configpb.OpsGenie
	httpClient *http.Client
	logger     *logger.Logger
	apiURL     string
	ogKey      string
}

const (
	// DefaultOpsgenieAPIURL is the base URL for the Opsgenie API.
	DefaultOpsgenieAPIURL = "https://api.opsgenie.com/v2/alerts"

	// DEFAULT_opsgenie_ROUTING_KEY_ENV_VAR is the default environment variable
	// to use for the Opsgenie routing key.
	DefaultGenieKeyEnvVar = "OPSGENIE_KEY"
)

func New(cfg *configpb.OpsGenie, l *logger.Logger) (*Client, error) {
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
		logger:     l,
		apiURL:     apiURL,
		ogKey:      ogKey,
	}, nil
}

// lookupOpsgenieKey looks up the routing key to use for the Opsgenie client,
// in order of precendence:
// 1. Routing key supplied by the user in the config
// 2. Routing key environment variable
func lookupOpsgenieKey(cfg *configpb.OpsGenie) (string, error) {
	// check if the user supplied a routing key
	if cfg.GetGenieKey() != "" {
		return cfg.GetGenieKey(), nil
	}

	envVar := cfg.GetGenieKeyEnvVar()
	if envVar == "" {
		envVar = DefaultGenieKeyEnvVar
	}
	// check if the environment variable is set for the routing key
	if routingKey, exists := os.LookupEnv(envVar); exists {
		return routingKey, nil
	}

	return "", errors.New("no genie key found")
}

func (c *Client) Notify(ctx context.Context, alertInfo *alertinfo.AlertInfo, alertFields map[string]string) error {
	// Create the event
	event := c.createAlertMessage(alertInfo, alertFields)

	// Send the event
	_, err := c.sendAlert(event)
	if err != nil {
		return err
	}

	// c.logger.Debugf("Opsgenie: trigger event sent successfully. Dedupe key: %s, message: %s", response.DedupKey, response.Message)

	return nil
}
