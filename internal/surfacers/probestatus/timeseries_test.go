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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeseries(t *testing.T) {
	total, success := 200, 180
	inputInterval := 30 * time.Second
	testDurations := []time.Duration{time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute}

	var tests = []struct {
		n           int // Number of metrics
		latest      int
		oldest      int
		oldestTotal int
		totalDeltas []int64 // for durations 1, 2, 5, 10 min
	}{
		{
			// No data yet
			n: 0, oldest: 0, latest: 0,
			totalDeltas: []int64{-1, -1, -1, -1},
		},
		{
			// [1]
			n: 1, oldest: 0, latest: 0,
			totalDeltas: []int64{0, 0, 0, 0},
		},
		{
			// [1], [2, 3]
			n: 3, oldest: 0, latest: 1,
			totalDeltas: []int64{2, 2, 2, 2},
		},
		{
			// [1], [2,3], [4,5], [6,7], [8,9], [10, 11], [12, 13], -, -, -
			n: 13, oldest: 0, latest: 6, oldestTotal: 200,
			totalDeltas: []int64{2, 4, 10, 12},
		},
		//
		// Edge conditions around first rotation.
		//
		{
			// [1, 2], [3,4], [5,6], [7,8], [9,10], [11,12], [13,14], [15,16]
			//                                           ... [17,18],   -
			n: 18, oldest: 0, latest: 8, oldestTotal: 201,
			totalDeltas: []int64{2, 4, 10, 16},
		},
		{
			// [1, 2], [3,4], [5,6], [7,8], [9,10], [11,12], [13,14], [15,16]
			//                                           ... [17,18], [19,20]
			n: 20, oldest: 0, latest: 9, oldestTotal: 201,
		},
		{
			// [21,22], [3,4], [5,6], [7,8], [9,10], [11,12], [13,14], [15,16]
			//                                            ... [17,18], [19,20]
			n: 22, oldest: 1, latest: 0, oldestTotal: 203,
		},
		//
		// Edge conditions around double rotation.
		//
		{
			// [21,22], [23,24], [25,26], [27,28], [29,30], [31,32], [33,34],
			//                                 ... [35,36], [37,38], [19,20]
			n: 38, oldest: 9, latest: 8, oldestTotal: 219,
		},
		{
			// [21,22], [23,24], [25,26], [27,28], [29,30], [31,32], [33,34],
			//                                 ... [35,36], [37,38], [39,40]
			n: 40, oldest: 0, latest: 9, oldestTotal: 221,
		},
		{
			// [41,42], [23,24], [25,26], [27,28], [29,30], [31,32], [33,34],
			//                                 ... [35,36], [37,38], [39,40]
			n: 42, oldest: 1, latest: 0, oldestTotal: 223,
		},
	}

	defaultTotalDelta := []int64{2, 4, 10, 18}

	for _, test := range tests {
		t.Run(fmt.Sprintf("numMetrics:%d", test.n), func(t *testing.T) {
			ts := newTimeseries(time.Minute, 10, nil)

			// Start test.n ago.
			baseTime := time.Now().Truncate(time.Minute).Add(time.Duration(-test.n) * inputInterval)
			for i := 0; i < test.n; i++ {
				ts.addDatum(baseTime.Add(time.Duration(i)*inputInterval), &datum{total: int64(total + i), success: int64(success + i)})
			}

			if ts.oldestIdx != test.oldest || ts.latestIdx != test.latest {
				t.Errorf("timeseries oldest=%d,want=%d latest=%d,want=%d", ts.oldestIdx, test.oldest, ts.latestIdx, test.latest)
			}

			if test.oldestTotal != 0 {
				if ts.a[ts.oldestIdx].total != int64(test.oldestTotal) {
					t.Errorf("timeseries oldestTotal=%d,want=%d", ts.a[ts.oldestIdx].total, test.oldestTotal)
				}
			}

			if test.totalDeltas == nil {
				test.totalDeltas = defaultTotalDelta
			}
			for i, td := range testDurations {
				totalDelta, _ := ts.computeDelta(time.Time{}, td)
				if totalDelta != test.totalDeltas[i] {
					t.Errorf("Total delta for duration (%s)=%v, wanted=%v", td, totalDelta, test.totalDeltas[i])
				}
			}
		})
	}
}

