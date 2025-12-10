// Copyright 2017-2023 The Cloudprober Authors.
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

package stackdriver

import (
	"context"
	"net/http"
	"testing"
	"time"

	surfacerpb "github.com/cloudprober/cloudprober/internal/surfacers/proto"
	configpb "github.com/cloudprober/cloudprober/internal/surfacers/stackdriver/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/options"
	"github.com/stretchr/testify/assert"
	monitoring "google.golang.org/api/monitoring/v3"
	"google.golang.org/protobuf/proto"
)

func newTestSurfacer() SDSurfacer {
	return SDSurfacer{
		cache:       make(map[string]*monitoring.TimeSeries),
		onGCE:       true,
		projectName: "test-project",
		resource: &monitoring.MonitoredResource{
			Type: "gce_instance",
			Labels: map[string]string{
				"instance_id": "test-instance",
				"zone":        "us-central1-a",
			},
		},
	}
}

func TestBaseMetric(t *testing.T) {
	s := newTestSurfacer()
	s.c = &configpb.SurfacerConf{
		MetricsPrefix: configpb.SurfacerConf_PTYPE_PROBE.Enum(),
		MonitoringUrl: proto.String("custom.googleapis.com/cloudprober/"),
	}
	testTimestamp := time.Now()
	testProbe := "test_probe"
	testPtype := "external"

	tests := []struct {
		description        string
		metricPrefixConfig *configpb.SurfacerConf_MetricPrefix
		em                 *metrics.EventMetrics
		wantKeys           string
		wantMetricPrefix   string
	}{
		{
			description:        "metrics prefix with ptype and probe",
			metricPrefixConfig: configpb.SurfacerConf_PTYPE_PROBE.Enum(),
			em: metrics.NewEventMetrics(testTimestamp).
				AddMetric("test_metric", metrics.NewString("metval")).
				AddLabel("keyA", "valueA").
				AddLabel("keyB", "valueB").
				AddLabel("probe", testProbe).
				AddLabel("ptype", testPtype),
			wantKeys:         "keyA=valueA,keyB=valueB",
			wantMetricPrefix: "external/test_probe/",
		},
		{
			description:        "metrics prefix with only probe",
			metricPrefixConfig: configpb.SurfacerConf_PROBE.Enum(),
			em: metrics.NewEventMetrics(testTimestamp).
				AddMetric("test_metric", metrics.NewString("metval")).
				AddLabel("keyA", "valueA").
				AddLabel("keyB", "valueB").
				AddLabel("probe", testProbe).
				AddLabel("ptype", testPtype),
			wantKeys:         "keyA=valueA,keyB=valueB,ptype=external",
			wantMetricPrefix: "test_probe/",
		},
		{
			description:        "metrics prefix with none",
			metricPrefixConfig: configpb.SurfacerConf_NONE.Enum(),
			em: metrics.NewEventMetrics(testTimestamp).
				AddMetric("test_metric", metrics.NewString("metval")).
				AddLabel("keyA", "valueA").
				AddLabel("keyB", "valueB").
				AddLabel("probe", testProbe).
				AddLabel("ptype", testPtype),
			wantKeys:         "keyA=valueA,keyB=valueB,probe=test_probe,ptype=external",
			wantMetricPrefix: "",
		},
	}

	for _, tt := range tests {
		s.c = &configpb.SurfacerConf{
			MetricsPrefix: tt.metricPrefixConfig,
		}
		bm, metricPrefix := s.baseMetric(tt.em)
		assert.Equal(t, tt.wantKeys, bm.cacheKey)
		assert.Equal(t, tt.wantMetricPrefix, metricPrefix)
	}
}

