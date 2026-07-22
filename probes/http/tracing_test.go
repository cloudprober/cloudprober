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

package http

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	proberconfigpb "github.com/cloudprober/cloudprober/config/proto"
	tracingpb "github.com/cloudprober/cloudprober/internal/tracing/proto"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/protobuf/proto"
)

func setupTestTracing(t *testing.T) (*tracetest.InMemoryExporter, *sdktrace.TracerProvider) {
	t.Helper()

	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))

	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	return exp, tp
}

func runTracedProbe(t *testing.T, tracingConfig *tracingpb.TracingConfig) (gotTraceparent string, spans tracetest.SpanStubs) {
	t.Helper()

	exp, tp := setupTestTracing(t)

	var mu sync.Mutex
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotTraceparent = r.Header.Get("Traceparent")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	_, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)

	opts := options.DefaultOptions()
	opts.Targets = targets.StaticTargets("localhost")
	opts.Interval = 10 * time.Second
	opts.Timeout = time.Second
	opts.ProbeConf = &configpb.ProbeConf{Port: proto.Int32(int32(port))}
	if tracingConfig != nil {
		opts.ProberConfig = &proberconfigpb.ProberConfig{Tracing: tracingConfig}
	}

	p := &Probe{}
	if err := p.Init("tracing_test", opts); err != nil {
		t.Fatalf("probe init error: %v", err)
	}
	p.tracerProvider = tp
	p.propagators = propagation.TraceContext{}

	runReq := &sched.RunProbeForTargetRequest{Target: p.targets[0]}
	p.runProbe(context.Background(), runReq)

	result := runReq.Result.(*probeResult)
	if result.success != 1 {
		t.Fatalf("probe did not succeed: success=%d, total=%d", result.success, result.total)
	}

	return gotTraceparent, exp.GetSpans()
}

func TestProbeTracingEnabled(t *testing.T) {
	gotTraceparent, spans := runTracedProbe(t, &tracingpb.TracingConfig{})

	assert.NotEmpty(t, gotTraceparent, "expected traceparent header to be propagated to target")

	// Find the client span produced by otelhttp; the connection-level spans
	// added by otelhttptrace (e.g. http.connect, http.send) are children.
	var clientSpan *tracetest.SpanStub
	var connSpans int
	for i := range spans {
		if spans[i].Name == "GET /" {
			clientSpan = &spans[i]
		} else {
			connSpans++
		}
	}

	if assert.NotNil(t, clientSpan, "expected a client span named %q", "GET /") {
		var probeAttr string
		for _, kv := range clientSpan.Attributes {
			if kv.Key == "probe" {
				probeAttr = kv.Value.AsString()
			}
		}
		assert.Equal(t, "tracing_test", probeAttr)
	}

	// otelhttptrace should add connection-level sub-spans to the trace.
	assert.NotZero(t, connSpans, "expected connection-level spans from otelhttptrace")
}

func TestProbeTracingDisabled(t *testing.T) {
	gotTraceparent, spans := runTracedProbe(t, nil)

	assert.Empty(t, gotTraceparent, "did not expect traceparent header when tracing is disabled")
	assert.Empty(t, spans, "did not expect any exported spans when tracing is disabled")
}

func TestProbeTracingZeroSamplingFraction(t *testing.T) {
	// A sampling_fraction of 0 means every request is unsampled, so the
	// transport shouldn't be wrapped or inject trace context headers.
	gotTraceparent, spans := runTracedProbe(t, &tracingpb.TracingConfig{
		SamplingFraction: proto.Float64(0),
	})

	assert.Empty(t, gotTraceparent, "did not expect traceparent header when sampling_fraction is 0")
	assert.Empty(t, spans, "did not expect any exported spans when sampling_fraction is 0")
}

func TestProbeTracingSDKDisabled(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true")

	gotTraceparent, spans := runTracedProbe(t, &tracingpb.TracingConfig{})

	assert.Empty(t, gotTraceparent, "did not expect traceparent header when OTEL_SDK_DISABLED=true")
	assert.Empty(t, spans, "did not expect any exported spans when OTEL_SDK_DISABLED=true")
}
