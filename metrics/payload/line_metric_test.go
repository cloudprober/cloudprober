// Copyright 2017-2024 The Cloudprober Authors.
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
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/metrics/payload/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var (
	testTarget = "test-target"
)

func parserForTest(t *testing.T, agg bool, additionalLabels string) *Parser {
	testConf := `
	  metrics_kind: CUMULATIVE
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
	p, err := NewParser(&c, nil)
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
		if td.labels[i][0] != "" {
			em.AddLabel(td.labels[i][0], td.labels[i][1])
		}
	}

	return results
}

func (td *testData) testPayload(quoteLabelValues bool) []byte {
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
	return []byte(strings.Join(payloadLines, "\n"))
}

func testPayloadMetrics(t *testing.T, p *Parser, etd *testData) {
	t.Helper()

	ems := p.PayloadMetrics(&Input{Text: etd.testPayload(false)}, testTarget)
	expectedMetrics := etd.multiEM(ems[0].Timestamp)
	for i, em := range ems {
		if em.String() != expectedMetrics[i].String() {
			t.Errorf("Output metrics not correct:\nGot:      %s\nExpected: %s", em.String(), expectedMetrics[i].String())
		}
	}

	// Test with quoted label values
	ems = p.PayloadMetrics(&Input{Text: etd.testPayload(true)}, testTarget)
	for i, em := range ems {
		if em.String() != expectedMetrics[i].String() {
			t.Errorf("Output metrics not correct:\nGot:      %s\nExpected: %s", em.String(), expectedMetrics[i].String())
		}
	}
}

func TestAdditionalLabels(t *testing.T) {
	p := parserForTest(t, false, "alk=alv")
	ems := p.PayloadMetrics(&Input{Text: []byte("run_count{dc=xx} 1")}, testTarget)
	if len(ems) != 1 {
		t.Fatalf("Got %d EventMetrics, wanted only 1", len(ems))
	}
	em := ems[0]
	wantMetric := fmt.Sprintf("%d labels=alk=alv,dc=xx run_count=1.000", em.Timestamp.Unix())
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
				"dc=xx queries=100.000",
				"dc=yy queries=100.000",
				"op=get,dc=xx op_latency=dist:sum:12|count:1|lb:-Inf,1,10,100|bc:0,0,1,0",
				"op=set,dc=xx op_latency=dist:sum:24|count:1|lb:-Inf,1,10,100|bc:0,0,1,0",
				"dc=xx req_latency=dist:sum:42|count:1|lb:-Inf,1,10,25,100|bc:0,0,0,1,0",
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
				"dc=xx queries=199.000",
				"dc=yy queries=200.000",
				"op=get,dc=xx op_latency=dist:sum:23|count:2|lb:-Inf,1,10,100|bc:0,0,2,0",
				"op=set,dc=xx op_latency=dist:sum:47|count:2|lb:-Inf,1,10,100|bc:0,0,2,0",
				"dc=xx req_latency=dist:sum:74|count:2|lb:-Inf,1,10,25,100|bc:0,0,0,2,0",
			},
		},
	}

	p := parserForTest(t, true, "")

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ems := p.PayloadMetrics(&Input{Text: []byte(strings.Join(test.payload, "\n"))}, testTarget)
			for i, em := range ems {
				wantMetric := fmt.Sprintf("%d labels=%s", em.Timestamp.Unix(), test.wantMetrics[i])
				if em.String() != wantMetric {
					t.Errorf("Output metrics not correct:\nGot:      %s\nExpected: %s", em.String(), wantMetric)
				}
			}
		})

	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		desc       string
		line       string
		wantMetric string
		wantLabels [][2]string
		wantValue  string
		wantErr    bool
	}{
		{
			desc:       "metric with no labels",
			line:       "op_total 56",
			wantMetric: "op_total",
			wantValue:  "56",
		},
		{
			desc:       "metric with no labels, but more spaces",
			line:       "op_total   56",
			wantMetric: "op_total",
			wantValue:  "56",
		},
		{
			desc:       "standard metric with labels",
			line:       "op_total{svc=serviceA,dc=xx} 56",
			wantMetric: "op_total",
			wantLabels: [][2]string{{"svc", "serviceA"}, {"dc", "xx"}},
			wantValue:  "56",
		},
		{
			desc:       "quoted labels, same result",
			line:       "op_total{svc=\"serviceA\",dc=\"xx\"} 56",
			wantMetric: "op_total",
			wantLabels: [][2]string{{"svc", "serviceA"}, {"dc", "xx"}},
			wantValue:  "56",
		},
		{
			desc:       "a label value has space and more spaces",
			line:       "op_total{svc=\"svc A\", dcs= \"xx,yy\"} 56",
			wantMetric: "op_total",
			wantLabels: [][2]string{{"svc", "svc A"}, {"dcs", "xx,yy"}},
			wantValue:  "56",
		},
		{
			desc:       "label and value have a space",
			line:       "version{svc=\"svc A\",dc=xx} \"version 1.5\"",
			wantMetric: "version",
			wantLabels: [][2]string{{"svc", "svc A"}, {"dc", "xx"}},
			wantValue:  "\"version 1.5\"",
		},
		{
			desc:    "invalid line",
			line:    "op_total ",
			wantErr: true,
		},
		{
			desc:    "only one brace, invalid line",
			line:    "total{svc=\"svc A\",dc=\"xx\" 56",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			m, v, l, err := parseLine(test.line)
			if err != nil {
				if !test.wantErr {
					t.Errorf("Unexpected error: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("Expected error, but didn't get any.")
			}
			assert.Equal(t, test.wantMetric, m)
			assert.Equal(t, test.wantLabels, l)
			assert.Equal(t, test.wantValue, v)
		})
	}
}

func TestParseLabels(t *testing.T) {
	invalidLabelLines := []string{
		`svc=A,dc="xx"56`,    // invalid dc label value
		`svc=A,dc=44"xx"`,    // invalid dc label value
		`svc=A,dc=44 "xx"`,   // invalid dc label value
		`svc=A,dc=xx" 56`,    // spurious quote
		`svcs=""A, B",dc=xx`, // single doublequote in quoted value

		// Bad character in unquoted value
		`svc=A,dc=xx 56`,
		`svc=A,dc=xx,56`,
		`svc=A,dc=x/x,`,
		`svc=A,dc=x\x,`,
		`svc=A,dc=x:x,`,
	}
	for _, line := range invalidLabelLines {
		labels, err := parseLabels(line)
		if err == nil {
			t.Errorf("Expected error, got nil, labels: %v", labels)
		}
	}

	inputToLabels := map[string][][2]string{
		"":             nil,
		"svc=,dc=":     {{"svc", ""}, {"dc", ""}},
		"svc=A,dc=":    {{"svc", "A"}, {"dc", ""}},
		"svc=,dc=xx ":  {{"svc", ""}, {"dc", "xx"}},
		"svc=, dc=xx ": {{"svc", ""}, {"dc", "xx"}},
		"svc=A,dc=xx":  {{"svc", "A"}, {"dc", "xx"}},
		"svc=A,dc=xx,": {{"svc", "A"}, {"dc", "xx"}},

		// More allowed characters inside quotes
		`svc=A,dc="x/y"`:                {{"svc", "A"}, {"dc", "x/y"}},
		`svc="svc A",dc="xx",`:          {{"svc", "svc A"}, {"dc", "xx"}},
		`svc="svc	A",dc="xx",`:          {{"svc", "svc	A"}, {"dc", "xx"}},
		`svcs="svc A, svc B",dc=xx`:     {{"svcs", "svc A, svc B"}, {"dc", "xx"}},
		`svcs="svc \"A\", svc B",dc=xx`: {{"svcs", `svc \"A\", svc B`}, {"dc", "xx"}},

		// Unquoted allowed chars
		"svc=A_B,dc=xx": {{"svc", "A_B"}, {"dc", "xx"}},
		"svc=A-B,dc=xx": {{"svc", "A-B"}, {"dc", "xx"}},
		"svc=A.B,dc=xx": {{"svc", "A.B"}, {"dc", "xx"}},
	}
	for input, wantLabels := range inputToLabels {
		labels, err := parseLabels(input)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		assert.Equal(t, wantLabels, labels, "input: %v", input)
	}
}

