// Copyright 2017-2020 The Cloudprober Authors.
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
	"bytes"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/common/iputils"
	"github.com/cloudprober/cloudprober/internal/alerting"
	alerting_configpb "github.com/cloudprober/cloudprober/internal/alerting/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	configpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type intf struct {
	addrs []net.Addr
}

func (i *intf) Addrs() ([]net.Addr, error) {
	return i.addrs, nil
}

func mockInterfaceByName(iname string, addrs []string) {
	ips := make([]net.Addr, len(addrs))
	for i, a := range addrs {
		ips[i] = &net.IPAddr{IP: net.ParseIP(a)}
	}
	i := &intf{addrs: ips}
	iputils.InterfaceByName = func(name string) (iputils.Addr, error) {
		if name != iname {
			return nil, errors.New("device not found")
		}
		return i, nil
	}
}

var ipVersionToEnum = map[int]*configpb.ProbeDef_IPVersion{
	4: configpb.ProbeDef_IPV4.Enum(),
	6: configpb.ProbeDef_IPV6.Enum(),
}

func TestGetSourceIPFromConfig(t *testing.T) {
	rows := []struct {
		name       string
		sourceIP   string
		sourceIntf string
		intf       string
		intfAddrs  []string
		ipVer      int
		want       string
		wantError  bool
	}{
		{
			name:     "Use IP",
			sourceIP: "1.1.1.1",
			want:     "1.1.1.1",
		},
		{
			name:      "Source IP doesn't match IP version",
			sourceIP:  "1.1.1.1",
			ipVer:     6,
			wantError: true,
		},
		{
			name:     "Use IPv6",
			sourceIP: "::1",
			ipVer:    6,
			want:     "::1",
		},
		{
			name:      "Invalid IP",
			sourceIP:  "12ab",
			wantError: true,
		},
		{
			name:       "Interface with no adders fails",
			sourceIntf: "eth1",
			intf:       "eth1",
			wantError:  true,
		},
		{
			name:       "Unknown interface fails",
			sourceIntf: "eth1",
			intf:       "eth0",
			wantError:  true,
		},
		{
			name:       "Uses first addr for interface",
			sourceIntf: "eth1",
			intf:       "eth1",
			intfAddrs:  []string{"1.1.1.1", "2.2.2.2"},
			want:       "1.1.1.1",
		},
		{
			name:       "Uses first IPv6 addr for interface",
			sourceIntf: "eth1",
			intf:       "eth1",
			intfAddrs:  []string{"1.1.1.1", "::1"},
			ipVer:      6,
			want:       "::1",
		},
	}

	for _, r := range rows {
		p := &configpb.ProbeDef{
			IpVersion: ipVersionToEnum[r.ipVer],
		}

		if r.sourceIP != "" {
			p.SourceIpConfig = &configpb.ProbeDef_SourceIp{SourceIp: r.sourceIP}
		} else if r.sourceIntf != "" {
			p.SourceIpConfig = &configpb.ProbeDef_SourceInterface{SourceInterface: r.sourceIntf}
			mockInterfaceByName(r.intf, r.intfAddrs)
		}

		source, err := getSourceIPFromConfig(p)

		if (err != nil) != r.wantError {
			t.Errorf("Row %q: getSourceIPFromConfig() gave error %q, want error is %v", r.name, err, r.wantError)
			continue
		}
		if r.wantError {
			continue
		}
		if source.String() != r.want {
			t.Errorf("Row %q: source= %q, want %q", r.name, source, r.want)
		}
	}
}

var testTargets = &targetspb.TargetsDef{
	Type: &targetspb.TargetsDef_HostNames{HostNames: "testHost"},
}

func TestIPVersionFromSourceIP(t *testing.T) {
	rows := []struct {
		name     string
		sourceIP string
		ipVer    int
	}{
		{
			name:  "No source IP",
			ipVer: 0,
		},
		{
			name:     "IPv4 from source IP",
			sourceIP: "1.1.1.1",
			ipVer:    4,
		},
		{
			name:     "IPv6 from source IP",
			sourceIP: "::1",
			ipVer:    6,
		},
	}

	for _, r := range rows {
		p := &configpb.ProbeDef{
			Targets: testTargets,
		}

		if r.sourceIP != "" {
			p.SourceIpConfig = &configpb.ProbeDef_SourceIp{SourceIp: r.sourceIP}
		}

		opts, err := BuildProbeOptions(p, nil, nil, nil)
		if err != nil {
			t.Errorf("got unexpected error: %v", err)
			continue
		}

		if opts.IPVersion != r.ipVer {
			t.Errorf("Unexpected IPVersion (test case: %s) want=%d, got=%d", r.name, r.ipVer, opts.IPVersion)
		}
	}
}

