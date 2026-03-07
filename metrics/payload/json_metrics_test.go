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
	"testing"

	"github.com/cloudprober/cloudprober/metrics"
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
		wantJM  []*jsonMetric
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
				json_metric: [
				{
					jq_filter: "[{key: (.tokenType|ascii_downcase+\"-token-valid\"), value: .valid}, {key: \"foo\", value: (.tokenId + 1)}] | from_entries",
					labels_jq_filter: "{\"token_type\":.tokenType}",
				},
				{
					jq_filter: "{\"token-expires-in-msec\": (.expiresInSec * 1000)}",
				}]
			`,
			wantJM: []*jsonMetric{
				{
					metricsJQ: testMustJQParse("[{key: (.tokenType|ascii_downcase+\"-token-valid\"), value: .valid}, {key: \"foo\", value: (.tokenId + 1)}] | from_entries"),
					labelsJQ:  testMustJQParse("{\"token_type\":.tokenType}"),
				},
				{
					metricsJQ: testMustJQParse("{\"token-expires-in-msec\": (.expiresInSec * 1000)}"),
				},
			},
			wantEMs: []string{
				"labels=token_type=Bearer bearer-token-valid=1 foo=144.000",
				"labels= token-expires-in-msec=1736560.000",
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
					jq_filter: "del(.deployment)",
					labels_jq_filter: "{\"deployment\":.deployment}",
				}]
			`,
			wantJM: []*jsonMetric{
				{
					metricsJQ: testMustJQParse("del(.deployment)"),
					labelsJQ:  testMustJQParse("{\"deployment\":.deployment}"),
				},
			},
			wantEMs: []string{
				"labels=deployment=dep1 dequeue_deplay_p50=1733.000 dequeue_deplay_p95=16899.399",
			},
		},
		{
			name: "multiple labels sorting",
			input: `{
				"deployment": "dep1",
				"region": "us-central1",
				"zone": "us-central1-a",
				"app": "cloudprober",
				"reqs": 100
			}`,
			config: `
				json_metric: [{
					jq_filter: "{\"reqs\":.reqs}",
					labels_jq_filter: "{\"deployment\":.deployment, \"region\":.region, \"zone\":.zone, \"app\":.app}",
				}]
			`,
			wantJM: []*jsonMetric{
				{
					metricsJQ: testMustJQParse("{\"reqs\":.reqs}"),
					labelsJQ:  testMustJQParse("{\"deployment\":.deployment, \"region\":.region, \"zone\":.zone, \"app\":.app}"),
				},
			},
			wantEMs: []string{
				"labels=app=cloudprober,deployment=dep1,region=us-central1,zone=us-central1-a reqs=100.000",
			},
		},
		{
			name: "bad filter",
			config: `
			json_metric: [{
				jq_filter: "{{tokenId",
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

			assert.Equal(t, tt.wantJM, p.jsonMetrics, "json metric groups")

			ems := p.processJSONMetric([]byte(tt.input))

			assert.Equal(t, len(tt.wantEMs), len(ems), "number of event metrics")

			for i, em := range ems {
				assert.Equal(t, tt.wantEMs[i], em.String(metrics.StringerNoTimestamp()), "metrics for %d", i)
			}
		})
	}
}
