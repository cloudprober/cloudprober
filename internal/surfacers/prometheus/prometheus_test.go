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
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/internal/surfacers/prometheus/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/state"
	"github.com/cloudprober/cloudprober/surfacers/options"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protodelim"
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

func TestExpirationAge(t *testing.T) {
	const stale = 42 * time.Minute

	tests := []struct {
		desc                     string
		includeTimestamp         defaultBoolEnum
		disableMetricsExpiration *bool
		staleOverride            *int32
		wantGauge                time.Duration
		wantCounter              time.Duration
	}{
		{
			desc:             "default: gauges on the timestamp deadline, counters on staleness",
			includeTimestamp: defaultBehavior,
			wantGauge:        metricExpirationTime,
			wantCounter:      stale,
		},
		{
			desc:             "all metrics timestamped: everything on the timestamp deadline",
			includeTimestamp: explicitTrue,
			wantGauge:        metricExpirationTime,
			wantCounter:      metricExpirationTime,
		},
		{
			desc:             "no metrics timestamped: everything on staleness",
			includeTimestamp: explicitFalse,
			wantGauge:        stale,
			wantCounter:      stale,
		},
		{
			desc:                     "expiration explicitly disabled",
			includeTimestamp:         defaultBehavior,
			disableMetricsExpiration: proto.Bool(true),
			wantGauge:                0,
			wantCounter:              0,
		},
		{
			desc:                     "expiration explicitly enabled: same as leaving it unset",
			includeTimestamp:         defaultBehavior,
			disableMetricsExpiration: proto.Bool(false),
			wantGauge:                metricExpirationTime,
			wantCounter:              stale,
		},
		{
			desc:             "staleness expiration disabled, gauges still bounded by the timestamp deadline",
			includeTimestamp: defaultBehavior,
			staleOverride:    proto.Int32(0),
			wantGauge:        metricExpirationTime,
			wantCounter:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			staleExpiration := stale
			if tt.staleOverride != nil {
				staleExpiration = time.Duration(*tt.staleOverride)
			}
			ps := &PromSurfacer{
				c: &configpb.SurfacerConf{
					DisableMetricsExpiration: tt.disableMetricsExpiration,
				},
				includeTimestamp:       tt.includeTimestamp,
				staleMetricsExpiration: staleExpiration,
			}
			assert.Equal(t, tt.wantGauge, ps.expirationAge("gauge"), "gauge")
			assert.Equal(t, tt.wantCounter, ps.expirationAge("counter"), "counter")
		})
	}
}

func TestStaleMetricsExpiration(t *testing.T) {
	scrape := func(ps *PromSurfacer) string {
		var b bytes.Buffer
		ps.writeData(&b)
		return b.String()
	}
	staleEM := func() *metrics.EventMetrics {
		return metrics.NewEventMetrics(time.Now().Add(-30*time.Minute)).
			AddMetric("total", metrics.NewInt(20)).
			AddLabel("probe", "stale-probe")
	}

	// Counters are not timestamped by default, so they are expired only after
	// the staleness deadline, not the 10m timestamp deadline.
	t.Run("under the default deadline", func(t *testing.T) {
		ps := testPromSurfacerNoErr(t, &configpb.SurfacerConf{})
		assert.Equal(t, 4*time.Hour, ps.staleMetricsExpiration, "default from the config proto")

		ps.record(staleEM())
		ps.deleteExpiredMetrics()
		assert.Contains(t, scrape(ps), "stale-probe", "counter stale for 30m, but under the 4h deadline")
	})

	t.Run("past the deadline", func(t *testing.T) {
		ps := testPromSurfacerNoErr(t, &configpb.SurfacerConf{
			StaleMetricsExpirationSec: proto.Int32(600),
		})
		ps.record(staleEM())
		ps.deleteExpiredMetrics()
		assert.NotContains(t, scrape(ps), "stale-probe", "counter stale beyond the deadline")
	})

	t.Run("zero disables staleness expiration", func(t *testing.T) {
		ps := testPromSurfacerNoErr(t, &configpb.SurfacerConf{
			StaleMetricsExpirationSec: proto.Int32(0),
		})
		ps.record(staleEM())
		ps.deleteExpiredMetrics()
		assert.Contains(t, scrape(ps), "stale-probe", "staleness expiration disabled")
	})
}

