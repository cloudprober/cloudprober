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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	pb "github.com/cloudprober/cloudprober/prober/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	endpointpb "github.com/cloudprober/cloudprober/targets/endpoint/proto"
	"google.golang.org/protobuf/proto"
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
	Latency time.Duration
	Error   error
}

func statusString(success bool, latency time.Duration) string {
	if !success {
		return "failed"
	}
	return fmt.Sprintf("ok (latency: %s)", latency)
}

func textFormatProbeRunResults(probeResults map[string][]*ProbeRunResult, indent string) string {
	out := make([]string, 0, len(probeResults))

	// Make sure order is deterministic.
	probeNameKeys := make([]string, 0, len(probeResults))
	for name := range probeResults {
		probeNameKeys = append(probeNameKeys, name)
	}
	sort.Strings(probeNameKeys)

	for _, name := range probeNameKeys {
		out = append(out, fmt.Sprintf("Probe: %s", name))
		for _, prr := range probeResults[name] {
			out = append(out, fmt.Sprintf("%sTarget: %s", indent, prr.Target.Dst()))
			out = append(out, fmt.Sprintf("%sStatus: %s", indent+indent, statusString(prr.Success, prr.Latency)))
			if !prr.Success {
				out = append(out, fmt.Sprintf("%sError: %v", indent+indent, prr.Error))
			}

			var metricLines []string
			for _, m := range prr.Metrics {
				metricLines = append(metricLines, m.String())
			}
			if len(metricLines) > 0 {
				out = append(out, fmt.Sprintf("%sMetrics: \n%s%s", indent+indent, indent+indent+indent, strings.Join(metricLines, "\n"+indent+indent+indent)))
			}
			out = append(out, "\n")
		}
	}
	return strings.Join(out, "\n")
}

func jsonFormatProbeRunResults(probeResults map[string][]*ProbeRunResult, indent string) string {
	type probeTargetResult struct {
		Success bool     `json:"success"`
		Latency string   `json:"latency,omitempty"`
		Error   string   `json:"error,omitempty"`
		Metrics []string `json:"metrics,omitempty"`
	}

	results := make(map[string]map[string]probeTargetResult)

	for name := range probeResults {
		results[name] = make(map[string]probeTargetResult)

		for _, prr := range probeResults[name] {
			tr := probeTargetResult{
				Success: prr.Success,
			}

			if prr.Latency > 0 {
				tr.Latency = prr.Latency.String()
			}

			if prr.Error != nil {
				tr.Error = prr.Error.Error()
			}

			if len(prr.Metrics) > 0 {
				tr.Metrics = make([]string, 0, len(prr.Metrics))
				for _, m := range prr.Metrics {
					tr.Metrics = append(tr.Metrics, m.String())
				}
			}

			results[name][prr.Target.Dst()] = tr
		}
	}

	// TODO(manugarg): Migrate to encoding/json/v2 once it's available.
	// v2's marshaler provides an option to make output deterministic.
	jsonData, err := json.MarshalIndent(results, "", indent)
	if err != nil {
		return fmt.Sprintf("{\"error\": \"%v\"}", err)
	}

	return string(jsonData)
}

// FormatProbeRunResults formats probe run results into text or JSON.
// The indent string is used for pretty-printing.
func FormatProbeRunResults(probeResults map[string][]*ProbeRunResult, format Format, indent string) string {
	switch format {
	case FormatText:
		return textFormatProbeRunResults(probeResults, indent)
	case FormatJSON:
		return jsonFormatProbeRunResults(probeResults, indent)
	}
	return ""
}

// ProbeRunResultsToProto converts probe run results to their protobuf form.
func ProbeRunResultsToProto(results map[string][]*ProbeRunResult) map[string]*pb.ProbeResults {
	ret := make(map[string]*pb.ProbeResults)

	for probeName, probeRunResults := range results {
		pbProbeResults := &pb.ProbeResults{}
		for _, r := range probeRunResults {
			pbRunResult := &pb.ProbeRunResult{
				Target: &endpointpb.Endpoint{
					Name:   proto.String(r.Target.Name),
					Labels: r.Target.Labels,
				},
				Success:     proto.Bool(r.Success),
				LatencyUsec: proto.Int64(r.Latency.Microseconds()),
			}
			if r.Target.IP != nil {
				pbRunResult.Target.Ip = proto.String(r.Target.IP.String())
			}
			if r.Target.Port != 0 {
				pbRunResult.Target.Port = proto.Int32(int32(r.Target.Port))
			}
			if r.Error != nil {
				pbRunResult.Error = proto.String(r.Error.Error())
			}
			pbProbeResults.RunResult = append(pbProbeResults.RunResult, pbRunResult)
		}
		ret[probeName] = pbProbeResults
	}
	return ret
}
