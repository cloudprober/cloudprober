// Copyright 2018 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this postgres except in compliance with the License.
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
Package postgres implements "postgres" surfacer. This surfacer type is in
experimental phase right now.

To use this surfacer, add a stanza similar to the following to your
cloudprober config:

	surfacer {
	  type: POSTGRES
		postgres_surfacer {
		  connection_string: "postgresql://root:${PASSWORD}@localhost/cloudprober?sslmode=disable"
		  metrics_table_name: "metrics"
	  }
	}
*/
package postgres

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/common/options"
	"github.com/jackc/pgx/v5"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/postgres/proto"
)

// pgMetric represents a single metric and corresponds to a single row in the
// metrics table.
type pgMetric struct {
	time       time.Time
	metricName string
	value      string
	labels     map[string]string
}

func updateLabelMap(labels map[string]string, extraLabels ...[2]string) map[string]string {
	if len(extraLabels) == 0 {
		return labels
	}
	labelsCopy := make(map[string]string)
	for k, v := range labels {
		labelsCopy[k] = v
	}
	for _, extraLabel := range extraLabels {
		labelsCopy[extraLabel[0]] = extraLabel[1]
	}
	return labelsCopy
}

// labelsJSON takes the labels array and formats it for insertion into
// postgres jsonb labels column, storing each label as k,v json object
func labelsJSON(labels map[string]string) (string, error) {
	bs, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func newPGMetric(t time.Time, metricName, val string, labels map[string]string) pgMetric {
	return pgMetric{
		time:       t,
		metricName: metricName,
		value:      val,
		labels:     labels,
	}
}

func distToPGMetrics(d *metrics.DistributionData, metricName string, labels map[string]string, t time.Time) []pgMetric {
	pgMerics := []pgMetric{
		newPGMetric(t, metricName+"_sum", strconv.FormatFloat(d.Sum, 'f', -1, 64), labels),
		newPGMetric(t, metricName+"_count", strconv.FormatInt(d.Count, 10), labels),
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
		labelsWithBucket := updateLabelMap(labels, [2]string{"le", lb})
		pgMerics = append(pgMerics, newPGMetric(t, metricName+"_bucket", strconv.FormatInt(val, 10), labelsWithBucket))
	}

	return pgMerics
}

func mapToPGMetrics[T int64 | float64](m *metrics.Map[T], em *metrics.EventMetrics, metricName string, baseLabels map[string]string) []pgMetric {
	pgMerics := []pgMetric{}
	for _, k := range m.Keys() {
		labels := updateLabelMap(baseLabels, [2]string{m.MapName, k})
		pgMerics = append(pgMerics, newPGMetric(em.Timestamp, metricName, metrics.MapValueToString[T](m.GetKey(k)), labels))
	}
	return pgMerics
}

// emToPGMetrics converts an EventMetrics struct into a list of pgMetrics.
func (s *Surfacer) emToPGMetrics(em *metrics.EventMetrics) []pgMetric {
	baseLabels := make(map[string]string)
	for _, k := range em.LabelsKeys() {
		baseLabels[k] = em.Label(k)
	}

	pgMerics := []pgMetric{}
	for _, metricName := range em.MetricsKeys() {
		if !s.opts.AllowMetric(metricName) {
			continue
		}

		val := em.Metric(metricName)

		// Map metric
		if mapVal, ok := val.(*metrics.Map[int64]); ok {
			pgMerics = append(pgMerics, mapToPGMetrics(mapVal, em, metricName, baseLabels)...)
			continue
		}
		if mapVal, ok := val.(*metrics.Map[float64]); ok {
			pgMerics = append(pgMerics, mapToPGMetrics(mapVal, em, metricName, baseLabels)...)
			continue
		}

		// Distribution metric
		if distVal, ok := val.(*metrics.Distribution); ok {
			pgMerics = append(pgMerics, distToPGMetrics(distVal.Data(), metricName, baseLabels, em.Timestamp)...)
			continue
		}

		// Convert string metrics to a numeric metric by moving metric value to
		// the "val" label and setting the metric value to 1.
		// For example: version="1.11" becomes version{val="1.11"}=1
		if _, ok := val.(metrics.String); ok {
			labels := updateLabelMap(baseLabels, [2]string{"val", val.String()})
			pgMerics = append(pgMerics, newPGMetric(em.Timestamp, metricName, "1", labels))
			continue
		}

		pgMerics = append(pgMerics, newPGMetric(em.Timestamp, metricName, val.String(), baseLabels))
	}
	return pgMerics
}

// writeMetrics parses events metrics into postgres rows, starts a transaction
// and inserts all discreet metric rows represented by the EventMetrics
func (s *Surfacer) dbRows(ems []*metrics.EventMetrics) ([][]any, error) {
	var rows [][]any

	for _, em := range ems {
		// Transaction for defined columns
		if len(s.c.GetLabelToColumn()) > 0 {
			for _, pgMetric := range s.emToPGMetrics(em) {
				row := []any{pgMetric.time, pgMetric.metricName, pgMetric.value}
				row = append(row, generateValues(pgMetric.labels, s.c.GetLabelToColumn())...)
				rows = append(rows, row)
			}
		} else {
			for _, pgMetric := range s.emToPGMetrics(em) {
				s, err := labelsJSON(pgMetric.labels)
				if err != nil {
					return nil, err
				}
				rows = append(rows, []any{pgMetric.time, pgMetric.metricName, pgMetric.value, s})
			}
		}
	}

	return rows, nil
}

// writeMetrics parses events metrics into postgres rows, starts a transaction
// and inserts all discreet metric rows represented by the EventMetrics
func (s *Surfacer) writeMetrics(ctx context.Context, ems []*metrics.EventMetrics) error {
	rows, err := s.dbRows(ems)
	if err != nil {
		return err
	}

	_, err = s.dbconn.CopyFrom(ctx, pgx.Identifier{s.c.GetMetricsTableName()}, s.columns, pgx.CopyFromRows(rows))
	return err
}

// init connects to postgres
func (s *Surfacer) init(ctx context.Context) error {
	s.l.Info("Initializing postgres surfacer")

	var err error

	if s.dbconn, err = s.openDB(s.c.GetConnectionString()); err != nil {
		return err
	}
	if err = s.dbconn.Ping(ctx); err != nil {
		return err
	}
	s.writeChan = make(chan *metrics.EventMetrics, s.c.GetMetricsBufferSize())

	// Generate the desired columns either with 'labels' by default
	// or select 'labels' based on the label_to_column fields
	s.columns = colName(s.c.GetLabelToColumn())

	// Start a goroutine to run forever, polling on the writeChan. Allows
	// for the surfacer to write asynchronously to the serial port.
	go func() {
		defer s.dbconn.Close(ctx)

		metricsBatchSize, batchTimerSec := s.c.GetMetricsBatchSize(), s.c.GetBatchTimerSec()

		buffer := make([]*metrics.EventMetrics, 0, metricsBatchSize)
		flushInterval := time.Duration(batchTimerSec) * time.Second

		flushTicker := time.NewTicker(flushInterval)
		defer flushTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.l.Infof("Context canceled, stopping the surfacer write loop")
				return
			case em := <-s.writeChan:
				if em.Kind != metrics.CUMULATIVE && em.Kind != metrics.GAUGE {
					continue
				}
				buffer = append(buffer, em)
				if int32(len(buffer)) >= metricsBatchSize {
					if err := s.writeMetrics(ctx, buffer); err != nil {
						s.l.Warningf("Error while writing metrics: %v", err)
					}
					buffer = buffer[:0]
					flushTicker.Reset(flushInterval)
				}
			case <-flushTicker.C:
				if len(buffer) > 0 {
					if err := s.writeMetrics(ctx, buffer); err != nil {
						s.l.Warningf("Error while writing metrics: %v", err)
					}
					buffer = buffer[:0]
				}
			}
		}
	}()

	return nil
}

