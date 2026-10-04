// Copyright 2026 The Cloudprober Authors.
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

package config

import (
	"fmt"
	"testing"

	surfacerpb "github.com/cloudprober/cloudprober/internal/surfacers/proto"
	distpb "github.com/cloudprober/cloudprober/metrics/proto"
	probes_configpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func testProbeDef(name, metricName string, dist bool) *probes_configpb.ProbeDef {
	p := &probes_configpb.ProbeDef{Name: proto.String(name)}
	if metricName != "" {
		p.LatencyMetricName = proto.String(metricName)
	}
	if dist {
		p.LatencyDistribution = &distpb.Dist{}
	}
	return p
}

func TestLatencyTypeClashes(t *testing.T) {
	mixedProbes := []*probes_configpb.ProbeDef{
		testProbeDef("p3", "", false),
		testProbeDef("p2", "", true),
		testProbeDef("p1", "latency", false),
		testProbeDef("p4", "latency_dist", true),
		{
			Name: proto.String("sys_metrics"),
			Type: probes_configpb.ProbeDef_SYSTEM.Enum(),
		},
	}
	latencyClash := []*LatencyTypeClash{{
		MetricName:   "latency",
		DistProbes:   []string{"p2"},
		NumberProbes: []string{"p1", "p3"},
	}}

	tests := []struct {
		name      string
		probes    []*probes_configpb.ProbeDef
		surfacers []*surfacerpb.SurfacerDef
		want      []*LatencyTypeClash
	}{
		{
			name:   "default_surfacers",
			probes: mixedProbes,
			want:   latencyClash,
		},
		{
			name:   "prometheus_by_type",
			probes: mixedProbes,
			surfacers: []*surfacerpb.SurfacerDef{
				{Type: surfacerpb.Type_FILE.Enum()},
				{Type: surfacerpb.Type_PROMETHEUS.Enum()},
			},
			want: latencyClash,
		},
		{
			name:   "otel_by_config",
			probes: mixedProbes,
			surfacers: []*surfacerpb.SurfacerDef{{
				Surfacer: &surfacerpb.SurfacerDef_OtelSurfacer{},
			}},
			want: latencyClash,
		},
		{
			name:   "other_surfacers",
			probes: mixedProbes,
			surfacers: []*surfacerpb.SurfacerDef{
				{Type: surfacerpb.Type_FILE.Enum()},
				{Surfacer: &surfacerpb.SurfacerDef_StackdriverSurfacer{}},
			},
		},
		{
			name:      "surfacers_disabled",
			probes:    mixedProbes,
			surfacers: []*surfacerpb.SurfacerDef{{}},
		},
		{
			name: "two_clashes_sorted_by_name",
			probes: []*probes_configpb.ProbeDef{
				testProbeDef("p1", "rtt", true),
				testProbeDef("p2", "rtt", false),
				testProbeDef("p3", "", true),
				testProbeDef("p4", "", false),
			},
			want: []*LatencyTypeClash{
				{MetricName: "latency", DistProbes: []string{"p3"}, NumberProbes: []string{"p4"}},
				{MetricName: "rtt", DistProbes: []string{"p1"}, NumberProbes: []string{"p2"}},
			},
		},
		{
			name: "different_names",
			probes: []*probes_configpb.ProbeDef{
				testProbeDef("p1", "", false),
				testProbeDef("p2", "latency_dist", true),
			},
		},
		{
			name: "all_distributions",
			probes: []*probes_configpb.ProbeDef{
				testProbeDef("p1", "", true),
				testProbeDef("p2", "", true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, LatencyTypeClashes(tt.probes, tt.surfacers))
		})
	}
}

func TestLatencyTypeClashString(t *testing.T) {
	c := &LatencyTypeClash{MetricName: "latency", DistProbes: []string{"d1"}}
	for i := 1; i <= 7; i++ {
		c.NumberProbes = append(c.NumberProbes, fmt.Sprintf("n%d", i))
	}

	want := `Metric "latency" is exported as a distribution by some probes (d1) and as a number by others (n1, n2, n3, n4, n5 and 2 more). ` +
		`This doesn't work well with Prometheus and OpenTelemetry and will become an error in a future release. ` +
		`To fix this, set latency_metric_name to a different name (e.g. "latency_dist") for the probes with latency_distribution.`
	assert.Equal(t, want, c.String())
}
