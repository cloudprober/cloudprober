// Copyright 2026 The Cloudprober Authors.
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
	"html"
	"net/http/httptest"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/probestatus/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestRatioHealth(t *testing.T) {
	tests := []struct {
		total, success int64
		want           health
	}{
		{0, 0, healthNoData},
		{-1, -1, healthNoData},
		{12, 12, healthOK},
		{1000, 999, healthWarn},
		{100, 99, healthWarn},
		{1000, 989, healthBad},
		{12, 0, healthBad},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_of_%d", tt.success, tt.total), func(t *testing.T) {
			assert.Equal(t, tt.want, ratioHealth(tt.total, tt.success))
		})
	}
}

func newTestSurfacer() *Surfacer {
	return &Surfacer{
		c: &configpb.SurfacerConf{
			TimeseriesSize:     proto.Int32(100),
			MaxTargetsPerProbe: proto.Int32(20),
		},
		resolution:    time.Minute,
		dashDurations: []time.Duration{time.Minute},
		pageCache:     newPageCache(1),
		metrics:       make(map[string]map[string]*timeseries),
		probeTargets:  make(map[string][]string),
	}
}

// recordRatio records two data points, two minutes apart, so that the
// target's ratio over the summary window is success/total.
func recordRatio(t *testing.T, ps *Surfacer, probe, target string, total, success int) {
	t.Helper()
	now := time.Now()
	ps.record(testEM(t, now.Add(-2*time.Minute), probe, target, 1000, 1000, 0))
	ps.record(testEM(t, now, probe, target, 1000+total, 1000+success, 0))
}

func TestSummary(t *testing.T) {
	ps := newTestSurfacer()

	recordRatio(t, ps, "p_ok", "t1", 10, 10)
	recordRatio(t, ps, "p_ok", "t2", 10, 10)

	// A single data point gives no delta: the probe has no recent data.
	ps.record(testEM(t, time.Now(), "p_nodata", "t1", 10, 10, 0))

	recordRatio(t, ps, "p_warn", "t1", 10, 10)
	recordRatio(t, ps, "p_warn", "t2", 1000, 999)

	recordRatio(t, ps, "p_bad", "t1", 10, 10)
	recordRatio(t, ps, "p_bad", "t2", 10, 0)

	for i := range 5 {
		recordRatio(t, ps, "p_many", fmt.Sprintf("t%d", i), 10, 5)
	}

	got := ps.summary([]string{"p_ok", "p_nodata", "p_warn", "p_bad", "p_many", "p_unknown"}, summaryWindow)

	want := []*probeSummary{
		{Name: "p_bad", Health: "bad", OK: 1, Total: 2, Failing: []string{"t2"}},
		{Name: "p_many", Health: "bad", OK: 0, Total: 5, Failing: []string{"t0", "t1", "t2"}, MoreFailing: 2},
		{Name: "p_warn", Health: "warn", OK: 1, Total: 2, Failing: []string{"t2"}},
		{Name: "p_ok", Health: "ok", OK: 2, Total: 2},
		{Name: "p_nodata", Health: "nodata"},
	}
	assert.Equal(t, want, got)
}

func TestStatusPageSummary(t *testing.T) {
	ps := newTestSurfacer()

	target := `<b>t2</b>`
	recordRatio(t, ps, "p1", "t1", 10, 10)
	recordRatio(t, ps, "p1", target, 10, 0)
	ps.probeNames = []string{"p1"}

	hw := &httpWriter{
		w: httptest.NewRecorder(),
		r: httptest.NewRequest("", "/", nil),
	}
	ps.writeData(hw)
	got := hw.w.(*httptest.ResponseRecorder).Body.String()

	assert.Contains(t, got, "<h3>Summary (last 5m)</h3>")
	assert.Contains(t, got, `<span class="health-dot health-bad" title="bad"></span>`)
	assert.Contains(t, got, `<a href="#probe-p1">p1</a>`)
	assert.Contains(t, got, "1/2 targets OK")
	assert.Contains(t, got, `<td>`+html.EscapeString(target)+`</td>`)
	assert.Contains(t, got, `<b id="probe-p1">Probe: p1</b>`)
}
