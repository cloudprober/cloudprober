// Copyright 2021 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datadog

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const defaultServer = "api.datadoghq.com"

type ddClient struct {
	apiKey         string
	appKey         string
	server         string
	c              http.Client
	useCompression bool
}

// ddSeries A metric to submit to Datadog. See:
// https://docs.datadoghq.com/developers/metrics/#custom-metrics-properties
type ddSeries struct {
	// The name of the host that produced the metric.
	Host *string `json:"host,omitempty"`
	// The name of the timeseries.
	Metric string `json:"metric"`
	// Points relating to a metric. All points must be tuples with timestamp and
	// a scalar value (cannot be a string). Timestamps should be in POSIX time in
	// seconds, and cannot be more than ten minutes in the future or more than
	// one hour in the past.
	Points [][]float64 `json:"points"`
	// A list of tags associated with the metric.
	Tags *[]string `json:"tags,omitempty"`
	// The type of the metric either `count`, `gauge`, or `rate`.
	Type *string `json:"type,omitempty"`
}

func newClient(server, apiKey, appKey string, disableCompression bool) *ddClient {
	// to avoid the double negative boolean evaluation in logic branches, and improve
	// readability, flip the value of the configuration option from disabling compression to
	// using compression.
	useCompression := !disableCompression

	c := &ddClient{
		apiKey:         apiKey,
		appKey:         appKey,
		server:         server,
		c:              http.Client{},
		useCompression: useCompression,
	}
	if c.apiKey == "" {
		c.apiKey = os.Getenv("DD_API_KEY")
	}

	if c.appKey == "" {
		c.appKey = os.Getenv("DD_APP_KEY")
	}

	if c.server == "" {
		c.server = defaultServer
	}

	return c
}

func (c *ddClient) newRequest(series []ddSeries) (*http.Request, error) {
	url := fmt.Sprintf("https://%s/api/v1/series", c.server)

	// JSON encoding of the datadog series.
	// {
	//   "series": [{..},{..}]
	// }
	json_body, err := json.Marshal(map[string][]ddSeries{"series": series})
	if err != nil {
		return nil, err
	}

	var payload *bytes.Buffer

	if c.useCompression {
		payload, err = compressPayload(json_body)
		if err != nil {
			return nil, err
		}
	} else {
		payload = bytes.NewBuffer(json_body)
	}

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("DD-API-KEY", c.apiKey)
	req.Header.Set("DD-APP-KEY", c.appKey)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	if c.useCompression {
		req.Header.Set("Content-Encoding", "gzip")
	}

	return req, nil
}

func (c *ddClient) submitMetrics(ctx context.Context, series []ddSeries) error {
	req, err := c.newRequest(series)
	if err != nil {
		return nil
	}

	resp, err := c.c.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error, HTTP status: %d, full response: %s", resp.StatusCode, string(b))
	}

	return nil
}

func compressPayload(b []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	_, err = gz.Write(b)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}
