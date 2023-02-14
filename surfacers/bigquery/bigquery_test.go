package bigquery

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/surfacers/bigquery/proto"
)

const (
	projectName = "test-project"
	bqdataset   = "test-dataset"
	bqtable     = "test-table"
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
			Name: &colName,
			Type: &colType,
		}
		bqCols = append(bqCols, bqCol)
	}

	surfacerConf := &configpb.SurfacerConf{
		ProjectName:     &projectName,
		BigqueryDataset: &bqdataset,
		BigqueryTable:   &bqtable,
		Columns:         bqCols,
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

func TestMin(t *testing.T) {
	tests := []struct {
		a    int
		b    int
		want int
	}{
		{
			a:    10,
			b:    20,
			want: 10,
		},
		{
			a:    20,
			b:    10,
			want: 10,
		},
		{
			a:    10,
			b:    10,
			want: 10,
		},
	}

	for _, tc := range tests {
		res := min(tc.a, tc.b)
		if res != tc.want {
			t.Fatalf("min(%d, %d) = %d, expected %d", tc.a, tc.b, res, tc.want)
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

	expectedBqCols := map[string]bigquery.Value{
		"name":   "test",
		"result": "success",
		"size":   int64(1),
	}

	parseBqCols, err := s.parseBQCols(em)

	if err != nil {
		t.Fatalf("Error while parsing bq columns: %v", err)
	}

	for k, v := range expectedBqCols {
		if parseBqCols[k] != v {
			t.Fatalf("Mismatch in column values!\n Got=%v, Expected=%v", parseBqCols[k], v)
		}
	}
}

func TestParseBQColsForInvalidRow(t *testing.T) {
	colValueMap := map[string]string{
		"name":   "test",
		"result": "success",
		"sizes":  "1",
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

	_, err := s.parseBQCols(em)

	if err == nil {
		t.Fatalf("No invalid row found!")
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

		for i := 0; i < tc; i++ {
			s.Write(ctx, em)
		}

		inserter := &fakeInserter{
			batchCount: 0,
		}

		s.batchInsertRowsToBQ(ctx, inserter)

		if inserter.batchCount != expectedCounts[i] {
			t.Fatalf("Error in batch insertion for test case: %v!", tc)
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
