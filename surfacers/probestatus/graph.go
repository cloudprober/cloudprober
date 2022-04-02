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
	"time"

	"github.com/cloudprober/cloudprober/sysvars"
)

func graphResolution(endTime time.Time, td time.Duration) time.Duration {
	dataTime := endTime.Sub(sysvars.StartTime())
	if dataTime < td {
		td = dataTime
	}

	if td <= 6*time.Hour {
		return time.Minute
	}
	if td <= 24*time.Hour {
		return 5 * time.Minute
	}
	return 15 * time.Minute
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

func computeGraphPoints(baseTS *timeseries, endTime time.Time, td, res time.Duration) *graphPoints {
	// Return nothing if duration is too small.
	if td < baseTS.res {
		return nil
	}

	ts := baseTS.shallowCopy()

	gp := &graphPoints{endTime: endTime}

	// Truncate latest if endTime is before the current timeseries time.
	ts.l.Debugf("timeseries before any change: ts.oldest=%d, ts.latest=%d", ts.oldest, ts.latest)
	if endTime.Before(ts.currentTS) {
		ts.latest = ts.agoIndex(int(ts.currentTS.Sub(endTime) / ts.res))
	} else {
		gp.endTime = ts.currentTS
	}
	ts.l.Debugf("timeseries after endTime truncated: ts.oldest=%d, ts.latest=%d", ts.oldest, ts.latest)

	if ts.latest == ts.oldest {
		return nil
	}

	// Let's move oldest now if needed.
	if int(td/ts.res) < ts.size() {
		ts.oldest = ts.agoIndex(int(td / ts.res))
		gp.startTime = gp.endTime.Add(-td)
	} else {
		gp.startTime = gp.endTime.Add(-time.Duration(ts.size()) * ts.res)
	}

	ts.l.Debugf("timeseries after startTime truncated: ts.oldest=%d, ts.latest=%d", ts.oldest, ts.latest)

	step := int(res / ts.res)
	if step == 0 {
		step = 1
	}
	// Note gd.ResCount will be 1, until we've a bunch of points.
	numValues := ts.size() / step
	var lastVal float64
	for i := numValues - 1; i >= 0; i = i - 1 {
		// Using agoIndex makes sure tts.latest won't jump beyond tts.oldest.
		if ts.latest == ts.oldest {
			break
		}

		currentD := ts.a[ts.latest]
		ts.latest = ts.agoIndex(step)
		lastD := ts.a[ts.latest]

		val := float64(currentD.success-lastD.success) / float64(currentD.total-lastD.total)
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

func (gd *graphData) JSONBytes() []byte {
	gdJSON, _ := json.Marshal(gd)
	return gdJSON
}

func computeGraphData(metrics map[string]*timeseries, endTime time.Time, td time.Duration) *graphData {
	gd := &graphData{
		Values:     make(map[string][]float64),
		Freqs:      make(map[string][]int64),
		ResSeconds: int(graphResolution(endTime, td).Seconds()),
	}

	startTimes, endTimes := make(map[string]time.Time), make(map[string]time.Time)

	for targetName, ts := range metrics {
		gp := computeGraphPoints(ts, endTime, td, time.Duration(gd.ResSeconds)*time.Second)
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
