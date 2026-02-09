// Copyright 2022-2023 The Cloudprober Authors.
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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/probestatus/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers/options"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func testEM(t *testing.T, tm time.Time, probe, target string, total, success int, latency float64) *metrics.EventMetrics {
	t.Helper()
	return metrics.NewEventMetrics(tm).
		AddLabel("probe", probe).
		AddLabel("dst", target).
		AddMetric("total", metrics.NewInt(int64(total))).
		AddMetric("success", metrics.NewInt(int64(success))).
		AddMetric("latency", metrics.NewFloat(latency))
}

func TestNewAndRecord(t *testing.T) {
	state.SetDefaultHTTPServeMux(http.NewServeMux())
	defer state.SetDefaultHTTPServeMux(nil)

	total, success := 200, 180
	latency := float64(2000)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ps, _ := New(ctx, &configpb.SurfacerConf{
		TimeseriesSize:     proto.Int32(10),
		MaxTargetsPerProbe: proto.Int32(2),
	}, &options.Options{}, nil)

	probeTargets := map[string][]string{
		"p":  {"t1", "t2"},
		"p2": {"t1"},
		"p3": {"t1", "t2", "t3"},
	}

	for probe, targets := range probeTargets {
		for _, target := range targets {
			ps.record(testEM(t, time.Now(), probe, target, total, success, latency))
		}
	}

	for probe, targets := range probeTargets {
		// For p3 one target will be dropped due to capacity limit.
		if probe == "p3" {
			targets = targets[:len(targets)-1]
		}
		for _, target := range targets {
			if ps.metrics[probe][target] == nil {
				t.Errorf("Unexpected nil timeseries for: probe(%s), target(%s)", probe, target)
			}
		}
	}
}

func TestPageCache(t *testing.T) {
	pc := newPageCache(1)

	c, valid := pc.contentIfValid("test-url")
	if valid {
		t.Errorf("Got valid content from new cache: %s", string(c))
	}

	testContent := map[string][]byte{
		"url1": []byte("test-content1"),
		"url2": []byte("test-content2"),
	}
	for url, c := range testContent {
		pc.setContent(url, c)
	}

	for url, tc := range testContent {
		c, valid = pc.contentIfValid(url)
		assert.True(t, valid, "Got unexpected invalid")
		assert.Equal(t, string(tc), string(c))
	}

	time.Sleep(time.Second)
	for url := range testContent {
		_, valid = pc.contentIfValid(url)
		assert.False(t, valid, "Got unexpected valid")
	}
}

func TestDisabledAndHandlers(t *testing.T) {
	tests := []struct {
		desc         string
		disabled     bool
		url          string
		patternMatch map[string]string
	}{
		{
			desc:     "disabled",
			disabled: true,
			patternMatch: map[string]string{
				"/status": "",
			},
		},
		{
			desc: "default",
			patternMatch: map[string]string{
				"/":        "/",
				"/status2": "/",
				"/status":  "/status",
			},
		},
		{
			desc: "different_url",
			url:  "/status2",
			patternMatch: map[string]string{
				"/status":  "/", // Default HTTP Handler
				"/status2": "/status2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancelFunc := context.WithCancel(context.Background())
			defer cancelFunc()

			httpServerMux := http.NewServeMux()
			state.SetDefaultHTTPServeMux(httpServerMux)

			conf := &configpb.SurfacerConf{
				Disable: proto.Bool(test.disabled),
			}
			if test.url != "" {
				conf.Url = proto.String(test.url)
			}
			ps, err := New(ctx, conf, &options.Options{}, nil)
			if err != nil {
				t.Fatalf("Error creating surfacer: %v", err)
			}

			if test.disabled {
				if ps != nil {
					t.Errorf("surfacer should be nil if it's disabled")
				}
				// Verify no panic on writing to a nil surfacer.
				ps.Write(ctx, &metrics.EventMetrics{Timestamp: time.Now()})
			}

			for url, pattern := range test.patternMatch {
				_, matchedPattern := httpServerMux.Handler(httptest.NewRequest("", url, nil))
				assert.Equal(t, pattern, matchedPattern)
			}
		})
	}
}

