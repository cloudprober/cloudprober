// Copyright 2025 The Cloudprober Authors.
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

package singlerun

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
)

func testPrrs(em *metrics.EventMetrics) map[string][]*ProbeRunResult {
	return map[string][]*ProbeRunResult{
		"ping": {
			{
				Target:  endpoint.Endpoint{Name: "1.2.3.4"},
				Metrics: []*metrics.EventMetrics{em},
				Success: true,
				Latency: 123 * time.Millisecond,
			},
		},
		"http": {
			{
				Target:  endpoint.Endpoint{Name: "1.2.3.4"},
				Metrics: nil,
				Success: false,
				Latency: 123 * time.Millisecond,
				Error:   errors.New("connection timeout"),
			},
		},
	}
}

var expectedTextFormatOutput = `Probe: http
  Target: 1.2.3.4
    Status: failed
    Error: connection timeout


Probe: ping
  Target: 1.2.3.4
    Status: ok (latency: 123ms)
    Metrics: 
      %s

`

var expectedJsonFormatOutput = `{
  "http": {
    "1.2.3.4": {
      "target": "1.2.3.4",
      "success": false,
      "latency": "123ms",
      "error": "connection timeout"
    }
  },
  "ping": {
    "1.2.3.4": {
      "target": "1.2.3.4",
      "success": true,
      "latency": "123ms",
      "metrics": [
        %s
      ]
    }
  }
}
`

func TestTextFormatProbeRunResults(t *testing.T) {
	em := metrics.NewEventMetrics(time.Now()).AddMetric("rtt", metrics.NewFloat(123))
	assert.Equal(t, fmt.Sprintf(expectedTextFormatOutput, em.String()), textFormatProbeRunResults(testPrrs(em), "  "))
}

func TestJsonFormatProbeRunResults(t *testing.T) {
	em := metrics.NewEventMetrics(time.Now()).AddMetric("rtt", metrics.NewFloat(123))
	assert.Equal(t, fmt.Sprintf(expectedJsonFormatOutput, em.String()), jsonFormatProbeRunResults(testPrrs(em), "  "))
}
