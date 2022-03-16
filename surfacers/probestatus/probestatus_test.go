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

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/surfacers/probestatus/proto"
	"google.golang.org/protobuf/proto"
)

const (
	testProbe = "test-probe"
)

func testEM(t *testing.T, tm time.Time, target string, total, success int, latency float64) *metrics.EventMetrics {
	t.Helper()
	return metrics.NewEventMetrics(tm).
		AddLabel("probe", testProbe).
		AddLabel("dst", target).
		AddMetric("total", metrics.NewInt(int64(total))).
		AddMetric("success", metrics.NewInt(int64(total))).
		AddMetric("latency", metrics.NewFloat(latency))
}

func TestRecord(t *testing.T) {
	total, success := 200, 180
	latency := float64(2000)
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
			//                                 ........[15,16], [17,18]
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
			ps := &Surfacer{
				c: &configpb.SurfacerConf{
					MaxPoints: proto.Int32(10),
				},
				emChan:       make(chan *metrics.EventMetrics, metricsBufferSize),
				queryChan:    make(chan *httpWriter, queriesQueueSize),
				metrics:      make(map[string]map[string]*timeseries),
				probeTargets: make(map[string][]string),
				resolution:   time.Minute,
				l:            &logger.Logger{},
			}

			// Start test.numMetrics ago.
			baseTime := time.Now().Truncate(time.Minute).Add(time.Duration(-test.numMetrics) * inputInterval)
			for i := 0; i < test.numMetrics; i++ {
				ps.record(testEM(t, baseTime.Add(time.Duration(i)*inputInterval), "t1", total+i, success+i, latency+9.9*float64(i)))
				ps.record(testEM(t, baseTime.Add(time.Duration(i)*inputInterval), "t2", total+i, success+i, latency+10*float64(i)))
			}

			for _, target := range []string{"t1", "t2"} {
				ts := ps.metrics[testProbe][target]
				if ts == nil {
					t.Errorf("Unexpected nil timeseries for target: %s", target)
				}
			}

			// Let's work on just one target's series.
			ts := ps.metrics[testProbe]["t1"]
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
				totalDelta, _ := ps.computeDelta(data, td)
				if totalDelta != test.totalDeltas[i] {
					t.Errorf("total delta for duration (%s)=%v, wanted=%v", td, totalDelta, test.totalDeltas[i])
				}
			}
		})
	}
}