func TestDeleteExpiredMetricsKeepsKeyOrder(t *testing.T) {
	ps := testPromSurfacerNoErr(t, &configpb.SurfacerConf{
		StaleMetricsExpirationSec: proto.Int32(3600),
	})

	// Three series on the same metric; only the middle one is stale enough to
	// be expired, so the surviving dataKeys must stay in insertion order.
	for _, tc := range []struct {
		probe string
		age   time.Duration
	}{
		{"probe-a", 0},
		{"probe-b", 2 * time.Hour},
		{"probe-c", 0},
	} {
		ps.record(metrics.NewEventMetrics(time.Now().Add(-tc.age)).
			AddMetric("total", metrics.NewInt(1)).
			AddLabel("probe", tc.probe))
	}

	pm := ps.metrics["total"]
	assert.Len(t, pm.dataKeys, 3, "before expiration")

	ps.deleteExpiredMetrics()

	assert.Equal(t, []string{
		"total{probe=\"probe-a\"}",
		"total{probe=\"probe-c\"}",
	}, pm.dataKeys)
	assert.Len(t, pm.data, 2, "data map and dataKeys must stay in sync")
}

func TestNegativeStaleMetricsExpiration(t *testing.T) {
	_, err := testPromSurfacer(&configpb.SurfacerConf{
		StaleMetricsExpirationSec: proto.Int32(-1),
	})
	assert.Error(t, err, "negative stale_metrics_expiration_sec should be rejected")

	ps, err := testPromSurfacer(&configpb.SurfacerConf{
		StaleMetricsExpirationSec: proto.Int32(0),
	})
	assert.NoError(t, err, "zero means never expire, and is valid")
	assert.Equal(t, time.Duration(0), ps.staleMetricsExpiration)
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

// goldenTestEMs returns EventMetrics that cover all the metric types, more
// than one series per metric, and updates to already recorded series.
func goldenTestEMs(ts time.Time) []*metrics.EventMetrics {
	latency := func(samples ...float64) *metrics.Distribution {
		d := metrics.NewDistribution([]float64{1, 4, 16})
		for _, s := range samples {
			d.AddSample(s)
		}
		return d
	}
	probeEM := func(ts time.Time, dst string, total int64, respCode *metrics.Map[int64], samples ...float64) *metrics.EventMetrics {
		return metrics.NewEventMetrics(ts).
			AddMetric("total", metrics.NewInt(total)).
			AddMetric("latency", latency(samples...)).
			AddMetric("resp-code", respCode).
			AddMetric("avg_latency", metrics.NewFloat(float64(total)/7)).
			AddLabel("ptype", "http").
			AddLabel("probe", "p1").
			AddLabel("dst", dst)
	}

	sysEM := metrics.NewEventMetrics(ts).
		AddMetric("goroutines", metrics.NewInt(22)).
		AddMetric("version", metrics.NewString("v0.14.2")).
		AddMetric("load", metrics.NewMapFloat("period").IncKeyBy("1m", 0.25).IncKeyBy("5m", 1.5)).
		AddMetric("fd_dist", latency(2, 20)).
		AddLabel("probe", "sysvars")
	sysEM.Kind = metrics.GAUGE

	return []*metrics.EventMetrics{
		probeEM(ts, "a.com", 2, metrics.NewMap("code").IncKeyBy("200", 2), 0.5, 5),
		probeEM(ts, "b.com", 3, metrics.NewMap("code").IncKeyBy("200", 2).IncKeyBy("503", 1), 2, 3, 100),
		sysEM,
		// No labels.
		metrics.NewEventMetrics(ts).AddMetric("total", metrics.NewInt(9)).AddMetric("latency", latency(1)),
		// Update for a.com, with a new resp-code key.
		probeEM(ts.Add(10*time.Second), "a.com", 4, metrics.NewMap("code").IncKeyBy("200", 3).IncKeyBy("503", 1), 0.5, 5, 5, -1),
	}
}

// TestWriteDataGolden verifies the complete /metrics output, including the
// order of the lines, against the files in testdata.
func TestWriteDataGolden(t *testing.T) {
	ts := time.UnixMilli(1790000000000)

	for _, mode := range []string{"default", "true", "false"} {
		t.Run("include_timestamp_"+mode, func(t *testing.T) {
			conf := &configpb.SurfacerConf{}
			if mode != "default" {
				conf.IncludeTimestamp = proto.Bool(mode == "true")
			}
			ps := testPromSurfacerNoErr(t, conf)
			for _, em := range goldenTestEMs(ts) {
				ps.record(em)
			}

			var b bytes.Buffer
			ps.writeData(&b)

			want, err := os.ReadFile("testdata/golden_timestamp_" + mode + ".txt")
			assert.NoError(t, err)
			// Git may check out the golden files with CRLF line endings on Windows.
			assert.Equal(t, strings.ReplaceAll(string(want), "\r\n", "\n"), b.String())
		})
	}
}

// benchSurfacer returns a surfacer with metrics for 100 targets: 1,700 series,
// 400 of them histograms. If native is true, it has only histograms, 400 of
// them, with native buckets: 50 buckets in 9 spans each.
func benchSurfacer(b *testing.B, native bool) *PromSurfacer {
	ps, err := testPromSurfacer(nil)
	if err != nil {
		b.Fatal(err)
	}
	ts := time.Now()
	for i := range 100 {
		if !native {
			for _, em := range goldenTestEMs(ts) {
				ps.record(em.AddLabel("target", strconv.Itoa(i)))
			}
			continue
		}
		for _, dst := range []string{"a.com", "b.com", "c.com", "d.com"} {
			d, _ := metrics.NewNativeDistribution(3)
			for j := range 300 {
				d.AddSample(1 + float64(j%60)*float64(j%7))
			}
			ps.record(metrics.NewEventMetrics(ts).AddMetric("latency", d).
				AddLabel("ptype", "http").AddLabel("probe", "p1").AddLabel("dst", dst).AddLabel("target", strconv.Itoa(i)))
		}
	}
	return ps
}

func BenchmarkWriteData(b *testing.B) {
	ps := benchSurfacer(b, false)

	var buf bytes.Buffer
	b.ReportAllocs()
	for b.Loop() {
		buf.Reset()
		ps.writeData(&buf)
	}
}

func BenchmarkWriteProtobuf(b *testing.B) {
	for name, native := range map[string]bool{"classic": false, "native": true} {
		b.Run(name, func(b *testing.B) {
			ps := benchSurfacer(b, native)

			var buf bytes.Buffer
			b.ReportAllocs()
			for b.Loop() {
				buf.Reset()
				ps.writeProtobuf(&buf)
			}
		})
	}
}

func TestProtobufMixedTypes(t *testing.T) {
	ps := testPromSurfacerNoErr(t, nil)
	dist := func() *metrics.Distribution { return metrics.NewDistribution([]float64{1}) }

	// Same metric name used for numbers and distributions, e.g. latency when
	// only some of the probes have latency_distribution. Metric's type is the
	// type of the first value we see.
	ps.record(metrics.NewEventMetrics(time.Now()).AddMetric("num_first", metrics.NewInt(1)).AddLabel("a", "1"))
	ps.record(metrics.NewEventMetrics(time.Now()).AddMetric("num_first", dist()).AddLabel("a", "2"))
	ps.record(metrics.NewEventMetrics(time.Now()).AddMetric("dist_first", dist()).AddLabel("a", "1"))
	ps.record(metrics.NewEventMetrics(time.Now()).AddMetric("dist_first", metrics.NewInt(1)).AddLabel("a", "2"))

	var b bytes.Buffer
	ps.writeProtobuf(&b)

	// Histograms and numbers go in separate metric families.
	var got []string
	for _, mf := range readProtobuf(t, &b) {
		assert.Len(t, mf.Metric, 1, mf.GetName())
		m := mf.Metric[0]
		got = append(got, fmt.Sprintf("%s %s a=%s histogram=%v", mf.GetName(), mf.GetType(), m.Label[0].GetValue(), m.Histogram != nil))
	}
	assert.Equal(t, []string{
		"num_first HISTOGRAM a=2 histogram=true",
		"num_first COUNTER a=1 histogram=false",
		"dist_first HISTOGRAM a=1 histogram=true",
		"dist_first UNTYPED a=2 histogram=false",
	}, got)
}

func TestAcceptsProtobuf(t *testing.T) {
	const pb = "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited"
	tests := map[string]struct {
		accept string
		want   bool
	}{
		"empty":     {"", false},
		"text_only": {"text/plain;version=0.0.4;q=0.5,*/*;q=0.1", false},
		// Prometheus 3.15 when scrape_native_histograms is true, and by default.
		"prometheus_native_histograms": {pb + ";q=0.7,application/openmetrics-text;version=1.0.0;escaping=allow-utf-8;q=0.6,application/openmetrics-text;version=0.0.1;q=0.5,text/plain;version=1.0.0;escaping=allow-utf-8;q=0.4,text/plain;version=0.0.4;q=0.3,*/*;q=0.2", true},
		"prometheus_default":           {"application/openmetrics-text;version=1.0.0;escaping=allow-utf-8;q=0.6,application/openmetrics-text;version=0.0.1;q=0.5,text/plain;version=1.0.0;escaping=allow-utf-8;q=0.4,text/plain;version=0.0.4;q=0.3,*/*;q=0.2", false},
		"protobuf_lower_q":             {"text/plain;version=0.0.4;q=0.6," + pb + ";q=0.5", false},
		// We can't serve OpenMetrics, so it's between protobuf and text.
		"openmetrics_first":      {"application/openmetrics-text;version=1.0.0;q=0.7," + pb + ";q=0.6,text/plain;version=0.0.4;q=0.5", true},
		"any_over_protobuf":      {"*/*;q=0.7," + pb + ";q=0.6", false},
		"protobuf_no_q":          {pb, true},
		"protobuf_same_q_later":  {"text/plain," + pb, false},
		"protobuf_no_encoding":   {"application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily", true},
		"protobuf_text_encoding": {"application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=text", false},
		"protobuf_other_proto":   {"application/vnd.google.protobuf;proto=foo.Bar", false},
		"malformed":              {";;;," + pb + ";q=x", false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, acceptsProtobuf(tt.accept))
		})
	}
}

