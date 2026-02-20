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
				totalDelta, _ := ts.computeDelta(td)
				if totalDelta != test.totalDeltas[i] {
					t.Errorf("Total delta for duration (%s)=%v, wanted=%v", td, totalDelta, test.totalDeltas[i])
				}
			}
		})
	}
}
