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
)

type timeseries struct {
	a              []*datum
	latest, oldest int
	res            time.Duration
	currentTS      time.Time
}

type datum struct {
	success, total int64
}

func newTimeseries(resolution time.Duration, size int) *timeseries {
	if resolution == 0 {
		resolution = time.Minute
	}
	return &timeseries{
		a:   make([]*datum, size),
		res: resolution,
	}
}

func (ts *timeseries) addDatum(t time.Time, d *datum) {
	tt := t.Truncate(ts.res)
	// Need a new bucket
	if tt.After(ts.currentTS) {
		// Move
		ts.latest = (ts.latest + 1) % len(ts.a)
		if ts.latest == ts.oldest {
			ts.oldest++
		}
	}
	// Same bucket but newer data
	if t.After(ts.currentTS) {
		ts.currentTS = tt
		ts.a[ts.latest] = d
		return
	}
}

func (ts *timeseries) oldestIndex() int {
	if ts.oldest == 0 {
		return 1
	}
	return ts.oldest
}

func (ts *timeseries) ago(td time.Duration) *datum {
	numPoints := int(td / ts.res)
	if ts.oldest == 0 {
		if numPoints > ts.latest-1 {
			return ts.a[1]
		}
		return ts.a[ts.latest-numPoints]
	}

	// One side is enough.
	if numPoints <= ts.latest {
		return ts.a[ts.latest-numPoints]
	}

	if numPoints > len(ts.a)-1 {
		numPoints = len(ts.a) - 1
	}
	// Flatten the array (add first section to the end) and subtract numPoints.
	index := (len(ts.a) + ts.latest) - numPoints
	return ts.a[index]
}

func (ts *timeseries) getData() []*datum {
	// We haven't rotated yet.
	if ts.oldest == 0 {
		return append([]*datum{}, ts.a[1:ts.latest+1]...)
	}
	return append([]*datum{}, append(ts.a[ts.latest+1:], ts.a[:ts.latest+1]...)...)
}

func (ts *timeseries) computeDelta(td time.Duration) (int64, int64) {
	lastData := ts.a[ts.latest]
	d := ts.ago(td)
	return lastData.total - d.total, lastData.success - d.success
}
