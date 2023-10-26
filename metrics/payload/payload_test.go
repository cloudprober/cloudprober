// Copyright 2017-2022 The Cloudprober Authors.
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

package payload

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var (
	testPtype  = "external"
	testProbe  = "testprobe"
	testTarget = "test-target"
)

func parserForTest(t *testing.T, agg bool, additionalLabels string) *Parser {
	testConf := `
	  aggregate_in_cloudprober: %v
		dist_metric {
			key: "op_latency"
			value {
				explicit_buckets: "1,10,100"
			}
		}
 `

	var c configpb.OutputMetricsOptions
	if err := prototext.Unmarshal([]byte(fmt.Sprintf(testConf, agg)), &c); err != nil {
		t.Error(err)
	}
	if additionalLabels != "" {
		c.AdditionalLabels = proto.String(additionalLabels)
	}
	p, err := NewParser(&c, testPtype, testProbe, metrics.CUMULATIVE, nil)
	if err != nil {
		t.Error(err)
	}

	return p
}

// testData encapsulates the test data.
type testData struct {
	varA, varB float64
	lat        []float64
	labels     [3][2]string
}

// multiEM returns EventMetrics structs corresponding to the provided testData.
func (td *testData) multiEM(ts time.Time) []*metrics.EventMetrics {
	var results []*metrics.EventMetrics

	d := metrics.NewDistribution([]float64{1, 10, 100})
	for _, sample := range td.lat {
		d.AddSample(sample)
	}

	results = append(results, []*metrics.EventMetrics{
		metrics.NewEventMetrics(ts).AddMetric("op_latency", d),
		metrics.NewEventMetrics(ts).AddMetric("time_to_running", metrics.NewFloat(td.varA)),
		metrics.NewEventMetrics(ts).AddMetric("time_to_ssh", metrics.NewFloat(td.varB)),
	}...)

	for i, em := range results {
		em.AddLabel("ptype", testPtype).
			AddLabel("probe", testProbe).
			AddLabel("dst", testTarget)

		if td.labels[i][0] != "" {
			em.AddLabel(td.labels[i][0], td.labels[i][1])
		}
	}

	return results
}

func (td *testData) testPayload(quoteLabelValues bool) string {
	var labelStrs [3]string
	for i, kv := range td.labels {
		if kv[0] == "" {
			continue
		}
		if quoteLabelValues {
			labelStrs[i] = fmt.Sprintf("{%s=\"%s\"}", kv[0], kv[1])
		} else {
			labelStrs[i] = fmt.Sprintf("{%s=%s}", kv[0], kv[1])
		}
	}

	var latencyStrs []string
	for _, f := range td.lat {
		latencyStrs = append(latencyStrs, fmt.Sprintf("%f", f))
	}
	payloadLines := []string{
		fmt.Sprintf("op_latency%s %s", labelStrs[0], strings.Join(latencyStrs, ",")),
		fmt.Sprintf("time_to_running%s %f", labelStrs[1], td.varA),
		fmt.Sprintf("time_to_ssh%s %f", labelStrs[2], td.varB),
	}
	return strings.Join(payloadLines, "\n")
}

func testPayloadMetrics(t *testing.T, p *Parser, etd *testData) {
	t.Helper()

	ems := p.PayloadMetrics(etd.testPayload(false), testTarget)
	expectedMetrics := etd.multiEM(ems[0].Timestamp)
	for i, em := range ems {
		if em.String() != expectedMetrics[i].String() {
			t.Errorf("Output metrics not correct:\nGot:      %s\nExpected: %s", em.String(), expectedMetrics[i].String())
		}
	}

	// Test with quoted label values
	ems = p.PayloadMetrics(etd.testPayload(true), testTarget)
	for i, em := range ems {
		if em.String() != expectedMetrics[i].String() {
			t.Errorf("Output metrics not correct:\nGot:      %s\nExpected: %s", em.String(), expectedMetrics[i].String())
		}
	}
}

func TestAdditionalLabels(t *testing.T) {
	p := parserForTest(t, false, "alk=alv")
	ems := p.PayloadMetrics("run_count{dc=xx} 1", testTarget)
	if len(ems) != 1 {
		t.Fatalf("Got %d EventMetrics, wanted only 1", len(ems))
	}
	em := ems[0]
	wantMetric := fmt.Sprintf("%d labels=ptype=%s,probe=%s,alk=alv,dst=%s,dc=xx run_count=1.000", em.Timestamp.Unix(), testPtype, testProbe, testTarget)
	if em.String() != wantMetric {
		t.Errorf("Output metrics not correct:\nGot:      %s\nExpected: %s", em.String(), wantMetric)
	}
}

func TestNoAggregation(t *testing.T) {
	p := parserForTest(t, false, "")

	// First payload
	td := &testData{
		varA: 10,
		varB: 30,
		lat:  []float64{3.1, 4.0, 13},
		labels: [3][2]string{
			{"l1", "v1"},
			{"l2", "v2"},
			{"l3", "v3"},
		},
	}
	testPayloadMetrics(t, p, td)

	// Send another payload, cloudprober should not aggregate the metrics.
	td = &testData{
		varA: 8,
		varB: 45,
		lat:  []float64{6, 14.1, 2.1},
		labels: [3][2]string{
			{"l1", "v1"},
			{"l2", "v2"},
			{"l3", "v3"},
		},
	}
	testPayloadMetrics(t, p, td)
}

