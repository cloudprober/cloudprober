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
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/metrics/testutils"
	dnsconfigpb "github.com/cloudprober/cloudprober/probes/dns/proto"
	httpconfigpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	tcpconfigpb "github.com/cloudprober/cloudprober/probes/tcp/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type testProbeResult struct {
	total int
}

func (tpr *testProbeResult) Metrics(ts time.Time, _ int64, opts *options.Options) []*metrics.EventMetrics {
	return []*metrics.EventMetrics{metrics.NewEventMetrics(ts).AddMetric("total", metrics.NewInt(int64(tpr.total)))}
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
		Logger:              &logger.Logger{},
	}

	s := &Scheduler{
		Opts:              opts,
		DataChan:          make(chan *metrics.EventMetrics, 100),
		NewResult:         func(_ *endpoint.Endpoint) ProbeResult { return &testProbeResult{} },
		RunProbeForTarget: func(ctx context.Context, runReq *RunProbeForTargetRequest) { runReq.Result.(*testProbeResult).total++ },
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

func TestRunProbeForTargetTimeout(t *testing.T) {
	testTargets := [2]string{"test1.com", "test2.com"}

	opts := &options.Options{
		Targets:  targets.StaticTargets(strings.Join(testTargets[:], ",")),
		Interval: 10 * time.Millisecond,
		Timeout:  5 * time.Millisecond,
	}

	s := &Scheduler{
		Opts:      opts,
		DataChan:  make(chan *metrics.EventMetrics, 100),
		NewResult: func(_ *endpoint.Endpoint) ProbeResult { return &testProbeResult{} },
		RunProbeForTarget: func(ctx context.Context, runReq *RunProbeForTargetRequest) {
			select {
			case <-ctx.Done():
				if ctx.Err() != context.DeadlineExceeded {
					if ctx.Err() == context.Canceled {
						t.Log("Probe canceled by the test")
						return
					}
					t.Errorf("Probe did not timeout as expected")
				}
			case <-time.After(50 * time.Millisecond):
				t.Errorf("Probe did not timeout as expected")
			}
		},
	}
	s.init()

	ctx, cancelF := context.WithCancel(context.Background())

	s.refreshTargets(ctx)
	// Run the probe for a short duration to trigger the timeout.
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, len(testTargets), len(s.cancelFuncs), "len(s.cancelFunc)=%d, want=%d", len(s.cancelFuncs), len(testTargets))

	cancelF()
	s.Wait()
}

func TestSchedulerGapBetweenTargets(t *testing.T) {
	testTargets := []endpoint.Endpoint{{Name: "test1.com"}, {Name: "test2.com"}}
	tests := []struct {
		name string
		opts *options.Options
		want time.Duration
	}{
		{
			name: "default",
			opts: &options.Options{
				Interval: 10 * time.Second,
			},
			want: 500 * time.Millisecond,
		},
		{
			name: "tcp probe",
			opts: &options.Options{
				Interval: 10 * time.Second,
				ProbeConf: &tcpconfigpb.ProbeConf{
					IntervalBetweenTargetsMsec: proto.Int32(1000),
				},
			},
			want: 1 * time.Second,
		},
		{
			name: "http probe",
			opts: &options.Options{
				Interval: 10 * time.Second,
				ProbeConf: &httpconfigpb.ProbeConf{
					IntervalBetweenTargetsMsec: proto.Int32(2000),
				},
			},
			want: 2 * time.Second,
		},
		{
			name: "dns probe doesn't have interval between targets",
			opts: &options.Options{
				Interval:  10 * time.Second,
				ProbeConf: &dnsconfigpb.ProbeConf{},
			},
			want: 500 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Scheduler{
				Opts:    tt.opts,
				targets: testTargets,
			}
			if got := s.gapBetweenTargets(); got != tt.want {
				t.Errorf("Scheduler.gapBetweenTargets() = %v, want %v", got, tt.want)
			}
		})
	}
}
