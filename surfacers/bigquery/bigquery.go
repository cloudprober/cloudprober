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
	"github.com/cloudprober/cloudprober/surfacers/common/options"
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
	baseRowCopy := make(map[string]bigquery.Value)
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
		labelsWithBucket := copyMap(labels)
		labelsWithBucket["le"] = lb
		labelsWithBucket = updateMetricValues(labelsWithBucket, metricName+"_bucket", val, timestamp, conf)
		bqMetrics = append(bqMetrics, &bqrow{value: labelsWithBucket})
	}
	return bqMetrics
}

func (s *Surfacer) parseBQCols(em *metrics.EventMetrics) ([]*bqrow, error) {
	baseRow := make(map[string]bigquery.Value)
	var out []*bqrow

	if len(s.c.GetBigqueryColumns()) > 0 {
		for _, col := range s.c.GetBigqueryColumns() {
			colName := col.GetColumnName()
			label := em.Label(col.GetLabel())
			val, err := convertToBqType(col.GetColumnType(), label)
			if err != nil {
				return nil, fmt.Errorf("error occurred while parsing for field %v: %v", colName, err)
			}
			baseRow[colName] = val
		}
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
		if mapVal, ok := val.(*metrics.Map); ok {
			for _, k := range mapVal.Keys() {
				bqRowMap := copyMap(baseRow)
				bqRowMap[mapVal.MapName] = k
				bqRowMap = updateMetricValues(bqRowMap, metricName, mapVal.GetKey(k), em.Timestamp, s.c)
				out = append(out, &bqrow{value: bqRowMap})
			}
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
