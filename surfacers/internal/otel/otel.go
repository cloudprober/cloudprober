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

/*
Package otel provides an OpenTelemetry surfacer for Cloudprober. It can
export metrics to OpenTelemetry Collector over gRPC or HTTP.

This surfacer holds the incoming EventMetrics in memory and periodically
(default: 10s) exports them the configured HTTP or gRPC endpoint.
*/
package otel

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/cloudprober/cloudprober/common/tlsconfig"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/internal/common/options"
	configpb "github.com/cloudprober/cloudprober/surfacers/internal/otel/proto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"
)

// OtelSurfacer implements OpenTelemetry surfacer for Cloudprober.
type OtelSurfacer struct {
	c    *configpb.SurfacerConf // Configuration
	opts *options.Options
	l    *logger.Logger

	// Otel metrics
	mu           sync.Mutex
	scopeMetrics map[string]*metricdata.ScopeMetrics

	startTime time.Time
}

func getExporter(ctx context.Context, config *configpb.SurfacerConf, l *logger.Logger) (metric.Exporter, error) {
	if config.GetOtlpHttpExporter() != nil {
		expConf := config.GetOtlpHttpExporter()

		var opts []otlpmetrichttp.Option

		if expConf.GetEndpointUrl() != "" {
			u, err := url.Parse(expConf.GetEndpointUrl())
			if err != nil {
				return nil, fmt.Errorf("invalid http endpoint_url: %s, err: %v", expConf.GetEndpointUrl(), err)
			}

			opts = append(opts, otlpmetrichttp.WithEndpoint(net.JoinHostPort(u.Hostname(), u.Port())))
			if u.Scheme != "https" {
				opts = append(opts, otlpmetrichttp.WithInsecure())
			}
			opts = append(opts, otlpmetrichttp.WithURLPath(u.Path))
		}

		if expConf.GetHttpHeader() != nil {
			opts = append(opts, otlpmetrichttp.WithHeaders(expConf.GetHttpHeader()))
		}

		if expConf.GetCompression() == configpb.Compression_GZIP {
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
		}

		if expConf.GetTlsConfig() != nil {
			tlsConfig := &tls.Config{}
			err := tlsconfig.UpdateTLSConfig(nil, expConf.GetTlsConfig())
			if err != nil {
				return nil, fmt.Errorf("failed to create tls config: %v", err)
			}
			opts = append(opts, otlpmetrichttp.WithTLSClientConfig(tlsConfig))
		}

		exp, err := otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create http exporter: %v", err)
		}
		return exp, nil
	}

	if config.GetOtlpGrpcExporter() != nil {
		expConf := config.GetOtlpGrpcExporter()

		var opts []otlpmetricgrpc.Option

		if expConf.GetEndpoint() != "" {
			opts = append(opts, otlpmetricgrpc.WithEndpoint(expConf.GetEndpoint()))
		}

		if expConf.GetHttpHeader() != nil {
			opts = append(opts, otlpmetricgrpc.WithHeaders(expConf.GetHttpHeader()))
		}

		if expConf.GetCompression() == configpb.Compression_GZIP {
			opts = append(opts, otlpmetricgrpc.WithCompressor("gzip"))
		}

		if expConf.GetInsecure() {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}

		if expConf.GetTlsConfig() != nil {
			tlsConfig := &tls.Config{}
			err := tlsconfig.UpdateTLSConfig(nil, expConf.GetTlsConfig())
			if err != nil {
				return nil, fmt.Errorf("failed to create tls config: %v", err)
			}
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
		}

		exp, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create grpc exporter: %v", err)
		}
		return exp, nil
	}

	l.Warning("No exporter specified, using stdout exporter")
	return stdoutmetric.New()
}