func TestStatsExportInterval(t *testing.T) {
	rows := []struct {
		name         string
		pType        *configpb.ProbeDef_Type
		interval     string
		timeout      string
		intervalMsec int32
		timeoutMsec  int32
		configured   int32
		want         int
		wantError    bool
	}{
		{
			name:         "Interval bigger than default",
			intervalMsec: 15,
			timeoutMsec:  10,
			want:         15,
		},
		{
			name:         "Timeout bigger than intervalMsec",
			intervalMsec: 10,
			timeoutMsec:  12,
			wantError:    true,
		},
		{
			name:         "Interval and timeout less than default",
			intervalMsec: 2,
			timeoutMsec:  1,
			want:         int(defaultStatsExtportIntv.Seconds()),
		},
		{
			name:         "UDP probe: default twice of timeout- I",
			intervalMsec: 10,
			timeoutMsec:  12,
			pType:        configpb.ProbeDef_UDP.Enum(),
			want:         24,
		},
		{
			name:         "UDP probe: default twice of timeout - II",
			intervalMsec: 5,
			timeoutMsec:  6,
			pType:        configpb.ProbeDef_UDP.Enum(),
			want:         12,
		},
		{
			name:         "Error: stats export intervalMsec smaller than intervalMsec",
			intervalMsec: 2,
			timeoutMsec:  1,
			configured:   1,
			wantError:    true,
		},
		{
			name:         "Both intervalMsec_msec and intervalMsec specified",
			interval:     "2s",
			intervalMsec: 2,
			configured:   10,
			wantError:    true,
		},
		{
			name:        "Both timeoutMsec and timeout specified",
			timeout:     "2s",
			timeoutMsec: 2,
			configured:  10,
			wantError:   true,
		},
		{
			name: "No intervalMsec or timeout values specified",
			want: 10,
		},
		{
			name:      "Invalid interval string specified",
			interval:  "2j",
			wantError: true,
		},
		{
			name:      "Invalid timeout string specified",
			timeout:   "2j",
			wantError: true,
		},
		{
			name:         "Configured value is good",
			intervalMsec: 2,
			timeoutMsec:  1,
			configured:   10,
			want:         10,
		},
	}

	for _, r := range rows {
		p := &configpb.ProbeDef{
			Targets:      testTargets,
			IntervalMsec: proto.Int32(r.intervalMsec * 1000),
			TimeoutMsec:  proto.Int32(r.timeoutMsec * 1000),
			Interval:     proto.String(r.interval),
			Timeout:      proto.String(r.timeout),
		}

		if r.pType != nil {
			p.Type = r.pType
		}

		if r.configured != 0 {
			p.StatsExportIntervalMsec = proto.Int32(r.configured * 1000)
		}

		opts, err := BuildProbeOptions(p, nil, nil, nil)
		if (err != nil) != r.wantError {
			t.Errorf("Row %q: BuildProbeOptions() gave error %q, want error is %v", r.name, err, r.wantError)
			continue
		}
		if r.wantError {
			continue
		}

		want := time.Duration(r.want) * time.Second
		if opts.StatsExportInterval != want {
			t.Errorf("Unexpected stats export interval (test case: %s): want=%s, got=%s", r.name, want, opts.StatsExportInterval)
		}
	}
}

func TestDefaultOptions(t *testing.T) {
	// Most of all, it verifies that DefaultOptions() doesn't generate panic.
	opts := DefaultOptions()
	if opts == nil {
		t.Errorf("Got nil default options")
	}
}

func TestNegativeTestSupport(t *testing.T) {
	supportedType := []configpb.ProbeDef_Type{
		configpb.ProbeDef_PING,
		configpb.ProbeDef_TCP,
	}
	unsupportedType := []configpb.ProbeDef_Type{
		configpb.ProbeDef_DNS,
		configpb.ProbeDef_EXTERNAL,
		configpb.ProbeDef_HTTP,
		configpb.ProbeDef_UDP,
		configpb.ProbeDef_UDP_LISTENER,
	}

	probeConf := func(ptype configpb.ProbeDef_Type) *configpb.ProbeDef {
		return &configpb.ProbeDef{
			Type:         ptype.Enum(),
			Targets:      testTargets,
			NegativeTest: proto.Bool(true),
		}
	}

	for _, ptype := range supportedType {
		t.Run(ptype.String(), func(t *testing.T) {
			_, err := BuildProbeOptions(probeConf(ptype), nil, nil, nil)
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			}
		})
	}

	for _, ptype := range unsupportedType {
		t.Run(ptype.String(), func(t *testing.T) {
			_, err := BuildProbeOptions(probeConf(ptype), nil, nil, nil)
			if err == nil {
				t.Errorf("Didn't get error for unsupported probe type: %v", ptype)
			}
		})
	}
}

