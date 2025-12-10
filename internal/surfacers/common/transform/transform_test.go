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

package transform

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
)

func TestFailureCountForDefaultMetrics(t *testing.T) {
	var tests = []struct {
		total, success, failure metrics.Value
		wantFailure             *metrics.Int
		wantErr                 bool
	}{
		{
			total:       metrics.NewInt(1000),
			success:     metrics.NewInt(990),
			wantFailure: metrics.NewInt(10),
		},
		{
			total:       metrics.NewInt(1000),
			success:     metrics.NewInt(990),
			failure:     metrics.NewInt(20),
			wantFailure: metrics.NewInt(20), // Not computed, original count.
		},
		{
			total:       metrics.NewInt(1000),
			success:     metrics.NewInt(1000),
			wantFailure: metrics.NewInt(0),
		},
		{
			success: metrics.NewInt(990),
		},
		{
			total: metrics.NewInt(1000),
		},
		{
			total:   metrics.NewInt(1000),
			success: metrics.NewString("100"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
			em := metrics.NewEventMetrics(time.Now()).
				AddMetric("total", test.total).
				AddMetric("success", test.success)

			if test.failure != nil {
				em.AddMetric("failure", test.failure)
			}

			if err := AddFailureMetric(em); err != nil {
				if !test.wantErr {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("didn't get expected error")
			}

			gotFailure := em.Metric("failure")

			if test.wantFailure == nil {
				if gotFailure != nil {
					t.Errorf("Unexpected failure count metric (with value: %d)", gotFailure.(metrics.NumValue).Int64())
				}
				return
			}

			if test.wantFailure != nil && gotFailure == nil {
				t.Errorf("Not creating failure count metric; expected metric with value: %d", test.wantFailure.Int64())
				return
			}

			if gotFailure.(metrics.NumValue).Int64() != test.wantFailure.Int64() {
				t.Errorf("Failure count=%v, want=%v", test.wantFailure.Int64(), test.wantFailure.Int64())
			}
		})
	}
}

func TestCumulativeToGauge(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() (*metrics.EventMetrics, map[string]*metrics.EventMetrics)
		wantVal   int64
		wantCache bool
		wantErr   bool
	}{
		{
			name: "first_reading",
			setup: func() (*metrics.EventMetrics, map[string]*metrics.EventMetrics) {
				em := metrics.NewEventMetrics(time.Now())
				em.AddMetric("test_metric", metrics.NewInt(100))
				em.Kind = metrics.CUMULATIVE
				return em, make(map[string]*metrics.EventMetrics)
			},
			wantVal:   100,
			wantCache: true,
		},
		{
			name: "subsequent_reading",
			setup: func() (*metrics.EventMetrics, map[string]*metrics.EventMetrics) {
				// First reading
				firstEM := metrics.NewEventMetrics(time.Now().Add(-time.Second))
				firstEM.AddMetric("test_metric", metrics.NewInt(100))
				firstEM.Kind = metrics.CUMULATIVE

				// Current reading
				currentEM := metrics.NewEventMetrics(time.Now())
				currentEM.AddMetric("test_metric", metrics.NewInt(150))
				currentEM.Kind = metrics.CUMULATIVE

				// Setup cache with first reading
				cache := make(map[string]*metrics.EventMetrics)
				cache[firstEM.Key()] = firstEM

				return currentEM, cache
			},
			wantVal:   50, // 150 - 100
			wantCache: true,
		},
		{
			name: "error_different_metrics",
			setup: func() (*metrics.EventMetrics, map[string]*metrics.EventMetrics) {
				// Cached metrics with different type
				cachedEM := metrics.NewEventMetrics(time.Now().Add(-time.Second))
				cachedEM.AddMetric("test_metric", metrics.NewString("100"))
				cachedEM.Kind = metrics.CUMULATIVE

				// Current metrics
				currentEM := metrics.NewEventMetrics(time.Now())
				currentEM.AddMetric("test_metric", metrics.NewInt(150))
				currentEM.Kind = metrics.CUMULATIVE

				cache := make(map[string]*metrics.EventMetrics)
				cache[cachedEM.Key()] = cachedEM

				return currentEM, cache
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em, cache := tt.setup()

			result, err := CumulativeToGauge(em, cache, nil)

			if (err != nil) != tt.wantErr {
				t.Fatalf("CumulativeToGauge() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Check the result
			if result.Kind != metrics.GAUGE {
				t.Errorf("Expected result kind to be GAUGE, got %v", result.Kind)
			}

			// Check the metric value
			val := result.Metric("test_metric").(metrics.NumValue).Int64()
			if val != tt.wantVal {
				t.Errorf("Expected metric value %d, got %d", tt.wantVal, val)
			}

			// Check cache was updated
			_, exists := cache[em.Key()]
			if exists != tt.wantCache {
				t.Errorf("Cache update check failed: exists=%v, wantCache=%v", exists, tt.wantCache)
			}
		})
	}
}
