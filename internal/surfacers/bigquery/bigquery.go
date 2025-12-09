// Copyright 2022-2023 The Cloudprober Authors.
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

// Package bigquery implements surfacer for bigquery insertion.
package bigquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	configpb "github.com/cloudprober/cloudprober/internal/surfacers/bigquery/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/common/options"
)

type bqrow struct {
	value map[string]bigquery.Value
}

type iInserter interface {
	Put(context.Context, any) error
}

// Surfacer structures for writing to bigquery.
type Surfacer struct {
	// Configuration
	c    *configpb.SurfacerConf
	opts *options.Options

	// Channel for incoming data.
	writeChan chan *metrics.EventMetrics

	// Cloud logger
	l *logger.Logger
}

// New initializes a bigquery surfacer. bigquery surfacer inserts probe results
// into a bigquery database.
// ctx is used to manage background goroutine.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*Surfacer, error) {
	s := &Surfacer{
		c:    config,
		opts: opts,
		l:    l,
	}
	return s, s.init(ctx)
}

func (row *bqrow) Save() (map[string]bigquery.Value, string, error) {
	return row.value, "", nil
}

// Write takes the data to be written
func (s *Surfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.writeChan <- em:
	default:
		s.l.Errorf("Surfacer's write channel is full, dropping new data.")
	}
}

func convertToBqType(colType, label string) (bigquery.Value, error) {
	if label == "" {
		return "", nil
	}
	colType = strings.ToLower(colType)
	switch colType {
	case "string":
		return label, nil
	case "integer", "int", "numeric":
		val, err := strconv.ParseInt(label, 10, 64)
		if err != nil {
			return nil, err
		}
		return val, nil
	case "float", "double":
		val, err := strconv.ParseFloat(label, 64)
		if err != nil {
			return nil, err
		}
		return val, nil
	case "timestamp":
		milliSec, err := strconv.ParseInt(label, 10, 64)
		if err != nil {
			return nil, err
		}
		timestamp := time.UnixMilli(milliSec).UTC()
		return timestamp, nil
	default:
		return nil, errors.New("invalid column type")
	}
}