func TestNativeSpansAndDeltas(t *testing.T) {
	span := func(offset int32, length uint32) *dto.BucketSpan {
		return &dto.BucketSpan{Offset: proto.Int32(offset), Length: proto.Uint32(length)}
	}
	tests := map[string]struct {
		buckets    map[int]int64
		wantSpans  []*dto.BucketSpan
		wantDeltas []int64
	}{
		"empty": {},
		"one_span": {
			buckets:    map[int]int64{-2: 3, -1: 5, 0: 1},
			wantSpans:  []*dto.BucketSpan{span(-2, 3)},
			wantDeltas: []int64{3, 2, -4},
		},
		"gaps": {
			buckets:    map[int]int64{1: 2, 2: 2, 5: 1, 9: 4},
			wantSpans:  []*dto.BucketSpan{span(1, 2), span(2, 1), span(3, 1)},
			wantDeltas: []int64{2, 0, -1, 3},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			spans, deltas := nativeSpansAndDeltas(tt.buckets)
			if !assert.Len(t, spans, len(tt.wantSpans)) {
				return
			}
			for i := range tt.wantSpans {
				assert.True(t, proto.Equal(tt.wantSpans[i], spans[i]), "span %d: got %v, want %v", i, spans[i], tt.wantSpans[i])
			}
			assert.Equal(t, tt.wantDeltas, deltas)
		})
	}
}