func TestAggreagateInCloudprober(t *testing.T) {
	tests := []struct {
		desc        string
		payload     []string
		wantMetrics []string
	}{
		{
			desc: "first-run",
			payload: []string{
				"run_count 1",
				"queries{dc=xx} 100",
				"queries{dc=yy} 100",
				"op_latency{op=get,dc=xx} 12",
				"op_latency{op=set,dc=xx} 24",
				"req_latency{dc=xx} dist:sum:42|count:1|lb:-Inf,1,10,25,100|bc:0,0,0,1,0",
			},
			wantMetrics: []string{
				" run_count=1.000",
				",dc=xx queries=100.000",
				",dc=yy queries=100.000",
				",op=get,dc=xx op_latency=dist:sum:12|count:1|lb:-Inf,1,10,100|bc:0,0,1,0",
				",op=set,dc=xx op_latency=dist:sum:24|count:1|lb:-Inf,1,10,100|bc:0,0,1,0",
				",dc=xx req_latency=dist:sum:42|count:1|lb:-Inf,1,10,25,100|bc:0,0,0,1,0",
			},
		},
		{
			desc: "second-run",
			payload: []string{
				"run_count 1",
				"queries{dc=xx} 99",
				"queries{dc=yy} 100",
				"op_latency{op=get,dc=xx} 11",
				"op_latency{dc=xx,op=set} 23",
				"req_latency{dc=xx} dist:sum:32|count:1|lb:-Inf,1,10,25,100|bc:0,0,0,1,0",
			},
			wantMetrics: []string{
				" run_count=2.000",
				",dc=xx queries=199.000",
				",dc=yy queries=200.000",
				",op=get,dc=xx op_latency=dist:sum:23|count:2|lb:-Inf,1,10,100|bc:0,0,2,0",
				",op=set,dc=xx op_latency=dist:sum:47|count:2|lb:-Inf,1,10,100|bc:0,0,2,0",
				",dc=xx req_latency=dist:sum:74|count:2|lb:-Inf,1,10,25,100|bc:0,0,0,2,0",
			},
		},
	}

	p := parserForTest(t, true, "")

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ems := p.PayloadMetrics(strings.Join(test.payload, "\n"), testTarget)
			for i, em := range ems {
				wantMetric := fmt.Sprintf("%d labels=ptype=%s,probe=%s,dst=%s%s", em.Timestamp.Unix(), testPtype, testProbe, testTarget, test.wantMetrics[i])
				if em.String() != wantMetric {
					t.Errorf("Output metrics not correct:\nGot:      %s\nExpected: %s", em.String(), wantMetric)
				}
			}
		})

	}
}

func TestMetricValueLabels(t *testing.T) {
	tests := []struct {
		desc    string
		line    string
		metric  string
		labels  [][2]string
		value   string
		wantErr bool
	}{
		{
			desc:   "metric with no labels",
			line:   "total 56",
			metric: "total",
			value:  "56",
		},
		{
			desc:   "metric with no labels, but more spaces",
			line:   "total   56",
			metric: "total",
			value:  "56",
		},
		{
			desc:   "standard metric with labels",
			line:   "total{service=serviceA,dc=xx} 56",
			metric: "total",
			labels: [][2]string{
				{"service", "serviceA"},
				{"dc", "xx"},
			},
			value: "56",
		},
		{
			desc:   "quoted labels, same result",
			line:   "total{service=\"serviceA\",dc=\"xx\"} 56",
			metric: "total",
			labels: [][2]string{
				{"service", "serviceA"},
				{"dc", "xx"},
			},
			value: "56",
		},
		{
			desc:   "a label value has space and more spaces",
			line:   "total{service=\"service A\", dc= \"xx\"} 56",
			metric: "total",
			labels: [][2]string{
				{"service", "service A"},
				{"dc", "xx"},
			},
			value: "56",
		},
		{
			desc:   "label and value have a space",
			line:   "version{service=\"service A\",dc=xx} \"version 1.5\"",
			metric: "version",
			labels: [][2]string{
				{"service", "service A"},
				{"dc", "xx"},
			},
			value: "\"version 1.5\"",
		},
		{
			desc:    "only one brace, invalid line",
			line:    "total{service=\"service A\",dc=\"xx\" 56",
			wantErr: true,
		},
	}

	p := &Parser{}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			m, v, l, err := p.metricValueLabels(test.line)
			if err != nil {
				if !test.wantErr {
					t.Errorf("Unexpected error: %v", err)
				}
				return
			}
			if err == nil && test.wantErr {
				t.Errorf("Expected error, but didn't get any.")
			}
			if m != test.metric {
				t.Errorf("Metric name: got=%s, wanted=%s", m, test.metric)
			}
			if v != test.value {
				t.Errorf("Metric value: got=%s, wanted=%s", v, test.value)
			}
			if !reflect.DeepEqual(l, test.labels) {
				t.Errorf("Metric labels: got=%v, wanted=%v", l, test.labels)
			}
		})
	}
}

func BenchmarkMetricValueLabels(b *testing.B) {
	payload := []string{
		"total 50",
		"total   56",
		"total{service=serviceA,dc=xx} 56",
		"total{service=\"serviceA\",dc=\"xx\"} 56",
		"version{service=\"service A\",dc=xx} \"version 1.5\"",
	}
	// run the em.String() function b.N times
	for n := 0; n < b.N; n++ {
		p := &Parser{}
		for _, s := range payload {
			p.metricValueLabels(s)
		}
	}
}