func BenchmarkMetricValueLabels(b *testing.B) {
	payload := []string{
		"total 50",
		"total   56",
		"total{svc=serviceA,dc=xx} 56",
		"total{svc=\"serviceA\",dc=\"xx\"} 56",
		"version{svc=\"svc A\",dc=xx} \"version 1.5\"",
	}
	// run the em.String() function b.N times
	for n := 0; n < b.N; n++ {
		for _, s := range payload {
			parseLine(s)
		}
	}
}

func TestAcceptRejectRegex(t *testing.T) {
	ts := time.Now()
	tsString := fmt.Sprintf("%d", ts.Unix())
	input := "op_total{probe=\"p1\"} 50\nop_success{probe=\"p1\"} 48"
	tests := []struct {
		desc     string
		acceptRe string
		rejectRe string
		wantEM   []string
	}{
		{
			desc:   "no regex",
			wantEM: []string{"labels=probe=p1 op_total=50.000", "labels=probe=p1 op_success=48.000"},
		},
		{
			desc:     "accept regex",
			acceptRe: "op_total",
			wantEM:   []string{"labels=probe=p1 op_total=50.000"},
		},
		{
			desc:     "reject regex",
			rejectRe: "op_total",
			wantEM:   []string{"labels=probe=p1 op_success=48.000"},
		},
		{
			desc:     "accept and reject regex",
			acceptRe: "op_success",
			rejectRe: "op_total",
			wantEM:   []string{"labels=probe=p1 op_success=48.000"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			c := &configpb.OutputMetricsOptions{
				LineAcceptRegex: proto.String(test.acceptRe),
				LineRejectRegex: proto.String(test.rejectRe),
			}
			p, err := NewParser(c, nil)
			if err != nil {
				t.Error(err)
			}

			ems := p.PayloadMetrics(&Input{Text: []byte(input)}, testTarget)

			got := make([]string, len(ems))
			for i, em := range ems {
				em.Timestamp = ts
				got[i] = em.String()
			}
			for i := range test.wantEM {
				test.wantEM[i] = tsString + " " + test.wantEM[i]
			}
			assert.Equal(t, test.wantEM, got)
		})
	}
}