// New returns a prometheus surfacer based on the config provided. It sets up a
// goroutine to process both the incoming EventMetrics and the web requests for
// the URL handler /metrics.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*OtelSurfacer, error) {
	if config == nil {
		config = &configpb.SurfacerConf{}
	}

	os := &OtelSurfacer{
		c:            config,
		opts:         opts,
		scopeMetrics: make(map[string]*metricdata.ScopeMetrics),
		startTime:    time.Now(),
		l:            l,
	}

	exp, err := getExporter(ctx, config, l)
	if err != nil {
		return nil, err
	}

	// Reader is sort of a binding between exporter and producer. It collects
	// from the producer and exports to the exporter.
	exportInterval := time.Second * time.Duration(config.GetExportIntervalSec())
	r := metric.NewPeriodicReader(exp, metric.WithProducer(os), metric.WithInterval(exportInterval))

	var attrKVs []attribute.KeyValue
	for _, attr := range config.GetResourceAttribute() {
		attrKVs = append(attrKVs, attribute.String(attr.GetKey(), attr.GetValue()))
	}
	res, err := resource.New(ctx, resource.WithHost(), resource.WithFromEnv(), resource.WithAttributes(attrKVs...))
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %v", err)
	}

	// This step registers the reader and pipelines behind the scene.
	// manugarg: This step seems kind of unnecessary right now but it's
	// required.
	metric.NewMeterProvider(metric.WithReader(r), metric.WithResource(res))

	l.Infof("Initialized opentelemetry surfacer with config: %s", config.String())
	return os, nil
}

func (os *OtelSurfacer) Produce(_ context.Context) ([]metricdata.ScopeMetrics, error) {
	os.mu.Lock()
	defer os.mu.Unlock()

	var scopeMetrics []metricdata.ScopeMetrics

	var keys []string
	for k := range os.scopeMetrics {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sm := os.scopeMetrics[k]
		if len(sm.Metrics) == 0 {
			continue
		}
		scopeMetrics = append(scopeMetrics, *sm)
		sm.Metrics = nil
	}

	return scopeMetrics, nil
}

func mapDataPoints[T int64 | float64](baseAttrs attribute.Set, m *metrics.Map[T], startTime, ts time.Time) []metricdata.DataPoint[T] {
	var dataPoints []metricdata.DataPoint[T]
	for _, k := range m.Keys() {
		attrs := attribute.NewSet(append(baseAttrs.ToSlice(), attribute.String(m.MapName, k))...)
		dataPoints = append(dataPoints, metricdata.DataPoint[T]{
			Attributes: attrs,
			StartTime:  startTime,
			Time:       ts,
			Value:      m.GetKey(k),
		})
	}
	return dataPoints
}

func convertDistribution(dist *metrics.Distribution, kind metrics.Kind, baseAttrs attribute.Set, startTime, ts time.Time) metricdata.Histogram[float64] {
	d := dist.Data()

	hdp := metricdata.HistogramDataPoint[float64]{
		Attributes:   baseAttrs,
		StartTime:    startTime,
		Time:         ts,
		Count:        uint64(d.Count),
		Sum:          d.Sum,
		Bounds:       append([]float64{}, d.LowerBounds[1:]...),
		BucketCounts: make([]uint64, len(d.BucketCounts)),
	}

	for i := range d.BucketCounts {
		hdp.BucketCounts[i] = uint64(d.BucketCounts[i])
	}

	hist := metricdata.Histogram[float64]{
		DataPoints: []metricdata.HistogramDataPoint[float64]{hdp},
	}

	if kind == metrics.GAUGE {
		hist.Temporality = metricdata.DeltaTemporality
	} else {
		hist.Temporality = metricdata.CumulativeTemporality
	}

	return hist
}

func otelAttributes(em *metrics.EventMetrics) attribute.Set {
	var attrs []attribute.KeyValue
	for _, k := range em.LabelsKeys() {
		attrs = append(attrs, attribute.String(k, em.Label(k)))
	}
	return attribute.NewSet(attrs...)
}

