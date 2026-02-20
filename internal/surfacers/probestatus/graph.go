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
	"encoding/json"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
)

type graphOptions struct {
	endTime  time.Time
	duration time.Duration
	res      time.Duration
}

func graphResolution(endTime time.Time, td time.Duration) time.Duration {
	dataTime := endTime.Sub(sysvars.StartTime())
	if dataTime < td {
		td = dataTime
	}

	if td <= 12*time.Hour {
		return time.Minute
	}
	if td <= 72*time.Hour {
		return 5 * time.Minute
	}
	return 15 * time.Minute
}

func graphOptsFromURL(qv url.Values, maxDuration time.Duration, l *logger.Logger) *graphOptions {
	gopts := &graphOptions{
		endTime:  time.Now(),
		duration: maxDuration,
	}

	for key, vals := range qv {
		val := vals[len(vals)-1]
		switch key {
		case "graph_endtime":
			iv, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				l.Errorf("Error parsing graph_endtime value: %v", err)
				continue
			}
			gopts.endTime = time.Unix(iv, 0)
		case "graph_duration":
			dv, err := time.ParseDuration(val)
			if err != nil {
				l.Errorf("Error parsing graph_duration value: %v", err)
				continue
			}
			gopts.duration = dv
		case "graph_res":
			dv, err := time.ParseDuration(val)
			if err != nil {
				l.Errorf("Error parsing graph_res value: %v", err)
				continue
			}
			gopts.res = dv
		}
	}

	if gopts.res == 0 {
		gopts.res = graphResolution(gopts.endTime, gopts.duration)
	}

	return gopts
}

func (gd *graphData) syncGraphLines(startTime, endTime map[string]time.Time) {
	var minStartTime, maxEndTime time.Time
	var needLeftPadding, needRightPadding bool

	for _, t := range startTime {
		if minStartTime.IsZero() {
			minStartTime = t
			continue
		}
		if t != minStartTime {
			if t.Before(minStartTime) {
				minStartTime = t
			}
			needLeftPadding = true
		}
	}
	for _, t := range endTime {
		if maxEndTime.IsZero() {
			maxEndTime = t
			continue
		}
		if t != maxEndTime {
			if t.After(maxEndTime) {
				maxEndTime = t
			}
			needRightPadding = true
		}
	}

	if needLeftPadding {
		for tgt, t := range startTime {
			if t.After(minStartTime) {
				lpadding := int(t.Sub(minStartTime).Seconds()) / gd.ResSeconds
				gd.Values[tgt] = append([]float64{-1}, gd.Values[tgt]...)
				gd.Freqs[tgt] = append([]int64{int64(lpadding)}, gd.Freqs[tgt]...)
			}
		}
	}
	if needRightPadding {
		for tgt, t := range endTime {
			if t.Before(maxEndTime) {
				rpadding := int(maxEndTime.Sub(t).Seconds()) / gd.ResSeconds
				gd.Values[tgt] = append(gd.Values[tgt], -1)
				gd.Freqs[tgt] = append(gd.Freqs[tgt], int64(rpadding))
			}
		}
	}
	gd.StartTime = minStartTime.Unix()
	gd.EndTime = maxEndTime.Unix()
}

type graphPoints struct {
	startTime, endTime time.Time
	values             []float64
	freqs              []int64
}

func computeGraphPoints(baseTS *timeseries, gopts *graphOptions) *graphPoints {
	// Return nothing if duration is too small.
	if gopts.duration < baseTS.res {
		return nil
	}

	ts := baseTS.shallowCopy()

	gp := &graphPoints{endTime: gopts.endTime}

	// Truncate latest if endTime is before the current timeseries time.
	ts.l.Debugf("timeseries before any change: ts.oldest=%d, ts.latest=%d", ts.oldestIdx, ts.latestIdx)
	if gopts.endTime.Before(ts.currentTS) {
		ts.latestIdx = ts.agoIndex(int(ts.currentTS.Sub(gopts.endTime) / ts.res))
	} else {
		gp.endTime = ts.currentTS
	}
	ts.l.Debugf("timeseries after endTime truncated: ts.oldest=%d, ts.latest=%d", ts.oldestIdx, ts.latestIdx)

	if ts.latestIdx == ts.oldestIdx {
		return nil
	}

	// Let's move oldest now if needed.
	if int(gopts.duration/ts.res) < ts.size() {
		ts.oldestIdx = ts.agoIndex(int(gopts.duration / ts.res))
		gp.startTime = gp.endTime.Add(-gopts.duration)
	} else {
		gp.startTime = gp.endTime.Add(-time.Duration(ts.size()) * ts.res)
	}

	ts.l.Debugf("timeseries after startTime truncated: ts.oldest=%d, ts.latest=%d", ts.oldestIdx, ts.latestIdx)

	step := int(gopts.res / ts.res)
	if step == 0 {
		step = 1
	}
	// Note gd.ResCount will be 1, until we've a bunch of points.
	numValues := ts.size() / step
	var lastVal float64
	for i := numValues - 1; i >= 0; i = i - 1 {
		// Using agoIndex makes sure tts.latest won't jump beyond tts.oldest.
		if ts.latestIdx == ts.oldestIdx {
			break
		}

		currentD := ts.a[ts.latestIdx]
		ts.latestIdx = ts.agoIndex(step)
		lastD := ts.a[ts.latestIdx]

		val := float64(currentD.success-lastD.success) / float64(currentD.total-lastD.total)
		// For graphing purpose, we convert NaNs to -1. We ignore -1 while
		// drawing the graph.
		if math.IsNaN(val) {
			val = -1
		}
		if val == lastVal && len(gp.freqs) > 0 {
			gp.freqs[0]++
		} else {
			gp.values = append([]float64{val}, gp.values...)
			gp.freqs = append([]int64{1}, gp.freqs...)
			lastVal = val
		}
	}

	return gp
}

type graphData struct {
	StartTime, EndTime int64
	ResSeconds         int // Used by HTML template
	Values             map[string][]float64
	Freqs              map[string][]int64
}

func (gd *graphData) JSONBytes(l *logger.Logger) []byte {
	gdJSON, err := json.Marshal(gd)
	// In a bug free code, this should never happen.
	if err != nil {
		l.Errorf("Error converting graph data to JSON: %v", err)
		return nil
	}
	return gdJSON
}

func computeGraphData(metrics map[string]*timeseries, gopts *graphOptions) *graphData {
	gd := &graphData{
		Values:     make(map[string][]float64),
		Freqs:      make(map[string][]int64),
		ResSeconds: int(gopts.res.Seconds()),
	}

	startTimes, endTimes := make(map[string]time.Time), make(map[string]time.Time)

	for targetName, ts := range metrics {
		gp := computeGraphPoints(ts, gopts)
		if gp == nil {
			continue
		}
		startTimes[targetName], endTimes[targetName] = gp.startTime, gp.endTime
		gd.Values[targetName], gd.Freqs[targetName] = gp.values, gp.freqs
	}

	gd.syncGraphLines(startTimes, endTimes)

	//ps.l.Debugf("graphData[%s]: %s", probeName, string(gd.JSONBytes()))
	return gd
}
