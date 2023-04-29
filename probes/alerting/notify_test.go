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

package alerting

import (
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func TestAlertFields(t *testing.T) {
	testTarget := endpoint.Endpoint{
		Name: "test-target",
		Labels: map[string]string{
			"apptype":  "backend",
			"language": "go",
		},
	}

	tests := []struct {
		name string
		ai   *AlertInfo
		want map[string]string
	}{
		{
			name: "simple",
			ai: &AlertInfo{
				Name:             "test-alert",
				ProbeName:        "test-probe",
				Target:           testTarget,
				FailureRatio:     0.9,
				FailureThreshold: 0.5,
				FailingSince:     time.Time{}.Add(time.Second),
			},
			want: map[string]string{
				"alert":                 "test-alert",
				"probe":                 "test-probe",
				"target":                "test-target",
				"alerting_value":        "0.90",
				"alerting_threshold":    "0.50",
				"failing_since":         "0001-01-01T00:00:01Z",
				"target.label.apptype":  "backend",
				"target.label.language": "go",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, alertFields(tt.ai), "Fields don't match")
		})
	}
}
