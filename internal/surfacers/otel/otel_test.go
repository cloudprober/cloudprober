// Copyright 2023 The Cloudprober Authors.
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

package otel

import (
	"context"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/otel/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/protobuf/proto"
)

func testEMs(ts time.Time) []*metrics.EventMetrics {
	d := metrics.NewDistribution([]float64{1, 10, 100})
	d.AddSample(2)
	d.AddSample(20)

	respCode := metrics.NewMap("code")
	respCode.IncKeyBy("200", 2)
	respCode.IncKey("500")

	ems := []*metrics.EventMetrics{
		metrics.NewEventMetrics(ts).
			AddMetric("failures", metrics.NewInt(20)).
			AddMetric("latency", metrics.NewFloat(9.2)).
			AddLabel("probe", "p1"),
		metrics.NewEventMetrics(ts.Add(time.Second)).
			AddMetric("resp_code", respCode).
			AddMetric("latency", d).
			AddLabel("probe", "p2"),
		metrics.NewEventMetrics(ts.Add(2*time.Second)).
			AddMetric("failures", metrics.NewInt(25)).
			AddMetric("latency", metrics.NewFloat(8.7)).
			AddMetric("dns_latency", metrics.NewFloat(1.2)).
			AddLabel("probe", "p1"),
	}

	ems[1].LatencyUnit = time.Millisecond
	return ems
}

func testMetric(name, unit string, data metricdata.Aggregation) metricdata.Metrics {
	return metricdata.Metrics{
		Name:        name,
		Description: name + " metric for Cloudprober",
		Unit:        unit,
		Data:        data,
	}
}

func dataPoint[T int64 | float64](val T, kvs [][2]string, startTime, ts time.Time) metricdata.DataPoint[T] {
	var attrs []attribute.KeyValue
	for _, kv := range kvs {
		attrs = append(attrs, attribute.String(kv[0], kv[1]))
	}
	return metricdata.DataPoint[T]{
		Attributes: attribute.NewSet(attrs...),
		StartTime:  startTime,
		Time:       ts,
		Value:      val,
	}
}

func sumData[T int64 | float64](dataPoints ...metricdata.DataPoint[T]) metricdata.Sum[T] {
	return metricdata.Sum[T]{
		DataPoints:  dataPoints,
		Temporality: metricdata.CumulativeTemporality,
		IsMonotonic: true,
	}
}

