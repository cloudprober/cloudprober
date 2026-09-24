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
	"strings"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

// targetsAggregator aggregates cumulative EventMetrics across targets. It
// drops the "dst" label and sums metrics for each remaining label set.
//
// Per-target metrics are cumulative, so we keep the last EventMetrics seen
// for each target and fold only the delta into the aggregate. This keeps the
// aggregate monotonic: a target's contribution stays after it goes away.
type targetsAggregator struct {
	mu sync.Mutex

	// Last EventMetrics seen for a target and label set.
	last map[string]*lastTargetEM
	// Aggregated EventMetrics, keyed by label set without "dst".
	agg map[string]*metrics.EventMetrics

	// Snapshots of targets that are no longer active are removed once they
	// have not been updated for staleAfter.
	listEndpoints func() []endpoint.Endpoint
	staleAfter    time.Duration
	lastGC        time.Time
	now           func() time.Time

	l *logger.Logger
}

type lastTargetEM struct {
	em        *metrics.EventMetrics
	targetKey string
	aggKey    string
	updated   time.Time
}

func newTargetsAggregator(listEndpoints func() []endpoint.Endpoint, staleAfter time.Duration, l *logger.Logger) *targetsAggregator {
	return &targetsAggregator{
		last:          make(map[string]*lastTargetEM),
		agg:           make(map[string]*metrics.EventMetrics),
		listEndpoints: listEndpoints,
		staleAfter:    staleAfter,
		lastGC:        time.Now(),
		now:           time.Now,
		l:             l,
	}
}

// labelsKey returns a key for em's label set, skipping the given label.
func labelsKey(em *metrics.EventMetrics, skipLabel string) string {
	var b strings.Builder
	for _, k := range em.LabelsKeys() {
		if k == skipLabel {
			continue
		}
		b.WriteString(k + "=" + em.Label(k) + ",")
	}
	return b.String()
}

// record folds em into the aggregate and returns the EventMetrics to export.
// GAUGE EventMetrics are returned as is.
func (ta *targetsAggregator) record(ep endpoint.Endpoint, em *metrics.EventMetrics) *metrics.EventMetrics {
	if em.Kind != metrics.CUMULATIVE {
		return em
	}

	ta.mu.Lock()
	defer ta.mu.Unlock()

	now := ta.now()
	ta.gc(now)

	targetKey, aggKey := ep.Key(), labelsKey(em, "dst")
	lastKey := targetKey + "|" + labelsKey(em, "")

	var prev *metrics.EventMetrics
	if last := ta.last[lastKey]; last != nil {
		prev = last.em
	}
	ta.last[lastKey] = &lastTargetEM{em: em, targetKey: targetKey, aggKey: aggKey, updated: now}

	aggEM := ta.agg[aggKey]
	if aggEM == nil {
		aggEM = metrics.NewEventMetrics(em.Timestamp)
		for _, k := range em.LabelsKeys() {
			if k != "dst" {
				aggEM.AddLabel(k, em.Label(k))
			}
		}
		ta.agg[aggKey] = aggEM
	}

	for _, name := range em.MetricsKeys() {
		val := em.Metric(name)
		// String metrics can't be summed.
		if metrics.IsString(val) {
			continue
		}

		// SubtractCounter leaves delta unchanged if the counter was reset,
		// e.g. target was removed and added again, which is what we want.
		delta := val.Clone()
		if prev != nil && prev.Metric(name) != nil {
			if _, err := delta.SubtractCounter(prev.Metric(name)); err != nil {
				ta.l.Warningf("Error aggregating metric %s across targets: %v", name, err)
				continue
			}
		}

		if aggVal := aggEM.Metric(name); aggVal != nil {
			if err := aggVal.Add(delta); err != nil {
				ta.l.Warningf("Error aggregating metric %s across targets: %v", name, err)
			}
			continue
		}
		aggEM.AddMetric(name, delta)
	}

	// Keep aggregate's timestamp monotonic.
	if em.Timestamp.After(aggEM.Timestamp) {
		aggEM.Timestamp = em.Timestamp
	}

	out := aggEM.Clone()
	out.LatencyUnit = em.LatencyUnit
	return out
}

// gc removes snapshots of targets that are gone, and aggregates that have
// no targets left. We don't remove snapshots of targets that are still
// listed, even if stale (e.g. probe is disabled by a schedule), as their
// next update would otherwise be counted in full again.
func (ta *targetsAggregator) gc(now time.Time) {
	if now.Sub(ta.lastGC) < ta.staleAfter {
		return
	}
	ta.lastGC = now

	active := make(map[string]bool)
	if ta.listEndpoints != nil {
		for _, ep := range ta.listEndpoints() {
			active[ep.Key()] = true
		}
	}

	liveAggKeys := make(map[string]bool)
	for k, last := range ta.last {
		if !active[last.targetKey] && now.Sub(last.updated) > ta.staleAfter {
			delete(ta.last, k)
			continue
		}
		liveAggKeys[last.aggKey] = true
	}

	for k := range ta.agg {
		if !liveAggKeys[k] {
			delete(ta.agg, k)
		}
	}
}
