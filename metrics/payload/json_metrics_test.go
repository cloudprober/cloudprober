// Copyright 2024 The Cloudprober Authors.
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
	"strings"
	"testing"

	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/itchyny/gojq"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
)

func testMustJQParse(s string) *gojq.Query {
	q, err := gojq.Parse(s)
	if err != nil {
		panic(err)
	}
	return q
}

func TestJSONMetrics(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		input   string
		wantJM  []*jsonMetricGroup
		wantEMs []string
		wantErr bool
	}{
		{
			name: "valid metrics",
			input: `{
				"tokenId": 143,
				"tokenType": "Bearer",
				"expiresInSec": 1736.56,
				"valid": true
			}`,
			config: `
				json_metric: [{
					metric_name: "foo",
					value_jq_filter: ".tokenId",
					labels_jq_filter: "{\"token_type\":.tokenType}",
				},
				{
					metrics_jq_filter: "[{key: (.tokenType|ascii_downcase+\"-token-valid\"), value: .valid}] | from_entries",
					labels_jq_filter: "{\"token_type\":.tokenType}",
				},
				{
					metric_name: "token-expires-in-sec",
					value_jq_filter: ".expiresInSec",
				}]
			`,
			wantJM: []*jsonMetricGroup{
				{
					metrics: []*jsonMetric{
						{
							metricName: "token-expires-in-sec",
							valueJQ:    testMustJQParse(".expiresInSec"),
						},
					},
				},
				{
					metrics: []*jsonMetric{
						{
							metricName: "foo",
							valueJQ:    testMustJQParse(".tokenId"),
						},
						{
							metricsJQ: testMustJQParse("[{key: (.tokenType|ascii_downcase+\"-token-valid\"), value: .valid}] | from_entries"),
						},
					},
					labelsJQ: testMustJQParse("{\"token_type\":.tokenType}"),
				},
			},
			wantEMs: []string{
				"labels= token-expires-in-sec=1736.560",
				"labels=token_type=Bearer foo=143.000 bearer-token-valid=1",
			},
		},
		{
			name: "valid metrics - 2",
			input: `{
				"deployment": "dep1",
				"dequeue_deplay_p50": 1733,
				"dequeue_deplay_p95": 16899.399
			}`,
			config: `
				json_metric: [{
					metrics_jq_filter: "del(.deployment)",
					labels_jq_filter: "{\"deployment\":.deployment}",
				}]
			`,
			wantJM: []*jsonMetricGroup{
				{
					metrics: []*jsonMetric{
						{
							metricsJQ: testMustJQParse("del(.deployment)"),
						},
					},
					labelsJQ: testMustJQParse("{\"deployment\":.deployment}"),
				},
			},
			wantEMs: []string{
				"labels=deployment=dep1 dequeue_deplay_p50=1733.000 dequeue_deplay_p95=16899.399",
			},
		},
		{
			name: "bad config",
			config: `
			json_metric: [{
				value_jq_filter: ".tokenId",
			}]
			`,
			wantErr: true,
		},
		{
			name: "bad filter",
			config: `
			json_metric: [{
				metric_name: "foo",
				value_jq_filter: "{{tokenId",
			}]
			`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &configpb.OutputMetricsOptions{}
			if err := prototext.Unmarshal([]byte(tt.config), cfg); err != nil {
				t.Fatalf("Failed to unmarshal config: %v", err)
			}
			p, err := NewParser(cfg, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			assert.Equal(t, tt.wantJM, p.jmGroups, "json metric groups")

			ems := p.processJSONMetric([]byte(tt.input))
			assert.Equal(t, len(tt.wantEMs), len(ems), "number of event metrics")

			for i, em := range ems {
				got := strings.Join(strings.Split(em.String(), " ")[1:], " ")
				assert.Equal(t, tt.wantEMs[i], got, "metrics for %d", i)
			}
		})
	}
}
