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

package probestatus

import (
	"sort"
	"time"
)

// summaryWindow is the window the status page summary is based on. 1m flaps
// on a single failed probe, and 30m keeps showing a problem long after it's
// fixed.
const summaryWindow = 5 * time.Minute

// maxFailingInSummary is how many failing targets a summary line names before
// it switches to "+N more".
const maxFailingInSummary = 3

// health is a success ratio's level, used for both the summary dots and the
// status table cells.
type health int

// Declared from best to worst, so that the worst of several is the max.
const (
	healthNoData health = iota
	healthOK
	healthWarn
	healthBad
)

func (h health) String() string {
	return [...]string{"nodata", "ok", "warn", "bad"}[h]
}

// ratioHealth classifies a success ratio: OK at 1, warn down to 0.99, bad
// below that.
func ratioHealth(total, success int64) health {
	switch {
	case total <= 0:
		return healthNoData
	case success == total:
		return healthOK
	case success*1000/total >= 990:
		return healthWarn
	default:
		return healthBad
	}
}

// probeSummary is one probe's line in the status page summary.
type probeSummary struct {
	Name        string
	Health      string
	OK, Total   int // Targets that are OK, out of those with data.
	Failing     []string
	MoreFailing int
}

// summary returns the summary lines for probeNames, with probes that need
// attention first. Within a health level, probes keep their order. Must be
// called from the event loop goroutine.
func (ps *Surfacer) summary(probeNames []string, window time.Duration) []*probeSummary {
	type entry struct {
		h health
		s *probeSummary
	}
	var entries []entry

	for _, probeName := range probeNames {
		probeTS := ps.metrics[probeName]
		if probeTS == nil {
			continue
		}

		s, worst := &probeSummary{Name: probeName}, healthNoData
		for _, targetName := range ps.probeTargets[probeName] {
			ts := probeTS[targetName]
			if ts == nil {
				continue
			}
			h := ratioHealth(ts.computeDelta(time.Time{}, window))
			if h == healthNoData {
				continue
			}

			s.Total++
			worst = max(worst, h)
			if h == healthOK {
				s.OK++
				continue
			}
			if len(s.Failing) < maxFailingInSummary {
				s.Failing = append(s.Failing, targetName)
			} else {
				s.MoreFailing++
			}
		}
		s.Health = worst.String()
		entries = append(entries, entry{worst, s})
	}

	// Bad, then warn, then OK, then no data.
	rank := map[health]int{healthBad: 0, healthWarn: 1, healthOK: 2, healthNoData: 3}
	sort.SliceStable(entries, func(i, j int) bool {
		return rank[entries[i].h] < rank[entries[j].h]
	})

	out := make([]*probeSummary, len(entries))
	for i, e := range entries {
		out[i] = e.s
	}
	return out
}
