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

// Package singlerun provides utilities to handle results of a single run.
package singlerun

import (
	"fmt"
	"strings"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

// Format specifies the format of the output.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// ProbeRunResult encapsulates the results of a single probe run for
// a specific target.
type ProbeRunResult struct {
	Target  endpoint.Endpoint
	Metrics []*metrics.EventMetrics
	Success bool // Did probe run succeed?
	Error   error
}

func textFormatProbeRunResults(probeResults map[string][]*ProbeRunResult, indent string) string {
	out := make([]string, 0, len(probeResults))
	for name, prrs := range probeResults {
		out = append(out, fmt.Sprintf("Probe: %s", name))
		for _, prr := range prrs {
			var metricLines []string
			for _, m := range prr.Metrics {
				metricLines = append(metricLines, m.String())
			}
			out = append(out, fmt.Sprintf("%sTarget: %s: \n%s%s", indent, prr.Target.Dst(), indent+indent, strings.Join(metricLines, "\n"+indent+indent)))
		}
		out = append(out, "\n")
	}
	return strings.Join(out, "\n")
}

func jsonFormatProbeRunResults(probeResults map[string][]*ProbeRunResult) string {
	return "json format not supported yet"
}

func FormatProbeRunResults(probeResults map[string][]*ProbeRunResult, format Format, indent string) string {
	switch format {
	case FormatText:
		return textFormatProbeRunResults(probeResults, indent)
	case FormatJSON:
		return jsonFormatProbeRunResults(probeResults)
	}
	return ""
}
