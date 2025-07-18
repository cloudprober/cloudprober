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
	"net/http"
	"testing"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestParserProcessHeaderMetrics(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{
			"X-Server-Latency": []string{"30585.915"},
			"Content-Type":     []string{"application/json"},
			"Date":             []string{"Thu, 01 Sep 2022 00:00:00 GMT"},
			"Content-Length":   []string{"101"},
		},
	}
	opts := &configpb.OutputMetricsOptions{
		HeaderMetric: []*configpb.HeaderMetric{
			{
				HeaderName: proto.String("Content-Type"),
			},
			{
				MetricName: proto.String("server_time"),
				HeaderName: proto.String("Date"),
				Type:       configpb.HeaderMetric_HTTP_TIME.Enum(),
			},
			{
				HeaderName: proto.String("Content-Length"),
				Type:       configpb.HeaderMetric_INT.Enum(),
			},
			{
				// Not found
				HeaderName: proto.String("X-Job-Latency"),
				Type:       configpb.HeaderMetric_FLOAT.Enum(),
			},
			{
				HeaderName: proto.String("X-Server-Latency"),
				MetricName: proto.String("server_latency"),
				Type:       configpb.HeaderMetric_FLOAT.Enum(),
			},
		},
	}

	tests := []struct {
		name       string
		cfgs       any
		resp       any
		wantNilEM  bool
		wantKeys   []string
		wantValues []metrics.Value
	}{
		{
			name:       "main",
			resp:       resp,
			wantKeys:   []string{"content_type", "server_time", "content_length", "server_latency"},
			wantValues: []metrics.Value{metrics.NewString("application/json"), metrics.NewInt(1661990400), metrics.NewInt(101), metrics.NewFloat(30585.915)},
		},
		{
			name:      "empty",
			resp:      &http.Response{},
			wantNilEM: true,
		},
		{
			name:      "bad-response",
			resp:      "bad-response",
			wantNilEM: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(opts, nil)
			if err != nil {
				t.Fatal(err)
			}
			em := p.processHeaderMetrics(tt.resp)
			if em == nil {
				if !tt.wantNilEM {
					t.Fatal("processHeaderMetrics: got nil")
				}
				return
			}
			if tt.wantNilEM {
				t.Error("processHeaderMetrics: got non-nil, want nil")
			}
			assert.Equal(t, tt.wantKeys, em.MetricsKeys(), "metrics keys mismatch")
			for i, name := range tt.wantKeys {
				assert.Equalf(t, tt.wantValues[i], em.Metric(name), "value mismatch for metric%s", name)
			}
		})
	}
}
