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

const (
	batchSize = 1000
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

func (s *Surfacer) parseBQCols(em *metrics.EventMetrics) (map[string]bigquery.Value, error) {
	bqRowMap := make(map[string]bigquery.Value)
	for _, col := range s.c.GetBigqueryColumns() {
		colName := col.GetName()
		val, err := convertToBqType(col.GetType(), em.Label(colName))
		if err != nil {
			return nil, fmt.Errorf("error occurred while parsing for field %v: %v", colName, err)
		}
		bqRowMap[colName] = val
	}
	return bqRowMap, nil
}

func (s *Surfacer) batchInsertRowsToBQ(ctx context.Context, inserter iInserter) {
	chanLen := len(s.writeChan)
	bigqueryTimeout := time.Duration(s.c.GetBigqueryTimeoutSec()) * time.Second
	bqctx, cancel := context.WithTimeout(ctx, bigqueryTimeout)
	defer cancel()

	for i := 0; i < chanLen; i += batchSize {
		var results []*bqrow

		for j := i; j < min(i+batchSize, chanLen); j++ {
			em := <-s.writeChan

			bqRowMap, err := s.parseBQCols(em)

			if err != nil {
				s.l.Errorf("%v", err)
				continue
			}
			results = append(results, &bqrow{value: bqRowMap})
		}

		if len(results) > 0 {
			if err := inserter.Put(bqctx, results); err != nil {
				s.l.Errorf("failed uploading probe results to Bigquery due to surfacer: %v", err)
			}
		}
	}
}

func (s *Surfacer) writeToBQ(ctx context.Context, inserter iInserter) {
	bigqueryInsertionTime := s.c.GetBatchInsertionIntervalSec()
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
