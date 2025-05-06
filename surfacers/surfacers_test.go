// Copyright 2017-2021 The Cloudprober Authors.
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

package surfacers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/config/runconfig"
	"github.com/cloudprober/cloudprober/metrics"
	surfacerpb "github.com/cloudprober/cloudprober/surfacers/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestDefaultConfig(t *testing.T) {
	runconfig.SetDefaultHTTPServeMux(http.NewServeMux())

	surfacers, err := Init(context.Background(), []*surfacerpb.SurfacerDef{})
	if err != nil {
		t.Fatal(err)
	}

	var wantSurfacers, gotSurfacers []string
	for _, s := range defaultSurfacers {
		wantSurfacers = append(wantSurfacers, s.GetType().String())
	}
	wantSurfacers = append(wantSurfacers, surfacerpb.Type_PROBESTATUS.String())

	for _, s := range surfacers {
		gotSurfacers = append(gotSurfacers, s.Type)
	}

	assert.Equal(t, wantSurfacers, gotSurfacers)
}

func TestEmptyConfig(t *testing.T) {
	runconfig.SetDefaultHTTPServeMux(http.NewServeMux())

	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(requiredSurfacers) {
		t.Errorf("Got non-required surfacers for zero config: %v", s)
	}
}

func TestInferType(t *testing.T) {
	typeToConf := map[string]*surfacerpb.SurfacerDef{
		"CLOUDWATCH":  {Surfacer: &surfacerpb.SurfacerDef_CloudwatchSurfacer{}},
		"DATADOG":     {Surfacer: &surfacerpb.SurfacerDef_DatadogSurfacer{}},
		"FILE":        {Surfacer: &surfacerpb.SurfacerDef_FileSurfacer{}},
		"POSTGRES":    {Surfacer: &surfacerpb.SurfacerDef_PostgresSurfacer{}},
		"PROBESTATUS": {Surfacer: &surfacerpb.SurfacerDef_ProbestatusSurfacer{}},
		"PROMETHEUS":  {Surfacer: &surfacerpb.SurfacerDef_PrometheusSurfacer{}},
		"PUBSUB":      {Surfacer: &surfacerpb.SurfacerDef_PubsubSurfacer{}},
		"STACKDRIVER": {Surfacer: &surfacerpb.SurfacerDef_StackdriverSurfacer{}},
		"BIGQUERY":    {Surfacer: &surfacerpb.SurfacerDef_BigquerySurfacer{}},
		"OTEL":        {Surfacer: &surfacerpb.SurfacerDef_OtelSurfacer{}},
	}

	for k := range surfacerpb.Type_value {
		if k == "NONE" || k == "USER_DEFINED" {
			continue
		}
		if typeToConf[k] == nil {
			t.Errorf("Missing infertype test for %s", k)
		}
	}

	for ctype, sdef := range typeToConf {
		t.Run(ctype, func(t *testing.T) {
			stype := inferType(sdef)
			assert.Equal(t, ctype, stype.String())
		})
	}
}

type testSurfacer struct {
	received []*metrics.EventMetrics
}

func (ts *testSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	ts.received = append(ts.received, em)
}

var testEventMetrics = []*metrics.EventMetrics{
	metrics.NewEventMetrics(time.Now()).
		AddMetric("total", metrics.NewInt(20)).
		AddMetric("timeout", metrics.NewInt(2)).
		AddLabel("ptype", "http").
		AddLabel("probe", "google_homepage"),
	metrics.NewEventMetrics(time.Now()).
		AddMetric("memory", metrics.NewInt(20)).
		AddMetric("num_goroutines", metrics.NewInt(2)).
		AddLabel("probe", "sysvars"),
}

func TestUserDefinedAndFiltering(t *testing.T) {
	runconfig.SetDefaultHTTPServeMux(http.NewServeMux())

	ts1, ts2 := &testSurfacer{}, &testSurfacer{}
	Register("s1", ts1)
	Register("s2", ts2)

	configs := []*surfacerpb.SurfacerDef{
		{
			Name: proto.String("s1"),
			Type: surfacerpb.Type_USER_DEFINED.Enum(),
		},
		{
			Name: proto.String("s2"),
			IgnoreMetricsWithLabel: []*surfacerpb.LabelFilter{
				{
					Key:   proto.String("probe"),
					Value: proto.String("sysvars"),
				},
			},
			Type: surfacerpb.Type_USER_DEFINED.Enum(),
		},
	}
	wantSurfacers := []string{"s1", "s2"}

	si, err := Init(context.Background(), configs)
	if err != nil {
		t.Fatalf("Unexpected initialization error: %v", err)
	}

	gotSurfacers := make(map[string]*SurfacerInfo)
	for _, s := range si {
		gotSurfacers[s.Name] = s
	}

	for _, name := range wantSurfacers {
		if gotSurfacers[name] == nil {
			t.Errorf("Didn't get the surfacer: %s, all surfacers: %v", name, gotSurfacers)
		}
	}

	for _, em := range testEventMetrics {
		for _, s := range si {
			s.Surfacer.Write(context.Background(), em)
		}
	}

	wantEventMetrics := [][]*metrics.EventMetrics{
		testEventMetrics,      // No filtering.
		testEventMetrics[0:1], // One EM is ignored for the 2nd surfacer.
	}

	for i, ts := range []*testSurfacer{ts1, ts2} {
		wantEMs := wantEventMetrics[i]
		assert.Equal(t, len(wantEMs), len(ts.received))
		for i, em := range wantEMs {
			assert.Equal(t, em.String(), ts.received[i].String())
		}
	}
}

func TestFailureMetric(t *testing.T) {
	runconfig.SetDefaultHTTPServeMux(http.NewServeMux())

	ts1, ts2 := &testSurfacer{}, &testSurfacer{}
	Register("s1", ts1)
	Register("s2", ts2)

	var testEventMetrics = []*metrics.EventMetrics{
		metrics.NewEventMetrics(time.Now()).
			AddMetric("total", metrics.NewInt(20)).
			AddMetric("success", metrics.NewInt(18)).
			AddMetric("timeout", metrics.NewInt(2)).
			AddLabel("ptype", "http"),
		metrics.NewEventMetrics(time.Now()).
			AddMetric("num_goroutines", metrics.NewInt(2)),
	}

	configs := []*surfacerpb.SurfacerDef{
		{
			Name: proto.String("s1"),
			Type: surfacerpb.Type_USER_DEFINED.Enum(),
		},
		{
			Name:             proto.String("s2"),
			Type:             surfacerpb.Type_USER_DEFINED.Enum(),
			AddFailureMetric: proto.Bool(true),
		},
	}

	si, err := Init(context.Background(), configs)
	if err != nil {
		t.Fatalf("Unexpected initialization error: %v", err)
	}

	for _, em := range testEventMetrics {
		for _, s := range si {
			s.Surfacer.Write(context.Background(), em)
		}
	}

	wantEventMetrics := [][]*metrics.EventMetrics{
		testEventMetrics, // s1
		{
			testEventMetrics[0].Clone().
				AddMetric("failure", metrics.NewInt(2)),
			testEventMetrics[1], // unchanged
		}, // s2
	}

	for i, ts := range []*testSurfacer{ts1, ts2} {
		wantEMs := wantEventMetrics[i]
		assert.Equal(t, len(wantEMs), len(ts.received))
		for i, em := range wantEMs {
			assert.Equal(t, em.String(), ts.received[i].String())
		}
	}
}
