// Copyright 2022 The Cloudprober Authors.
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
	a              []*datum
	latest, oldest int
	res            time.Duration
	currentTS      time.Time
	startTime      time.Time
	l              *logger.Logger
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
	tt := t.Truncate(ts.res)
	// Need a new bucket
	if tt.After(ts.currentTS) && !ts.currentTS.IsZero() {
		// Move
		ts.latest = (ts.latest + 1) % len(ts.a)
		if ts.latest == ts.oldest {
			ts.oldest = (ts.latest + 1) % len(ts.a)
		}
	}
	// Same bucket but newer data
	if t.After(ts.currentTS) {
		ts.currentTS = tt
		ts.a[ts.latest] = d
		return
	}
}

func (ts *timeseries) agoIndex(durationCount int) int {
	// This happens before first rotation, and after that whenever rotation
	// happens.
	if ts.oldest <= ts.latest {
		if ts.latest-durationCount < ts.oldest {
			return ts.oldest
		}
		return ts.latest - durationCount
	}

	// One side is enough.
	if durationCount <= ts.latest {
		return ts.latest - durationCount
	}

	if durationCount > len(ts.a)-1 {
		durationCount = len(ts.a) - 1
	}

	// Flatten the array (add first section to the end) and subtract numPoints.
	return (len(ts.a) + ts.latest) - durationCount
}

func (ts *timeseries) size() int {
	if ts.oldest <= ts.latest {
		return ts.latest - ts.oldest
	}
	return len(ts.a)
}

func (ts *timeseries) computeDelta(td time.Duration) (int64, int64) {
	// If current data is older than what we're looking for, return -1. We use
	// this information to decide whether to show availability data for this
	// period or not. This is to take care of the deleted targets.
	// Note we add a grace period of 2*ts.res before returning -1, so the data
	// in the status tables can be up to 2 min old.
	if time.Since(ts.currentTS) > td+2*ts.res {
		return -1, -1
	}

	startIndex := ts.agoIndex(int(td / ts.res))
	if startIndex == ts.latest {
		return 0, 0
	}

	tD := ts.a[ts.latest].total - ts.a[startIndex].total
	sD := ts.a[ts.latest].success - ts.a[startIndex].success
	return tD, sD
}
