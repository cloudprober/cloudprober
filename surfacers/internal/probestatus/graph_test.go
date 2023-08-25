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
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/stretchr/testify/assert"
)

func testTimeSeries(inputInterval, res time.Duration, size int, errorIndex int) *timeseries {
	ts := newTimeseries(res, size, nil)

	baseTime := time.Now().Truncate(res).Add(-time.Duration(size) * inputInterval)
	total, success := 200, 180
	for i := 0; i < size; i++ {
		total++
		if i != errorIndex {
			success++
		}
		ts.addDatum(baseTime.Add(time.Duration(i)*inputInterval), &datum{total: int64(total), success: int64(success)})
	}

	return ts
}

func TestComputeGraphPoints(t *testing.T) {
	ts := testTimeSeries(time.Minute, time.Minute, 10, 5)
	baseTime := ts.currentTS.Add(-time.Duration(9) * time.Minute)
	defaultGP := &graphPoints{
		endTime:   ts.currentTS,
		startTime: baseTime,
		values:    []float64{1, 0, 1},
		freqs:     []int64{4, 1, 4},
	}

	tests := []struct {
		desc    string
		endTime time.Time
		td      time.Duration
		gp      *graphPoints
	}{
		{desc: "too_small", td: 30 * time.Second, gp: nil},
		{desc: "too_long_ago", endTime: ts.currentTS.Add(-12 * time.Minute), gp: nil},
		{desc: "default_params", gp: defaultGP},
		{desc: "two_minutes_ahead", endTime: ts.currentTS.Add(2 * time.Minute), gp: defaultGP},
		{
			desc:    "one_minute_ago",
			endTime: ts.currentTS.Add(-time.Minute),
			gp: &graphPoints{
				endTime:   ts.currentTS.Add(-time.Minute),
				startTime: baseTime,
				values:    []float64{1, 0, 1},
				freqs:     []int64{4, 1, 3},
			},
		},
		{
			desc:    "one_minute_ago_5min_duration",
			endTime: ts.currentTS.Add(-time.Minute),
			td:      5 * time.Minute,
			gp: &graphPoints{
				endTime:   ts.currentTS.Add(-time.Minute),
				startTime: ts.currentTS.Add(-6 * time.Minute),
				values:    []float64{1, 0, 1},
				freqs:     []int64{1, 1, 3},
			},
		},
		{
			desc: "large_duration",
			td:   time.Hour,
			gp: &graphPoints{
				endTime:   ts.currentTS,
				startTime: baseTime,
				values:    []float64{1, 0, 1},
				freqs:     []int64{4, 1, 4},
			},
		},
		{
			desc: "very_small",
			td:   time.Minute,
			gp: &graphPoints{
				endTime:   ts.currentTS,
				startTime: ts.currentTS.Add(-time.Minute),
				values:    []float64{1},
				freqs:     []int64{1},
			},
		},
	}

	for _, test := range tests {
		if test.endTime.IsZero() {
			test.endTime = time.Now()
		}
		if test.td == 0 {
			test.td = time.Hour
		}
		res := graphResolution(test.endTime, test.td)
		t.Run(test.desc, func(t *testing.T) {
			gp := computeGraphPoints(ts, &graphOptions{endTime: test.endTime, duration: test.td, res: res})
			assert.Equal(t, gp, test.gp, "")
		})
	}
}

