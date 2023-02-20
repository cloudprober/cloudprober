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
Package cloudwatch implements a surfacer to export metrics to AWS Cloudwatch.
*/
package cloudwatch

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/sysvars"

	configpb "github.com/cloudprober/cloudprober/surfacers/cloudwatch/proto"
	"github.com/cloudprober/cloudprober/surfacers/common/options"
)

// The dimension named used to identify distributions
const distributionDimensionName string = "le"

// CWSurfacer implements AWS Cloudwatch surfacer.
type CWSurfacer struct {
	c         *configpb.SurfacerConf
	opts      *options.Options
	writeChan chan *metrics.EventMetrics
	session   *cloudwatch.Client
	l         *logger.Logger

	// A cache of []types.MetricDatum's, used for batch writing to the
	// cloudwatch api.
	metricDatumCache       []types.MetricDatum
	metricDatumCacheTicker *time.Ticker
}

// New creates a new instance of a cloudwatch surfacer, based on the config
// passed in. It then hands off to a goroutine to surface metrics to cloudwatch
// across a buffered channel.
func New(ctx context.Context, conf *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*CWSurfacer, error) {
	region := getRegion(conf)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	cw := &CWSurfacer{
		c:                      conf,
		opts:                   opts,
		writeChan:              make(chan *metrics.EventMetrics, opts.MetricsBufferSize), // incoming internal metrics buffer
		session:                cloudwatch.NewFromConfig(cfg),
		l:                      l,
		metricDatumCache:       make([]types.MetricDatum, 0, *conf.MetricsBufferSize), // buffer between cloudprober and cloudwatch apis
		metricDatumCacheTicker: time.NewTicker(time.Duration(*conf.MetricsPublishTimerSec) * time.Second),
	}

	go cw.processIncomingMetrics(ctx)

	cw.l.Info("Initialised Cloudwatch surfacer")
	return cw, nil
}

// Write is a function defined to comply with the surfacer interface, and enables the
// cloudwatch surfacer to receive EventMetrics over the buffered channel.
func (cw *CWSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case cw.writeChan <- em:
	default:
		cw.l.Error("Surfacer's write channel is full, dropping new data.")
	}
}

func (cw *CWSurfacer) processIncomingMetrics(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			cw.l.Infof("Context canceled, stopping the surfacer write loop")
			return
		case em := <-cw.writeChan:
			cw.recordEventMetrics(ctx, em)
		case <-cw.metricDatumCacheTicker.C:
			cw.publishMetrics(ctx)
		}
	}
}

// recordEventMetrics takes an EventMetric, which can contain multiple metrics
// of varying types, and loops through each metric in the EventMetric, parsing
// each metric into a structure that is supported by Cloudwatch
func (cw *CWSurfacer) recordEventMetrics(ctx context.Context, em *metrics.EventMetrics) {
	for _, metricKey := range em.MetricsKeys() {
		if !cw.opts.AllowMetric(metricKey) {
			continue
		}

		switch value := em.Metric(metricKey).(type) {
		case metrics.NumValue:
			cw.addMetricAndPublish(ctx, cw.newCWMetricDatum(metricKey, value.Float64(), emLabelsToDimensions(em), em.Timestamp, em.LatencyUnit))

		case *metrics.Map:
			for _, mapKey := range value.Keys() {
				dimensions := emLabelsToDimensions(em)
				dimensions = append(dimensions, types.Dimension{
					Name:  aws.String(value.MapName),
					Value: aws.String(mapKey),
				})
				cw.addMetricAndPublish(ctx, cw.newCWMetricDatum(metricKey, value.GetKey(mapKey).Float64(), dimensions, em.Timestamp, em.LatencyUnit))
			}

		case *metrics.Distribution:
			for i, distributionBound := range value.Data().LowerBounds {
				dimensions := append(emLabelsToDimensions(em), types.Dimension{
					Name:  aws.String(distributionDimensionName),
					Value: aws.String(strconv.FormatFloat(distributionBound, 'f', -1, 64)),
				})

				cw.addMetricAndPublish(ctx, cw.newCWMetricDatum(metricKey, float64(value.Data().BucketCounts[i]), dimensions, em.Timestamp, em.LatencyUnit))
			}
		}
	}
}

// Add the metric to the local buffer, and if the buffer is full, publish the
// metrics to cloudwatch
func (cw *CWSurfacer) addMetricAndPublish(ctx context.Context, md types.MetricDatum) {
	cw.metricDatumCache = append(cw.metricDatumCache, md)
	if len(cw.metricDatumCache) == int(*cw.c.MetricsBufferSize) {
		cw.publishMetrics(ctx)
	}
}

// Check if we should publish the metric buffer to cloudwatch
func (cw *CWSurfacer) publishMetrics(ctx context.Context) {
	if len(cw.metricDatumCache) == 0 {
		return
	}

	_, err := cw.session.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace:  aws.String(cw.c.GetNamespace()),
		MetricData: cw.metricDatumCache,
	})
	if err != nil {
		cw.l.Errorf("Error publishing metrics to cloudwatch: %v", err)
	}

	cw.metricDatumCache = cw.metricDatumCache[:0]                                              // reset the buffer
	cw.metricDatumCacheTicker.Reset(time.Duration(*cw.c.MetricsPublishTimerSec) * time.Second) // reset the ticker

	return
}

// Create a new cloudwatch metriddatum using the values passed in.
func (cw *CWSurfacer) newCWMetricDatum(metricname string, value float64, dimensions []types.Dimension, timestamp time.Time, latencyUnit time.Duration) types.MetricDatum {
	storageResolution := aws.Int32(cw.c.GetResolution())

	// define the metric datum with default values
	metricDatum := types.MetricDatum{
		Dimensions:        dimensions,
		MetricName:        aws.String(metricname),
		Value:             aws.Float64(value),
		StorageResolution: storageResolution,
		Timestamp:         aws.Time(timestamp),
		Unit:              types.StandardUnitCount,
	}

	// the cloudwatch api will throw warnings when a timeseries has multiple
	// units, to avoid this always calculate the value as milliseconds.
	if metricname == "latency" {
		metricDatum.Unit = types.StandardUnitMilliseconds
		metricDatum.Value = aws.Float64(value * float64(latencyUnit) / float64(time.Millisecond))
	}

	return metricDatum
}

// Take metric labels from an event metric and parse them into a Cloudwatch Dimension struct.
func emLabelsToDimensions(em *metrics.EventMetrics) []types.Dimension {
	dimensions := make([]types.Dimension, 0, len(em.LabelsKeys()))

	for _, k := range em.LabelsKeys() {
		dimensions = append(dimensions, types.Dimension{
			Name:  aws.String(k),
			Value: aws.String(em.Label(k)),
		})
	}

	return dimensions
}

// getRegion provides an order of precedence for the lookup of the AWS Region
// Precendence:
// 1. The region passed in via config
// 2. The region discovered from the metadata endpoint
// 3. By returning nil, the environment variable AWS_REGION as evaluated by the
// the AWS SDK.
func getRegion(config *configpb.SurfacerConf) string {
	if config.Region != nil {
		return config.GetRegion()
	}

	vars := sysvars.Vars()
	if v, exists := vars["EC2_Region"]; exists {
		return v
	}

	return ""
}