// Write takes the data to be written
func (s *Surfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	select {
	case s.writeChan <- em:
	default:
		s.l.Errorf("Surfacer's write channel is full, dropping new data.")
	}
}

// generateValues generates column values or places NULL
// in the event label/value does not exist
func generateValues(labels map[string]string, ltc []*configpb.LabelToColumn) []any {
	var args []any

	for _, v := range ltc {
		if val, ok := labels[v.GetLabel()]; ok {
			args = append(args, val)
		} else {
			args = append(args, nil)
		}
	}

	return args
}

// colName figures out postgres table column names, based on the
// label_to_column configuration.
func colName(ltc []*configpb.LabelToColumn) []string {
	var columns []string
	if len(ltc) > 0 {
		columns = append([]string{"time", "metric_name", "value"}, make([]string, len(ltc))...)
		for i, v := range ltc {
			columns[i+3] = v.GetColumn()
		}
	} else {
		columns = []string{"time", "metric_name", "value", "labels"}
	}
	return columns
}

// Surfacer structures for writing to postgres.
type Surfacer struct {
	// Configuration
	c       *configpb.SurfacerConf
	opts    *options.Options
	columns []string

	// Channel for incoming data.
	writeChan chan *metrics.EventMetrics

	// Cloud logger
	l *logger.Logger

	openDB func(connectionString string) (*pgx.Conn, error)
	dbconn *pgx.Conn
}

// New initializes a Postgres surfacer. Postgres surfacer inserts probe results
// into a postgres database.
func New(ctx context.Context, config *configpb.SurfacerConf, opts *options.Options, l *logger.Logger) (*Surfacer, error) {
	s := &Surfacer{
		c:    config,
		opts: opts,
		l:    l,
		openDB: func(cs string) (*pgx.Conn, error) {
			return pgx.Connect(ctx, cs)
		},
	}
	return s, s.init(ctx)
}