func nativeTestEM(ts time.Time, probe string, samples ...float64) *metrics.EventMetrics {
	d, _ := metrics.NewNativeDistribution(0)
	for _, s := range samples {
		d.AddSample(s)
	}
	return metrics.NewEventMetrics(ts).AddMetric("native_latency", d).AddLabel("probe", probe)
}

func TestNativeHistogramText(t *testing.T) {
	ps := testPromSurfacerNoErr(t, nil)
	ps.record(nativeTestEM(time.Now(), "p2", 0.75, 1.5, 3, 3))

	var b bytes.Buffer
	ps.writeData(&b)
	assert.Equal(t, strings.Join([]string{
		"# TYPE native_latency histogram",
		`native_latency_sum{probe="p2"} 8.25`,
		`native_latency_count{probe="p2"} 4`,
		`native_latency_bucket{probe="p2",le="+Inf"} 4`,
	}, "\n")+"\n", b.String())
}

func readProtobuf(t *testing.T, r io.Reader) []*dto.MetricFamily {
	t.Helper()
	var mfs []*dto.MetricFamily
	br := bufio.NewReader(r)
	for {
		mf := &dto.MetricFamily{}
		if err := protodelim.UnmarshalFrom(br, mf); err != nil {
			if err != io.EOF {
				t.Fatalf("Error reading protobuf output: %v", err)
			}
			return mfs
		}
		mfs = append(mfs, mf)
	}
}

