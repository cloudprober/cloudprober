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
	"reflect"
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

	var tests = []struct {
		numMetrics  int
		latest      int
		oldest      int
		oldestTotal int
		totalDeltas []int64 // for durations 1, 2, 5, 10 min
	}{
		{
			// _, [1], [2,3], [4,5], [6,7], [8,9], [10, 11], [12, 13]
			numMetrics:  13,
			oldest:      0,
			latest:      7, // Leave first empty, fill next 7
			oldestTotal: 187,
			totalDeltas: []int64{2, 4, 10, 12}, // 7 buckets max.
		},
		{
			// _, [1, 2], [3,4], [5,6], [7,8], [9,10], [11,12], [13,14],
			//                                 ........[15,16], [17,18]
			numMetrics:  18,
			oldest:      0,
			latest:      9,                     // Leave first empty, fill next 9
			oldestTotal: 183,                   // First gets overwritten
			totalDeltas: []int64{2, 4, 10, 16}, // 9 buckets for 10m window.
		},
		{
			// [19,20], [21,22], [3,4], [5,6], [7,8], [9,10], [11,12], [13,14],
			//                                            ... [15,16], [17,18]
			numMetrics: 22, // It will rotate.
			oldest:     2,
			latest:     1,

			// Start at 178, but [1,2] is lost, and 3 is overwritten by 4.
			oldestTotal: 181,
			totalDeltas: []int64{2, 4, 10, 18}, // 10 buckets for 10m window.
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

			for i := test.numMetrics; i >= 1; i-- {
				tm := time.Now().Truncate(time.Minute).Add(time.Duration(-i) * inputInterval)
				ps.record(testEM(t, tm, "t1", total-i, success-i, latency-9.9*float64(i)))
				ps.record(testEM(t, tm, "t2", total-i, success-2*i, latency-10*float64(i)))
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

			durations := []time.Duration{time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute}
			data := ts.getRecentData(10 * time.Minute)
			totalDeltas, _ := ps.computeDelta(durations, data)
			t.Logf("%+v", len(data))
			if !reflect.DeepEqual(totalDeltas, test.totalDeltas) {
				t.Errorf("total deltas=%v, want deltas=%v", totalDeltas, test.totalDeltas)
			}
		})
	}
}
