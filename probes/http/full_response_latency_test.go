// Copyright 2026 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/internal/validators"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	configpb "github.com/cloudprober/cloudprober/probes/http/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestFullResponseLatency(t *testing.T) {
	const bodyDelay = 100 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(bodyDelay)
		fmt.Fprint(w, "response")
	}))
	defer server.Close()

	for _, unit := range []time.Duration{time.Millisecond, time.Second} {
		for _, histogram := range []bool{false, true} {
			t.Run(fmt.Sprintf("unit=%s/histogram=%t", unit, histogram), func(t *testing.T) {
				opts := options.DefaultOptions()
				opts.LatencyUnit = unit
				opts.LatencyMetricName = "duration"
				opts.ProbeConf = &configpb.ProbeConf{LatencyBreakdown: []configpb.ProbeConf_LatencyBreakdown{
					configpb.ProbeConf_RESPONSE_READ_LATENCY, configpb.ProbeConf_TOTAL_LATENCY,
				}}
				buckets := []float64{0, bodyDelay.Seconds() / (2 * unit.Seconds()), 10 / unit.Seconds()}
				if histogram {
					opts.LatencyDist = metrics.NewDistribution(buckets)
				}
				p := &Probe{}
				require.NoError(t, p.Init("full_response", opts))
				result := p.newResult()
				const requests = 3
				for i := 0; i < requests; i++ {
					req, err := http.NewRequest(http.MethodGet, server.URL, nil)
					require.NoError(t, err)
					require.NoError(t, p.doHTTPRequest(req, server.Client(), endpoint.Endpoint{Name: "test"}, result, nil))
				}
				assert.EqualValues(t, requests, result.success)
				em := result.Metrics(time.Now(), 0, opts)[0]
				sums := map[string]float64{}
				for _, name := range []string{"duration", "response_read_duration", "total_duration"} {
					value := em.Metric(name)
					require.NotNil(t, value)
					if histogram {
						data := value.(*metrics.Distribution).Data()
						assert.EqualValues(t, requests, data.Count)
						assert.Equal(t, append([]float64{math.Inf(-1)}, buckets...), data.LowerBounds)
						assert.EqualValues(t, requests, data.BucketCounts[0]+data.BucketCounts[1]+data.BucketCounts[2]+data.BucketCounts[3])
						if name != "duration" {
							assert.Zero(t, data.BucketCounts[1], "body delay must exceed the first nonnegative bucket")
						}
						sums[name] = data.Sum
					} else {
						sums[name] = value.(metrics.NumValue).Float64()
					}
				}
				assert.GreaterOrEqual(t, sums["response_read_duration"]*unit.Seconds(), requests*bodyDelay.Seconds()*0.9)
				assert.GreaterOrEqual(t, sums["total_duration"], sums["response_read_duration"]+sums["duration"])
				// Header latency must not acquire the deliberate body delay.
				assert.Less(t, sums["duration"], sums["response_read_duration"])
				assert.Nil(t, em.Metric("dns_duration"), "only selected breakdowns should be exported")
			})
		}
	}
}

