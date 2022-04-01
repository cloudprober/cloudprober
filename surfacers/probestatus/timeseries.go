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

func (ts *timeseries) getRecentData(td time.Duration) []*datum {
	size := int(td / ts.res)
	if size > len(ts.a) || size == 0 {
		size = len(ts.a)
	}

	oldestIndex := ts.latest - (size - 1)
	if oldestIndex >= 0 {
		if oldestIndex == 0 {
			oldestIndex = 1
		}
		return append([]*datum{}, ts.a[oldestIndex:ts.latest+1]...)
	}

	if ts.oldest == 0 {
		return append([]*datum{}, ts.a[1:ts.latest+1]...)
	}

	otherSide := len(ts.a) + oldestIndex
	return append([]*datum{}, append(ts.a[otherSide:], ts.a[:ts.latest+1]...)...)
}

func (ts *timeseries) computeDelta(data []*datum, td time.Duration) (int64, int64) {
	lastData := data[len(data)-1]
	numPoints := int(td / ts.res)
	start := len(data) - 1 - numPoints
	if start < 0 {
		start = 0
	}
	return lastData.total - data[start].total, lastData.success - data[start].success
}
