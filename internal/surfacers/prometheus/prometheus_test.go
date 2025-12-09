// Copyright 2017-2025 The Cloudprober Authors.
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

package prometheus

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/prometheus/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers/common/options"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func newEventMetrics(sent, rcvd int64, respCodes map[string]int64, ptype, probe string) *metrics.EventMetrics {
	respCodesVal := metrics.NewMap("code")
	for k, v := range respCodes {
		respCodesVal.IncKeyBy(k, v)
	}
	return metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", metrics.NewInt(sent)).
		AddMetric("rcvd", metrics.NewInt(rcvd)).
		AddMetric("resp-code", respCodesVal).
		AddLabel("ptype", ptype).
		AddLabel("probe", probe)
}

func verify(t *testing.T, ps *PromSurfacer, expectedMetrics map[string]testData) {
	for k, td := range expectedMetrics {
		pm := ps.metrics[td.metricName]
		if pm == nil {
			t.Errorf("Metric %s not found in the prometheus metrics: %v", k, ps.metrics)
			continue
		}
		if pm.data[k] == nil {
			t.Errorf("Data key %s not found in the prometheus metrics: %v", k, pm.data)
			continue
		}
		if pm.data[k].value != td.value {
			t.Errorf("Didn't get expected metrics. Got: %s, Expected: %s", pm.data[k].value, td.value)
		}
	}
	var dataCount int
	for _, pm := range ps.metrics {
		dataCount += len(pm.data)
	}
	if dataCount != len(expectedMetrics) {
		t.Errorf("Prometheus doesn't have expected number of data keys. Got: %d, Expected: %d", dataCount, len(expectedMetrics))
	}
}

// mergeMap is helper function to build expectedMetrics by merging newly
// added expectedMetrics with the existing ones.
func mergeMap(recv map[string]testData, newmap map[string]testData) {
	for k, v := range newmap {
		recv[k] = v
	}
}

// testData encapsulates expected value for a metric key and metric name.
type testData struct {
	metricName string // To access data row in a 2-level data structure.
	value      string
}

func testPromSurfacer(baseConf *configpb.SurfacerConf) (*PromSurfacer, error) {
	c := &configpb.SurfacerConf{}
	if baseConf != nil {
		c = proto.Clone(baseConf).(*configpb.SurfacerConf)
	}
	// Attach a random integer to metrics URL so that multiple
	// tests can run in parallel without handlers clashing with
	// each other.
	c.MetricsUrl = proto.String(fmt.Sprintf("/metrics_%d", rand.Int()))
	return New(context.Background(), c, &options.Options{}, nil)
}

func testPromSurfacerNoErr(t *testing.T, baseConf *configpb.SurfacerConf) *PromSurfacer {
	ps, err := testPromSurfacer(baseConf)
	if err != nil {
		t.Fatal("Error while initializing prometheus surfacer", err)
	}
	return ps
}

