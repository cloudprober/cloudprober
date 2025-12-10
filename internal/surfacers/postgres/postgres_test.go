package postgres

import (
	"reflect"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/postgres/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestEMToPGMetricsNoDistribution(t *testing.T) {
	respCodesVal := metrics.NewMap("code")
	respCodesVal.IncKeyBy("200", 19)
	ts := time.Now()
	em := metrics.NewEventMetrics(ts).
		AddMetric("sent", metrics.NewInt(32)).
		AddMetric("rcvd", metrics.NewInt(22)).
		AddMetric("latency", metrics.NewFloat(10.11111)).
		AddMetric("resp_code", respCodesVal).
		AddLabel("ptype", "http")

	s := &Surfacer{}
	rows := s.emToPGMetrics(em)

	if len(rows) != 4 {
		t.Errorf("Expected %d rows, received: %d\n", 4, len(rows))
	}

	if !isRowExpected(rows[0], ts, "sent", "32", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[0])
	}

	if !isRowExpected(rows[1], ts, "rcvd", "22", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[1])
	}

	if !isRowExpected(rows[2], ts, "latency", "10.111", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[2])
	}

	if !isRowExpected(rows[3], ts, "resp_code", "19", map[string]string{"ptype": "http", "code": "200"}) {
		t.Errorf("Incorrect Row found %+v", rows[3])
	}
}

func TestEMToPGMetricsWithDistribution(t *testing.T) {
	respCodesVal := metrics.NewMap("code")
	respCodesVal.IncKeyBy("200", 19)
	latencyVal := metrics.NewDistribution([]float64{1, 4})
	latencyVal.AddSample(0.5)
	latencyVal.AddSample(5)
	ts := time.Now()
	em := metrics.NewEventMetrics(ts).
		AddMetric("latency", latencyVal).
		AddLabel("ptype", "http")

	s := &Surfacer{}
	rows := s.emToPGMetrics(em)

	if len(rows) != 5 {
		t.Errorf("Expected %d rows, received: %d\n", 5, len(rows))
	}

	if !isRowExpected(rows[0], ts, "latency_sum", "5.5", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[0])
	}

	if !isRowExpected(rows[1], ts, "latency_count", "2", map[string]string{"ptype": "http"}) {
		t.Errorf("Incorrect Row found %+v", rows[1])
	}

	if !isRowExpected(rows[2], ts, "latency_bucket", "1", map[string]string{"ptype": "http", "le": "1"}) {
		t.Errorf("Incorrect Row found %+v", rows[2])
	}

	if !isRowExpected(rows[3], ts, "latency_bucket", "1", map[string]string{"ptype": "http", "le": "4"}) {
		t.Errorf("Incorrect Row found %+v", rows[3])
	}

	if !isRowExpected(rows[4], ts, "latency_bucket", "2", map[string]string{"ptype": "http", "le": "+Inf"}) {
		t.Errorf("Incorrect Row found %+v", rows[4])
	}

}

func isRowExpected(row pgMetric, t time.Time, metricName string, value string, labels map[string]string) bool {
	if row.time != t {
		return false
	}
	if row.metricName != metricName {
		return false
	}
	if row.value != value {
		return false
	}
	if !reflect.DeepEqual(row.labels, labels) {
		return false
	}

	return true
}

func TestGenerateValues(t *testing.T) {
	label1 := "dst"
	label2 := "code"
	column1 := "dst"
	column2 := "code"

	type args struct {
		labels map[string]string
		ltc    []*configpb.LabelToColumn
	}
	tests := []struct {
		name string
		args args
		want []interface{}
	}{
		{
			name: "test",
			args: args{
				labels: map[string]string{label2: "200", label1: "google.com"},
				ltc: []*configpb.LabelToColumn{{
					Label:  &label2,
					Column: &column2,
				}, {
					Label:  &label1,
					Column: &column1,
				}},
			},
			want: []interface{}{"200", "google.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateValues(tt.args.labels, tt.args.ltc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColColumns(t *testing.T) {
	label1 := "dst"
	label2 := "code"
	column1 := "dst"
	column2 := "code"

	type args struct {
		ltc []*configpb.LabelToColumn
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test-1",
			args: args{ltc: []*configpb.LabelToColumn{{
				Label:  &label2,
				Column: &column2,
			}, {
				Label:  &label1,
				Column: &column1,
			}},
			},
			want: []string{
				"time", "metric_name", "value", "code", "dst",
			},
		},
		{
			name: "test-2",
			args: args{ltc: nil},
			want: []string{
				"time", "metric_name", "value", "labels",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colName(tt.args.ltc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("colNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBRows(t *testing.T) {
	ts := time.Now()
	ems := []*metrics.EventMetrics{
		metrics.NewEventMetrics(ts).AddMetric("sent", metrics.NewInt(32)).AddMetric("rcvd", metrics.NewInt(22)).AddLabel("dst", "dst1"),
		metrics.NewEventMetrics(ts).AddMetric("sent", metrics.NewInt(33)).AddMetric("rcvd", metrics.NewInt(32)).AddLabel("dst", "dst2"),
	}
	tests := []struct {
		name          string
		columns       []string
		labelToColumn []*configpb.LabelToColumn
		want          [][]any
		wantErr       bool
	}{
		{
			name:    "test-1",
			columns: []string{"time", "metric_name", "value", "dst"},
			labelToColumn: []*configpb.LabelToColumn{{
				Label:  proto.String("dst"),
				Column: proto.String("dst"),
			}},
			want: [][]any{
				{ts, "sent", "32", "dst1"},
				{ts, "rcvd", "22", "dst1"},
				{ts, "sent", "33", "dst2"},
				{ts, "rcvd", "32", "dst2"},
			},
		},
		{
			name:          "test-2",
			columns:       []string{"time", "metric_name", "value", "labels"},
			labelToColumn: nil,
			want: [][]any{
				{ts, "sent", "32", "{\"dst\":\"dst1\"}"},
				{ts, "rcvd", "22", "{\"dst\":\"dst1\"}"},
				{ts, "sent", "33", "{\"dst\":\"dst2\"}"},
				{ts, "rcvd", "32", "{\"dst\":\"dst2\"}"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Surfacer{
				columns: tt.columns,
				c:       &configpb.SurfacerConf{LabelToColumn: tt.labelToColumn},
			}
			got, err := s.dbRows(ems)
			if (err != nil) != tt.wantErr {
				t.Errorf("Surfacer.dbRows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got, "dbRows() = %v, want %v", got, tt.want)
		})
	}
}
