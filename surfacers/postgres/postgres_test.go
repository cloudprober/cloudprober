package postgres

import (
	configpb "github.com/cloudprober/cloudprober/surfacers/postgres/proto"
	"reflect"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
)

func Test_emToPGMetrics_No_Distribution(t *testing.T) {
	respCodesVal := metrics.NewMap("code", metrics.NewInt(0))
	respCodesVal.IncKeyBy("200", metrics.NewInt(19))
	ts := time.Now()
	em := metrics.NewEventMetrics(ts).
		AddMetric("sent", metrics.NewInt(32)).
		AddMetric("rcvd", metrics.NewInt(22)).
		AddMetric("latency", metrics.NewFloat(10.11111)).
		AddMetric("resp_code", respCodesVal).
		AddLabel("ptype", "http")

	rows := emToPGMetrics(em)

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

func Test_emToPGMetrics_With_Distribution(t *testing.T) {
	respCodesVal := metrics.NewMap("code", metrics.NewInt(0))
	respCodesVal.IncKeyBy("200", metrics.NewInt(19))
	latencyVal := metrics.NewDistribution([]float64{1, 4})
	latencyVal.AddSample(0.5)
	latencyVal.AddSample(5)
	ts := time.Now()
	em := metrics.NewEventMetrics(ts).
		AddMetric("latency", latencyVal).
		AddLabel("ptype", "http")

	rows := emToPGMetrics(em)

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

func Test_generateColumns(t *testing.T) {
	label := "test-label"
	column := "test-column"

	type args struct {
		columns []*configpb.LabelToColumn
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test",
			args: args{
				[]*configpb.LabelToColumn{{
					Label:  &label,
					Column: &column,
				}},
			},
			want: []string{
				"time", "metric_name", "value", "test-column",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateColumns(tt.args.columns); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateColumns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateSortedValues(t *testing.T) {
	label1 := "test-label-1"
	label2 := "test-label-2"
	column1 := "test-column-1"
	column2 := "test-column-2"

	type args struct {
		labels map[string]string
		ltc    []*configpb.LabelToColumn
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test",
			args: args{
				labels: map[string]string{label2: "value-2", label1: "value-1"},
				ltc: []*configpb.LabelToColumn{{
					Label:  &label2,
					Column: &column2,
				}, {
					Label:  &label1,
					Column: &column1,
				}},
			},
			want: []string{"value-1", "value-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateSortedValues(tt.args.labels, tt.args.ltc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateSortedValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