func TestRecord(t *testing.T) {
	ps := testPromSurfacerNoErr(t, nil)

	// Record first EventMetrics
	ps.record(newEventMetrics(32, 22, map[string]int64{
		"200": 22,
	}, "http", "vm-to-google"))
	expectedMetrics := map[string]testData{
		"sent{ptype=\"http\",probe=\"vm-to-google\"}":                   {"sent", "32"},
		"rcvd{ptype=\"http\",probe=\"vm-to-google\"}":                   {"rcvd", "22"},
		"resp_code{ptype=\"http\",probe=\"vm-to-google\",code=\"200\"}": {"resp_code", "22"},
	}
	verify(t, ps, expectedMetrics)

	// Record second EventMetrics, no overlap.
	ps.record(newEventMetrics(500, 492, map[string]int64{}, "ping", "vm-to-vm"))
	mergeMap(expectedMetrics, map[string]testData{
		"sent{ptype=\"ping\",probe=\"vm-to-vm\"}": {"sent", "500"},
		"rcvd{ptype=\"ping\",probe=\"vm-to-vm\"}": {"rcvd", "492"},
	})
	verify(t, ps, expectedMetrics)

	// Record third EventMetrics, replaces first EventMetrics' metrics.
	ps.record(newEventMetrics(62, 50, map[string]int64{
		"200": 42,
		"204": 8,
	}, "http", "vm-to-google"))
	mergeMap(expectedMetrics, map[string]testData{
		"sent{ptype=\"http\",probe=\"vm-to-google\"}":                   {"sent", "62"},
		"rcvd{ptype=\"http\",probe=\"vm-to-google\"}":                   {"rcvd", "50"},
		"resp_code{ptype=\"http\",probe=\"vm-to-google\",code=\"200\"}": {"resp_code", "42"},
		"resp_code{ptype=\"http\",probe=\"vm-to-google\",code=\"204\"}": {"resp_code", "8"},
	})
	verify(t, ps, expectedMetrics)

	// Check with float map
	pLat := metrics.NewMapFloat("platency").IncKeyBy("p95", 0.083).IncKeyBy("p99", 0.134)
	ps.record(metrics.NewEventMetrics(time.Now()).AddMetric("app_latency", pLat))
	mergeMap(expectedMetrics, map[string]testData{
		"app_latency{platency=\"p95\"}": {"app_latency", "0.083"},
		"app_latency{platency=\"p99\"}": {"app_latency", "0.134"},
	})
	verify(t, ps, expectedMetrics)

	// Test string metrics.
	em := metrics.NewEventMetrics(time.Now()).
		AddMetric("instance_id", metrics.NewString("23152113123131")).
		AddMetric("version", metrics.NewString("cloudradar-20170606-RC00")).
		AddLabel("module", "sysvars")
	em.Kind = metrics.GAUGE
	ps.record(em)
	mergeMap(expectedMetrics, map[string]testData{
		"instance_id{module=\"sysvars\",val=\"23152113123131\"}":       {"instance_id", "1"},
		"version{module=\"sysvars\",val=\"cloudradar-20170606-RC00\"}": {"version", "1"},
	})
	verify(t, ps, expectedMetrics)
}

func TestInvalidNames(t *testing.T) {
	ps := testPromSurfacerNoErr(t, nil)

	ps.record(metrics.NewEventMetrics(time.Now()).
		AddMetric("sent", metrics.NewInt(32)).
		AddMetric("rcvd/sent", metrics.NewInt(22)).
		AddMetric("resp", metrics.NewMap("resp-code").IncKeyBy("200", 19)).
		AddLabel("probe-type", "http").
		AddLabel("probe/name", "vm-to-google"))

	// Metric rcvd/sent is dropped
	// Label probe-type is converted to probe_type
	// Label probe/name is dropped
	// Map value key resp-code is converted to resp_code label name
	expectedMetrics := map[string]testData{
		"sent{probe_type=\"http\"}":                   {"sent", "32"},
		"resp{probe_type=\"http\",resp_code=\"200\"}": {"resp", "19"},
	}
	verify(t, ps, expectedMetrics)
}