func TestSyncGraphLines(t *testing.T) {
	metrics := map[string]*timeseries{
		"t1": testTimeSeries(time.Minute, time.Minute, 10, 5),
		"t2": testTimeSeries(time.Minute, time.Minute, 10, -1),
	}
	baseET := metrics["t1"].currentTS
	baseST := baseET.Add(-9 * time.Minute)

	tests := []struct {
		desc       string
		startTimes []time.Time
		endTimes   []time.Time
		st, et     int64
		vals       map[string][]float64
		freqs      map[string][]int64
	}{
		{desc: "default"},
		{
			desc:       "t2_starts_early",
			startTimes: []time.Time{baseST, baseST.Add(-time.Minute)},
			st:         baseST.Add(-time.Minute).Unix(),
			vals:       map[string][]float64{"t1": {-1, 1, 0, 1}},
			freqs:      map[string][]int64{"t1": {1, 4, 1, 4}},
		},
		{
			desc:       "t2_starts_late",
			startTimes: []time.Time{baseST, baseST.Add(time.Minute)},
			vals:       map[string][]float64{"t2": {-1, 1}},
			freqs:      map[string][]int64{"t2": {1, 9}},
		},
		{
			desc:     "t2_finishes_early",
			endTimes: []time.Time{baseET, baseET.Add(-time.Minute)},
			vals:     map[string][]float64{"t2": {1, -1}},
			freqs:    map[string][]int64{"t2": {9, 1}},
		},
		{
			desc:     "t1_finishes_late",
			endTimes: []time.Time{baseET.Add(time.Minute), baseET},
			et:       baseET.Add(time.Minute).Unix(),
			vals:     map[string][]float64{"t2": {1, -1}},
			freqs:    map[string][]int64{"t2": {9, 1}},
		},
		{
			desc:     "t1_finishes_early",
			endTimes: []time.Time{baseET.Add(-2 * time.Minute), baseET},
			vals:     map[string][]float64{"t1": {1, 0, 1, -1}},
			freqs:    map[string][]int64{"t1": {4, 1, 4, 2}},
		},
		{
			desc:     "t2_finishes_late",
			endTimes: []time.Time{baseET, baseET.Add(2 * time.Minute)},
			et:       baseET.Add(2 * time.Minute).Unix(),
			vals:     map[string][]float64{"t1": {1, 0, 1, -1}},
			freqs:    map[string][]int64{"t1": {4, 1, 4, 2}},
		},
		{
			desc:       "t2_starts_early_t1_finishes_late",
			startTimes: []time.Time{baseST, baseST.Add(-time.Minute)},
			endTimes:   []time.Time{baseET.Add(time.Minute), baseET},
			st:         baseST.Add(-time.Minute).Unix(),
			et:         baseET.Add(time.Minute).Unix(),
			vals: map[string][]float64{
				"t1": {-1, 1, 0, 1},
				"t2": {1, -1},
			},
			freqs: map[string][]int64{
				"t1": {1, 4, 1, 4},
				"t2": {9, 1},
			},
		},
	}

	for _, test := range tests {
		startTimes := map[string]time.Time{"t1": baseST, "t2": baseST}
		endTimes := map[string]time.Time{"t1": baseET, "t2": baseET}

		if test.startTimes != nil {
			startTimes["t1"], startTimes["t2"] = test.startTimes[0], test.startTimes[1]
		}
		if test.endTimes != nil {
			endTimes["t1"], endTimes["t2"] = test.endTimes[0], test.endTimes[1]
		}
		t.Run(test.desc, func(t *testing.T) {
			gd := computeGraphData(metrics, &graphOptions{endTime: baseET, duration: time.Hour, res: time.Minute})

			if test.st == 0 {
				test.st = gd.StartTime
			}
			if test.et == 0 {
				test.et = gd.EndTime
			}
			if test.vals == nil {
				test.vals = make(map[string][]float64)
			}
			if test.freqs == nil {
				test.freqs = make(map[string][]int64)
			}
			for tgt := range gd.Values {
				if test.vals[tgt] == nil {
					test.vals[tgt] = gd.Values[tgt]
				}
				if test.freqs[tgt] == nil {
					test.freqs[tgt] = gd.Freqs[tgt]
				}
			}

			gd.syncGraphLines(startTimes, endTimes)

			assert.Equal(t, test.st, gd.StartTime, "")
			assert.Equal(t, test.et, gd.EndTime, "")
			assert.Equal(t, test.vals, gd.Values, "")
			assert.Equal(t, test.freqs, gd.Freqs, "")
		})
	}
}

func TestGraphOptsFromURL(t *testing.T) {
	et := time.Now().Truncate(time.Second)
	etStr := strconv.FormatInt(et.Unix(), 10)
	maxDuration := 72 * time.Hour

	tests := []struct {
		q     url.Values
		gopts *graphOptions
	}{
		{
			q: map[string][]string{
				"graph_endtime":  {"13141", etStr},
				"graph_duration": {"10s", "6h"},
			},
			gopts: &graphOptions{
				endTime:  et,
				duration: 6 * time.Hour,
				res:      time.Minute,
			},
		},
		{
			// We'll use default endtime(=time.Now()) and duration.
			q: map[string][]string{
				"graph_endtime": {"as1231"},
			},
			gopts: &graphOptions{
				duration: maxDuration,
				res:      5 * time.Minute,
			},
		},
		{
			// Force 1 min resolution for maxDuration
			q: map[string][]string{
				"graph_duration": {"21d"}, // Parse error
				"graph_res":      {"1m"},
			},
			gopts: &graphOptions{
				duration: maxDuration,
				res:      time.Minute,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.q.Encode(), func(t *testing.T) {
			gopts := graphOptsFromURL(test.q, maxDuration, &logger.Logger{})
			if test.gopts.endTime.IsZero() {
				if gopts.endTime.IsZero() || gopts.endTime.After(time.Now()) {
					t.Errorf("Default end time was not set up properly: now=%v, endTime=%v", time.Now(), gopts.endTime)
				}
				test.gopts.endTime = gopts.endTime
			}
			assert.Equal(t, test.gopts, gopts, "")
		})
	}
}
