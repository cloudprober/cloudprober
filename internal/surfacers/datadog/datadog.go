// Copyright 2021 The Cloudprober Authors.
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
Package datadog implements a surfacer to export metrics to Datadog.
*/
package datadog

import (
	"context"
	"fmt"
	"os"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/datadog/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/options"
	"google.golang.org/protobuf/proto"
)

/*
	The datadog surfacer presents EventMetrics to the datadog SubmitMetrics APIs,
	using the config passed in.

	Some EventMetrics are not supported here, as the datadog SubmitMetrics API only
	supports float64 type values as the metric value.
*/

var datadogKind = map[metrics.Kind]string{
	metrics.GAUGE:      "gauge",
	metrics.CUMULATIVE: "count",
}

// DDSurfacer implements a datadog surfacer for datadog metrics.
type DDSurfacer struct {
	c         *configpb.SurfacerConf
	opts      *options.Options
	writeChan chan *metrics.EventMetrics
	client    *ddClient
	l         *logger.Logger
	prefix    string

	// A cache of []*ddSeries, used for batch writing to datadog
	ddSeriesCache []ddSeries
}

// New creates a new instance of a datadog surfacer, based on the config passed in. It then hands off
// to a goroutine to surface metrics to datadog across a buffered channel.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*DDSurfacer, error) {
	if config.GetApiKey() != "" {
		os.Setenv("DD_API_KEY", config.GetApiKey())
	}
	if config.GetAppKey() != "" {
		os.Setenv("DD_APP_KEY", config.GetAppKey())
	}

	p := config.GetPrefix()
	if p[len(p)-1] != '.' {
		p += "."
	}

	dd := &DDSurfacer{
		c:             config,
		writeChan:     make(chan *metrics.EventMetrics, config.GetMetricsBatchSize()),
		client:        newClient(config.GetServer(), config.GetApiKey(), config.GetAppKey(), config.GetDisableCompression()),
		l:             l,
		prefix:        p,
		ddSeriesCache: make([]ddSeries, 0, config.GetMetricsBatchSize()),
	}

	go dd.receiveMetricsFromEvent(ctx)

	dd.l.Info("Initialised Datadog surfacer")
	return dd, nil
}

// Write is a function defined to comply with the surfacer interface, and enables the
// datadog surfacer to receive EventMetrics over the buffered channel.
func (dd *DDSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case dd.writeChan <- em:
	default:
		dd.l.Error("Surfacer's write channel is full, dropping new data.")
	}
}

func (dd *DDSurfacer) receiveMetricsFromEvent(ctx context.Context) {
	publishTimer := time.NewTicker(time.Duration(dd.c.GetBatchTimerSec()) * time.Second)
	defer publishTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			dd.l.Infof("Context canceled, stopping the surfacer write loop")
			return
		case em := <-dd.writeChan:
			dd.recordEventMetrics(ctx, publishTimer, em)
		case <-publishTimer.C:
			if len(dd.ddSeriesCache) != 0 {
				dd.publishMetrics(ctx)
			}
		}
	}
}

func recordMapValue[T int64 | float64](dd *DDSurfacer, m *metrics.Map[T], baseTags []string, key string, em *metrics.EventMetrics) []ddSeries {
	var series []ddSeries
	for _, k := range m.Keys() {
		series = append(series, dd.newDDSeries(key, float64(m.GetKey(k)), append(baseTags, fmt.Sprintf("%s:%s", m.MapName, k)), em.Timestamp, em.Kind))
	}
	return series
}

func (dd *DDSurfacer) recordEventMetrics(ctx context.Context, publishTimer *time.Ticker, em *metrics.EventMetrics) {
	for _, metricKey := range em.MetricsKeys() {
		if !dd.opts.AllowMetric(metricKey) {
			continue
		}

		var series []ddSeries
		switch value := em.Metric(metricKey).(type) {
		case metrics.NumValue:
			series = []ddSeries{dd.newDDSeries(metricKey, value.Float64(), emLabelsToTags(em), em.Timestamp, em.Kind)}
		case *metrics.Map[int64]:
			series = recordMapValue(dd, value, emLabelsToTags(em), metricKey, em)
		case *metrics.Map[float64]:
			series = recordMapValue(dd, value, emLabelsToTags(em), metricKey, em)
		case *metrics.Distribution:
			series = dd.distToDDSeries(value.Data(), metricKey, emLabelsToTags(em), em.Timestamp, em.Kind)
		}
		dd.addMetricsAndPublish(ctx, publishTimer, series...)
	}
}

// publish the metrics to datadog, buffering as necessary
func (dd *DDSurfacer) addMetricsAndPublish(ctx context.Context, publishTimer *time.Ticker, series ...ddSeries) {
	for i := range series {
		if len(dd.ddSeriesCache) >= int(dd.c.GetMetricsBatchSize()) {
			dd.publishMetrics(ctx)
			publishTimer.Reset(time.Duration(dd.c.GetBatchTimerSec()) * time.Second)
		}

		dd.ddSeriesCache = append(dd.ddSeriesCache, series[i])
	}
}

func (dd *DDSurfacer) publishMetrics(ctx context.Context) {
	if err := dd.client.submitMetrics(ctx, dd.ddSeriesCache); err != nil {
		dd.l.Errorf("Failed to publish %d series to datadog: %v", len(dd.ddSeriesCache), err)
	}

	dd.ddSeriesCache = dd.ddSeriesCache[:0]
}

// Create a new datadog series using the values passed in.
func (dd *DDSurfacer) newDDSeries(metricName string, value float64, tags []string, timestamp time.Time, kind metrics.Kind) ddSeries {
	return ddSeries{
		Metric: dd.prefix + metricName,
		Points: [][]float64{{float64(timestamp.Unix()), value}},
		Tags:   &tags,
		Type:   proto.String(datadogKind[kind]),
	}
}

// Take metric labels from an event metric and parse them into a Datadog Dimension struct.
func emLabelsToTags(em *metrics.EventMetrics) []string {
	tags := []string{}

	for _, k := range em.LabelsKeys() {
		tags = append(tags, fmt.Sprintf("%s:%s", k, em.Label(k)))
	}

	return tags
}

func (dd *DDSurfacer) distToDDSeries(d *metrics.DistributionData, metricName string, tags []string, t time.Time, kind metrics.Kind) []ddSeries {
	ret := []ddSeries{
		{
			Metric: dd.prefix + metricName + ".sum",
			Points: [][]float64{{float64(t.Unix()), d.Sum}},
			Tags:   &tags,
			Type:   proto.String(datadogKind[kind]),
		}, {
			Metric: dd.prefix + metricName + ".count",
			Points: [][]float64{{float64(t.Unix()), float64(d.Count)}},
			Tags:   &tags,
			Type:   proto.String(datadogKind[kind]),
		},
	}

	// Add one point at the value of the Lower Bound per count in the bucket. Each point represents the
	// minimum poissible value that it could have been.
	var points [][]float64
	for i := range d.LowerBounds {
		for n := 0; n < int(d.BucketCounts[i]); n++ {
			points = append(points, []float64{float64(t.Unix()), d.LowerBounds[i]})
		}
	}

	ret = append(ret, ddSeries{Metric: dd.prefix + metricName, Points: points, Tags: &tags, Type: proto.String(datadogKind[kind])})
	return ret
}