func testWebOutput(t *testing.T, config *configpb.SurfacerConf, expectTimestamp string) {
	t.Helper()

	ps := testPromSurfacerNoErr(t, config)
	latencyVal := metrics.NewDistribution([]float64{1, 4})
	latencyVal.AddSample(0.5)
	latencyVal.AddSample(5)
	ts := time.Now()
	counterEM := metrics.NewEventMetrics(ts).
		AddMetric("sent", metrics.NewInt(32)).
		AddMetric("latency", latencyVal).
		AddMetric("resp_code", metrics.NewMap("code").IncKeyBy("200", 19)).
		AddLabel("ptype", "http")
	ps.record(counterEM)

	gaugeEM := metrics.NewEventMetrics(ts).
		AddMetric("num_goroutines", metrics.NewInt(22)).
		AddLabel("system", "sysvars")
	gaugeEM.Kind = metrics.GAUGE
	ps.record(gaugeEM)

	var b bytes.Buffer
	ps.writeData(&b)
	data := b.String()
	var counterSuffix string
	var gaugeSuffix string
	tsSuffix := fmt.Sprintf(" %d", ts.UnixNano()/(1000*1000))
	switch expectTimestamp {
	case "default":
		gaugeSuffix = tsSuffix
	case "true":
		counterSuffix = tsSuffix
		gaugeSuffix = tsSuffix
	case "false":
		counterSuffix = ""
		gaugeSuffix = ""
	}
	for _, d := range []string{
		"# TYPE sent counter",
		"# TYPE resp_code counter",
		"# TYPE latency histogram",
		"sent{ptype=\"http\"} 32" + counterSuffix,
		"resp_code{ptype=\"http\",code=\"200\"} 19" + counterSuffix,
		"latency_sum{ptype=\"http\"} 5.5" + counterSuffix,
		"latency_count{ptype=\"http\"} 2" + counterSuffix,
		"latency_bucket{ptype=\"http\",le=\"1\"} 1" + counterSuffix,
		"latency_bucket{ptype=\"http\",le=\"4\"} 1" + counterSuffix,
		"latency_bucket{ptype=\"http\",le=\"+Inf\"} 2" + counterSuffix,
		"# TYPE num_goroutines gauge",
		"num_goroutines{system=\"sysvars\"} 22" + gaugeSuffix,
	} {
		if !strings.Contains(data, d+"\n") {
			t.Errorf("String \"%s\" not found in output data: %s", d, data)
		}
	}
}

func TestScrapeOutput(t *testing.T) {
	// Save and restore the global flag
	oldIncludeTimestampFlag := *includeTimestampFlag

	t.Run("IncludeTimestamp config default", func(t *testing.T) {
		testWebOutput(t, nil, "default")
	})

	t.Run("IncludeTimestamp config true", func(t *testing.T) {
		defer func() { *includeTimestampFlag = oldIncludeTimestampFlag }()
		*includeTimestampFlag = false
		testWebOutput(t, &configpb.SurfacerConf{IncludeTimestamp: proto.Bool(true)}, "true")
	})

	t.Run("IncludeTimestamp config false", func(t *testing.T) {
		testWebOutput(t, &configpb.SurfacerConf{IncludeTimestamp: proto.Bool(false)}, "false")
	})
}

func TestScrapeOutputWithExpiredTimeMetrics(t *testing.T) {
	ps := testPromSurfacerNoErr(t, &configpb.SurfacerConf{IncludeTimestamp: proto.Bool(true)})

	nowTime := time.Now()
	timeBeforeTenMin := nowTime.Add(-10 * time.Minute)
	promTS := fmt.Sprintf("%d", nowTime.UnixNano()/(1000*1000))

	em := metrics.NewEventMetrics(nowTime).
		AddMetric("success", metrics.NewInt(6)).
		AddMetric("total", metrics.NewInt(10)).
		AddLabel("ptype", "ping").
		AddLabel("probe", "ping-probe").
		AddLabel("dst", "www.google.com")
	ps.record(em)

	expiredEm := metrics.NewEventMetrics(timeBeforeTenMin).
		AddMetric("success", metrics.NewInt(12)).
		AddMetric("total", metrics.NewInt(20)).
		AddLabel("ptype", "ping").
		AddLabel("probe", "expired-ping-probe").
		AddLabel("dst", "www.google.com/2")
	ps.record(expiredEm)

	expiredEm2 := metrics.NewEventMetrics(timeBeforeTenMin).
		AddMetric("success", metrics.NewInt(18)).
		AddMetric("total", metrics.NewInt(30)).
		AddLabel("ptype", "ping").
		AddLabel("probe", "expired-ping-probe-2").
		AddLabel("dst", "www.google.com/3")
	ps.record(expiredEm2)

	var b bytes.Buffer
	ps.deleteExpiredMetrics()
	ps.writeData(&b)
	data := b.String()

	for _, d := range []string{
		"success{ptype=\"ping\",probe=\"expired-ping-probe\",dst=\"www.google.com/2\"} 12 " + promTS,
		"success{ptype=\"ping\",probe=\"expired-ping-probe-2\",dst=\"www.google.com/3\"} 18 " + promTS,
		"total{ptype=\"ping\",probe=\"expired-ping-probe\",dst=\"www.google.com/2\"} 20 " + promTS,
		"total{ptype=\"ping\",probe=\"expired-ping-probe-2\",dst=\"www.google.com/3\"} 30 " + promTS,
	} {
		if strings.Contains(data, d) {
			t.Errorf("String \"%s\" contains expired data in output data: %s", d, data)
		}
	}
}

