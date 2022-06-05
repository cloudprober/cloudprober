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
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	fileconfigpb "github.com/cloudprober/cloudprober/surfacers/file/proto"
	surfacerpb "github.com/cloudprober/cloudprober/surfacers/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func resetHTTPHandlers(t *testing.T) {
	t.Helper()
	http.DefaultServeMux = new(http.ServeMux)
}

func TestDefaultConfig(t *testing.T) {
	resetHTTPHandlers(t)

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
	resetHTTPHandlers(t)

	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(requiredSurfacers) {
		t.Errorf("Got non-required surfacers for zero config: %v", s)
	}
}

func TestInferType(t *testing.T) {
	resetHTTPHandlers(t)

	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatalf("error creating tempfile for test")
	}

	defer os.Remove(tmpfile.Name()) // clean up

	s, err := Init(context.Background(), []*surfacerpb.SurfacerDef{
		{
			Surfacer: &surfacerpb.SurfacerDef_FileSurfacer{
				FileSurfacer: &fileconfigpb.SurfacerConf{
					FilePath: proto.String(tmpfile.Name()),
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(s) < 1 {
		t.Errorf("len(s)=%d, expected>=1", len(s))
	}

	if s[0].Type != "FILE" {
		t.Errorf("Surfacer type: %s, expected: FILE", s[0].Type)
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
	resetHTTPHandlers(t)

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
		if !reflect.DeepEqual(ts.received, wantEMs) {
			t.Errorf("ts[%d]: Received EventMetrics: %v, want EventMetrics: %v", i, ts.received, wantEMs)
		}
	}
}

func TestFailureMetric(t *testing.T) {
	resetHTTPHandlers(t)

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
		if !reflect.DeepEqual(ts.received, wantEMs) {
			t.Errorf("ts[%d]: Received EventMetrics: %v, want EventMetrics: %v", i, ts.received, wantEMs)
		}
	}
}
