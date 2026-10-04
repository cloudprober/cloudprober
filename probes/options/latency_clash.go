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

package options

import (
	"fmt"
	"slices"
	"strings"

	surfacerpb "github.com/cloudprober/cloudprober/internal/surfacers/proto"
	configpb "github.com/cloudprober/cloudprober/probes/proto"
)

// LatencyTypeClash is a latency metric name that is exported as a
// distribution by some probes and as a number by others.
type LatencyTypeClash struct {
	MetricName   string
	DistProbes   []string // Probes with latency_distribution, sorted.
	NumberProbes []string // Probes without latency_distribution, sorted.
}

// joinProbeNames joins the probe names, showing only the first few.
func joinProbeNames(names []string) string {
	const maxNames = 5
	if len(names) <= maxNames {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:maxNames], ", "), len(names)-maxNames)
}

func (c *LatencyTypeClash) String() string {
	return fmt.Sprintf("Metric %q is exported as a distribution by some probes (%s) and as a number by others (%s). "+
		"This doesn't work well with Prometheus and OpenTelemetry and will become an error in a future release. "+
		"To fix this, set latency_metric_name to a different name (e.g. %q) for the probes with latency_distribution.",
		c.MetricName, joinProbeNames(c.DistProbes), joinProbeNames(c.NumberProbes), c.MetricName+"_dist")
}

// hasPrometheusOrOTel returns true if the given surfacers config results in
// a Prometheus or an OTel surfacer.
func hasPrometheusOrOTel(surfacers []*surfacerpb.SurfacerDef) bool {
	// Prometheus is one of the default surfacers.
	if len(surfacers) == 0 {
		return true
	}
	return slices.ContainsFunc(surfacers, func(s *surfacerpb.SurfacerDef) bool {
		switch s.GetType() {
		case surfacerpb.Type_PROMETHEUS, surfacerpb.Type_OTEL:
			return true
		case surfacerpb.Type_NONE:
			// Type is inferred from the surfacer config in this case.
			switch s.Surfacer.(type) {
			case *surfacerpb.SurfacerDef_PrometheusSurfacer, *surfacerpb.SurfacerDef_OtelSurfacer:
				return true
			}
		}
		return false
	})
}

// LatencyTypeClashes returns the latency metric names that are exported as a
// distribution by some probes and as a number by others, sorted by metric
// name. Prometheus and OTel don't handle one metric name with two types
// well, so we look for clashes only if one of those surfacers is configured.
func LatencyTypeClashes(probes []*configpb.ProbeDef, surfacers []*surfacerpb.SurfacerDef) []*LatencyTypeClash {
	if !hasPrometheusOrOTel(surfacers) {
		return nil
	}

	byName := make(map[string]*LatencyTypeClash)
	for _, p := range probes {
		// System and UDP listener probes don't export latency.
		if t := p.GetType(); t == configpb.ProbeDef_SYSTEM || t == configpb.ProbeDef_UDP_LISTENER {
			continue
		}
		mn := p.GetLatencyMetricName()
		if byName[mn] == nil {
			byName[mn] = &LatencyTypeClash{MetricName: mn}
		}
		if p.GetLatencyDistribution() != nil {
			byName[mn].DistProbes = append(byName[mn].DistProbes, p.GetName())
		} else {
			byName[mn].NumberProbes = append(byName[mn].NumberProbes, p.GetName())
		}
	}

	var clashes []*LatencyTypeClash
	for _, c := range byName {
		if len(c.DistProbes) == 0 || len(c.NumberProbes) == 0 {
			continue
		}
		slices.Sort(c.DistProbes)
		slices.Sort(c.NumberProbes)
		clashes = append(clashes, c)
	}
	slices.SortFunc(clashes, func(a, b *LatencyTypeClash) int {
		return strings.Compare(a.MetricName, b.MetricName)
	})
	return clashes
}