func TestMetricsPrefix(t *testing.T) {
	tests := []struct {
		name       string
		confPrefix string
		flagPrefix string
		wantPrefix string
		wantErr    bool
	}{
		{
			name:       "No prefix",
			wantPrefix: "",
		},
		{
			name:       "conf prefix",
			confPrefix: "cloudprober_c_",
			wantPrefix: "cloudprober_c_",
		},
		{
			name:       "flag prefix",
			flagPrefix: "cloudprober_f_",
			wantPrefix: "cloudprober_f_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			*metricsPrefixFlag = tt.flagPrefix
			defer func() {
				*metricsPrefixFlag = ""
			}()

			c := &configpb.SurfacerConf{}
			if tt.confPrefix != "" {
				c.MetricsPrefix = proto.String(tt.confPrefix)
			}
			ps, err := testPromSurfacer(c)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Error while initializing prometheus surfacer: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Errorf("Expected error, got none")
			}

			assert.Equal(t, ps.prefix, tt.wantPrefix, "prefix mismatch")
		})
	}

	// Make sure that the prefix is applied to the metrics.
	ps := testPromSurfacerNoErr(t, nil)
	ps.prefix = "cloudprober_"

	// Record first EventMetrics
	ps.record(newEventMetrics(32, 22, map[string]int64{
		"200": 22,
	}, "http", "vm-to-google"))

	expectedMetrics := map[string]testData{
		"cloudprober_sent{ptype=\"http\",probe=\"vm-to-google\"}":                   {"cloudprober_sent", "32"},
		"cloudprober_rcvd{ptype=\"http\",probe=\"vm-to-google\"}":                   {"cloudprober_rcvd", "22"},
		"cloudprober_resp_code{ptype=\"http\",probe=\"vm-to-google\",code=\"200\"}": {"cloudprober_resp_code", "22"},
	}
	verify(t, ps, expectedMetrics)
}

func TestMain(m *testing.M) {
	state.SetDefaultHTTPServeMux(http.NewServeMux())
	defer state.SetDefaultHTTPServeMux(nil)

	m.Run()
}

func TestDisableMetricsExpiration(t *testing.T) {
	tests := []struct {
		includeTimestamp         []defaultBoolEnum
		disableMetricsExpiration *bool
		want                     []defaultBoolEnum
	}{
		// Not explicitly set
		{
			includeTimestamp: []defaultBoolEnum{defaultBehavior, explicitTrue, explicitFalse},
			want:             []defaultBoolEnum{defaultBehavior, explicitFalse, explicitTrue},
		},
		{
			includeTimestamp:         []defaultBoolEnum{defaultBehavior, explicitTrue, explicitFalse},
			disableMetricsExpiration: proto.Bool(true),
			want:                     []defaultBoolEnum{explicitTrue, explicitTrue, explicitTrue},
		},
		{
			includeTimestamp:         []defaultBoolEnum{defaultBehavior, explicitTrue, explicitFalse},
			disableMetricsExpiration: proto.Bool(false),
			want:                     []defaultBoolEnum{explicitFalse, explicitFalse, explicitFalse},
		},
	}

	for _, tt := range tests {
		for i, includeTimestamp := range tt.includeTimestamp {
			t.Run(fmt.Sprintf("includeTimestamp=%v, disableMetricsExpiration=%v", includeTimestamp, tt.disableMetricsExpiration), func(t *testing.T) {
				conf := &configpb.SurfacerConf{
					DisableMetricsExpiration: tt.disableMetricsExpiration,
				}
				ps := &PromSurfacer{
					c:                conf,
					includeTimestamp: includeTimestamp,
				}
				assert.Equal(t, tt.want[i], ps.shouldDisableMetricsExpiration())
			})
		}
	}
}

