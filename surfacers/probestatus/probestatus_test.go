// Copyright 2022 The Cloudprober Authors.
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

package probestatus

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/surfacers/common/options"
	configpb "github.com/cloudprober/cloudprober/surfacers/probestatus/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

const (
	testProbe = "test-probe"
)

func testEM(t *testing.T, tm time.Time, probe, target string, total, success int, latency float64) *metrics.EventMetrics {
	t.Helper()
	return metrics.NewEventMetrics(tm).
		AddLabel("probe", probe).
		AddLabel("dst", target).
		AddMetric("total", metrics.NewInt(int64(total))).
		AddMetric("success", metrics.NewInt(int64(total))).
		AddMetric("latency", metrics.NewFloat(latency))
}

func TestNewAndRecord(t *testing.T) {
	total, success := 200, 180
	latency := float64(2000)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ps := New(ctx, &configpb.SurfacerConf{
		TimeseriesSize:     proto.Int32(10),
		MaxTargetsPerProbe: proto.Int32(2),
	}, &options.Options{HTTPServeMux: http.NewServeMux()}, nil)

	probeTargets := map[string][]string{
		"p":  {"t1", "t2"},
		"p2": {"t1"},
		"p3": {"t1", "t2", "t3"},
	}

	for probe, targets := range probeTargets {
		for _, target := range targets {
			ps.record(testEM(t, time.Now(), probe, target, total, success, latency))
		}
	}

	for probe, targets := range probeTargets {
		// For p3 one target will be dropped due to capacity limit.
		if probe == "p3" {
			targets = targets[:len(targets)-1]
		}
		for _, target := range targets {
			if ps.metrics[probe][target] == nil {
				t.Errorf("Unexpected nil timeseries for: probe(%s), target(%s)", probe, target)
			}
		}
	}
}

func TestPageCache(t *testing.T) {
	pc := &pageCache{
		maxAge: time.Second,
	}

	c, valid := pc.contentIfValid()
	if valid {
		t.Errorf("Got valid content from new cache: %s", string(c))
	}

	testContent := []byte("test-content")
	pc.setContent(testContent)
	c, valid = pc.contentIfValid()
	if !valid {
		t.Errorf("Got unexpected invalid")
	}
	if !bytes.Equal(c, testContent) {
		t.Errorf("Got=%s, wanted=%s", string(c), string(testContent))
	}

	time.Sleep(time.Second)
	c, valid = pc.contentIfValid()
	if valid {
		t.Errorf("Got unexpected valid content from pageCache: %s", string(c))
	}
}

func TestDisabledAndHandlers(t *testing.T) {
	tests := []struct {
		desc         string
		disabled     bool
		url          string
		patternMatch map[string]string
	}{
		{
			desc:     "disabled",
			disabled: true,
			patternMatch: map[string]string{
				"/probestatus": "",
			},
		},
		{
			desc: "default",
			patternMatch: map[string]string{
				"/probestatus": "/probestatus",
			},
		},
		{
			desc: "different_url",
			url:  "/status",
			patternMatch: map[string]string{
				"/probestatus": "",
				"/status":      "/status",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancelFunc := context.WithCancel(context.Background())
			defer cancelFunc()

			conf := &configpb.SurfacerConf{
				Disable: proto.Bool(test.disabled),
			}
			if test.url != "" {
				conf.Url = proto.String(test.url)
			}
			opts := &options.Options{HTTPServeMux: http.NewServeMux()}
			ps := New(ctx, conf, opts, nil)

			if test.disabled {
				if ps != nil {
					t.Errorf("surfacer should be nil if it's disabled")
				}
				// Verify no panic on writing to a nil surfacer.
				ps.Write(ctx, &metrics.EventMetrics{Timestamp: time.Now()})
			}

			for url, pattern := range test.patternMatch {
				_, matchedPattern := opts.HTTPServeMux.Handler(httptest.NewRequest("", url, nil))
				assert.Equal(t, pattern, matchedPattern)
			}
		})
	}
}
