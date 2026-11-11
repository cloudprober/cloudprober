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

// summaryWindowFor returns the summary window for a timeseries resolution and
// how far back the timeseries goes. The window is rounded up to whole
// resolution steps, as computeDelta rounds it down: to nothing at all if the
// resolution is larger than the window.
func summaryWindowFor(res, maxDuration time.Duration) time.Duration {
	win := summaryWindow
	if res > 0 && win%res != 0 {
		win += res - win%res
	}
	return min(win, maxDuration)
}

// maxFailingInSummary is how many failing targets a summary line names before
// it switches to "+N more".
const maxFailingInSummary = 3

// health is a success ratio's level, used for both the summary dots and the
// status table cells.
type health int

// Ordered so that the worst of several levels is their max, and so that
// sorting by descending level puts what needs attention first, and what has
// no data last.
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
	Health      health
	OK, Total   int // Targets that are OK, out of those with data.
	Failing     []string
	MoreFailing int
}

// summary returns the summary lines for probeNames, with probes that need
// attention first. Within a health level, probes keep their order. Must be
// called from the event loop goroutine.
func (ps *Surfacer) summary(probeNames []string, window time.Duration) []*probeSummary {
	var out []*probeSummary

	for _, probeName := range probeNames {
		probeTS := ps.metrics[probeName]
		if probeTS == nil {
			continue
		}

		s := &probeSummary{Name: probeName}
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
			s.Health = max(s.Health, h)
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
		out = append(out, s)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Health > out[j].Health
	})
	return out
}