func TestTimeSeries(t *testing.T) {
	t.Helper()

	testTimestamp := time.Now()

	// Generate a time series and check that it is correct
	s := newTestSurfacer()
	s.opts = options.BuildOptionsForTest(&surfacerpb.SurfacerDef{
		IgnoreMetricsWithName: proto.String(".*sysvars_ec2_name"),
	})

	tests := []struct {
		description   string
		labels        [][2]string
		metricName    string
		metricValue   metrics.Value
		latencyUnit   time.Duration
		tsValue       []float64
		tsUnit        []string
		tsExtraLabels [][2]string
		wantCacheKeys []string
	}{
		{
			description: "int64-metric",
			metricName:  "success",
			metricValue: metrics.NewInt(123456),
			tsValue:     []float64{123456},
		},
		{
			description:   "ignore-metric",
			metricName:    "sysvars_ec2_name",
			metricValue:   metrics.NewString("xyz"),
			tsValue:       []float64{},
			wantCacheKeys: []string{"-"}, // want nothing
		},
		{
			description: "int64-metric-2",
			metricName:  "success",
			metricValue: metrics.NewInt(123456),
			tsValue:     []float64{123456},
		},
		{
			description: "float-metric",
			metricName:  "latency",
			metricValue: metrics.NewFloat(1.176),
			latencyUnit: time.Millisecond,
			tsValue:     []float64{1.176},
			tsUnit:      []string{"ms"},
		},
		{
			description:   "string-value",
			metricName:    "version",
			metricValue:   metrics.NewString("versionXX"),
			labels:        [][2]string{{"keyA", "valueA"}, {"keyB", "valueB"}},
			tsValue:       []float64{1},
			tsExtraLabels: [][2]string{{"val", "versionXX"}},
			wantCacheKeys: []string{"version,keyA=valueA,keyB=valueB"},
		},
		{
			description:   "int64-map",
			metricName:    "resp-code",
			metricValue:   metrics.NewMap("code").IncKeyBy("200", 98).IncKeyBy("500", 2),
			labels:        [][2]string{{"keyC", "valueC"}},
			tsValue:       []float64{98, 2},
			tsExtraLabels: [][2]string{{"code", "200"}, {"code", "500"}},
			wantCacheKeys: []string{"resp-code,keyC=valueC,code=200", "resp-code,keyC=valueC,code=500"},
		},
		{
			description:   "float64-map",
			metricName:    "app-latency",
			metricValue:   metrics.NewMapFloat("percentile").IncKeyBy("p95", 0.05).IncKeyBy("p99", 0.9),
			labels:        [][2]string{{"keyD", "valueD"}},
			tsValue:       []float64{0.05, 0.9},
			tsExtraLabels: [][2]string{{"percentile", "p95"}, {"percentile", "p99"}},
			wantCacheKeys: []string{"app-latency,keyD=valueD,percentile=p95", "app-latency,keyD=valueD,percentile=p99"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			em := metrics.NewEventMetrics(testTimestamp).
				AddMetric(tt.metricName, tt.metricValue)
			for _, l := range tt.labels {
				em.AddLabel(l[0], l[1])
			}
			if tt.latencyUnit != 0 {
				em.LatencyUnit = tt.latencyUnit
			}

			var wantTimeSeries []*monitoring.TimeSeries

			for i, v := range tt.tsValue {
				f := float64(v)
				labelsMap := make(map[string]string)
				for _, label := range tt.labels {
					labelsMap[label[0]] = label[1]
				}
				if tt.tsExtraLabels != nil {
					labelsMap[tt.tsExtraLabels[i][0]] = tt.tsExtraLabels[i][1]
				}

				unit := "1"
				if tt.tsUnit != nil {
					unit = tt.tsUnit[i]
				}

				wantTimeSeries = append(wantTimeSeries, &monitoring.TimeSeries{
					Metric: &monitoring.Metric{
						Type:   "custom.googleapis.com/cloudprober/" + tt.metricName,
						Labels: labelsMap,
					},
					Resource: &monitoring.MonitoredResource{
						Type: "gce_instance",
						Labels: map[string]string{
							"instance_id": "test-instance",
							"zone":        "us-central1-a",
						},
					},
					MetricKind: "CUMULATIVE",
					ValueType:  "DOUBLE",
					Unit:       unit,
					Points: []*monitoring.Point{
						{
							Interval: &monitoring.TimeInterval{
								StartTime: "0001-01-01T00:00:00Z",
								EndTime:   em.Timestamp.Format(time.RFC3339Nano),
							},
							Value: &monitoring.TypedValue{
								DoubleValue: &f,
							},
						},
					},
				})
			}

			gotTimeSeries := s.recordEventMetrics(em)

			if len(tt.wantCacheKeys) == 0 {
				tt.wantCacheKeys = []string{tt.metricName + ","}
			}

			assert.Equal(t, wantTimeSeries, gotTimeSeries, "Got timeseries")
			for i, wantTS := range wantTimeSeries {
				assert.Equal(t, wantTS, s.cache[tt.wantCacheKeys[i]], "Cache timeseries")
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name                    string
		config                  *configpb.SurfacerConf
		wantAllowedMetricsRegex string
	}{
		{
			name: "surfacer with default config",
		},
		{
			name: "surfacer with a custom config",
			config: &configpb.SurfacerConf{
				AllowedMetricsRegex: proto.String("total|success"),
			},
			wantAllowedMetricsRegex: "total|success",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			s, err := New(ctx, tt.config, &options.Options{}, http.DefaultClient, nil)
			assert.NoError(t, err, "New() error = %v", err)
			assert.LessOrEqual(t, time.Since(s.startTime), time.Minute, "surfacer start time is too far in the past")

			if tt.wantAllowedMetricsRegex == "" {
				assert.Nil(t, s.allowedMetricsRegex, "allowed metrics regex")
			} else {
				assert.Equal(t, tt.wantAllowedMetricsRegex, s.allowedMetricsRegex.String(), "allowed metrics regex")
			}
		})
	}
}
