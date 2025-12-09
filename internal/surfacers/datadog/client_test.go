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
	"compress/gzip"
	"encoding/json"
	"io"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cAPIKey, cAppKey := "c-apiKey", "c-appKey"
	eAPIKey, eAppKey := "e-apiKey", "e-appKey"

	tests := []struct {
		desc               string
		apiKey             string
		appKey             string
		server             string
		disableCompression bool
		env                map[string]string
		wantClient         *ddClient
	}{
		{
			desc:               "keys-from-config",
			apiKey:             cAPIKey,
			appKey:             cAppKey,
			server:             "",
			disableCompression: false,
			wantClient: &ddClient{
				apiKey:         cAPIKey,
				appKey:         cAppKey,
				server:         defaultServer,
				useCompression: true,
			},
		},
		{
			desc: "keys-from-env",
			env: map[string]string{
				"DD_API_KEY": eAPIKey,
				"DD_APP_KEY": eAppKey,
			},
			server:             "test-server",
			disableCompression: false,
			wantClient: &ddClient{
				apiKey:         eAPIKey,
				appKey:         eAppKey,
				server:         "test-server",
				useCompression: true,
			},
		},
		{
			desc: "compression-disabled",
			env: map[string]string{
				"DD_API_KEY": eAPIKey,
				"DD_APP_KEY": eAppKey,
			},
			server:             "test-server",
			disableCompression: true,
			wantClient: &ddClient{
				apiKey:         eAPIKey,
				appKey:         eAppKey,
				server:         "test-server",
				useCompression: false,
			},
		},
		{
			desc: "compression-enabled",
			env: map[string]string{
				"DD_API_KEY": eAPIKey,
				"DD_APP_KEY": eAppKey,
			},
			server:             "test-server",
			disableCompression: false,
			wantClient: &ddClient{
				apiKey:         eAPIKey,
				appKey:         eAppKey,
				server:         "test-server",
				useCompression: true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			for k, v := range test.env {
				os.Setenv(k, v)
			}

			c := newClient(test.server, test.apiKey, test.appKey, test.disableCompression)
			if !reflect.DeepEqual(c, test.wantClient) {
				t.Errorf("got client: %v, want client: %v", c, test.wantClient)
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	ts := time.Now().Unix()
	tags := []string{"probe:cloudprober_http"}
	metricType := "count"

	generateMetricsFunc := func(n int) []ddSeries {
		var result []ddSeries

		for i := 0; i < n; i++ {
			var metricName string

			switch i % 5 {
			case 0:
				metricName = "cloudprober.success"
			case 1:
				metricName = "cloudprober.total"
			case 2:
				metricName = "cloudprober.latency"
			case 3:
				metricName = "cloudprober.packet_loss"
			case 4:
				metricName = "cloudprober.error"
			}

			point := rand.Float64()*100 + 1

			result = append(result, ddSeries{
				Metric: metricName,
				Points: [][]float64{{float64(ts), point}},
				Tags:   &tags,
				Type:   &metricType,
			})
		}

		return result
	}

	tests := map[string]struct {
		ddSeries              []ddSeries
		disableCompression    bool
		wantCompressedPayload bool
	}{
		"disable-compression-with-small-payload": {
			ddSeries:              generateMetricsFunc(1),
			disableCompression:    true,
			wantCompressedPayload: false,
		},
		"disable-compression-with-large-payload": {
			ddSeries:              generateMetricsFunc(1000),
			disableCompression:    true,
			wantCompressedPayload: false,
		},
		"enable-compression-with-small-payload": {
			ddSeries:              generateMetricsFunc(1),
			disableCompression:    false,
			wantCompressedPayload: true,
		},
		"enable-compression-large-payload": {
			ddSeries:              generateMetricsFunc(1000),
			disableCompression:    false,
			wantCompressedPayload: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testClient := newClient("", "test-api-key", "test-app-key", test.disableCompression)
			req, err := testClient.newRequest(test.ddSeries)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check URL
			wantURL := "https://api.datadoghq.com/api/v1/series"
			if req.URL.String() != wantURL {
				t.Fatalf("Got URL: %s, wanted: %s", req.URL.String(), wantURL)
			}

			baseHeaders := map[string]string{
				"Content-Type": "application/json; charset=utf-8",
				"DD-API-KEY":   "test-api-key",
				"DD-APP-KEY":   "test-app-key",
			}

			// check that we instruct upstream servers that we have transferred this file
			// with gzip compression
			if test.wantCompressedPayload {
				baseHeaders["Content-Encoding"] = "gzip"
			}

			// Check request headers
			for k, v := range baseHeaders {
				if req.Header.Get(k) != v {
					t.Fatalf("%s header: %s, wanted: %s", k, req.Header.Get(k), v)
				}
			}

			// Check request body
			var body []byte
			if test.wantCompressedPayload {
				gz, err := gzip.NewReader(req.Body)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				body, err = io.ReadAll(gz)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			} else {
				body, err = io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			data := map[string][]ddSeries{}
			if err := json.Unmarshal(body, &data); err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(data["series"], test.ddSeries) {
				t.Fatalf("Got series: %v, wanted: %v", data, test.ddSeries)
			}
		})
	}
}
