// Copyright 2023 The Cloudprober Authors.
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

// Package alertinfo implements AlertInfo struct for sharing alert data
// across modules.
package alertinfo

import (
	"net"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestAlertInfoFields(t *testing.T) {
	testTarget := endpoint.Endpoint{
		Name: "test-target",
		IP:   net.ParseIP("10.11.12.13"),
		Labels: map[string]string{
			"apptype":  "backend",
			"language": "go",
		},
	}

	tests := []struct {
		name            string
		ai              *AlertInfo
		templateDetails map[string]string
		want            map[string]string
	}{
		{
			name: "no_template_details",
			ai: &AlertInfo{
				Name:            "test-alert",
				ProbeName:       "test-probe",
				DeduplicationID: "122333444",
				Target:          testTarget,
				Failures:        8,
				Total:           12,
				FailingSince:    time.Time{}.Add(time.Second),
			},
			want: map[string]string{
				"alert":                 "test-alert",
				"probe":                 "test-probe",
				"target":                "test-target",
				"target_ip":             "10.11.12.13",
				"failures":              "8",
				"total":                 "12",
				"since":                 "0001-01-01T00:00:01Z",
				"target.label.apptype":  "backend",
				"target.label.language": "go",
			},
		},
		{
			name: "with_template_details",
			ai: &AlertInfo{
				Name:            "test-alert",
				ProbeName:       "test-probe",
				DeduplicationID: "122333444",
				Target:          testTarget,
				Failures:        8,
				Total:           12,
				FailingSince:    time.Time{}.Add(time.Second),
			},
			templateDetails: map[string]string{
				"summary":       "Cloudprober alert \"@alert@\" for \"@target@\"",
				"dashboard_url": "https://my-dashboard.com/probe=@probe@&target=@target@",
				"details":       "Dashboard: @dashboard_url@",
			},
			want: map[string]string{
				"alert":                 "test-alert",
				"probe":                 "test-probe",
				"target":                "test-target",
				"target_ip":             "10.11.12.13",
				"failures":              "8",
				"total":                 "12",
				"since":                 "0001-01-01T00:00:01Z",
				"target.label.apptype":  "backend",
				"target.label.language": "go",
				"summary":               "Cloudprober alert \"test-alert\" for \"test-target\"",
				"dashboard_url":         "https://my-dashboard.com/probe=test-probe&target=test-target",
				"details":               "Dashboard: https://my-dashboard.com/probe=test-probe&target=test-target",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.ai.Fields(tt.templateDetails), "Fields don't match")
		})
	}
}

func TestFieldsToString(t *testing.T) {
	fields := map[string]string{
		"alert": "test-alert",
		"probe": "test-probe",
	}

	tests := []struct {
		name     string
		skipKeys []string
		want     string
	}{
		{
			name: "skip_none",
			want: "alert: test-alert\nprobe: test-probe",
		},
		{
			name:     "skip_probe",
			skipKeys: []string{"probe"},
			want:     "alert: test-alert",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FieldsToString(fields, tt.skipKeys...), "Fields don't match")
		})
	}
}
