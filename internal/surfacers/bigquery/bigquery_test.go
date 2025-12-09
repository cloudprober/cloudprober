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

package bigquery

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	configpb "github.com/cloudprober/cloudprober/internal/surfacers/bigquery/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
)

const (
	metricNameCol  = "metric_name"
	metricValueCol = "metric_value"
	metricTimeCol  = "metric_time"
)

type fakeInserter struct {
	batchCount int
}

func (i *fakeInserter) Put(ctx context.Context, values any) error {
	i.batchCount++
	return nil
}

func newSurfacerConfig(col map[string]string) *configpb.SurfacerConf {
	projectName := "test-project"
	bqdataset := "test-dataset"
	bqtable := "test-table"

	var bqCols []*configpb.BQColumn
	for k, v := range col {
		colName := k
		colType := v
		bqCol := &configpb.BQColumn{
			Label:      &colName,
			ColumnName: &colName,
			ColumnType: &colType,
		}
		bqCols = append(bqCols, bqCol)
	}

	surfacerConf := &configpb.SurfacerConf{
		ProjectName:     &projectName,
		BigqueryDataset: &bqdataset,
		BigqueryTable:   &bqtable,
		BigqueryColumns: bqCols,
	}

	return surfacerConf
}

func TestWriteWhenSurfacerChannelIsNotFull(t *testing.T) {
	numMetrics := [5]int{12, 1, 103, 3500, 10000}
	for _, num := range numMetrics {
		s := &Surfacer{}
		s.writeChan = make(chan *metrics.EventMetrics, num)
		oldChanLen := len(s.writeChan)
		ctx := context.Background()

		for i := 0; i < num; i++ {
			s.Write(ctx, &metrics.EventMetrics{})
		}

		if len(s.writeChan) != oldChanLen+num {
			t.Fatalf("Metric not inserted in surfacer channel!\nChannel length: %d, Expected length: %d", len(s.writeChan), oldChanLen+num)
		}

	}
}

func TestWriteWhenSurfacerChannelIsFull(t *testing.T) {
	numMetrics := [3]int{25, 1, 2300}
	for _, num := range numMetrics {
		s := &Surfacer{}
		s.writeChan = make(chan *metrics.EventMetrics, num)
		ctx := context.Background()

		for i := 0; i < num; i++ {
			s.Write(ctx, &metrics.EventMetrics{})
		}
		oldChanLen := len(s.writeChan)

		s.Write(ctx, &metrics.EventMetrics{})

		if len(s.writeChan) != oldChanLen {
			t.Fatal("Metric inserted even though surfacer capacity should be full!")
		}
	}
}

func TestConvertToBQTypeForSuccess(t *testing.T) {
	tests := []struct {
		colType string
		value   string
		want    bigquery.Value
	}{
		{
			colType: "string",
			value:   "This is a test string",
			want:    "This is a test string",
		},
		{
			colType: "int",
			value:   "43251",
			want:    int64(43251),
		},
		{
			colType: "INTEGER",
			value:   "94318",
			want:    int64(94318),
		},
		{
			colType: "FloAt",
			value:   "432.97",
			want:    432.97,
		},
		{
			colType: "double",
			value:   "1.1",
			want:    1.1,
		},
		{
			colType: "timestamp",
			value:   "1438019908",
			want:    time.UnixMilli(1438019908).UTC(),
		},
	}

	for _, tc := range tests {
		val, err := convertToBqType(tc.colType, tc.value)
		if err != nil {
			t.Fatalf("Error while parsing column with value %v and type %v", tc.value, tc.colType)
		}

		if val != tc.want {
			t.Fatalf("Incorrect parsed value for column type %v!\n Got=%v, Expected=%v", tc.colType, val, tc.want)
		}
	}
}

func TestConvertToBQTypeForFailure(t *testing.T) {
	tests := []struct {
		colType string
		value   string
	}{
		{
			colType: "test",
			value:   "This is a test string",
		},
		{
			colType: "int",
			value:   "23.1",
		},
		{
			colType: "INTEGER",
			value:   "90.43",
		},
		{
			colType: "float",
			value:   "432.97a",
		},
		{
			colType: "double",
			value:   "1b.1",
		},
		{
			colType: "timestamp",
			value:   "time",
		},
	}

	for _, tc := range tests {
		val, err := convertToBqType(tc.colType, tc.value)
		if err == nil || val != nil {
			t.Fatalf("Expected error but got nil for test case %v!", tc)
		}
	}
}