func convertToJSON(labels map[string]string) (string, error) {
	bs, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func getJSON(em *metrics.EventMetrics) (string, error) {
	labels := make(map[string]string)
	for _, k := range em.LabelsKeys() {
		labels[k] = em.Label(k)
	}
	return convertToJSON(labels)
}

func copyMap(baseRow map[string]bigquery.Value) map[string]bigquery.Value {
	baseRowCopy := make(map[string]bigquery.Value, len(baseRow))
	for k, v := range baseRow {
		baseRowCopy[k] = v
	}
	return baseRowCopy
}

func updateMetricValues(bqRowMap map[string]bigquery.Value, metricName string, value bigquery.Value, timestamp time.Time, conf *configpb.SurfacerConf) map[string]bigquery.Value {
	bqRowMap[conf.GetMetricNameColName()] = metricName
	bqRowMap[conf.GetMetricValueColName()] = value
	bqRowMap[conf.GetMetricTimeColName()] = timestamp
	return bqRowMap
}

func distToBqMetrics(d *metrics.DistributionData, metricName string, labels map[string]bigquery.Value, timestamp time.Time, conf *configpb.SurfacerConf) []*bqrow {
	sumMetric := updateMetricValues(labels, metricName+"_sum", d.Sum, timestamp, conf)
	countMetric := updateMetricValues(labels, metricName+"_count", d.Count, timestamp, conf)

	bqMetrics := []*bqrow{
		{value: sumMetric},
		{value: countMetric},
	}

	// Create and format all metrics for each bucket in this distribution. Each
	// bucket is assigned a metric name suffixed with "_bucket" and labeled with
	// the corresponding bucket as "le: {bucket}"
	var val int64
	for i := range d.LowerBounds {
		val += d.BucketCounts[i]
		var lb string
		if i == len(d.LowerBounds)-1 {
			lb = "+Inf"
		} else {
			lb = strconv.FormatFloat(d.LowerBounds[i+1], 'f', -1, 64)
		}
		labelsWithBucket := copyMap(labels)
		labelsWithBucket["le"] = lb
		labelsWithBucket = updateMetricValues(labelsWithBucket, metricName+"_bucket", val, timestamp, conf)
		bqMetrics = append(bqMetrics, &bqrow{value: labelsWithBucket})
	}
	return bqMetrics
}

func (s *Surfacer) bqLabels(em *metrics.EventMetrics) (map[string]bigquery.Value, error) {
	baseRow := make(map[string]bigquery.Value)
	for _, col := range s.c.GetBigqueryColumns() {
		colName := col.GetColumnName()
		val, err := convertToBqType(col.GetColumnType(), em.Label(col.GetLabel()))
		if err != nil {
			return nil, fmt.Errorf("error occurred while parsing for field %v: %v", colName, err)
		}
		baseRow[colName] = val
	}
	return baseRow, nil
}

func parseMapToBQCols[T int64 | float64](m *metrics.Map[T], baseRow map[string]bigquery.Value, metricName string, t time.Time, c *configpb.SurfacerConf) []*bqrow {
	var out []*bqrow
	for _, k := range m.Keys() {
		bqRowMap := copyMap(baseRow)
		bqRowMap[m.MapName] = k
		bqRowMap = updateMetricValues(bqRowMap, metricName, m.GetKey(k), t, c)
		out = append(out, &bqrow{value: bqRowMap})
	}
	return out
}

func (s *Surfacer) parseBQCols(em *metrics.EventMetrics) ([]*bqrow, error) {
	baseRow := make(map[string]bigquery.Value)
	var out []*bqrow

	if len(s.c.GetBigqueryColumns()) > 0 {
		bqLabels, err := s.bqLabels(em)
		if err != nil {
			return nil, err
		}
		baseRow = bqLabels
	} else {
		jsonVal, err := getJSON(em)
		if err != nil {
			return nil, err
		}
		baseRow["labels"] = jsonVal
	}

	for _, metricName := range em.MetricsKeys() {
		if !s.opts.AllowMetric(metricName) {
			continue
		}

		val := em.Metric(metricName)

		// Map metric
		if mapVal, ok := val.(*metrics.Map[int64]); ok {
			out = append(out, parseMapToBQCols(mapVal, baseRow, metricName, em.Timestamp, s.c)...)
			continue
		}

		bqRowMap := copyMap(baseRow)
		// Distribution metric
		if distVal, ok := val.(*metrics.Distribution); ok {
			out = append(out, distToBqMetrics(distVal.Data(), metricName, bqRowMap, em.Timestamp, s.c)...)
			continue
		}

		// Convert string metrics to a numeric metric by moving metric value to
		// the "val" label and setting the metric value to 1.
		// For example: version="1.11" becomes version{val="1.11"}=1
		if _, ok := val.(metrics.String); ok {
			bqRowMap["val"] = val
			bqRowMap = updateMetricValues(bqRowMap, metricName, "1", em.Timestamp, s.c)
			out = append(out, &bqrow{value: bqRowMap})
			continue
		}

		bqMetric := updateMetricValues(bqRowMap, metricName, val.String(), em.Timestamp, s.c)
		out = append(out, &bqrow{value: bqMetric})
	}
	return out, nil
}

func (s *Surfacer) batchInsertRowsToBQ(ctx context.Context, inserter iInserter) {
	chanLen := len(s.writeChan)
	bigqueryTimeout := time.Duration(s.c.GetBigqueryTimeoutSec()) * time.Second
	bqctx, cancel := context.WithTimeout(ctx, bigqueryTimeout)
	defer cancel()
	batchSize := int(s.c.GetMetricsBatchSize())

	for i := 0; i < chanLen; i += batchSize {
		var bqRowsArr []*bqrow

		for j := i; j < i+batchSize && j < chanLen; j++ {
			em := <-s.writeChan

			bqMetrics, err := s.parseBQCols(em)

			if err != nil {
				s.l.Errorf("%v", err)
				continue
			}
			bqRowsArr = append(bqRowsArr, bqMetrics...)
		}
		if len(bqRowsArr) > 0 {
			if err := inserter.Put(bqctx, bqRowsArr); err != nil {
				for _, row := range bqRowsArr {
					s.l.Errorf("failed uploading row to Bigquery: %v, row: %v", err, row.value)
				}
			}
		}
	}
}

func (s *Surfacer) writeToBQ(ctx context.Context, inserter iInserter) {
	ticker := time.NewTicker(time.Duration(s.c.GetBatchTimerSec()) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.l.Infof("Context canceled, stopping the surfacer write loop")
			return
		case <-ticker.C:
			s.batchInsertRowsToBQ(ctx, inserter)
		}
	}
}

func (s *Surfacer) init(ctx context.Context) error {
	s.writeChan = make(chan *metrics.EventMetrics, s.c.GetMetricsBufferSize())

	client, err := bigquery.NewClient(ctx, s.c.GetProjectName())
	if err != nil {
		s.l.Errorf("bigquery client can't be created: %v", err)
		return err
	}
	defer client.Close()

	inserter := client.Dataset(s.c.GetBigqueryDataset()).Table(s.c.GetBigqueryTable()).Inserter()
	if inserter == nil {
		return fmt.Errorf("error bigquery inserter cannot be created")
	}

	// Start a goroutine to run forever, polling on the writeChan. Allows
	// for the surfacer to write asynchronously to the serial port.
	go func() {
		s.writeToBQ(ctx, inserter)
	}()

	return nil
}