func TestWriteProtobuf(t *testing.T) {
	ts := time.UnixMilli(1790000000000)
	ps := testPromSurfacerNoErr(t, nil)
	for _, em := range goldenTestEMs(ts) {
		ps.record(em)
	}
	ps.record(nativeTestEM(ts, "p2", 0, 0.75, 1, 3, 3, 40, -2))
	ps.record(nativeTestEM(ts, "p3", 0, 0.75, 1, 3, 3, 40, -2))
	// No samples.
	ps.record(nativeTestEM(ts, "p4"))

	var b bytes.Buffer
	ps.writeProtobuf(&b)
	mfs := readProtobuf(t, &b)

	// Same metric families, in the same order, as the text format.
	var names []string
	for _, mf := range mfs {
		names = append(names, mf.GetName()+" "+strings.ToLower(mf.GetType().String()))
	}
	assert.Equal(t, []string{
		"total counter", "latency histogram", "resp_code counter", "avg_latency counter",
		"goroutines gauge", "version gauge", "load gauge", "fd_dist histogram", "native_latency histogram",
	}, names)

	lp := func(kv ...string) []*dto.LabelPair {
		var pairs []*dto.LabelPair
		for i := 0; i < len(kv); i += 2 {
			pairs = append(pairs, &dto.LabelPair{Name: proto.String(kv[i]), Value: proto.String(kv[i+1])})
		}
		return pairs
	}
	span := func(offset int32, length uint32) *dto.BucketSpan {
		return &dto.BucketSpan{Offset: proto.Int32(offset), Length: proto.Uint32(length)}
	}
	bucket := func(count uint64, upperBound float64) *dto.Bucket {
		return &dto.Bucket{CumulativeCount: proto.Uint64(count), UpperBound: proto.Float64(upperBound)}
	}
	nativeHist := &dto.Histogram{
		SampleCount:   proto.Uint64(7),
		SampleSum:     proto.Float64(45.75),
		Schema:        proto.Int32(0),
		ZeroThreshold: proto.Float64(0),
		ZeroCount:     proto.Uint64(1),
		// 0.75, 1 -> 0; 3, 3 -> 2; 40 -> 6.
		PositiveSpan:  []*dto.BucketSpan{span(0, 1), span(1, 1), span(3, 1)},
		PositiveDelta: []int64{2, 0, -1},
		NegativeSpan:  []*dto.BucketSpan{span(1, 1)},
		NegativeDelta: []int64{1},
	}

	wantMetrics := map[string][]*dto.Metric{
		"total": {
			{Label: lp("ptype", "http", "probe", "p1", "dst", "a.com"), Counter: &dto.Counter{Value: proto.Float64(4)}},
			{Label: lp("ptype", "http", "probe", "p1", "dst", "b.com"), Counter: &dto.Counter{Value: proto.Float64(3)}},
			{Counter: &dto.Counter{Value: proto.Float64(9)}},
		},
		"avg_latency": {
			{Label: lp("ptype", "http", "probe", "p1", "dst", "a.com"), Counter: &dto.Counter{Value: proto.Float64(0.571)}},
			{Label: lp("ptype", "http", "probe", "p1", "dst", "b.com"), Counter: &dto.Counter{Value: proto.Float64(0.429)}},
		},
		// Gauges have timestamps by default.
		"version": {
			{Label: lp("probe", "sysvars", "val", "v0.14.2"), Gauge: &dto.Gauge{Value: proto.Float64(1)}, TimestampMs: proto.Int64(1790000000000)},
		},
		"load": {
			{Label: lp("probe", "sysvars", "period", "1m"), Gauge: &dto.Gauge{Value: proto.Float64(0.25)}, TimestampMs: proto.Int64(1790000000000)},
			{Label: lp("probe", "sysvars", "period", "5m"), Gauge: &dto.Gauge{Value: proto.Float64(1.5)}, TimestampMs: proto.Int64(1790000000000)},
		},
		"fd_dist": {
			{Label: lp("probe", "sysvars"), Histogram: &dto.Histogram{
				SampleCount: proto.Uint64(2),
				SampleSum:   proto.Float64(22),
				Bucket:      []*dto.Bucket{bucket(0, 1), bucket(1, 4), bucket(1, 16)},
			}},
		},
		"native_latency": {
			{Label: lp("probe", "p2"), Histogram: nativeHist},
			{Label: lp("probe", "p3"), Histogram: nativeHist},
			{Label: lp("probe", "p4"), Histogram: &dto.Histogram{
				SampleCount:   proto.Uint64(0),
				SampleSum:     proto.Float64(0),
				Schema:        proto.Int32(0),
				ZeroThreshold: proto.Float64(0),
				ZeroCount:     proto.Uint64(0),
				PositiveSpan:  []*dto.BucketSpan{span(0, 0)},
			}},
		},
	}

	for _, mf := range mfs {
		want, ok := wantMetrics[mf.GetName()]
		if !ok {
			continue
		}
		t.Run(mf.GetName(), func(t *testing.T) {
			if !assert.Equal(t, len(want), len(mf.Metric)) {
				return
			}
			for i := range want {
				assert.True(t, proto.Equal(want[i], mf.Metric[i]), "metric %d:\n got: %v\nwant: %v", i, mf.Metric[i], want[i])
			}
		})
	}
}

func TestProtobufScrape(t *testing.T) {
	ps := testPromSurfacerNoErr(t, nil)
	ps.record(nativeTestEM(time.Now(), "p2", 1.5))

	for _, tt := range []struct {
		accept          string
		wantContentType string
	}{
		{"text/plain;version=0.0.4", ""},
		{"application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.6,text/plain;version=0.0.4;q=0.5", protobufContentType},
	} {
		t.Run(tt.accept, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, ps.c.GetMetricsUrl(), nil)
			req.Header.Set("Accept", tt.accept)
			rec := httptest.NewRecorder()
			state.DefaultHTTPServeMux().ServeHTTP(rec, req)

			if tt.wantContentType == "" {
				assert.Contains(t, rec.Body.String(), `native_latency_bucket{probe="p2",le="+Inf"} 1`)
				return
			}
			assert.Equal(t, tt.wantContentType, rec.Header().Get("Content-Type"))
			mfs := readProtobuf(t, rec.Body)
			assert.Len(t, mfs, 1)
			assert.Equal(t, int32(1), mfs[0].Metric[0].Histogram.GetPositiveSpan()[0].GetOffset())
		})
	}
}