func TestFullResponseLatencyFailures(t *testing.T) {
	for _, scenario := range []string{"body_timeout", "header_timeout", "truncated", "validation", "negative"} {
		for _, histogram := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/histogram=%t", scenario, histogram), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if scenario == "header_timeout" {
						<-r.Context().Done()
						return
					}
					if scenario == "truncated" {
						w.Header().Set("Content-Length", "100")
					}
					w.WriteHeader(http.StatusOK)
					w.(http.Flusher).Flush()
					if scenario == "body_timeout" {
						<-r.Context().Done()
						return
					}
					fmt.Fprint(w, "short")
				}))
				defer server.Close()
				opts := options.DefaultOptions()
				opts.ProbeConf = &configpb.ProbeConf{LatencyBreakdown: []configpb.ProbeConf_LatencyBreakdown{configpb.ProbeConf_ALL_STAGES}}
				opts.NegativeTest = scenario == "negative"
				if histogram {
					opts.LatencyDist = metrics.NewDistribution([]float64{0, 1, 10})
				}
				if scenario == "validation" {
					opts.Validators = []*validators.Validator{{Name: "reject", Validate: func(*validators.Input, *logger.Logger) (bool, error) {
						return false, nil
					}}}
				}
				p := &Probe{}
				require.NoError(t, p.Init("full_response", opts))
				result := p.newResult()
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
				require.NoError(t, err)
				err = p.doHTTPRequest(req, server.Client(), endpoint.Endpoint{Name: "test"}, result, nil)
				require.Error(t, err)
				if scenario == "body_timeout" || scenario == "header_timeout" {
					require.ErrorIs(t, err, context.DeadlineExceeded)
				} else if scenario == "truncated" {
					require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				}
				assert.EqualValues(t, 1, result.total)
				assert.Zero(t, result.success)
				// Preserve existing behavior: only Do errors increment timeouts.
				if scenario == "header_timeout" {
					assert.EqualValues(t, 1, result.timeouts)
				} else {
					assert.Zero(t, result.timeouts)
				}
				for _, value := range []metrics.LatencyValue{result.latency, result.latencyBreakdown.responseReadLatency, result.latencyBreakdown.totalLatency} {
					if histogram {
						assert.Zero(t, value.(*metrics.Distribution).Data().Count)
					} else {
						assert.Zero(t, value.(metrics.NumValue).Float64())
					}
				}
			})
		}
	}
}

func TestFullResponseLatencyExcludesValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "response")
	}))
	defer server.Close()
	opts := options.DefaultOptions()
	opts.LatencyUnit = time.Second
	opts.ProbeConf = &configpb.ProbeConf{LatencyBreakdown: []configpb.ProbeConf_LatencyBreakdown{configpb.ProbeConf_TOTAL_LATENCY}}
	var validationDuration time.Duration
	opts.Validators = []*validators.Validator{{Name: "slow", Validate: func(*validators.Input, *logger.Logger) (bool, error) {
		start := time.Now()
		time.Sleep(50 * time.Millisecond)
		validationDuration = time.Since(start)
		return true, nil
	}}}
	p := &Probe{}
	require.NoError(t, p.Init("full_response", opts))
	result := p.newResult()
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	start := time.Now()
	require.NoError(t, p.doHTTPRequest(req, server.Client(), endpoint.Endpoint{Name: "test"}, result, nil))
	elapsed := time.Since(start)
	total := result.latencyBreakdown.totalLatency.(metrics.NumValue).Float64()
	assert.LessOrEqual(t, total, (elapsed - validationDuration).Seconds())
	assert.Nil(t, result.latencyBreakdown.responseReadLatency)
}

func TestFullResponseLatencyConcurrentRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		time.Sleep(50 * time.Millisecond)
		fmt.Fprint(w, "response")
	}))
	defer server.Close()
	u := server.Listener.Addr().(*net.TCPAddr)
	opts := options.DefaultOptions()
	opts.ProbeConf = &configpb.ProbeConf{
		RequestsPerProbe: proto.Int32(3),
		LatencyBreakdown: []configpb.ProbeConf_LatencyBreakdown{configpb.ProbeConf_RESPONSE_READ_LATENCY, configpb.ProbeConf_TOTAL_LATENCY},
	}
	opts.LatencyDist = metrics.NewDistribution([]float64{0, 1, 1000})
	p := &Probe{}
	require.NoError(t, p.Init("full_response", opts))
	run := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: "127.0.0.1", Port: u.Port}}
	p.runProbe(context.Background(), run)
	result := run.Result.(*probeResult)
	assert.EqualValues(t, 3, result.success)
	for _, value := range []metrics.LatencyValue{result.latency, result.latencyBreakdown.responseReadLatency, result.latencyBreakdown.totalLatency} {
		assert.EqualValues(t, 3, value.(*metrics.Distribution).Data().Count)
	}
}
