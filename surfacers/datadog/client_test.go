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
	"encoding/json"
	"io"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cAPIKey, cAppKey := "c-apiKey", "c-appKey"
	eAPIKey, eAppKey := "e-apiKey", "e-appKey"

	tests := []struct {
		desc       string
		apiKey     string
		appKey     string
		server     string
		env        map[string]string
		wantClient *ddClient
	}{
		{
			desc:   "keys-from-config",
			apiKey: cAPIKey,
			appKey: cAppKey,
			server: "",
			wantClient: &ddClient{
				apiKey: cAPIKey,
				appKey: cAppKey,
				server: defaultServer,
			},
		},
		{
			desc: "keys-from-env",
			env: map[string]string{
				"DD_API_KEY": eAPIKey,
				"DD_APP_KEY": eAppKey,
			},
			server: "test-server",
			wantClient: &ddClient{
				apiKey: eAPIKey,
				appKey: eAppKey,
				server: "test-server",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			for k, v := range test.env {
				os.Setenv(k, v)
			}

			c := newClient(test.server, test.apiKey, test.appKey)
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

	testSeries := []ddSeries{
		{
			Metric: "cloudprober.success",
			Points: [][]float64{{float64(ts), 99}},
			Tags:   &tags,
			Type:   &metricType,
		},
		{
			Metric: "cloudprober.total",
			Points: [][]float64{{float64(ts), 100}},
			Tags:   &tags,
			Type:   &metricType,
		},
	}

	testClient := newClient("", "test-api-key", "test-app-key")
	req, err := testClient.newRequest(testSeries)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check URL
	wantURL := "https://api.datadoghq.com/api/v1/series"
	if req.URL.String() != wantURL {
		t.Errorf("Got URL: %s, wanted: %s", req.URL.String(), wantURL)
	}

	// Check request headers
	for k, v := range map[string]string{
		"DD-API-KEY": "test-api-key",
		"DD-APP-KEY": "test-app-key",
	} {
		if req.Header.Get(k) != v {
			t.Errorf("%s header: %s, wanted: %s", k, req.Header.Get(k), v)
		}
	}

	// Check request body
	b, err := io.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Error reading request body: %v", err)
	}
	data := map[string][]ddSeries{}

	// Check if the request body is compressed
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Errorf("Error creating gzip reader: %v", err)
	}
	defer r.Close()

	// Check if the request body is valid JSON
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		t.Errorf("Error decoding request body: %v", err)
	}

	// Check if the request body is valid
	if !reflect.DeepEqual(data["series"], testSeries) {
		t.Errorf("Got request body: %v, wanted: %v", data["series"], testSeries)
	}
}

func TestCompressPayload(t *testing.T) {
	tests := map[string]struct {
		series []ddSeries
		want   string
	}{
		"empty": {
			series: []ddSeries{},
			want:   "{\"series\":[]}",
		},
		"single": {
			series: []ddSeries{
				{
					Metric: "cloudprober.success",
					Points: [][]float64{{float64(1234567890), 99}},
					Tags:   &[]string{"probe:cloudprober_http"},
					Type:   &[]string{"count"}[0],
				},
			},
			want: "{\"series\":[{\"metric\":\"cloudprober.success\",\"points\":[[1234567890,99]],\"tags\":[\"probe:cloudprober_http\"],\"type\":\"count\"}]}",
		},
		"multiple": {
			series: []ddSeries{
				{
					Metric: "cloudprober.success",
					Points: [][]float64{{float64(1234567890), 99}},
					Tags:   &[]string{"probe:cloudprober_http"},
					Type:   &[]string{"count"}[0],
				},
				{
					Metric: "cloudprober.total",
					Points: [][]float64{{float64(1234567890), 100}},
					Tags:   &[]string{"probe:cloudprober_http"},
					Type:   &[]string{"count"}[0],
				},
			},
			want: "{\"series\":[{\"metric\":\"cloudprober.success\",\"points\":[[1234567890,99]],\"tags\":[\"probe:cloudprober_http\"],\"type\":\"count\"},{\"metric\":\"cloudprober.total\",\"points\":[[1234567890,100]],\"tags\":[\"probe:cloudprober_http\"],\"type\":\"count\"}]}",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := compressPayload(test.series)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			gotDecompressed, err := decompressPayload(got)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if gotDecompressed != test.want {
				t.Errorf("Got: %s, wanted: %s", gotDecompressed, test.want)
			}
		})
	}
}

func decompressPayload(payload *bytes.Buffer) (string, error) {
	r, err := gzip.NewReader(payload)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return "", err
	}
	return buf.String(), nil
}
