// Copyright 2021 The Cloudprober Authors.
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

package cloudwatch

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/surfacers/cloudwatch/proto"
)

func newTestCWSurfacer() CWSurfacer {
	l, _ := logger.New(context.TODO(), "test-logger")
	namespace := "sre/test/cloudprober"
	resolution := int32(60)

	return CWSurfacer{
		l: l,
		c: &configpb.SurfacerConf{
			Namespace:  &namespace,
			Resolution: &resolution,
		},
	}
}

func TestGetRegion(t *testing.T) {
	tests := map[string]struct {
		surfacer CWSurfacer
		region   string
		want     string
	}{
		"us-east-1_config": {
			surfacer: newTestCWSurfacer(),
			region:   "us-east-1",
			want:     "us-east-1",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.region != "" {
				tc.surfacer.c.Region = &tc.region
			}

			got := getRegion(tc.surfacer.c)
			if got != tc.want {
				t.Errorf("got: %v, want: %v", got, tc.want)
			}
		})
	}
}

func TestGetRegionDefault(t *testing.T) {
	s := newTestCWSurfacer()
	got := getRegion(s.c)
	if got != "" {
		t.Errorf("got: %v, want: \"\"", got)
	}
}

func TestEmLabelsToDimensions(t *testing.T) {
	timestamp := time.Now()

	tests := map[string]struct {
		em   *metrics.EventMetrics
		want []types.Dimension
	}{
		"no label": {
			em:   metrics.NewEventMetrics(timestamp),
			want: []types.Dimension{},
		},
		"one label": {
			em: metrics.NewEventMetrics(timestamp).
				AddLabel("ptype", "sysvars"),
			want: []types.Dimension{
				{
					Name:  aws.String("ptype"),
					Value: aws.String("sysvars"),
				},
			},
		},
		"three labels": {
			em: metrics.NewEventMetrics(timestamp).
				AddLabel("ptype", "sysvars").
				AddLabel("probe", "sysvars").
				AddLabel("test", "testing123"),
			want: []types.Dimension{
				{
					Name:  aws.String("ptype"),
					Value: aws.String("sysvars"),
				},
				{
					Name:  aws.String("probe"),
					Value: aws.String("sysvars"),
				},
				{
					Name:  aws.String("test"),
					Value: aws.String("testing123"),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := emLabelsToDimensions(tc.em)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got: %v, want: %v", got, tc.want)
			}
		})
	}
}

func TestNewCWMetricDatum(t *testing.T) {
	timestamp := time.Now()

	tests := map[string]struct {
		surfacer   CWSurfacer
		metricname string
		value      float64
		dimensions []types.Dimension
		timestamp  time.Time
		duration   time.Duration
		want       types.MetricDatum
	}{
		"simple": {
			surfacer:   newTestCWSurfacer(),
			metricname: "testingmetric",
			value:      float64(20),
			dimensions: []types.Dimension{
				{
					Name: aws.String("test"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			want: types.MetricDatum{
				Dimensions: []types.Dimension{
					{
						Name: aws.String("test"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("testingmetric"),
				Value:             aws.Float64(float64(20)),
				StorageResolution: aws.Int32(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              types.StandardUnitCount,
			},
		},
		"le_dimension_count_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "testingmetric",
			value:      float64(20),
			dimensions: []types.Dimension{
				{
					Name: aws.String("test"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			want: types.MetricDatum{
				Dimensions: []types.Dimension{
					{
						Name: aws.String("test"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("testingmetric"),
				Value:             aws.Float64(float64(20)),
				StorageResolution: aws.Int32(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              types.StandardUnitCount,
			},
		},
		"latency_name_nanosecond_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []types.Dimension{
				{
					Name: aws.String("name"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			duration:  time.Nanosecond,
			want: types.MetricDatum{
				Dimensions: []types.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(0.00002),
				StorageResolution: aws.Int32(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              types.StandardUnitMilliseconds,
			},
		},
		"latency_name_microseconds_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []types.Dimension{
				{
					Name: aws.String("name"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			duration:  time.Microsecond,
			want: types.MetricDatum{
				Dimensions: []types.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(0.02),
				StorageResolution: aws.Int32(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              types.StandardUnitMilliseconds,
			},
		},
		"latency_name_milliseconds_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []types.Dimension{
				{
					Name: aws.String("name"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			duration:  time.Millisecond,
			want: types.MetricDatum{
				Dimensions: []types.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(20),
				StorageResolution: aws.Int32(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              types.StandardUnitMilliseconds,
			},
		},
		"latency_name_seconds_unit": {
			surfacer:   newTestCWSurfacer(),
			metricname: "latency",
			value:      float64(20),
			dimensions: []types.Dimension{
				{
					Name: aws.String("name"), Value: aws.String("value"),
				},
			},
			timestamp: timestamp,
			duration:  time.Second,
			want: types.MetricDatum{
				Dimensions: []types.Dimension{
					{
						Name: aws.String("name"), Value: aws.String("value"),
					},
				},
				MetricName:        aws.String("latency"),
				Value:             aws.Float64(20000),
				StorageResolution: aws.Int32(60),
				Timestamp:         aws.Time(timestamp),
				Unit:              types.StandardUnitMilliseconds,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.surfacer.newCWMetricDatum(
				tc.metricname,
				tc.value,
				tc.dimensions,
				tc.timestamp,
				tc.duration,
			)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got: %v, want: %v", got, tc.want)
			}
		})
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
