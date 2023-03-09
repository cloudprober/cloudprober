// Package bigquery implements surfacer for bigquery insertion.
package bigquery

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/surfacers/bigquery/proto"
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
	c *configpb.SurfacerConf

	// Channel for incoming data.
	writeChan chan *metrics.EventMetrics

	// Cloud logger
	l *logger.Logger
}

// New initializes a bigquery surfacer. bigquery surfacer inserts probe results
// into a bigquery database.
// ctx is used to manage background goroutine.
func New(ctx context.Context, config *configpb.SurfacerConf, l *logger.Logger) (*Surfacer, error) {
	s := &Surfacer{
		c: config,
		l: l,
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func convertToBqType(colType, label string) (bigquery.Value, error) {
	colType = strings.ToLower(colType)
	switch colType {
	case "string":
		return label, nil
	case "integer", "int":
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

func updateLabels(labels map[string]bigquery.Value, key string, val bigquery.Value) map[string]bigquery.Value {
	labelsCopy := make(map[string]bigquery.Value)
	for k, v := range labels {
		labelsCopy[k] = v
	}
	labelsCopy[key] = val
	return labelsCopy
}

func updateMetricValues(bqRowMap map[string]bigquery.Value, metricName string, value bigquery.Value, timestamp time.Time, excludedColumns []string) map[string]bigquery.Value {
	mapCopy := make(map[string]bigquery.Value)
	for k, v := range bqRowMap {
		mapCopy[k] = v
	}
	if !slices.Contains(excludedColumns, "metric_name") {
		mapCopy["metric_name"] = metricName
	}
	if !slices.Contains(excludedColumns, "metric_value") {
		mapCopy["metric_value"] = value
	}
	if !slices.Contains(excludedColumns, "metric_time") {
		mapCopy["metric_time"] = timestamp
	}
	return mapCopy
}

func distToBqMetrics(d *metrics.DistributionData, metricName string, labels map[string]bigquery.Value, timestamp time.Time, excludedColumns []string) []*bqrow {
	sumMetric := updateMetricValues(labels, metricName+"_sum", d.Sum, timestamp, excludedColumns)
	countMetric := updateMetricValues(labels, metricName+"_count", d.Count, timestamp, excludedColumns)

	bqMetrics := []*bqrow{
		&bqrow{value: sumMetric},
		&bqrow{value: countMetric},
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
		labelsWithBucket := updateLabels(labels, "le", lb)
		labelsWithBucket = updateMetricValues(labelsWithBucket, metricName+"_bucket", val, timestamp, excludedColumns)
		bqMetrics = append(bqMetrics, &bqrow{value: labelsWithBucket})
	}
	return bqMetrics
}

func (s *Surfacer) parseBQCols(em *metrics.EventMetrics) ([]*bqrow, error) {
	labels := make(map[string]bigquery.Value)
	var bqMetrics []*bqrow

	if len(s.c.GetBigqueryColumns()) > 0 {
		for _, col := range s.c.GetBigqueryColumns() {
			colName := col.GetColumnName()
			label := col.GetLabel()
			val, err := convertToBqType(col.GetColumnType(), em.Label(label))
			if err != nil {
				return nil, fmt.Errorf("error occurred while parsing for field %v: %v", colName, err)
			}
			labels[colName] = val
		}
	} else {
		jsonVal, err := getJSON(em)
		if err != nil {
			return nil, err
		}
		labels["labels"] = jsonVal
	}

	for _, metricName := range em.MetricsKeys() {
		val := em.Metric(metricName)

		// Map metric
		if mapVal, ok := val.(*metrics.Map); ok {
			for _, k := range mapVal.Keys() {
				bqRowMap := updateLabels(labels, mapVal.MapName, k)
				bqRowMap = updateMetricValues(bqRowMap, metricName, mapVal.GetKey(k), em.Timestamp, s.c.GetExcludeMetricColumn())
				bqMetrics = append(bqMetrics, &bqrow{value: labels})
			}
			continue
		}

		// Distribution metric
		if distVal, ok := val.(*metrics.Distribution); ok {
			bqMetrics = append(bqMetrics, distToBqMetrics(distVal.Data(), metricName, labels, em.Timestamp, s.c.GetExcludeMetricColumn())...)
			continue
		}

		// Convert string metrics to a numeric metric by moving metric value to
		// the "val" label and setting the metric value to 1.
		// For example: version="1.11" becomes version{val="1.11"}=1
		if _, ok := val.(metrics.String); ok {
			bqRowMap := updateLabels(labels, "val", val)
			bqRowMap = updateMetricValues(bqRowMap, metricName, "1", em.Timestamp, s.c.GetExcludeMetricColumn())
			bqMetrics = append(bqMetrics, &bqrow{value: bqRowMap})
			continue
		}

		fmt.Printf("Expected statement!!")
		bqMetric := updateMetricValues(labels, metricName, val.String(), em.Timestamp, s.c.GetExcludeMetricColumn())
		bqMetrics = append(bqMetrics, &bqrow{value: bqMetric})
	}
	return bqMetrics, nil
}

func (s *Surfacer) batchInsertRowsToBQ(ctx context.Context, inserter iInserter) {
	chanLen := len(s.writeChan)
	bigqueryTimeout := time.Duration(s.c.GetBigqueryTimeoutSec()) * time.Second
	bqctx, cancel := context.WithTimeout(ctx, bigqueryTimeout)
	defer cancel()
	batchSize := int(s.c.GetMetricsBatchSize())

	for i := 0; i < chanLen; i += batchSize {
		var results []*bqrow

		for j := i; j < min(i+batchSize, chanLen); j++ {
			em := <-s.writeChan

			bqMetrics, err := s.parseBQCols(em)

			if err != nil {
				s.l.Errorf("%v", err)
				continue
			}
			results = append(results, bqMetrics...)
		}
		if len(results) > 0 {
			if err := inserter.Put(bqctx, results); err != nil {
				for _, row := range results {
					s.l.Errorf("failed uploading row to Bigquery: %v, row: %v", err, row.value)
				}
			}
		}
	}
}

func (s *Surfacer) writeToBQ(ctx context.Context, inserter iInserter) {
	bigqueryInsertionTime := s.c.GetBatchTimerSec()
	ticker := time.NewTicker(time.Duration(bigqueryInsertionTime) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.l.Infof("Context canceled, stopping the surfacer write loop")
			s.batchInsertRowsToBQ(ctx, inserter)
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