func TestOtelSurfacerWrite(t *testing.T) {
	startTime := time.Now()
	ems := testEMs(startTime.Add(time.Second))

	tests := []struct {
		name             string
		em               *metrics.EventMetrics
		wantScopeMetrics map[string]*metricdata.ScopeMetrics
	}{
		{
			name: "em0",
			em:   ems[0],
			wantScopeMetrics: map[string]*metricdata.ScopeMetrics{
				"probe.p1": {
					Scope: instrumentation.Scope{Name: "probe.p1"},
					Metrics: []metricdata.Metrics{
						testMetric("cloudprober_failures", "1", sumData(dataPoint[int64](20, [][2]string{{"probe", "p1"}}, startTime, ems[0].Timestamp))),
						testMetric("cloudprober_latency", "us", sumData(dataPoint[float64](9.2, [][2]string{{"probe", "p1"}}, startTime, ems[0].Timestamp))),
					},
				},
			},
		},
		{
			name: "em1",
			em:   ems[1],
			wantScopeMetrics: map[string]*metricdata.ScopeMetrics{
				"probe.p2": {
					Scope: instrumentation.Scope{Name: "probe.p2"},
					Metrics: []metricdata.Metrics{
						testMetric("cloudprober_resp_code", "1", sumData(
							dataPoint[int64](2, [][2]string{{"probe", "p2"}, {"code", "200"}}, startTime, ems[1].Timestamp),
							dataPoint[int64](1, [][2]string{{"probe", "p2"}, {"code", "500"}}, startTime, ems[1].Timestamp))),
						testMetric("cloudprober_latency", "ms", metricdata.Histogram[float64]{
							DataPoints: []metricdata.HistogramDataPoint[float64]{
								{
									Attributes:   attribute.NewSet(attribute.String("probe", "p2")),
									StartTime:    startTime,
									Time:         ems[1].Timestamp,
									Count:        2,
									Sum:          22,
									Bounds:       []float64{1, 10, 100},
									BucketCounts: []uint64{0, 1, 1, 0},
								},
							},
							Temporality: metricdata.CumulativeTemporality,
						}),
					},
				},
			},
		},
		{
			name: "em2",
			em:   ems[2],
			wantScopeMetrics: map[string]*metricdata.ScopeMetrics{
				"probe.p1": {
					Scope: instrumentation.Scope{Name: "probe.p1"},
					Metrics: []metricdata.Metrics{
						testMetric("cloudprober_failures", "1", sumData(dataPoint[int64](20, [][2]string{{"probe", "p1"}}, startTime, ems[0].Timestamp))),
						testMetric("cloudprober_latency", "us", sumData(dataPoint[float64](9.2, [][2]string{{"probe", "p1"}}, startTime, ems[0].Timestamp))),
						testMetric("cloudprober_failures", "1", sumData(dataPoint[int64](25, [][2]string{{"probe", "p1"}}, startTime, ems[0].Timestamp.Add(2*time.Second)))),
						testMetric("cloudprober_latency", "us", sumData(dataPoint[float64](8.7, [][2]string{{"probe", "p1"}}, startTime, ems[0].Timestamp.Add(2*time.Second)))),
						testMetric("cloudprober_dns_latency", "us", sumData(dataPoint[float64](1.2, [][2]string{{"probe", "p1"}}, startTime, ems[0].Timestamp.Add(2*time.Second)))),
					},
				},
			},
		},
	}

	wantScopeMetrics := make(map[string]*metricdata.ScopeMetrics)
	os := &OtelSurfacer{
		startTime:    startTime,
		scopeMetrics: make(map[string]*metricdata.ScopeMetrics),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Write(context.Background(), tt.em)
			for k, v := range tt.wantScopeMetrics {
				wantScopeMetrics[k] = v
			}
			assert.Equal(t, wantScopeMetrics, os.scopeMetrics)
		})
	}

	// First produce now will return all scope metrics
	var wantProducedMetrics []metricdata.ScopeMetrics
	for _, scope := range []string{"probe.p1", "probe.p2"} {
		wantProducedMetrics = append(wantProducedMetrics, *wantScopeMetrics[scope])
	}
	producedMetrics, _ := os.Produce(context.Background())
	assert.Equal(t, wantProducedMetrics, producedMetrics)

	// Next produce will be empty
	producedMetrics, _ = os.Produce(context.Background())
	assert.Empty(t, producedMetrics)
}

func TestGetExporterType(t *testing.T) {
	tests := []struct {
		name     string
		config   *configpb.SurfacerConf
		wantType metric.Exporter
		wantErr  bool
	}{
		{
			name: "otlp_http",
			config: &configpb.SurfacerConf{
				Exporter: &configpb.SurfacerConf_OtlpHttpExporter{
					OtlpHttpExporter: &configpb.HTTPExporter{},
				},
			},
			wantType: &otlpmetrichttp.Exporter{},
		},
		{
			name: "otlp_grpc",
			config: &configpb.SurfacerConf{
				Exporter: &configpb.SurfacerConf_OtlpGrpcExporter{
					OtlpGrpcExporter: &configpb.GRPCExporter{
						Endpoint: proto.String("localhost:1234"),
					},
				},
			},
			wantType: &otlpmetricgrpc.Exporter{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getExporter(context.Background(), tt.config, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("getExporter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.IsType(t, tt.wantType, got, "unexpected exporter type")
		})
	}
}
