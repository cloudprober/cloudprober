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

package starlark

import (
	"context"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	payloadconfigpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	metricspb "github.com/cloudprober/cloudprober/metrics/proto"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/starlark/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// metricsAfterRun drives p.runProbe with a stable runReq, then returns the
// EMs the scheduler would export via result.Metrics. Tests that need to
// inspect payload metrics use this rather than Probe.RunOnce, which doesn't
// surface them to the caller.
func metricsAfterRun(t *testing.T, p *Probe, target string) []*metrics.EventMetrics {
	t.Helper()
	runReq := &sched.RunProbeForTargetRequest{
		Target:  endpoint.Endpoint{Name: target},
		LastRun: &sched.LastRunResult{},
	}
	p.runProbe(context.Background(), runReq)
	if runReq.LastRun.Error != nil {
		t.Fatalf("runProbe: %v", runReq.LastRun.Error)
	}
	return runReq.Result.Metrics(time.Now(), 1, p.opts)
}

// findMetric returns the EventMetrics carrying the named metric, or nil.
func findMetric(ems []*metrics.EventMetrics, name string) *metrics.EventMetrics {
	for _, em := range ems {
		if em.Metric(name) != nil {
			return em
		}
	}
	return nil
}

func TestPrintMetric_Basic(t *testing.T) {
	source := `
def probe(target):
    print_metric("items_in_cart 5")
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-print-metric", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	ems := metricsAfterRun(t, p, "example.com")
	em := findMetric(ems, "items_in_cart")
	if em == nil {
		t.Fatalf("items_in_cart not found in %d EMs", len(ems))
	}
	// metrics.ParseValueFromString picks Float for unsuffixed numbers.
	assert.Equal(t, "5.000", em.Metric("items_in_cart").String())
	assert.Equal(t, "starlark", em.Label("ptype"))
}

func TestPrintMetric_WithLabels(t *testing.T) {
	source := `
def probe(target):
    print_metric('items_in_cart{user="alice"} 5')
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-print-metric-labels", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	ems := metricsAfterRun(t, p, "example.com")
	em := findMetric(ems, "items_in_cart")
	if em == nil {
		t.Fatalf("items_in_cart not found")
	}
	assert.Equal(t, "alice", em.Label("user"))
}

// TestPrintMetric_Distribution exercises the dist_metric proto config:
// emitting multiple values for a name registered as a distribution puts them
// in a Distribution metric, not separate gauges. Same semantics as external /
// http payload distributions.
func TestPrintMetric_Distribution(t *testing.T) {
	source := `
def probe(target):
    print_metric("lat_ms 12")
    print_metric("lat_ms 27")
    print_metric("lat_ms 4")
`
	opts := newOpts(t, "example.com", source)
	opts.ProbeConf.(*configpb.ProbeConf).OutputMetricsOptions = &payloadconfigpb.OutputMetricsOptions{
		// Aggregate so the three observations land in one distribution
		// rather than three separate single-observation EMs.
		AggregateInCloudprober: proto.Bool(true),
		DistMetric: map[string]*metricspb.Dist{
			"lat_ms": {Buckets: &metricspb.Dist_ExplicitBuckets{ExplicitBuckets: "1,5,10,50,100"}},
		},
	}

	p := &Probe{}
	if err := p.Init("script-print-metric-dist", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	ems := metricsAfterRun(t, p, "example.com")
	// With aggregation, each emit returns a clone of the running
	// distribution. The last EM holds all three observations.
	var dist *metrics.Distribution
	for _, em := range ems {
		if d, ok := em.Metric("lat_ms").(*metrics.Distribution); ok {
			dist = d
		}
	}
	if dist == nil {
		t.Fatalf("lat_ms distribution not found in %d EMs", len(ems))
	}
	assert.Equal(t, int64(3), dist.Data().Count)
}

// TestPrintMetric_AggregationCumulative exercises CUMULATIVE +
// aggregate_in_cloudprober: print_metric across runs accumulates server-side
// rather than emitting independent EMs.
func TestPrintMetric_AggregationCumulative(t *testing.T) {
	source := `
def probe(target):
    print_metric("hits 1")
`
	opts := newOpts(t, "example.com", source)
	opts.ProbeConf.(*configpb.ProbeConf).OutputMetricsOptions = &payloadconfigpb.OutputMetricsOptions{
		MetricsKind:            payloadconfigpb.OutputMetricsOptions_CUMULATIVE.Enum(),
		AggregateInCloudprober: proto.Bool(true),
	}

	p := &Probe{}
	if err := p.Init("script-print-metric-cumul", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}

	runReq := &sched.RunProbeForTargetRequest{
		Target:  endpoint.Endpoint{Name: "example.com"},
		LastRun: &sched.LastRunResult{},
	}
	for i := 0; i < 3; i++ {
		p.runProbe(context.Background(), runReq)
		assert.NoError(t, runReq.LastRun.Error)
	}
	ems := runReq.Result.Metrics(time.Now(), 3, p.opts)
	// With aggregate_in_cloudprober, each run emits a fresh EM with the
	// running total. The last EM holds the full count.
	var last *metrics.EventMetrics
	for _, em := range ems {
		if em.Metric("hits") != nil {
			last = em
		}
	}
	if last == nil {
		t.Fatalf("hits not found")
	}
	// 3 runs × 1 each, aggregated.
	assert.Equal(t, "3.000", last.Metric("hits").String())
}

// TestPrintMetric_DefaultGauge confirms print_metric works with no proto
// config — payload.NewParser tolerates nil opts.
func TestPrintMetric_DefaultGauge(t *testing.T) {
	source := `
def probe(target):
    print_metric("temp 78.3")
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-print-metric-default", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	ems := metricsAfterRun(t, p, "example.com")
	em := findMetric(ems, "temp")
	if em == nil {
		t.Fatalf("temp not found")
	}
	if em.Kind != metrics.GAUGE {
		t.Errorf("expected GAUGE kind, got %v", em.Kind)
	}
}

// TestPrintMetric_SurvivesScriptError pins the streaming contract: a
// print_metric call before the script errors still produces an EM. The
// buffered approach used to drop these.
func TestPrintMetric_SurvivesScriptError(t *testing.T) {
	source := `
def probe(target):
    print_metric("attempts 1")
    fail("boom")
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	if err := p.Init("script-print-metric-error", opts); err != nil {
		t.Fatalf("Init: %v", err)
	}
	runReq := &sched.RunProbeForTargetRequest{
		Target:  endpoint.Endpoint{Name: "example.com"},
		LastRun: &sched.LastRunResult{},
	}
	p.runProbe(context.Background(), runReq)
	assert.Error(t, runReq.LastRun.Error, "script should have failed")

	ems := runReq.Result.Metrics(time.Now(), 1, p.opts)
	if findMetric(ems, "attempts") == nil {
		t.Fatalf("attempts metric should survive script error; got %d EMs", len(ems))
	}
}

// TestPrintMetric_AtModuleLevel pins that top-level print_metric calls land
// in a scratch buffer that's discarded after ExecFile, not silently retained.
func TestPrintMetric_AtModuleLevel(t *testing.T) {
	source := `
print_metric("module_load_line 1")

def probe(target):
    pass
`
	opts := newOpts(t, "example.com", source)
	p := &Probe{}
	err := p.Init("script-print-metric-toplevel", opts)
	assert.NoError(t, err)

	ems := metricsAfterRun(t, p, "example.com")
	assert.Nil(t, findMetric(ems, "module_load_line"), "module-load metric should not leak into the run")
}
