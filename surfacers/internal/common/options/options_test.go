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

package options

import (
	"net/http"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/state"
	configpb "github.com/cloudprober/cloudprober/surfacers/proto"
	surfacerpb "github.com/cloudprober/cloudprober/surfacers/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

var testEventMetrics = []*metrics.EventMetrics{
	metrics.NewEventMetrics(time.Now()).
		AddMetric("total", metrics.NewInt(20)).
		AddMetric("timeout", metrics.NewInt(2)).
		AddLabel("ptype", "http").
		AddLabel("probe", "manugarg_homepage"),
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

func TestAllowEventMetrics(t *testing.T) {
	tests := []struct {
		desc         string
		allowFilter  [][2]string
		ignoreFilter [][2]string
		wantAllowed  []int
		wantErr      bool
	}{
		{
			desc:        "all",
			wantAllowed: []int{0, 1, 2},
		},
		{
			desc: "ignore-sysvars-and-google-homepage",
			ignoreFilter: [][2]string{
				{"probe", "sysvars"},
				{"probe", "google_homepage"},
			},
			wantAllowed: []int{0},
		},
		{
			desc: "allow-google-homepage-and-sysvars",
			allowFilter: [][2]string{
				{"probe", "sysvars"},
				{"probe", "google_homepage"},
			},
			wantAllowed: []int{1, 2},
		},
		{
			desc: "ignore-takes-precedence-for-sysvars",
			allowFilter: [][2]string{
				{"probe", "sysvars"},
				{"probe", "google_homepage"},
			},
			ignoreFilter: [][2]string{
				{"probe", "sysvars"},
			},
			wantAllowed: []int{1},
		},
		{
			desc:        "error-label-value-without-key",
			allowFilter: [][2]string{{"", "sysvars"}},
			wantErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &configpb.SurfacerDef{}
			for _, ignoreF := range test.ignoreFilter {
				config.IgnoreMetricsWithLabel = append(config.IgnoreMetricsWithLabel, &configpb.LabelFilter{Key: proto.String(ignoreF[0]), Value: proto.String(ignoreF[1])})
			}
			for _, allowF := range test.allowFilter {
				config.AllowMetricsWithLabel = append(config.AllowMetricsWithLabel, &configpb.LabelFilter{Key: proto.String(allowF[0]), Value: proto.String(allowF[1])})
			}

			opts, err := BuildOptionsFromConfig(config, nil)
			if err != nil {
				if !test.wantErr {
					t.Errorf("Unexpected building options from the config: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("Expected error, but there were none")
				return
			}

			var gotEM []int
			for i, em := range testEventMetrics {
				if opts.AllowEventMetrics(em) {
					gotEM = append(gotEM, i)
				}
			}

			if !reflect.DeepEqual(gotEM, test.wantAllowed) {
				t.Errorf("Got EMs (index): %v, want EMs (index): %v", gotEM, test.wantAllowed)
			}
		})
	}
}

func TestAllowMetric(t *testing.T) {
	tests := []struct {
		desc           string
		metricName     []string
		allow          string
		ignore         string
		disableFailure bool
		wantMetrics    []string
		wantErr        bool
	}{
		{
			desc:        "all",
			metricName:  []string{"total", "success"},
			wantMetrics: []string{"total", "success"},
		},
		{
			desc:       "bad-allow-regex",
			metricName: []string{"total", "success"},
			allow:      "(?badRe)",
			wantErr:    true,
		},
		{
			desc:       "bad-ignore-regex",
			metricName: []string{"total", "success"},
			ignore:     "(?badRe)",
			wantErr:    true,
		},
		{
			desc:        "ignore-total",
			metricName:  []string{"total", "success", "failure"},
			ignore:      "tot.*",
			wantMetrics: []string{"success", "failure"},
		},
		{
			desc:        "allow-total",
			metricName:  []string{"total", "success", "failure"},
			allow:       "tot.*",
			wantMetrics: []string{"total"},
		},
		{
			desc:           "disable failure metric",
			metricName:     []string{"total", "success", "failure"},
			disableFailure: true,
			wantMetrics:    []string{"total", "success"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &configpb.SurfacerDef{
				IgnoreMetricsWithName: proto.String(test.ignore),
				AllowMetricsWithName:  proto.String(test.allow),
			}
			if test.disableFailure {
				config.AddFailureMetric = proto.Bool(false)
			}

			opts, err := BuildOptionsFromConfig(config, nil)
			if err != nil {
				if !test.wantErr {
					t.Errorf("Unexpected building options from the config: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("Expected error, but there were none")
				return
			}

			var gotMetrics []string
			for _, m := range test.metricName {
				if opts.AllowMetric(m) {
					gotMetrics = append(gotMetrics, m)
				}
			}

			if !reflect.DeepEqual(gotMetrics, test.wantMetrics) {
				t.Errorf("Got metrics: %v, wanted: %v", gotMetrics, test.wantMetrics)
			}
		})
	}
}

func TestMain(m *testing.M) {
	state.SetDefaultHTTPServeMux(http.NewServeMux())
	code := m.Run()
	state.SetDefaultHTTPServeMux(nil)
	os.Exit(code)
}

func TestBuildOptions(t *testing.T) {
	tests := []struct {
		name    string
		sdef    *surfacerpb.SurfacerDef
		want    *Options
		wantErr bool
	}{
		{
			name: "default",
			sdef: &surfacerpb.SurfacerDef{
				Type: configpb.Type_DATADOG.Enum(),
			},
			want: &Options{AddFailureMetric: true},
		},
		{
			name: "default_file",
			sdef: &surfacerpb.SurfacerDef{
				Type: configpb.Type_FILE.Enum(),
			},
			want: &Options{AddFailureMetric: true},
		},
		{
			name: "disable_failure_metric_dd",
			sdef: &surfacerpb.SurfacerDef{
				Type:             configpb.Type_DATADOG.Enum(),
				AddFailureMetric: proto.Bool(false),
			},
			want: &Options{
				AddFailureMetric: false,
			},
		},
		{
			name: "enable_failure_metric_file",
			sdef: &surfacerpb.SurfacerDef{
				Type:             configpb.Type_FILE.Enum(),
				AddFailureMetric: proto.Bool(true),
			},
			want: &Options{AddFailureMetric: true},
		},
		{
			name: "custom_latency_regex",
			sdef: &surfacerpb.SurfacerDef{
				Type:                 configpb.Type_DATADOG.Enum(),
				LatencyMetricPattern: proto.String("latency_.*"),
			},
			want: &Options{AddFailureMetric: true, latencyMetricRe: regexp.MustCompile("latency_.*")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want.Config = tt.sdef

			if tt.want.MetricsBufferSize == 0 {
				tt.want.MetricsBufferSize = 10000
			}
			if tt.want.HTTPServeMux == nil {
				tt.want.HTTPServeMux = state.DefaultHTTPServeMux()
			}

			if tt.want.latencyMetricRe == nil {
				tt.want.latencyMetricRe = regexp.MustCompile("^(.+_|)latency$")
			}

			got, err := buildOptions(tt.sdef, true, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOptionsIsLatencyMetric(t *testing.T) {
	tests := []struct {
		name       string
		opts       *Options
		metricName []string
		want       []bool
	}{
		{
			name:       "nil",
			metricName: []string{"latency", "dns_latency", "latency_read"},
			want:       []bool{true, true, false},
		},
		{
			name:       "non-default",
			opts:       &Options{latencyMetricRe: regexp.MustCompile("latency_.*")},
			metricName: []string{"latency", "dns_latency", "latency_read"},
			want:       []bool{false, false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, m := range tt.metricName {
				assert.Equal(t, tt.want[i], tt.opts.IsLatencyMetric(m), "metricName: %s", m)
			}
		})
	}
}

func TestProcessAdditionalLabels(t *testing.T) {
	tests := []struct {
		name        string
		envVar      string
		envVarValue string
		want        [][2]string
	}{
		{
			name: "empty",
		},
		{
			name:        "default_name",
			envVar:      "CLOUDPROBER_ADDITIONAL_LABELS",
			envVarValue: "env=prod,app=identity",
			want: [][2]string{
				{"env", "prod"},
				{"app", "identity"},
			},
		},
		{
			name:        "kill_space",
			envVar:      "CLOUDPROBER_ADDITIONAL_LABELS",
			envVarValue: " env= prod, app=identity",
			want: [][2]string{
				{"env", "prod"},
				{"app", "identity"},
			},
		},
		{
			name:        "invalid_label1",
			envVar:      "CLOUDPROBER_ADDITIONAL_LABELS",
			envVarValue: "env=prod,=identity",
			want: [][2]string{
				{"env", "prod"},
			},
		},
		{
			name:        "invalid_label2",
			envVar:      "CLOUDPROBER_ADDITIONAL_LABELS",
			envVarValue: "env=,app=identity",
			want: [][2]string{
				{"app", "identity"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envVar, tt.envVarValue)
			defer os.Unsetenv(tt.envVar)

			if got := processAdditionalLabels(tt.envVar, nil); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processAdditionalLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