func sumOrGauge[T int64 | float64](kind metrics.Kind, dataPoints ...metricdata.DataPoint[T]) metricdata.Aggregation {
	if kind == metrics.GAUGE {
		return metricdata.Gauge[T]{DataPoints: dataPoints}
	}
	return metricdata.Sum[T]{
		Temporality: metricdata.CumulativeTemporality,
		DataPoints:  dataPoints,
		IsMonotonic: true,
	}
}

func numberData[T int64 | float64](kind metrics.Kind, v T, attrs attribute.Set, startTime, timestamp time.Time) metricdata.Aggregation {
	return sumOrGauge[T](kind, metricdata.DataPoint[T]{
		Attributes: attrs,
		StartTime:  startTime,
		Time:       timestamp,
		Value:      v,
	})
}

func (os *OtelSurfacer) convertMetric(em *metrics.EventMetrics, metricName string) (metricdata.Metrics, error) {
	baseAttrs := otelAttributes(em)

	unit := "1"
	if os.opts.IsLatencyMetric(metricName) {
		unit = metrics.LatencyUnitToString(em.LatencyUnit)
		os.l.Debugf("Latency metric (%s) unit: %s", metricName, unit)
	}

	otelmetrics := func(data metricdata.Aggregation) metricdata.Metrics {
		return metricdata.Metrics{
			Name:        os.c.GetMetricsPrefix() + metricName,
			Description: os.c.GetMetricsPrefix() + metricName + " metric for Cloudprober",
			Unit:        unit,
			Data:        data,
		}
	}

	switch v := em.Metric(metricName).(type) {
	case *metrics.Int:
		return otelmetrics(numberData[int64](em.Kind, v.Int64(), baseAttrs, os.startTime, em.Timestamp)), nil
	case *metrics.Float:
		return otelmetrics(numberData[float64](em.Kind, v.Float64(), baseAttrs, os.startTime, em.Timestamp)), nil
	case *metrics.Map[int64]:
		return otelmetrics(sumOrGauge[int64](em.Kind, mapDataPoints[int64](baseAttrs, v, os.startTime, em.Timestamp)...)), nil
	case *metrics.Map[float64]:
		return otelmetrics(sumOrGauge[float64](em.Kind, mapDataPoints[float64](baseAttrs, v, os.startTime, em.Timestamp)...)), nil
	case metrics.String:
		attrs := attribute.NewSet(append(baseAttrs.ToSlice(), attribute.String("value", v.String()))...)
		return otelmetrics(numberData[int64](em.Kind, 1, attrs, os.startTime, em.Timestamp)), nil
	case *metrics.Distribution:
		return otelmetrics(convertDistribution(v, em.Kind, baseAttrs, os.startTime, em.Timestamp)), nil
	}

	return metricdata.Metrics{}, fmt.Errorf("unsupported metric type: %T", em.Metric(metricName))
}

func getScope(em *metrics.EventMetrics) string {
	if em.Label("probe") != "" {
		return "probe." + em.Label("probe")
	}
	if em.Label("module") != "" {
		return "module." + em.Label("module")
	}
	return "global"
}

// record processes the incoming EventMetrics and updates the in-memory
// otel metrics database.
func (os *OtelSurfacer) Write(_ context.Context, em *metrics.EventMetrics) {
	os.mu.Lock()
	defer os.mu.Unlock()

	scope := getScope(em)
	sm := os.scopeMetrics[scope]
	if sm == nil {
		sm = &metricdata.ScopeMetrics{
			Scope: instrumentation.Scope{
				Name: scope,
			},
		}
		os.scopeMetrics[scope] = sm
	}

	for _, metricName := range em.MetricsKeys() {
		if !os.opts.AllowMetric(metricName) {
			continue
		}

		otelmetrics, err := os.convertMetric(em, metricName)
		if err != nil {
			os.l.Errorf("Error converting metric: %s, err: %v", metricName, err)
			continue
		}
		sm.Metrics = append(sm.Metrics, otelmetrics)
	}
}
