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

package options

import (
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func testAggEM(ts time.Time, ep endpoint.Endpoint, app string, total, success int64, codes map[string]int64) *metrics.EventMetrics {
	respCodes := metrics.NewMap("code")
	for k, v := range codes {
		respCodes.IncKeyBy(k, v)
	}
	return metrics.NewEventMetrics(ts).
		AddMetric("total", metrics.NewInt(total)).
		AddMetric("success", metrics.NewInt(success)).
		AddMetric("resp-code", respCodes).
		AddMetric("version", metrics.NewString("v1")).
		AddLabel("probe", "p1").
		AddLabel("dst", ep.Dst()).
		AddLabel("app", app)
}

func TestTargetsAggregator(t *testing.T) {
	epA, epB, epC := endpoint.Endpoint{Name: "a"}, endpoint.Endpoint{Name: "b"}, endpoint.Endpoint{Name: "c"}

	activeEPs := []endpoint.Endpoint{epA, epB, epC}
	ta := newTargetsAggregator(func() []endpoint.Endpoint { return activeEPs }, time.Minute, nil)

	now := time.Now()
	ta.now = func() time.Time { return now }

	type wantAgg struct {
		total, success int64
		codes          string
	}
	verify := func(t *testing.T, em *metrics.EventMetrics, app string, want wantAgg) {
		t.Helper()
		assert.Equal(t, []string{"probe", "app"}, em.LabelsKeys())
		assert.Equal(t, app, em.Label("app"))
		assert.Equal(t, []string{"total", "success", "resp-code"}, em.MetricsKeys(), "string metric should be skipped")
		assert.Equal(t, want.total, em.Metric("total").(*metrics.Int).Int64())
		assert.Equal(t, want.success, em.Metric("success").(*metrics.Int).Int64())
		assert.Equal(t, want.codes, em.Metric("resp-code").String())
	}

	// Targets a and b are for app1, c for app2.
	verify(t, ta.record(epA, testAggEM(now, epA, "app1", 10, 9, map[string]int64{"200": 9})), "app1", wantAgg{10, 9, "map:code,200:9"})
	verify(t, ta.record(epB, testAggEM(now, epB, "app1", 5, 5, map[string]int64{"200": 5})), "app1", wantAgg{15, 14, "map:code,200:14"})
	verify(t, ta.record(epC, testAggEM(now, epC, "app2", 3, 3, map[string]int64{"200": 3})), "app2", wantAgg{3, 3, "map:code,200:3"})

	// Only deltas are added. New map keys are merged in.
	verify(t, ta.record(epA, testAggEM(now, epA, "app1", 20, 18, map[string]int64{"200": 18, "500": 2})), "app1", wantAgg{25, 23, "map:code,200:23,500:2"})

	// Target b goes away; its contribution stays. Target a is re-added, with
	// counters starting over, so its new values are added in full.
	activeEPs = []endpoint.Endpoint{epA, epC}
	verify(t, ta.record(epA, testAggEM(now, epA, "app1", 2, 2, map[string]int64{"200": 2})), "app1", wantAgg{27, 25, "map:code,200:25,500:2"})

	// Gauge metrics are passed through as is.
	gaugeEM := testAggEM(now, epA, "app1", 1, 1, nil)
	gaugeEM.Kind = metrics.GAUGE
	assert.Same(t, gaugeEM, ta.record(epA, gaugeEM))

	// Aggregate timestamp never goes back.
	em := ta.record(epA, testAggEM(now.Add(-time.Second), epA, "app1", 2, 2, map[string]int64{"200": 2}))
	assert.Equal(t, now, em.Timestamp)

	// After staleAfter, snapshot for b (not active anymore) is removed, but
	// c's snapshot stays even though it's stale, as c is still active.
	now = now.Add(2 * time.Minute)
	ta.record(epA, testAggEM(now, epA, "app1", 3, 3, map[string]int64{"200": 3}))
	assert.Len(t, ta.last, 2)
	assert.NotContains(t, ta.last, epB.Key()+"|"+labelsKey(testAggEM(now, epB, "app1", 0, 0, nil), ""))
	verify(t, ta.record(epC, testAggEM(now, epC, "app2", 4, 4, map[string]int64{"200": 4})), "app2", wantAgg{4, 4, "map:code,200:4"})

	// Once all targets for an aggregate are gone, aggregate is removed.
	activeEPs = []endpoint.Endpoint{epA}
	now = now.Add(2 * time.Minute)
	ta.record(epA, testAggEM(now, epA, "app1", 4, 4, map[string]int64{"200": 4}))
	assert.Len(t, ta.agg, 1)
	assert.Contains(t, ta.agg, "probe=p1,app=app1,")
}

func TestRecordMetricsAggregateAcrossTargets(t *testing.T) {
	p := &configpb.ProbeDef{
		Name: proto.String("p1"),
		Type: configpb.ProbeDef_HTTP.Enum(),
		Targets: &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_HostNames{HostNames: "a,b"},
		},
		AggregateAcrossTargets: proto.Bool(true),
	}
	opts, err := BuildProbeOptions(p, nil, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, opts.aggregator)
	assert.Equal(t, 5*opts.StatsExportInterval, opts.aggregator.staleAfter)

	dataChan := make(chan *metrics.EventMetrics, 2)
	for _, ep := range opts.Targets.ListEndpoints() {
		opts.RecordMetrics(ep, testAggEM(time.Now(), ep, "app1", 10, 9, nil), dataChan)
	}

	<-dataChan
	em := <-dataChan
	assert.Equal(t, "", em.Label("dst"))
	assert.Equal(t, int64(20), em.Metric("total").(*metrics.Int).Int64())
	assert.Equal(t, int64(18), em.Metric("success").(*metrics.Int).Int64())
}
