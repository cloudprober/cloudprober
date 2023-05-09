// Copyright 2017-2024 The Cloudprober Authors.
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

package payload

import (
	"testing"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/itchyny/gojq"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func testMustJQParse(s string) *gojq.Query {
	q, err := gojq.Parse(s)
	if err != nil {
		panic(err)
	}
	return q
}

func TestJSONMetrics(t *testing.T) {
	testJSON := `{
	  "tokenId": 143,
	  "tokenType": "Bearer",
	  "expiresInSec": 1736.56,
	  "valid": true
	}`
	tests := []struct {
		name        string
		cfg         []*configpb.JSONMetric
		wantJM      []*jsonMetric
		wantMetrics []string
		wantValues  []metrics.Value
		wantErr     bool
	}{
		{
			name: "2 metrics",
			cfg: []*configpb.JSONMetric{
				{
					Name:          &configpb.JSONMetric_MetricName{MetricName: "foo"},
					ValueJqFilter: proto.String(".tokenId"),
				},
				{
					Name:          &configpb.JSONMetric_NameJqFilter{NameJqFilter: ".tokenType+\"-token-valid\" | ascii_downcase"},
					ValueJqFilter: proto.String(".valid"),
				},
				{
					Name:          &configpb.JSONMetric_MetricName{MetricName: "token-type"},
					ValueJqFilter: proto.String(".tokenType"),
				},
				{
					Name:          &configpb.JSONMetric_MetricName{MetricName: "token-expires-in-sec"},
					ValueJqFilter: proto.String(".expiresInSec"),
				},
			},
			wantJM: []*jsonMetric{
				{
					metricName: "foo",
					valueJQ:    testMustJQParse(".tokenId"),
				},
				{
					nameJQ:  testMustJQParse(".tokenType+\"-token-valid\" | ascii_downcase"),
					valueJQ: testMustJQParse(".valid"),
				},
				{
					metricName: "token-type",
					valueJQ:    testMustJQParse(".tokenType"),
				},
				{
					metricName: "token-expires-in-sec",
					valueJQ:    testMustJQParse(".expiresInSec"),
				},
			},
			wantMetrics: []string{"foo", "bearer-token-valid", "token-type", "token-expires-in-sec"},
			wantValues:  []metrics.Value{metrics.NewFloat(143), metrics.NewInt(1), metrics.NewString("Bearer"), metrics.NewFloat(1736.56)},
		},
		{
			name: "bad config",
			cfg: []*configpb.JSONMetric{
				{
					ValueJqFilter: proto.String(".tokenId"),
				},
			},
			wantErr: true,
		},
		{
			name: "bad filter",
			cfg: []*configpb.JSONMetric{
				{
					Name:          &configpb.JSONMetric_MetricName{MetricName: "foo"},
					ValueJqFilter: proto.String("{{tokenId"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(&configpb.OutputMetricsOptions{
				JsonMetric: tt.cfg,
			}, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			assert.Equal(t, tt.wantJM, p.jsonMetrics)

			em := p.processJSONMetric([]byte(testJSON))
			assert.Equal(t, tt.wantMetrics, em.MetricsKeys(), "metric keys")
			for i, v := range tt.wantValues {
				assert.Equal(t, v, em.Metric(tt.wantMetrics[i]), "metric values")
			}
		})
	}
}