func TestIncludeTimestamp(t *testing.T) {
	tests := []struct {
		name           string
		setFlag        bool
		flagValue      string
		configValue    *bool
		expectedResult defaultBoolEnum
	}{
		{
			name:           "flag_true",
			setFlag:        true,
			flagValue:      "true",
			expectedResult: explicitTrue,
		},
		{
			name:           "flag_false",
			setFlag:        true,
			flagValue:      "false",
			expectedResult: explicitFalse,
		},
		{
			name:           "config_true",
			setFlag:        true,
			flagValue:      "false",
			configValue:    proto.Bool(true),
			expectedResult: explicitTrue,
		},
		{
			name:           "config_false",
			setFlag:        true,
			flagValue:      "true",
			configValue:    proto.Bool(false),
			expectedResult: explicitFalse,
		},
		{
			name:           "default_behavior",
			setFlag:        false,
			configValue:    nil,
			expectedResult: defaultBehavior,
		},
	}

	// Save and restore the actual command line flags
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the command line flags for each test case
			os.Args = oldArgs
			flag.CommandLine = flag.NewFlagSet("", flag.ExitOnError)

			// Set up the flag if needed
			if tt.setFlag {
				// Need to parse the flag from command line arguments
				os.Args = append([]string{oldArgs[0]}, "-prometheus_include_timestamp="+tt.flagValue)
				f := flag.Bool("prometheus_include_timestamp", false, "")
				flag.Parse()
				*includeTimestampFlag = *f
			}

			// Create a config with the test case's config value
			conf := &configpb.SurfacerConf{
				IncludeTimestamp: tt.configValue,
			}

			// Call the function under test
			result := shouldIncludeTimestamp(conf)

			// Verify the result
			if result != tt.expectedResult {
				t.Errorf("includeTimestamp() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name                 string
		config               *configpb.SurfacerConf
		metricsPrefixFlag    string
		notInitServeMux      bool
		wantIncludeTimestamp defaultBoolEnum
		wantMetricsPrefix    string
		wantErr              bool
	}{
		{
			name:                 "Default",
			config:               nil,
			metricsPrefixFlag:    "",
			wantIncludeTimestamp: defaultBehavior,
			wantMetricsPrefix:    "",
		},
		{
			name:                 "Flags",
			config:               nil,
			metricsPrefixFlag:    "cloudprober_f_",
			wantIncludeTimestamp: defaultBehavior,
			wantMetricsPrefix:    "cloudprober_f_",
		},
		{
			name:                 "Config override",
			config:               &configpb.SurfacerConf{IncludeTimestamp: proto.Bool(false), MetricsPrefix: proto.String("cloudprober_")},
			metricsPrefixFlag:    "cloudprober_f_",
			wantIncludeTimestamp: explicitFalse,
			wantMetricsPrefix:    "cloudprober_",
		},
		{
			name:            "ServeMux not initialized",
			config:          nil,
			notInitServeMux: true,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldHTTPMux := state.DefaultHTTPServeMux()
			if tt.notInitServeMux {
				state.SetDefaultHTTPServeMux(nil)
			} else {
				state.SetDefaultHTTPServeMux(http.NewServeMux())
			}
			defer state.SetDefaultHTTPServeMux(oldHTTPMux)

			*metricsPrefixFlag = tt.metricsPrefixFlag
			defer func() {
				*metricsPrefixFlag = ""
			}()

			got, err := New(context.Background(), tt.config, nil, nil)
			if tt.wantErr {
				if err == nil {
					t.Errorf("New() error = %v, want non-nil", err)
					return
				}
				return
			}
			if err != nil {
				t.Errorf("New() error = %v, want nil", err)
				return
			}
			if got.includeTimestamp != tt.wantIncludeTimestamp {
				t.Errorf("includeTimestamp = %v, want %v", got.includeTimestamp, tt.wantIncludeTimestamp)
			}
			if got.prefix != tt.wantMetricsPrefix {
				t.Errorf("prefix = %v, want %v", got.prefix, tt.wantMetricsPrefix)
			}
		})
	}
}
