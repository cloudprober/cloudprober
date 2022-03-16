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
	"fmt"
	"testing"
	"time"
)

func TestTimeseries(t *testing.T) {
	total, success := 200, 180
	inputInterval := 30 * time.Second
	testDurations := []time.Duration{time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute}

	var tests = []struct {
		numMetrics  int
		latest      int
		oldest      int
		oldestTotal int
		lenData     int     // lenght of data for 10m query.
		totalDeltas []int64 // for durations 1, 2, 5, 10 min
	}{
		{
			// _, [1], [2,3], [4,5], [6,7], [8,9], [10, 11], [12, 13]
			numMetrics:  13,
			oldest:      0,
			latest:      7,   // Leave first empty, fill next 7
			oldestTotal: 200, // First input
			lenData:     7,   // First bucket empty.
			totalDeltas: []int64{2, 4, 10, 12},
		},
		{
			// _, [1, 2], [3,4], [5,6], [7,8], [9,10], [11,12], [13,14],
			//                                     ... [15,16], [17,18]
			numMetrics:  18,
			oldest:      0,
			latest:      9,   // Leave first empty, fill next 9
			oldestTotal: 201, // First gets overwritten
			lenData:     9,   // First bucket is empty.
			totalDeltas: []int64{2, 4, 10, 16},
		},
		{
			// [19,20], [21,22], [3,4], [5,6], [7,8], [9,10], [11,12], [13,14],
			//                                            ... [15,16], [17,18]
			numMetrics:  22, // It will rotate.
			oldest:      2,
			latest:      1,
			oldestTotal: 203, // First bucket is lost
			lenData:     10,
			totalDeltas: []int64{2, 4, 10, 18},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("numMetrics: %d", test.numMetrics), func(t *testing.T) {
			ts := newTimeseries(time.Minute, 10)

			// Start test.numMetrics ago.
			baseTime := time.Now().Truncate(time.Minute).Add(time.Duration(-test.numMetrics) * inputInterval)
			for i := 0; i < test.numMetrics; i++ {
				ts.addDatum(baseTime.Add(time.Duration(i)*inputInterval), &datum{total: int64(total + i), success: int64(success + i)})
			}

			if ts.oldest != test.oldest || ts.latest != test.latest {
				t.Errorf("timeseries oldest=%d,want=%d latest=%d,want=%d", ts.oldest, test.oldest, ts.latest, test.latest)
			}
			oldest := ts.oldest
			if oldest == 0 {
				oldest = 1
			}
			if ts.a[oldest].total != int64(test.oldestTotal) {
				t.Errorf("timeseries oldestTotal=%d,want=%d", ts.a[oldest].total, test.oldestTotal)
			}

			// Test data retrieval.
			for i := 0; i <= 15; i = i + 2 {
				t.Run(fmt.Sprintf("getRecentData(%d)", i), func(t *testing.T) {
					data := ts.getRecentData(time.Duration(i) * time.Minute)
					for j, d := range data {
						if d == nil {
							t.Errorf("Duration: %d minutes, Got nil data at %d", i, j)
						}
					}
				})
			}

			data := ts.getRecentData(10 * time.Minute)
			if len(data) != test.lenData {
				t.Errorf("len(data)=%d, wanted=%d", len(data), test.lenData)
			}

			for i, td := range testDurations {
				totalDelta, _ := ts.computeDelta(data, td)
				if totalDelta != test.totalDeltas[i] {
					t.Errorf("total delta for duration (%s)=%v, wanted=%v", td, totalDelta, test.totalDeltas[i])
				}
			}
		})
	}
}