func TestRecordMetrics(t *testing.T) {
	ep := endpoint.Endpoint{Name: "test_target"}
	opts := DefaultOptions()
	additionalLabel := &AdditionalLabel{
		Key: "test_additional_label",
		valueForTarget: map[string]string{
			ep.Key(): "test_value",
		},
	}

	tests := []struct {
		name    string
		noAlert bool
	}{
		{
			name: "default",
		},
		{
			name:    "no alert",
			noAlert: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := logger.New(logger.WithWriter(&buf))
			opts.AdditionalLabels = []*AdditionalLabel{additionalLabel}
			alertHandler, _ := alerting.NewAlertHandler(&alerting_configpb.AlertConf{}, "test-probe", l)
			opts.AlertHandlers = []*alerting.AlertHandler{alertHandler}

			dataChan := make(chan *metrics.EventMetrics, 3)
			inputEM := metrics.NewEventMetrics(time.Now()).
				AddMetric("total", metrics.NewInt(1)).
				AddMetric("success", metrics.NewInt(0))
			if tt.noAlert {
				inputEM.SetNotForAlerting()
			}
			opts.RecordMetrics(ep, inputEM, dataChan)
			em := <-dataChan

			// Verify that we got what we sent (there is no cloning)
			assert.Equal(t, em, inputEM)
			// Verify that additional label is set by RecordMetrics
			assert.Equal(t, "test_value", em.Label("test_additional_label"))

			em = metrics.NewEventMetrics(time.Now()).
				AddMetric("total", metrics.NewInt(2)).
				AddMetric("success", metrics.NewInt(0))
			if tt.noAlert {
				em.SetNotForAlerting()
			}
			opts.RecordMetrics(ep, em, dataChan)
			alertHandlerLog := buf.String()
			t.Log("buf:", alertHandlerLog)
			if tt.noAlert {
				assert.NotContains(t, alertHandlerLog, "ALERT (test-probe)")
			} else {
				assert.Contains(t, alertHandlerLog, "ALERT (test-probe)")
			}
		})
	}
}

func TestNilTargets(t *testing.T) {
	tests := []struct {
		cfg           *configpb.ProbeDef
		wantEndpoints []endpoint.Endpoint
		wantErr       bool
	}{
		{
			cfg: &configpb.ProbeDef{
				Type: configpb.ProbeDef_PING.Enum(),
				Name: proto.String("test-probe"),
			},
			wantErr: true,
		},
		{
			cfg: &configpb.ProbeDef{
				Type: configpb.ProbeDef_USER_DEFINED.Enum(),
				Name: proto.String("test-probe"),
			},
			wantEndpoints: []endpoint.Endpoint{{Name: ""}},
		},
		{
			cfg: &configpb.ProbeDef{
				Type: configpb.ProbeDef_EXTERNAL.Enum(),
				Name: proto.String("test-probe"),
			},
			wantEndpoints: []endpoint.Endpoint{{Name: ""}},
		},
		{
			cfg: &configpb.ProbeDef{
				Type: configpb.ProbeDef_EXTENSION.Enum(),
				Name: proto.String("test-probe"),
			},
			wantEndpoints: []endpoint.Endpoint{{Name: ""}},
		},
		{
			cfg: &configpb.ProbeDef{
				Type: configpb.ProbeDef_EXTERNAL.Enum(),
				Name: proto.String("test-probe"),
				Targets: &targetspb.TargetsDef{
					Type: &targetspb.TargetsDef_HostNames{HostNames: "testHost"},
				},
			},
			wantEndpoints: []endpoint.Endpoint{{Name: "testHost"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.cfg.String(), func(t *testing.T) {
			got, err := BuildProbeOptions(tt.cfg, nil, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildProbeOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			assert.Equal(t, tt.wantEndpoints, got.Targets.ListEndpoints())
		})
	}
}

func TestOptionsLogMetrics(t *testing.T) {
	var called int
	overrideFn := func(em *metrics.EventMetrics) {
		called++
	}

	tests := []struct {
		name              string
		pConf             *configpb.ProbeDef
		wantNilLogMetrics bool
		wantInc           int
	}{
		{
			name: "default",
			pConf: &configpb.ProbeDef{
				Targets: testTargets,
			},
			wantNilLogMetrics: true,
			wantInc:           0,
		},
		{
			name: "inc",
			pConf: &configpb.ProbeDef{
				Targets:      testTargets,
				DebugOptions: &configpb.DebugOptions{LogMetrics: proto.Bool(true)},
			},
			wantNilLogMetrics: false,
			wantInc:           1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := BuildProbeOptions(tt.pConf, nil, nil, nil)
			if err != nil {
				t.Errorf("Unexpected BuildProbeOptions() error = %v", err)
				return
			}

			assert.Equal(t, tt.wantNilLogMetrics, opts.logMetricsOverride == nil)

			// Try calling LogMetrics()
			if opts.logMetricsOverride != nil {
				opts.logMetricsOverride = overrideFn
			}
			oldCalled := called
			opts.LogMetrics(metrics.NewEventMetrics(time.Now()))
			assert.Equal(t, tt.wantInc, called-oldCalled)
		})
	}
}

func TestOptions_StatsExportFrequency(t *testing.T) {
	tests := []struct {
		name string
		opts *Options
		want int64
	}{
		{
			name: "default",
			opts: &Options{
				Interval:            2 * time.Second,
				StatsExportInterval: 10 * time.Second,
			},
			want: 5,
		},
		{
			name: "default",
			opts: &Options{
				Interval:            20 * time.Second,
				StatsExportInterval: 10 * time.Second,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.StatsExportFrequency(); got != tt.want {
				t.Errorf("Options.StatsExportFrequency() = %v, want %v", got, tt.want)
			}
		})
	}
}
