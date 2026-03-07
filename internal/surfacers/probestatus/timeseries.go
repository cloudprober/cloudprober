// Copyright 2022-2026 The Cloudprober Authors.
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
	"time"

	"github.com/cloudprober/cloudprober/logger"
)

type timeseries struct {
	a                    []*datum
	latestIdx, oldestIdx int
	res                  time.Duration
	currentTS            time.Time
	startTime            time.Time
	l                    *logger.Logger
}

func (ts *timeseries) shallowCopy() *timeseries {
	c := &timeseries{}
	*c = *ts
	return c
}

type datum struct {
	success, total int64
}

func newTimeseries(resolution time.Duration, size int, l *logger.Logger) *timeseries {
	if resolution == 0 {
		resolution = time.Minute
	}
	return &timeseries{
		a:         make([]*datum, size),
		res:       resolution,
		startTime: time.Now(),
		l:         l,
	}
}

func (ts *timeseries) addDatum(t time.Time, d *datum) {
	newDatumTime := t.Truncate(ts.res)
	// Need a new bucket
	if newDatumTime.After(ts.currentTS) && !ts.currentTS.IsZero() {
		numMissed := int(newDatumTime.Sub(ts.currentTS) / ts.res)
		if numMissed > len(ts.a) {
			numMissed = len(ts.a)
		}

		lastKnownDatum := ts.a[ts.latestIdx]
		for i := 0; i < numMissed; i++ {
			// Move
			ts.latestIdx = (ts.latestIdx + 1) % len(ts.a)
			if ts.latestIdx == ts.oldestIdx {
				ts.oldestIdx = (ts.latestIdx + 1) % len(ts.a)
			}
			// Fill intermediate buckets with the last known datum.
			// The last bucket will be overwritten by `d` below anyway,
			// but we can just fill it here for simplicity.
			if i < numMissed-1 {
				ts.a[ts.latestIdx] = lastKnownDatum
			}
		}
	}
	// Same bucket but newer data
	if t.After(ts.currentTS) {
		ts.currentTS = newDatumTime
		ts.a[ts.latestIdx] = d
		return
	}
}

func (ts *timeseries) agoIndex(durationCount int) int {
	// This happens before first rotation, and after that whenever rotation
	// happens.
	if ts.oldestIdx <= ts.latestIdx {
		if ts.latestIdx-durationCount < ts.oldestIdx {
			return ts.oldestIdx
		}
		return ts.latestIdx - durationCount
	}

	// One side is enough.
	if durationCount <= ts.latestIdx {
		return ts.latestIdx - durationCount
	}

	if durationCount > len(ts.a)-1 {
		durationCount = len(ts.a) - 1
	}

	// Flatten the array (add first section to the end) and subtract numPoints.
	return (len(ts.a) + ts.latestIdx) - durationCount
}

func (ts *timeseries) size() int {
	if ts.oldestIdx <= ts.latestIdx {
		return ts.latestIdx - ts.oldestIdx
	}
	return len(ts.a)
}

func (ts *timeseries) computeDelta(td time.Duration) (int64, int64) {
	// If current data is older than what we're looking for, return -1. We use
	// this information to decide whether to show availability data for this
	// period or not. This is to take care of the deleted targets.
	// Note we add a grace period of 10*ts.res before returning -1, so the data
	// in the status tables can be up to 10 min old.
	if time.Since(ts.currentTS) > td+10*ts.res {
		return -1, -1
	}

	startIndex := ts.agoIndex(int(td / ts.res))
	if startIndex == ts.latestIdx {
		return 0, 0
	}

	tD := ts.a[ts.latestIdx].total - ts.a[startIndex].total
	sD := ts.a[ts.latestIdx].success - ts.a[startIndex].success
	return tD, sD
}