func TestComputeDeltaEndTime(t *testing.T) {
	// Create a timeseries with 1-minute resolution and 10-slot buffer.
	// Add data at each minute: minute i has total=100+10*i, success=90+9*i.
	// This gives a constant delta of 10 total and 9 success per minute.
	ts := newTimeseries(time.Minute, 10, nil)
	start := time.Now().Truncate(time.Minute)
	ts.startTime = start

	for i := 0; i < 10; i++ {
		ts.addDatum(start.Add(time.Duration(i)*time.Minute), &datum{
			total:   int64(100 + 10*i),
			success: int64(90 + 9*i),
		})
	}
	// State: latestIdx=9, oldestIdx=0, size()=9, currentTS=start+9min
	// a[0]={100,90}, a[1]={110,99}, ..., a[9]={190,171}

	type deltaCheck struct {
		name        string
		endTime     time.Time
		window      time.Duration
		wantTotal   int64
		wantSuccess int64
	}

	t.Run("pre_rotation", func(t *testing.T) {
		checks := []deltaCheck{
			{
				name:        "endTime=current, window=5min",
				endTime:     start.Add(9 * time.Minute), // minute 9
				window:      5 * time.Minute,
				wantTotal:   50, // a[9]-a[4] = 190-140
				wantSuccess: 45, // 171-126
			},
			{
				name:        "endTime=minute7, window=3min",
				endTime:     start.Add(7 * time.Minute),
				window:      3 * time.Minute,
				wantTotal:   30, // a[7]-a[4] = 170-140
				wantSuccess: 27, // 153-126
			},
			{
				name:        "endTime=minute5, window=5min",
				endTime:     start.Add(5 * time.Minute),
				window:      5 * time.Minute,
				wantTotal:   50, // a[5]-a[0] = 150-100
				wantSuccess: 45, // 135-90
			},
			{
				name:        "endTime=minute3, window=3min",
				endTime:     start.Add(3 * time.Minute),
				window:      3 * time.Minute,
				wantTotal:   30, // a[3]-a[0] = 130-100
				wantSuccess: 27, // 117-90
			},
			{
				name:        "endTime=oldest, window=3min (start clamped)",
				endTime:     start, // minute 0
				window:      3 * time.Minute,
				wantTotal:   0, // start clamped to oldest, same as end
				wantSuccess: 0,
			},
			{
				name:        "endTime=future (clamped to current)",
				endTime:     start.Add(15 * time.Minute),
				window:      5 * time.Minute,
				wantTotal:   50, // same as endTime=current
				wantSuccess: 45,
			},
			{
				name:        "endTime before buffer",
				endTime:     start.Add(-1 * time.Minute),
				window:      5 * time.Minute,
				wantTotal:   -1,
				wantSuccess: -1,
			},
			{
				name:        "zero endTime (defaults to current)",
				endTime:     time.Time{},
				window:      5 * time.Minute,
				wantTotal:   50,
				wantSuccess: 45,
			},
		}

		for _, c := range checks {
			gotTotal, gotSuccess := ts.computeDelta(c.endTime, c.window)
			assert.Equal(t, c.wantTotal, gotTotal, c.name+" total")
			assert.Equal(t, c.wantSuccess, gotSuccess, c.name+" success")
		}
	})

	// Add 3 more minutes to cause rotation.
	// After this: buffer has minutes 3-12, latestIdx=2, oldestIdx=3.
	for i := 10; i < 13; i++ {
		ts.addDatum(start.Add(time.Duration(i)*time.Minute), &datum{
			total:   int64(100 + 10*i),
			success: int64(90 + 9*i),
		})
	}
	// a[0]={200,180}, a[1]={210,189}, a[2]={220,198}, a[3]={130,117}, ..., a[9]={190,171}

	t.Run("post_rotation", func(t *testing.T) {
		checks := []deltaCheck{
			{
				name:        "endTime=current(min12), window=5min",
				endTime:     start.Add(12 * time.Minute),
				window:      5 * time.Minute,
				wantTotal:   50, // a[2]-a[7] = 220-170
				wantSuccess: 45, // 198-153
			},
			{
				name:        "endTime=min10, window=5min",
				endTime:     start.Add(10 * time.Minute),
				window:      5 * time.Minute,
				wantTotal:   50, // a[0]-a[5] = 200-150
				wantSuccess: 45, // 180-135
			},
			{
				name:        "endTime=min8, window=3min",
				endTime:     start.Add(8 * time.Minute),
				window:      3 * time.Minute,
				wantTotal:   30, // a[8]-a[5] = 180-150
				wantSuccess: 27, // 162-135
			},
			{
				name:        "endTime=oldest(min3), window=2min (start clamped)",
				endTime:     start.Add(3 * time.Minute),
				window:      2 * time.Minute,
				wantTotal:   0, // both clamp to oldest
				wantSuccess: 0,
			},
			{
				name:        "endTime=min2 (before oldest, beyond buffer)",
				endTime:     start.Add(2 * time.Minute),
				window:      5 * time.Minute,
				wantTotal:   -1,
				wantSuccess: -1,
			},
			{
				name:        "zero endTime, window=9min (full buffer)",
				endTime:     time.Time{},
				window:      9 * time.Minute,
				wantTotal:   90, // a[2]-a[3] = 220-130
				wantSuccess: 81, // 198-117
			},
		}

		for _, c := range checks {
			gotTotal, gotSuccess := ts.computeDelta(c.endTime, c.window)
			assert.Equal(t, c.wantTotal, gotTotal, c.name+" total")
			assert.Equal(t, c.wantSuccess, gotSuccess, c.name+" success")
		}
	})
}
