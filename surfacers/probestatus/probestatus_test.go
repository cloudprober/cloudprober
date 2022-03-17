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
	"context"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/surfacers/probestatus/proto"
	"google.golang.org/protobuf/proto"
)

const (
	testProbe = "test-probe"
)

func testEM(t *testing.T, tm time.Time, probe, target string, total, success int, latency float64) *metrics.EventMetrics {
	t.Helper()
	return metrics.NewEventMetrics(tm).
		AddLabel("probe", probe).
		AddLabel("dst", target).
		AddMetric("total", metrics.NewInt(int64(total))).
		AddMetric("success", metrics.NewInt(int64(total))).
		AddMetric("latency", metrics.NewFloat(latency))
}

func TestNewAndRecord(t *testing.T) {
	total, success := 200, 180
	latency := float64(2000)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ps := New(ctx, &configpb.SurfacerConf{
		TimeseriesSize:     proto.Int32(10),
		MaxTargetsPerProbe: proto.Int32(2),
	}, nil, nil)

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
