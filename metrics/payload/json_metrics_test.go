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
	testJSON := `{
	  "tokenId": 143,
	  "tokenType": "Bearer",
	  "expiresInSec": 1736.56,
	  "valid": true
	}`
	tests := []struct {
		name    string
		config  string
		cfg     []*configpb.JSONMetric
		wantJM  []*jsonMetricGroup
		wantEMs []string
		wantErr bool
	}{
		{
			name: "valid metrics",
			config: `
				json_metric: [{
					metric_name: "foo",
					value_jq_filter: ".tokenId",
					labels_jq_filter: "{\"token_type\":.tokenType}",
				},
				{
					name_jq_filter: ".tokenType+\"-token-valid\" | ascii_downcase",
					value_jq_filter: ".valid",
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
							nameJQ:  testMustJQParse(".tokenType+\"-token-valid\" | ascii_downcase"),
							valueJQ: testMustJQParse(".valid"),
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

			ems := p.processJSONMetric([]byte(testJSON))
			assert.Equal(t, len(tt.wantEMs), len(ems), "number of event metrics")

			for i, em := range ems {
				got := strings.Join(strings.Split(em.String(), " ")[1:], " ")
				assert.Equal(t, tt.wantEMs[i], got, "metrics for %d", i)
			}
		})
	}
}