func TestQueryStatus(t *testing.T) {
	state.SetDefaultHTTPServeMux(http.NewServeMux())
	defer state.SetDefaultHTTPServeMux(nil)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	ps, err := New(ctx, &configpb.SurfacerConf{
		TimeseriesSize:     proto.Int32(20),
		MaxTargetsPerProbe: proto.Int32(10),
	}, &options.Options{}, nil)
	if err != nil {
		t.Fatalf("Error creating surfacer: %v", err)
	}

	// Record data across multiple minutes for probe "p1" with targets "t1"
	// and "t2", and probe "p2" with target "t3".
	baseTime := time.Now().Add(-5 * time.Minute).Truncate(time.Minute)

	// Simulate cumulative counters growing over 5 minutes.
	for i := 0; i < 5; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Minute)
		ps.record(testEM(t, ts, "p1", "t1", (i+1)*100, (i+1)*90, 0))
		ps.record(testEM(t, ts, "p1", "t2", (i+1)*50, (i+1)*45, 0))
		ps.record(testEM(t, ts, "p2", "t3", (i+1)*200, (i+1)*180, 0))
	}

	t.Run("all_probes_aggregated", func(t *testing.T) {
		results, err := ps.QueryStatus(ctx, nil, 60, false)
		assert.NoError(t, err)
		assert.Len(t, results, 2) // p1 and p2

		// Find p1 and p2 in results
		var p1, p2 *ProbeStatusData
		for _, r := range results {
			switch r.Name {
			case "p1":
				p1 = r
			case "p2":
				p2 = r
			}
		}

		assert.NotNil(t, p1)
		assert.Len(t, p1.TargetStatuses, 2) // t1 and t2

		assert.NotNil(t, p2)
		assert.Len(t, p2.TargetStatuses, 1)
		assert.Equal(t, "t3", p2.TargetStatuses[0].TargetName)

		// Verify no minute data when not requested
		for _, ts := range p1.TargetStatuses {
			assert.Nil(t, ts.MinuteData)
		}
	})

	t.Run("filter_by_probe_name", func(t *testing.T) {
		results, err := ps.QueryStatus(ctx, []string{"p1"}, 60, false)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "p1", results[0].Name)
	})

	t.Run("nonexistent_probe", func(t *testing.T) {
		results, err := ps.QueryStatus(ctx, []string{"nonexistent"}, 60, false)
		assert.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("per_minute_breakdown", func(t *testing.T) {
		results, err := ps.QueryStatus(ctx, []string{"p1"}, 60, true)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		for _, ts := range results[0].TargetStatuses {
			assert.NotNil(t, ts.MinuteData)
			assert.Greater(t, len(ts.MinuteData), 0, "expected per-minute data for target %s", ts.TargetName)
		}
	})

	t.Run("nil_surfacer", func(t *testing.T) {
		var nilPS *Surfacer
		results, err := nilPS.QueryStatus(ctx, nil, 60, false)
		assert.NoError(t, err)
		assert.Nil(t, results)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := ps.QueryStatus(canceledCtx, nil, 60, false)
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestSurfacerWriteData(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		wantContains    []string
		wantNotContains []string
		probeNames      []string
	}{
		{
			name: "default",
			wantContains: []string{
				"startTime", "allProbes", "chart",
			},
			probeNames: []string{"Probe: p1", "Probe: p2"},
		},
		{
			name:  "probe_filter",
			query: "probe=p1",
			wantContains: []string{
				"startTime", "allProbes", "chart",
			},
			wantNotContains: []string{"Probe: p2"},
			probeNames:      []string{"Probe: p1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &Surfacer{
				pageCache:  newPageCache(1),
				probeNames: tt.probeNames,
			}
			hw := &httpWriter{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("", "/", nil),
			}
			if tt.query != "" {
				hw.r.URL.RawQuery = tt.query
				t.Logf("Setting query: %v", hw.r.URL.Query())
			}
			ps.writeData(hw)
			got := hw.w.(*httptest.ResponseRecorder).Body.String()
			for _, keyword := range append(tt.wantContains, tt.probeNames...) {
				assert.Contains(t, got, keyword)
			}

			for _, keyword := range tt.wantNotContains {
				assert.NotContains(t, got, keyword)
			}
		})
	}
}