func TestParseBQColsForValidRow(t *testing.T) {
	colValueMap := map[string]string{
		"name":   "test",
		"result": "success",
		"size":   "1",
	}
	colTypeMap := map[string]string{
		"name":   "string",
		"result": "string",
		"size":   "integer",
	}
	s := &Surfacer{
		c:         newSurfacerConfig(colTypeMap),
		l:         &logger.Logger{},
		writeChan: make(chan *metrics.EventMetrics, 10),
	}

	em := metrics.NewEventMetrics(time.Now())
	for k, v := range colValueMap {
		em.AddLabel(k, v)
	}
	em.AddMetric("TestBqCols", metrics.NewInt(1))

	expectedBqCols := map[string]bigquery.Value{
		"name":   "test",
		"result": "success",
		"size":   int64(1),
	}
	parseBqCols, err := s.parseBQCols(em)

	if err != nil {
		t.Fatalf("Error while parsing bq columns: %v", err)
	}

	if len(parseBqCols) == 0 {
		t.Fatalf("Length of parsed columns should be greater than 0!")
	}

	for k, v := range expectedBqCols {
		for _, metric := range parseBqCols {
			if v != metric.value[k] {
				t.Fatalf("Mismatch in column values!\n Got=%v, Expected=%v", v, metric.value[k])
			}
		}
	}
}

func TestInsertRowsToBQ(t *testing.T) {
	tests := [4]int{134, 9, 43141, 34}

	colValueMap := map[string]string{
		"id":     "test",
		"result": "success",
		"size":   "1",
	}
	colTypeMap := map[string]string{
		"id":     "string",
		"result": "string",
		"size":   "integer",
	}
	ctx := context.Background()

	for _, tc := range tests {
		s := &Surfacer{
			c:         newSurfacerConfig(colTypeMap),
			l:         &logger.Logger{},
			writeChan: make(chan *metrics.EventMetrics, tc),
		}

		em := metrics.NewEventMetrics(time.Now())
		for k, v := range colValueMap {
			em.AddLabel(k, v)
		}
		em.AddMetric("TestInsertRows", metrics.NewInt(2))

		for i := 0; i < tc; i++ {
			s.Write(ctx, em)
		}

		s.batchInsertRowsToBQ(ctx, &fakeInserter{})

		if len(s.writeChan) != 0 {
			t.Fatalf("Error in inserting rows to BQ! Remaining rows: %v", len(s.writeChan))
		}
	}
}

func TestBatchInsertion(t *testing.T) {
	tests := [6]int{2999, 3000, 3001, 1, 1500, 432190}
	expectedCounts := [6]int{3, 3, 4, 1, 2, 433}

	colValueMap := map[string]string{
		"id": "test",
	}
	colTypeMap := map[string]string{
		"id": "string",
	}
	ctx := context.Background()

	for i, tc := range tests {
		s := &Surfacer{
			c:         newSurfacerConfig(colTypeMap),
			l:         &logger.Logger{},
			writeChan: make(chan *metrics.EventMetrics, tc),
		}

		em := metrics.NewEventMetrics(time.Now())
		for k, v := range colValueMap {
			em.AddLabel(k, v)
		}
		em.AddMetric("TestBatchInsertion", metrics.NewInt(3))

		for i := 0; i < tc; i++ {
			s.Write(ctx, em)
		}

		inserter := &fakeInserter{
			batchCount: 0,
		}

		s.batchInsertRowsToBQ(ctx, inserter)

		if inserter.batchCount != expectedCounts[i] {
			t.Fatalf("Number of expected put calls %v, Got=%v for tc=%v!", expectedCounts[i], inserter.batchCount, tc)
		}
	}
}

func TestWriteToBQ(t *testing.T) {
	colValueMap := map[string]string{
		"id": "test",
	}
	colTypeMap := map[string]string{
		"id": "string",
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*11)
	defer cancel()

	s := &Surfacer{
		c:         newSurfacerConfig(colTypeMap),
		l:         &logger.Logger{},
		writeChan: make(chan *metrics.EventMetrics, 4500),
	}

	em := metrics.NewEventMetrics(time.Now())
	for k, v := range colValueMap {
		em.AddLabel(k, v)
	}
	em.AddMetric("TestWriteToBQ", metrics.NewInt(5))

	for i := 0; i < 4500; i++ {
		s.Write(ctx, em)
	}

	inserter := &fakeInserter{
		batchCount: 0,
	}

	s.writeToBQ(ctx, inserter)

	if len(s.writeChan) != 0 {
		t.Fatalf("Error in writeToBQ!")
	}
}
