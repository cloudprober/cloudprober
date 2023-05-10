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

package sched

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/testutils"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

type testProbeResult struct {
	total int
}

func (tpr *testProbeResult) Metrics(ts time.Time, opts *options.Options) *metrics.EventMetrics {
	return metrics.NewEventMetrics(ts).AddMetric("total", metrics.NewInt(int64(tpr.total)))
}

func compareNumberOfMetrics(t *testing.T, ems []*metrics.EventMetrics, metricName string, targets [2]string, wantCloseRange bool) {
	t.Helper()

	mmap := testutils.MetricsMapByTarget(ems).Filter(metricName)
	num1 := len(mmap[targets[0]])
	num2 := len(mmap[targets[1]])

	diff := num1 - num2
	threshold := num1 / 2
	notCloseRange := diff < -(threshold) || diff > threshold

	if notCloseRange && wantCloseRange {
		t.Errorf("Number of metrics for two targets are not within a close range (%d, %d)", num1, num2)
	}
	if !notCloseRange && !wantCloseRange {
		t.Errorf("Number of metrics for two targets are within a close range (%d, %d)", num1, num2)
	}

	// Let's check the last value of the metric and make sure that it's greater
	// than (if stats_export_interval > interval), or equal to the number of
	// metrics received. Note: This test assumes that metric value is
	// incremented in each run.
	for _, mvs := range mmap {
		numMetrics := len(mvs)
		lastVal := int(mvs[numMetrics-1].(metrics.NumValue).Int64())
		if lastVal < numMetrics {
			t.Errorf("Metric (%s) last value: %d, less than: %d", metricName, lastVal, numMetrics)
		}
	}
}

func TestUpdateTargetsAndStartProbes(t *testing.T) {
	testTargets := [2]string{"test1.com", "test2.com"}

	opts := &options.Options{
		Targets:             targets.StaticTargets(fmt.Sprintf("%s,%s", testTargets[0], testTargets[1])),
		Interval:            10 * time.Millisecond,
		StatsExportInterval: 20 * time.Millisecond,
		LogMetrics:          func(_ *metrics.EventMetrics) {},
		Logger:              &logger.Logger{},
	}

	s := &Scheduler{
		Opts:              opts,
		DataChan:          make(chan *metrics.EventMetrics, 100),
		NewResult:         func() ProbeResult { return &testProbeResult{} },
		RunProbeForTarget: func(ctx context.Context, ep endpoint.Endpoint, r ProbeResult) { r.(*testProbeResult).total++ },
	}
	s.init()

	ctx, cancelF := context.WithCancel(context.Background())
	s.refreshTargets(ctx)
	if len(s.cancelFuncs) != 2 {
		t.Errorf("len(s.cancelFunc)=%d, want=2", len(s.cancelFuncs))
	}
	ems, _ := testutils.MetricsFromChannel(s.DataChan, 100, time.Second)
	wantCloseRange := true
	compareNumberOfMetrics(t, ems, "total", testTargets, wantCloseRange)

	// Updates targets to just one target. This should cause one probe loop to
	// exit. We should get only one data stream after that.
	opts.Targets = targets.StaticTargets(testTargets[0])
	s.refreshTargets(ctx)
	if len(s.cancelFuncs) != 1 {
		t.Errorf("len(s.cancelFunc)=%d, want=1", len(s.cancelFuncs))
	}
	ems, _ = testutils.MetricsFromChannel(s.DataChan, 100, time.Second)
	wantCloseRange = false
	compareNumberOfMetrics(t, ems, "total", testTargets, wantCloseRange)

	cancelF()
	s.Wait()
}
