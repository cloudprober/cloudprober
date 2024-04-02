// Copyright 2019 The Cloudprober Authors.
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

package testutils

import (
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/stretchr/testify/assert"
)

func TestMetricsFromChannel(t *testing.T) {
	dataChan := make(chan *metrics.EventMetrics, 10)

	var ts [2]time.Time

	ts[0] = time.Now()
	time.Sleep(time.Millisecond)
	ts[1] = time.Now()

	// Put 2 EventMetrics, get 2
	dataChan <- metrics.NewEventMetrics(ts[0])
	dataChan <- metrics.NewEventMetrics(ts[1])

	ems, err := MetricsFromChannel(dataChan, 2, time.Second)
	if err != nil {
		t.Error(err)
	}

	for i := range ems {
		if ems[i].Timestamp != ts[i] {
			t.Errorf("First EventMetrics has unexpected timestamp. Got=%s, Expected=%s", ems[i], ts[i])
		}
	}

	// Put 2 EventMetrics, try to get 3
	dataChan <- metrics.NewEventMetrics(ts[0])
	dataChan <- metrics.NewEventMetrics(ts[1])

	ems, err = MetricsFromChannel(dataChan, 3, time.Second)
	if err == nil {
		t.Error("expected error got none")
	}

	for i := range ems {
		if ems[i].Timestamp != ts[i] {
			t.Errorf("First EventMetrics has unexpected timestamp. Got=%s, Expected=%s", ems[i], ts[i])
		}
	}
}

func TestMetricsMap(t *testing.T) {
	var ems []*metrics.EventMetrics
	expectedValues := map[string][]int64{
		"success": {99, 98},
		"total":   {100, 100},
	}
	ems = append(ems, metrics.NewEventMetrics(time.Now()).
		AddMetric("success", metrics.NewInt(99)).
		AddMetric("total", metrics.NewInt(100)).
		AddLabel("dst", "target1"))
	ems = append(ems, metrics.NewEventMetrics(time.Now()).
		AddMetric("success", metrics.NewInt(98)).
		AddMetric("total", metrics.NewInt(100)).
		AddLabel("dst", "target2"))

	metricsMap := MetricsMapByTarget(ems)

	for i, tgt := range []string{"target1", "target2"} {
		for _, m := range []string{"success", "total"} {
			assert.Len(t, metricsMap[tgt][m], 1, "number of values mismatch")
			val := metricsMap[tgt][m][0].(metrics.NumValue).Int64()
			assert.Equal(t, expectedValues[m][i], val, "metric value mismatch")
		}
	}
}

func TestMetricsMapFilterAndLastValueInt64(t *testing.T) {
	mmap := MetricsMap{
		"target1": map[string][]metrics.Value{
			"success": {metrics.NewInt(98), metrics.NewInt(99)},
			"total":   {metrics.NewInt(100), metrics.NewInt(101)},
		},
		"target2": map[string][]metrics.Value{
			"success2": {metrics.NewInt(198), metrics.NewInt(199)},
			"total":    {metrics.NewInt(200), metrics.NewInt(201)},
		},
	}

	tests := []struct {
		name       string
		target     string
		metricName string
		filtered   map[string][]metrics.Value
		want       int64
	}{
		{
			name:       "target2",
			target:     "target1",
			metricName: "success",
			filtered: map[string][]metrics.Value{
				"target1": {metrics.NewInt(98), metrics.NewInt(99)},
			},
			want: 99,
		},
		{
			name:       "target2",
			target:     "target2",
			metricName: "success",
			filtered: map[string][]metrics.Value{
				"target1": {metrics.NewInt(98), metrics.NewInt(99)},
			},
			want: -1,
		},
		{
			name:       "not found",
			target:     "target1",
			metricName: "latency",
			filtered:   map[string][]metrics.Value{},
			want:       -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.filtered, mmap.Filter(tt.metricName), "filtered map mismatch")
			if got := mmap.LastValueInt64(tt.target, tt.metricName); got != tt.want {
				t.Errorf("MetricsMap.LastValueInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventMetricsByTargetMetric(t *testing.T) {
	ems := []*metrics.EventMetrics{
		metrics.NewEventMetrics(time.Now()).
			AddMetric("success", metrics.NewInt(99)).
			AddMetric("total", metrics.NewInt(100)).
			AddLabel("dst", "target1"),
		metrics.NewEventMetrics(time.Now()).
			AddMetric("success", metrics.NewInt(98)).
			AddMetric("total", metrics.NewInt(100)).
			AddLabel("dst", "target2"),
		metrics.NewEventMetrics(time.Now()).
			AddMetric("success", metrics.NewInt(99)).
			AddMetric("total", metrics.NewInt(101)).
			AddLabel("dst", "target1"),
	}
	emsMap := EventMetricsByTargetMetric(ems)

	wantEMs := map[string][]*metrics.EventMetrics{
		"target1": {ems[0], ems[2]},
		"target2": {ems[1]},
	}
	for _, tgt := range []string{"target1", "target2"} {
		for _, m := range []string{"success", "total"} {
			assert.Len(t, emsMap[tgt][m], len(wantEMs[tgt]), "number of ems mismatch")
			assert.Equal(t, wantEMs[tgt], emsMap[tgt][m], "eventmetrics mismatch")
		}
	}
}
